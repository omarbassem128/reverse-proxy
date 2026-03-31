package main

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

type RateLimiter struct {
	clients map[string]int
	mu      sync.Mutex
}

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

	rl := NewRateLimiter()
	// httputil.ReverseProxy implements next.serveHTTP(), which is defined in http.Handler.
	// This is why pr is accepted as an argument for rateLimiterMiddleware().
	protectedProxy := rl.rateLimiterMiddleware(pr)

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

func (rl *RateLimiter) rateLimiterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "Internal service error", http.StatusInternalServerError)
			return
		}

		rl.mu.Lock()
		requests := rl.clients[host]
		rl.clients[host]++
		// mutex is being unlocked here (and not using defer) because next.ServeHTTP() is a blocking call.
		// This means that if the call takes a while, it will block other users from having their IPs checked via the middleware, which defeats the purpose.
		rl.mu.Unlock()

		if requests < 5 {
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
		}

	})
}

func NewRateLimiter() *RateLimiter {
	//mutex is automatically ready to use.
	rl := &RateLimiter{
		clients: make(map[string]int),
	}

	go func() {
		ticker := time.NewTicker(1 * time.Minute)

		for range ticker.C {
			rl.mu.Lock()
			clear(rl.clients)
			rl.mu.Unlock()
		}

	}()

	return rl
}
