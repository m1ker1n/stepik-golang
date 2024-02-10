package main

import (
	"fmt"
	"github.com/xiegeo/coloredgoroutine"
	"github.com/xiegeo/coloredgoroutine/goid"
	"io"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var c io.Writer = coloredgoroutine.Colors(os.Stdout)

func ExecutePipeline(hashSignJobs ...job) {
	fmt.Fprintf(c, "procs=%d\n", runtime.GOMAXPROCS(0))
	if len(hashSignJobs) == 0 {
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(hashSignJobs))

	chans := make([]chan interface{}, len(hashSignJobs)+1)
	for i := 1; i < len(chans)-1; i++ {
		chans[i] = make(chan interface{}, MaxInputDataLen)
	}

	for i := 0; i < len(chans)-1; i++ {
		go func(i int, in, out chan interface{}) {
			hashSignJobs[i](in, out)
			wg.Done()
		}(i, chans[i], chans[i+1])
	}

	wg.Wait()
}

func SingleHash(in chan interface{}, out chan interface{}) {
	var wg sync.WaitGroup
	var md5Mu sync.Mutex

	for data := range in {
		intData, ok := (data).(int)
		if !ok {
			return
		}
		strData := strconv.Itoa(intData)
		fmt.Fprintf(c, "[%d] %s SingleHash data %s\n", goid.ID(), strData, strData)

		wg.Add(1)

		md5 := make(chan string)
		go func() {
			md5Mu.Lock()
			md5 <- DataSignerMd5(strData)
			md5Mu.Unlock()
		}()

		crc32 := make(chan string)
		go func() {
			crc32 <- DataSignerCrc32(strData)
		}()

		crc32Md5 := make(chan string)
		go func() {
			md5Res := <-md5
			fmt.Fprintf(c, "[%d] %s SingleHash md5(data) %s\n", goid.ID(), strData, md5Res)
			crc32Md5 <- DataSignerCrc32(md5Res)
		}()

		crc32Res := <-crc32
		fmt.Fprintf(c, "[%d] %s SingleHash crc32(data) %s\n", goid.ID(), strData, crc32Res)
		crc32Md5Res := <-crc32Md5
		fmt.Fprintf(c, "[%d] %s SingleHash crc32(md5(data)) %s\n", goid.ID(), strData, crc32Md5Res)
		out <- crc32Res + "~" + crc32Md5Res
		wg.Done()
	}
	wg.Wait()
	close(out)
}

func MultiHash(in chan interface{}, out chan interface{}) {
	var wg sync.WaitGroup
	for data := range in {
		strData, ok := (data).(string)
		if !ok {
			return
		}

		wg.Add(1)

		go func() {
			hashes := make([]string, 6)
			var wgHashes sync.WaitGroup
			wgHashes.Add(6)
			for i := 0; i < 6; i++ {
				go func(i int) {
					hashes[i] = DataSignerCrc32(strconv.Itoa(i) + strData)
					fmt.Fprintf(c, "[%d] %s MultiHash crc32(th+step1) %d %s\n", goid.ID(), strData, i, hashes[i])
					wgHashes.Done()
				}(i)
			}

			wgHashes.Wait()
			res := strings.Join(hashes, "")
			fmt.Fprintf(c, "[%d] %s MultiHash result %s\n", goid.ID(), strData, res)
			out <- res
			wg.Done()
		}()
	}
	wg.Wait()
	close(out)
}

func CombineResults(in chan interface{}, out chan interface{}) {
	var data []string
	for dataEl := range in {
		strDataEl, ok := (dataEl).(string)
		if !ok {
			return
		}
		data = append(data, strDataEl)
	}
	sort.Strings(data)
	res := strings.Join(data, "_")
	fmt.Fprintf(c, "[%d] CombineResults %s\n", goid.ID(), res)
	out <- res
	close(out)
}
