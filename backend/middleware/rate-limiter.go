package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	clients map[string]int
	mu      sync.Mutex
}

func (rl *RateLimiter) RateLimiterMiddleware(next http.Handler) http.Handler {
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
