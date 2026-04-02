package middleware

import (
	"fmt"
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
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "internal service error", http.StatusInternalServerError)
			return
		}

		currentClient := tbrl.clients[host]
		if currentClient == nil {
			tbrl.clients[host] = &ClientState{tokens: 4, lastVisited: time.Now()}
			next.ServeHTTP(w, r)
			return
		}

		lastVisited := currentClient.lastVisited
		difference := time.Since(lastVisited)
		currentClient.lastVisited = time.Now()
		differenceInt := int(difference.Seconds())
		addition := differenceInt + currentClient.tokens
		newTokens := min(addition, 5)
		currentClient.tokens = newTokens

		if currentClient.tokens > 0 {
			next.ServeHTTP(w, r)
			currentClient.tokens--
			fmt.Printf("remaining tokens: %d\n", currentClient.tokens)
		} else {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
		}

	})
}

func NewTokenBucketRateLimiter() *TokenBuckerRl {
	rl := &TokenBuckerRl{clients: make(map[string]*ClientState)}
	return rl
}
