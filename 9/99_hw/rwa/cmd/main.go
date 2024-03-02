package main

import (
	"fmt"
	"hw9/internal/app"
	"net/http"
)

// сюда код писать не надо

func main() {
	addr := ":8080"
	h := app.GetApp()
	fmt.Println("start server at", addr)
	http.ListenAndServe(addr, h)
}
