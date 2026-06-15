package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RateLimiter struct {
	ips map[string]*rate.Limiter
	mu  sync.RWMutex
	r   rate.Limit
	b   int
}

// NewRateLimiter creates a rate limiter that tracks individual IPs
func NewRateLimiter(r float64, b float64) *RateLimiter {
	return &RateLimiter{
		ips: make(map[string]*rate.Limiter),
		r:   rate.Limit(r),
		b:   int(b),
	}
}

// GetLimiter returns the rate limiter for the provided IP address
func (i *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.RLock()
	limiter, exists := i.ips[ip]
	i.mu.RUnlock()

	if !exists {
		i.mu.Lock()
		limiter, exists = i.ips[ip]
		if !exists {
			limiter = rate.NewLimiter(i.r, i.b)
			i.ips[ip] = limiter
		}
		i.mu.Unlock()
	}

	return limiter
}

func RateLimit(limiter *RateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		l := limiter.GetLimiter(ip)
		if !l.Allow() {
			slog.WarnContext(r.Context(), "Rate limit exceeded", "path", r.URL.Path, "ip", ip)
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	}
}

// Logging middleware
func Logging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.InfoContext(r.Context(), "HTTP Request handled",
			"method", r.Method,
			"path", r.URL.Path,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	}
}
