package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

func main() {
	destServer1URL, err1 := url.Parse("http://localhost:9090")
	if err1 != nil {
		log.Fatal(err1)
	}
	destServer2URL, err2 := url.Parse("http://localhost:9091")
	if err2 != nil {
		log.Fatal(err2)
	}

	urlsSlice := []*url.URL{destServer1URL, destServer2URL}
	
	pr := newProxyDest(urlsSlice)

	http.ListenAndServe(":3000", pr)
}

func newProxyDest(urls []*url.URL) *httputil.ReverseProxy {
	var mu sync.Mutex
	counter := 0
	rProxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			mu.Lock()
			counter = counter % len(urls)
			pr.SetURL(urls[counter])
			pr.SetXForwarded()
			counter++ 
			mu.Unlock()
		},
	}

	return rProxy
}
