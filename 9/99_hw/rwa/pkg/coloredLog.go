package pkg

import (
	"fmt"
	"log"
)

var (
	Info = LogTeal
	Warn = LogYellow
	Fata = LogRed
)

var (
	LogBlack   = Color("\033[1;30m%s\033[0m")
	LogRed     = Color("\033[1;31m%s\033[0m")
	LogGreen   = Color("\033[1;32m%s\033[0m")
	LogYellow  = Color("\033[1;33m%s\033[0m")
	LogPurple  = Color("\033[1;34m%s\033[0m")
	LogMagenta = Color("\033[1;35m%s\033[0m")
	LogTeal    = Color("\033[1;36m%s\033[0m")
	LogWhite   = Color("\033[1;37m%s\033[0m")
)

func Color(colorString string) func(...interface{}) {
	sprint := func(args ...interface{}) {
		log.Printf(colorString, fmt.Sprint(args...))
	}
	return sprint
}
