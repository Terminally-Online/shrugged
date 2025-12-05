package api

import (
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.Mutex
	requests map[string]*clientRequests
	limit    int
	window   time.Duration
}

type clientRequests struct {
	count     int
	windowEnd time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*clientRequests),
		limit:    limit,
		window:   window,
	}

	go rl.cleanup()

	return rl
}

func (rl *RateLimiter) Allow(clientIP string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	client, exists := rl.requests[clientIP]
	if !exists || now.After(client.windowEnd) {
		rl.requests[clientIP] = &clientRequests{
			count:     1,
			windowEnd: now.Add(rl.window),
		}
		return true
	}

	if client.count >= rl.limit {
		return false
	}

	client.count++
	return true
}

func (rl *RateLimiter) Remaining(clientIP string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	client, exists := rl.requests[clientIP]
	if !exists || time.Now().After(client.windowEnd) {
		return rl.limit
	}

	remaining := rl.limit - client.count
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, client := range rl.requests {
			if now.After(client.windowEnd) {
				delete(rl.requests, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)

		if !rl.Allow(clientIP) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		w.Header().Set("X-RateLimit-Remaining", string(rune(rl.Remaining(clientIP)+'0')))

		next.ServeHTTP(w, r)
	})
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}
