package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/reverse-proxy/backend/middleware"
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

	rl := middleware.NewTokenBucketRateLimiter()
	// httputil.ReverseProxy implements serveHTTP(), which is defined in http.Handler.
	// This is why pr is accepted as an argument for FixedWinowRateLimiter().
	protectedProxy := rl.TokenBucketRateLimiter(pr)

	

	server := http.Server{
		Addr: ":3000",
		Handler: protectedProxy,
		ReadTimeout: time.Second * 5,
		WriteTimeout: time.Second * 5,
		IdleTimeout: time.Second * 5,
	}

	go server.ListenAndServe()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	stopServer := <-quit
	fmt.Printf("Shutting down server... Reason: %v\n", stopServer)
	ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
	defer cancel()
	server.Shutdown(ctx)
	

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
