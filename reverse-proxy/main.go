package main

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
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
	godotenv.Load()
	secretKey := os.Getenv("JWT_SECRET")

	destServer1URL, err1 := url.Parse("http://localhost:9090")
	if err1 != nil {
		log.Fatal(err1)
	}

	destServer2URL, err2 := url.Parse("http://localhost:9091")
	if err2 != nil {
		log.Fatal(err2)
	}

	urlsSlice := []*url.URL{destServer1URL, destServer2URL}
	reverseProxy := newProxyDest(urlsSlice)

	backgroundCtx, backgroundCancel := context.WithCancel(context.Background())

	midHandler := wrapMiddlewares(reverseProxy, backgroundCtx, secretKey)

	server := http.Server{
		Addr:         ":3000",
		Handler:      midHandler,
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
		IdleTimeout:  time.Second * 5,
	}

	go func() {
		err := server.ListenAndServeTLS("cert.pem", "key.pem")
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("crash: %s", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	stopServer := <-quit
	backgroundCancel()
	fmt.Printf("Shutting down server... Reason: %v\n", stopServer)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second*10)
	defer shutdownCancel()
	server.Shutdown(shutdownCtx)

}

func newProxyDest(urls []*url.URL) *httputil.ReverseProxy {
	var mu sync.Mutex
	counter := 0
	rProxy := &httputil.ReverseProxy{
		Rewrite: func(proxyRequest *httputil.ProxyRequest) {
			mu.Lock()
			defer mu.Unlock()
			counter = counter % len(urls)
			proxyRequest.SetURL(urls[counter])
			proxyRequest.SetXForwarded()
			counter++
		},
		Transport: httpTransport(),
	}

	return rProxy
}

func wrapMiddlewares(reverseProxy *httputil.ReverseProxy, ctx context.Context, secretKey string) http.Handler {
	middleware.SetTrustedProxies([]string{"127.0.0.1", "::1"})
	rl := middleware.NewTokenBucketRateLimiter(ctx)

	midHandler := middleware.NewAuthMiddleware(secretKey)(reverseProxy)
	midHandler = rl.TokenBucketRateLimiter(midHandler)
	Logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	midHandler = middleware.NewLogRequest(Logger)(midHandler)
	midHandler = middleware.GenerateLogID(midHandler)
	return midHandler

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
