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

func (tokenBucketRlStruct *TokenBuckerRl) TokenBucketRateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "internal service error", http.StatusInternalServerError)
			return
		}
		allow := false
		tokenBucketRlStruct.mu.Lock()
		now := time.Now()

		currentClient := tokenBucketRlStruct.clients[host]
		if currentClient == nil {
			tokenBucketRlStruct.clients[host] = &ClientState{tokens: 5, lastVisited: now}
			currentClient = tokenBucketRlStruct.clients[host]
		} else {
			lastVisited := currentClient.lastVisited
			difference := time.Since(lastVisited)
			currentClient.lastVisited = now
			differenceInt := int(difference.Seconds())
			addition := differenceInt + currentClient.tokens
			newTokens := min(addition, 5)
			currentClient.tokens = newTokens
		}

		if currentClient.tokens > 0 {
			allow = true
			currentClient.tokens--
		}

		tokenBucketRlStruct.mu.Unlock()

		if allow {
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
		}

	})
}

func NewTokenBucketRateLimiter() *TokenBuckerRl {
	rl := &TokenBuckerRl{clients: make(map[string]*ClientState)}
	return rl
}
