package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimit struct {
	client *redis.Client
	prefix string
	limit  int64
	window time.Duration
}

func NewRateLimit(client *redis.Client, prefix string, limit int64, window time.Duration) *RateLimit {
	return &RateLimit{client: client, prefix: prefix, limit: limit, window: window}
}

func (l *RateLimit) Allow(ctx context.Context, key string) (bool, error) {
	if l.client == nil {
		return true, nil
	}
	redisKey := fmt.Sprintf("%s:%s", l.prefix, key)
	pipe := l.client.TxPipeline()
	count := pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, l.window)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, fmt.Errorf("rate limit Redis transaction: %w", err)
	}
	return count.Val() <= l.limit, nil
}

func (l *RateLimit) Middleware(key func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowed, err := l.Allow(r.Context(), key(r))
			if err != nil {
				WriteProblem(w, r, NewProblem(http.StatusServiceUnavailable, "rate_limiter_unavailable", "The rate limiter is unavailable."))
				return
			}
			if !allowed {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(l.window.Seconds())))
				WriteProblem(w, r, NewProblem(http.StatusTooManyRequests, "rate_limit_exceeded", "Too many requests. Try again later."))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func ClientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			for i := 0; i < len(forwarded); i++ {
				if forwarded[i] == ',' {
					return forwarded[:i]
				}
			}
			return forwarded
		}
	}
	host := r.RemoteAddr
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			return host[:i]
		}
	}
	return host
}
