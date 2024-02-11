package main

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"testing"
)

// запускаем перед основными функциями по разу чтобы файл остался в памяти в файловом кеше
// ioutil.Discard - это ioutil.Writer который никуда не пишет
func init() {
	SlowSearch(ioutil.Discard)
	FastSearch(ioutil.Discard)
}

// -----
// go test -v

func TestSearch(t *testing.T) {
	slowOut := new(bytes.Buffer)
	SlowSearch(slowOut)
	slowResult := slowOut.String()

	fastOut := new(bytes.Buffer)
	FastSearch(fastOut)
	fastResult := fastOut.String()

	if slowResult != fastResult {
		t.Errorf("results not match\nGot:\n%v\nExpected:\n%v", fastResult, slowResult)
	}
}

// -----
// go test -bench . -benchmem

func BenchmarkSlow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SlowSearch(ioutil.Discard)
	}
}

func BenchmarkFast(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FastSearch(ioutil.Discard)
	}
}

func BenchmarkRegexpReplaceString(b *testing.B) {
	r := regexp.MustCompile("@")
	for i := 0; i < b.N; i++ {
		r.ReplaceAllString("someEmail@dot.com", " [at] ")
	}
}

func BenchmarkStringsReplaceString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		strings.ReplaceAll("someEmail@dot.com", "@", " [at] ")
	}
}

func BenchmarkRegexpMatch(b *testing.B) {
	browser := "32Android4"
	for i := 0; i < b.N; i++ {
		regexp.MatchString("Android", browser)
	}
}

func BenchmarkStringContains(b *testing.B) {
	browser := "32Android4"
	for i := 0; i < b.N; i++ {
		strings.Contains(browser, "Android")
	}
}

func BenchmarkReadAll(b *testing.B) {
	for i := 0; i < b.N; i++ {
		file, err := os.Open(filePath)
		if err != nil {
			panic(err)
		}
		fileContents, err := ioutil.ReadAll(file)
		if err != nil {
			panic(err)
		}

		lines := strings.Split(string(fileContents), "\n")
		for range lines {
		}
	}
}

func BenchmarkScanner(b *testing.B) {
	for i := 0; i < b.N; i++ {
		file, err := os.Open(filePath)
		if err != nil {
			panic(err)
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
		}
		if err := scanner.Err(); err != nil {
			panic(err)
		}
	}
}
