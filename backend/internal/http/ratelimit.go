package http

import (
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// limiter allows `limit` requests per key per fixed `window`. In memory: the
// app runs as one instance, and a restart only resets the counters.
type limiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu   sync.Mutex
	hits map[string]*bucket
}

type bucket struct {
	start time.Time
	n     int
}

func newLimiter(limit int, window time.Duration) *limiter {
	return &limiter{limit: limit, window: window, now: time.Now, hits: map[string]*bucket{}}
}

// allow counts one request for key; when over the limit it says how long
// until the window resets.
func (l *limiter) allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if len(l.hits) > 10_000 { // keep memory bounded under a flood of keys
		for k, b := range l.hits {
			if now.Sub(b.start) >= l.window {
				delete(l.hits, k)
			}
		}
	}
	b := l.hits[key]
	if b == nil || now.Sub(b.start) >= l.window {
		b = &bucket{start: now}
		l.hits[key] = b
	}
	if b.n >= l.limit {
		return false, l.window - now.Sub(b.start)
	}
	b.n++
	return true, 0
}

// limit rejects requests over l's budget for the key (IP or user) with 429.
func (s *Server) limit(l *limiter, key func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ok, wait := l.allow(key(r)); !ok {
				w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
				writeError(w, http.StatusTooManyRequests, "rate_limited", "ทำรายการถี่เกินไป รอสักครู่แล้วลองใหม่")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// writesOnly applies a limiter to state-changing requests only.
func (s *Server) writesOnly(l *limiter, key func(*http.Request) string) func(http.Handler) http.Handler {
	lim := s.limit(l, key)
	return func(next http.Handler) http.Handler {
		limited := lim(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
			default:
				limited.ServeHTTP(w, r)
			}
		})
	}
}

// byUser keys on the signed-in user (use after requireUser).
func byUser(r *http.Request) string { return "u:" + userFrom(r.Context()).ID.String() }

// clientIP is the visitor's address. Behind Cloudflare (Render's edge) the
// connection comes from the proxy, so with TrustProxy we use the
// CF-Connecting-IP header, which Cloudflare sets and clients can't override.
func (s *Server) clientIP(r *http.Request) string {
	if s.TrustProxy {
		if ip := r.Header.Get("CF-Connecting-IP"); net.ParseIP(ip) != nil {
			s.logIPSource.Do(func() { slog.Info("client IPs come from CF-Connecting-IP") })
			return "ip:" + ip
		}
		s.logIPSource.Do(func() {
			slog.Warn("TRUST_PROXY is on but CF-Connecting-IP is missing; rate limits fall back to the proxy address")
		})
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return "ip:" + host
}

// limits are the per-server budgets. Generous for a group of friends,
// tight enough to stop scripted floods.
type limits struct {
	authIP   *limiter // signed-out sign-in endpoints, per IP
	signupIP *limiter // new passkey accounts, per IP
	writes   *limiter // any change, per user
	newTrips *limiter // trips created + join requests, per user
	slips    *limiter // slip uploads (each costs AI quota), per user
}

func defaultLimits() *limits {
	return &limits{
		authIP:   newLimiter(60, time.Minute),
		signupIP: newLimiter(20, time.Hour),
		writes:   newLimiter(120, time.Minute),
		newTrips: newLimiter(20, time.Hour),
		slips:    newLimiter(30, time.Hour),
	}
}
