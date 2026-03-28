package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func main() {
	destServerURL, err := url.Parse("http://localhost:9090")
	if err != nil {
		log.Fatal(err)
	}
	pr := newProxyDest(destServerURL)

	http.ListenAndServe(":3000", pr)
}

func newProxyDest(url *url.URL) *httputil.ReverseProxy {
	rProxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(url)
			pr.SetXForwarded()
		},
	}

	return rProxy
}
