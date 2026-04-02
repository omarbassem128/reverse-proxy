package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type ClientState struct {
	tokens      int
	lastVisited time.Time
}

type TokenBuckerRl struct {
	mu      sync.Mutex
	clients map[string]*ClientState
}

func (tbrl *TokenBuckerRl) TokenBucketRateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tbrl.mu.Lock()

		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "internal service error", http.StatusInternalServerError)
			tbrl.mu.Unlock()
			return
		}

		currentClient := tbrl.clients[host]
		if currentClient == nil {
			tbrl.clients[host] = &ClientState{tokens: 4, lastVisited: time.Now()}
			tbrl.mu.Unlock()
			next.ServeHTTP(w, r)
			return
		} else {
			lastVisited := currentClient.lastVisited
			difference := time.Since(lastVisited)
			currentClient.lastVisited = time.Now()
			differenceInt := int(difference.Seconds())
			addition := differenceInt + currentClient.tokens
			newTokens := min(addition, 5)
			currentClient.tokens = newTokens
		}
		
		if currentClient.tokens > 0 {
			currentClient.tokens--
			tbrl.mu.Unlock()
			next.ServeHTTP(w, r)
		} else {
			tbrl.mu.Unlock()
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
		}

	})
}

func NewTokenBucketRateLimiter() *TokenBuckerRl {
	rl := &TokenBuckerRl{clients: make(map[string]*ClientState)}
	return rl
}
