package httpserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
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
	return l.AllowPolicy(ctx, key, l.limit, l.window)
}

func (l *RateLimit) AllowPolicy(ctx context.Context, key string, limit int64, window time.Duration) (bool, error) {
	if l.client == nil {
		return true, nil
	}
	if limit <= 0 {
		limit = l.limit
	}
	if window <= 0 {
		window = l.window
	}
	redisKey := fmt.Sprintf("%s:%s", l.prefix, key)
	pipe := l.client.TxPipeline()
	count := pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, window)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, fmt.Errorf("rate limit Redis transaction: %w", err)
	}
	return count.Val() <= limit, nil
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

func ClientIP(r *http.Request, trustedProxyCIDRs []string) string {
	remote, ok := parseRequestIP(r.RemoteAddr)
	if !ok {
		return ""
	}
	if len(trustedProxyCIDRs) == 0 || !addressInCIDRs(remote, trustedProxyCIDRs) {
		return remote.String()
	}
	forwardedValues := r.Header.Values("X-Forwarded-For")
	if len(forwardedValues) == 0 {
		return remote.String()
	}
	parts := strings.Split(strings.Join(forwardedValues, ","), ",")
	forwarded := make([]netip.Addr, 0, len(parts))
	for _, part := range parts {
		addr, parseErr := netip.ParseAddr(strings.TrimSpace(part))
		if parseErr != nil {
			return remote.String()
		}
		forwarded = append(forwarded, addr.Unmap())
	}
	current := remote
	for index := len(forwarded) - 1; index >= 0; index-- {
		if !addressInCIDRs(current, trustedProxyCIDRs) {
			break
		}
		current = forwarded[index]
	}
	return current.String()
}

func parseRequestIP(remoteAddr string) (netip.Addr, bool) {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err != nil {
		host = strings.Trim(strings.TrimSpace(remoteAddr), "[]")
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, false
	}
	return addr.Unmap(), true
}

func addressInCIDRs(addr netip.Addr, cidrs []string) bool {
	for _, rawCIDR := range cidrs {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(rawCIDR))
		if err != nil {
			continue
		}
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
