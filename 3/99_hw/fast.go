package main

import (
	"fmt"
	"github.com/mailru/easyjson"
	"hw3/user"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

func FastSearch(out io.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	r := regexp.MustCompile("@")
	seenBrowsers := []string{}
	uniqueBrowsers := 0
	foundUsers := ""

	lines := strings.Split(string(fileContents), "\n")

	users := make([]user.User, 0)
	for _, line := range lines {
		var u user.User
		// fmt.Printf("%v %v\n", err, line)
		err := easyjson.Unmarshal([]byte(line), &u)
		if err != nil {
			panic(err)
		}
		users = append(users, u)
	}

	for i, u := range users {

		isAndroid := false
		isMSIE := false

		for _, browser := range u.Browsers {
			if strings.Contains(browser, "Android") {
				isAndroid = true
				notSeenBefore := true
				for _, item := range seenBrowsers {
					if item == browser {
						notSeenBefore = false
					}
				}
				if notSeenBefore {
					// log.Printf("FAST New browser: %s, first seen: %s", browser, u.Name)
					seenBrowsers = append(seenBrowsers, browser)
					uniqueBrowsers++
				}
			}
		}

		for _, browser := range u.Browsers {
			if strings.Contains(browser, "MSIE") {
				isMSIE = true
				notSeenBefore := true
				for _, item := range seenBrowsers {
					if item == browser {
						notSeenBefore = false
					}
				}
				if notSeenBefore {
					// log.Printf("FAST New browser: %s, first seen: %s", browser, u.Name)
					seenBrowsers = append(seenBrowsers, browser)
					uniqueBrowsers++
				}
			}
		}

		if !(isAndroid && isMSIE) {
			continue
		}

		// log.Println("Android and MSIE user:", u.Name, u.Email)
		email := r.ReplaceAllString(u.Email, " [at] ")
		foundUsers += fmt.Sprintf("[%d] %s <%s>\n", i, u.Name, email)
	}

	fmt.Fprintln(out, "found users:\n"+foundUsers)
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))
}
