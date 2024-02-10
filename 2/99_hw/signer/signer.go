package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func ExecutePipeline(hashSignJobs ...job) {
	if len(hashSignJobs) == 0 {
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(hashSignJobs))

	chans := make([]chan interface{}, len(hashSignJobs)+1)
	for i := 1; i < len(chans); i++ {
		chans[i] = make(chan interface{}, MaxInputDataLen)
	}

	for i := 0; i < len(chans)-1; i++ {
		go func(i int, in, out chan interface{}) {
			hashSignJobs[i](in, out)
			close(out)
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
		fmt.Printf("%s SingleHash data %s\n", strData, strData)

		wg.Add(1)
		go func() {
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
				fmt.Printf("%s SingleHash md5(data) %s\n", strData, md5Res)
				crc32Md5 <- DataSignerCrc32(md5Res)
			}()

			crc32Res := <-crc32
			fmt.Printf("%s SingleHash crc32(data) %s\n", strData, crc32Res)
			crc32Md5Res := <-crc32Md5
			fmt.Printf("%s SingleHash crc32(md5(data)) %s\n", strData, crc32Md5Res)
			out <- crc32Res + "~" + crc32Md5Res

			wg.Done()
		}()
	}
	wg.Wait()
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
					fmt.Printf("%s MultiHash crc32(th+step1) %d %s\n", strData, i, hashes[i])
					wgHashes.Done()
				}(i)
			}

			wgHashes.Wait()
			res := strings.Join(hashes, "")
			fmt.Printf("%s MultiHash result %s\n", strData, res)
			out <- res
			wg.Done()
		}()
	}
	wg.Wait()
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
	fmt.Printf("CombineResults %s\n", res)
	out <- res
}
