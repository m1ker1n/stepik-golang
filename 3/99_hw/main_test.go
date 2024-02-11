package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
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

const (
	pattern = "Android"
	browser = "123Android5363"
)

func BenchmarkRegexpMatchString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = regexp.MatchString(pattern, browser)
	}
}

func BenchmarkPrecompiledRegexpMatchString(b *testing.B) {
	r := regexp.MustCompile(pattern)
	for i := 0; i < b.N; i++ {
		r.MatchString(browser)
	}
}

func BenchmarkStringsContains(b *testing.B) {
	for i := 0; i < b.N; i++ {
		strings.Contains(browser, pattern)
	}
}

const unmarshalData = `{"browsers":["Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.32 (KHTML, like Gecko) Chromium/25.0.1349.2 Chrome/25.0.1349.2 Safari/537.32 Epiphany/3.8.2","Wget/1.9 cvs-stable (Red Hat modified)","Mozilla/3.0 (compatible; NetPositive/2.1.1; BeOS)","Mozilla/5.0 (X11; Linux 3.8-6.dmz.1-liquorix-686) KHTML/4.8.4 (like Gecko) Konqueror/4.8"],"company":"Meevee","country":"Fiji","email":"CarolynReyes@Mydeo.name","job":"Librarian","name":"Steven Burton","phone":"420-22-74"}`

func BenchmarkUnmarshalToMap(b *testing.B) {
	for i := 0; i < b.N; i++ {
		user := make(map[string]interface{})
		_ = json.Unmarshal([]byte(unmarshalData), &user)
	}
}

func BenchmarkUnmarshalToStruct(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var user User
		_ = json.Unmarshal([]byte(unmarshalData), &user)
	}
}
