package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"monolith/app/config"
)

// visitor tracks a client's rate limiter and last seen time.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var visitors = struct {
	sync.Mutex
	m map[string]*visitor
}{m: make(map[string]*visitor)}

func getVisitor(ip string) *rate.Limiter {
	visitors.Lock()
	defer visitors.Unlock()

	v, exists := visitors.m[ip]
	if !exists {
		v = &visitor{
			limiter:  rate.NewLimiter(rate.Every(time.Minute/time.Duration(config.RATE_LIMIT_REQUESTS_PER_MINUTE)), config.RATE_LIMIT_REQUESTS_PER_MINUTE),
			lastSeen: time.Now(),
		}
		visitors.m[ip] = v
	}
	v.lastSeen = time.Now()
	return v.limiter
}

// cleanupVisitors runs periodically and removes entries that haven't been seen for a while.
func cleanupVisitors() {
	for {
		time.Sleep(time.Minute)
		visitors.Lock()
		for ip, v := range visitors.m {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(visitors.m, ip)
			}
		}
		visitors.Unlock()
	}
}

func init() {
	go cleanupVisitors()
}

// RateLimitMiddleware limits the number of requests an IP can make per minute.
func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		limiter := getVisitor(ip)
		if !limiter.Allow() {
			slog.Warn("rate limit exceeded", "ip", ip, "path", r.URL.Path)
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
