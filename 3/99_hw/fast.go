package main

import (
	"bufio"
	"fmt"
	"github.com/mailru/easyjson"
	"hw3/user"
	"io"
	"os"
	"strings"
)

func FastSearch(out io.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(file)

	seenBrowsers := map[string]bool{}
	var foundUsersBuilder strings.Builder

	for i := 0; scanner.Scan(); i++ {
		var u user.User
		// fmt.Printf("%v %v\n", err, line)
		err := easyjson.Unmarshal(scanner.Bytes(), &u)
		if err != nil {
			panic(err)
		}

		isAndroid := false
		isMSIE := false

		for _, browser := range u.Browsers {
			if strings.Contains(browser, "Android") {
				isAndroid = true
				if _, seenBefore := seenBrowsers[browser]; !seenBefore {
					// log.Printf("FAST New browser: %s, first seen: %s", browser, u.Name)
					seenBrowsers[browser] = true
				}
			}

			if strings.Contains(browser, "MSIE") {
				isMSIE = true
				if _, seenBefore := seenBrowsers[browser]; !seenBefore {
					// log.Printf("FAST New browser: %s, first seen: %s", browser, u.Name)
					seenBrowsers[browser] = true
				}
			}
		}

		if !(isAndroid && isMSIE) {
			continue
		}

		// log.Println("Android and MSIE user:", u.Name, u.Email)
		email := strings.ReplaceAll(u.Email, "@", " [at] ")
		foundUsersBuilder.WriteString(fmt.Sprintf("[%d] %s <%s>\n", i, u.Name, email))
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	fmt.Fprintln(out, "found users:\n"+foundUsersBuilder.String())
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))
}
