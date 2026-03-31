package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", serverHandler)
	http.ListenAndServe(":9091", nil)
}

func serverHandler(rw http.ResponseWriter, request *http.Request) {
	clientIP := request.Header.Get("X-Forwarded-For")
	fmt.Fprintf(rw, "This is 9091. Your real IP is: %s\n", clientIP)
}
