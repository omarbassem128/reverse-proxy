package main

import (
	"github.com/reverse-proxy/backend/middleware"
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

	rl := middleware.NewRateLimiter()
	// httputil.ReverseProxy implements next.serveHTTP(), which is defined in http.Handler.
	// This is why pr is accepted as an argument for rateLimiterMiddleware().
	protectedProxy := rl.RateLimiterMiddleware(pr)

	http.ListenAndServe(":3000", protectedProxy)
}

func newProxyDest(urls []*url.URL) *httputil.ReverseProxy {
	var mu sync.Mutex
	counter := 0
	rProxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			mu.Lock()
			defer mu.Unlock()
			counter = counter % len(urls)
			pr.SetURL(urls[counter])
			pr.SetXForwarded()
			counter++
		},
	}

	return rProxy
}
