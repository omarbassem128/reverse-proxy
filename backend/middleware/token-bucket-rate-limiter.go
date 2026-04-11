package middleware

import (
	"context"
	"fmt"
	"hash/fnv"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ClientState struct {
	tokens      int
	lastVisited time.Time
}

type TokenBucketRl struct {
	shards [32]*Shard
}

// trustedProxies holds IPs of proxies whose X-Forwarded-For headers we accept.
var trustedProxies = make(map[string]struct{})

func SetTrustedProxies(ips []string) {
	m := make(map[string]struct{}, len(ips))
	for _, ip := range ips {
		m[ip] = struct{}{}
	}
	trustedProxies = m
}

type Shard struct {
	mu      sync.Mutex
	clients map[string]*ClientState
}

func (tokenBucketRlStruct *TokenBucketRl) TokenBucketRateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := clientIP(r)
		if host == "" {
			http.Error(w, "internal service error", http.StatusInternalServerError)
			return
		}

		shardIndex, shardErr := getShard(host)
		if shardErr != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		currShard := tokenBucketRlStruct.shards[shardIndex]

		allow := false
		currShard.mu.Lock()
		now := time.Now()

		currentClient := currShard.clients[host]
		if currentClient == nil {
			currShard.clients[host] = &ClientState{tokens: 5, lastVisited: now}
			currentClient = currShard.clients[host]
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

		currShard.mu.Unlock()

		if allow {
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
		}

	})
}

func NewTokenBucketRateLimiter(ctx context.Context) *TokenBucketRl {
	var shards [32]*Shard
	for i := 0; i < 32; i++ {
		shards[i] = &Shard{clients: make(map[string]*ClientState)}
	}

	rl := &TokenBucketRl{shards: shards}
	go func() {
		ticker := time.NewTicker(time.Minute * 2)
		for {
			select {
			case <-ticker.C:
				cleanupMap(rl)

			case <-ctx.Done():
				return
			}
		}
	}()
	return rl
}

func cleanupMap(tokenBucketRlStruct *TokenBucketRl) {
	for _, val := range tokenBucketRlStruct.shards {
		val.mu.Lock()

		for key2, val2 := range val.clients {

			if time.Since(val2.lastVisited) > time.Minute*5 {
				fmt.Printf("Deleting client: %s, with lastVisited time: %s\n", key2, val2.lastVisited.String())
				delete(val.clients, key2)
			}
		}
		val.mu.Unlock()
	}
}

func getShard(clientIP string) (int, error) {
	var b []byte = []byte(clientIP)
	hasher := fnv.New32()
	_, err := hasher.Write(b)
	if err != nil {
		return 0, err
	}
	hashNumber := hasher.Sum32()
	shardIndex := int(hashNumber % 32)

	return shardIndex, nil
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return ""
	}
	if _, ok := trustedProxies[host]; ok {
		if h := r.Header.Get("X-Forwarded-For"); h != "" {
			parts := strings.Split(h, ",")
			if len(parts) > 0 {
				return strings.TrimSpace(parts[len(parts)-1])
			}
		}
	}
	return host
}
