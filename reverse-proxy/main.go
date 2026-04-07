package main

import (
	"context"
	"fmt"
	"github.com/reverse-proxy/backend/middleware"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
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
	// This is why pr is accepted as an argument for FixedWinowRateLimiter() and TokenBucketRateLimiter().
	midHandler := rl.TokenBucketRateLimiter(pr)
	Logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	midHandler = middleware.NewLogRequest(Logger)(midHandler)
	midHandler = middleware.GenerateLogID(midHandler)

	server := http.Server{
		Addr:         ":3000",
		Handler:      midHandler,
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
		IdleTimeout:  time.Second * 5,
	}

	go server.ListenAndServe()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	stopServer := <-quit
	fmt.Printf("Shutting down server... Reason: %v\n", stopServer)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
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
		Transport: httpTransport(),
	}

	return rProxy
}

func httpTransport() *http.Transport {
	return &http.Transport{
		ResponseHeaderTimeout: time.Second * 3,
		TLSHandshakeTimeout:   time.Second * 2,
		MaxIdleConns:          5,
		MaxIdleConnsPerHost:   5,
		IdleConnTimeout:       time.Second * 4,
	}
}
