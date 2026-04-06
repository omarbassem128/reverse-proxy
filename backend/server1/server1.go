package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/", serverHandler)
	http.ListenAndServe(":9090", nil)
}

func serverHandler(rw http.ResponseWriter, request *http.Request) {
	clientIP := request.Header.Get("X-Forwarded-For")
	time.Sleep(time.Second*10)
	fmt.Fprintf(rw, "This is 9090. Your real IP is: %s\n", clientIP)
}
