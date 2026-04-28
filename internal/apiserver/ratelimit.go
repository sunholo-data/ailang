package apiserver

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// IPRateLimiter is a per-IP token-bucket limiter for write-side MCP tools
// (today: only submit_feedback). State lives in this process — each Cloud Run
// instance has its own table — so under autoscale the cap is "soft" up to
// `mcp_max_instances × rate`. For strict edge enforcement see
// design_docs/planned/v0_15_0/m-mcp-edge-throttle.md Path B (Cloud Armor).
type IPRateLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*tokenBucket
	rate      float64 // tokens added per second
	burst     float64 // max tokens (and starting balance for new IPs)
	now       func() time.Time
	lastSweep time.Time
}

type tokenBucket struct {
	tokens float64
	last   time.Time
}

// NewIPRateLimiter returns a limiter for `rpm` requests per minute with a
// configured burst. rpm <= 0 returns nil; nil limiters allow all traffic
// (operator escape hatch via AILANG_RATELIMIT_RPM=0).
func NewIPRateLimiter(rpm int, burst int) *IPRateLimiter {
	if rpm <= 0 {
		return nil
	}
	if burst < 1 {
		burst = 1
	}
	return &IPRateLimiter{
		buckets: make(map[string]*tokenBucket),
		rate:    float64(rpm) / 60.0,
		burst:   float64(burst),
		now:     time.Now,
	}
}

// Allow consumes a token for ip. Empty ip is bucketed as "unknown" so
// header-less probes still hit a cap rather than bypassing it.
func (l *IPRateLimiter) Allow(ip string) bool {
	if l == nil {
		return true
	}
	if ip == "" {
		ip = "unknown"
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	b, ok := l.buckets[ip]
	if !ok {
		b = &tokenBucket{tokens: l.burst, last: now}
		l.buckets[ip] = b
	} else {
		elapsed := now.Sub(b.last).Seconds()
		if elapsed > 0 {
			b.tokens += elapsed * l.rate
			if b.tokens > l.burst {
				b.tokens = l.burst
			}
			b.last = now
		}
	}

	if now.Sub(l.lastSweep) > 10*time.Minute {
		l.sweepLocked(now)
		l.lastSweep = now
	}

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true
	}
	return false
}

// sweepLocked drops idle buckets so a steady stream of unique IPs cannot
// grow the table without bound. A bucket idle for >1h is fully refilled
// regardless of its token count, so eviction is safe. Caller holds l.mu.
func (l *IPRateLimiter) sweepLocked(now time.Time) {
	cutoff := now.Add(-1 * time.Hour)
	for ip, b := range l.buckets {
		if b.last.Before(cutoff) {
			delete(l.buckets, ip)
		}
	}
}

// clientIPFromHeader extracts the trusted client IP from an HTTP header set.
//
// On a public Cloud Run service, X-Forwarded-For arrives as
// "<client-claimed>, <real-client-from-GFE>" — clients can prepend whatever
// they want, but Google's frontend always appends the actual TCP source as
// the rightmost entry. So we trust the rightmost. Falls back to X-Real-Ip,
// then to empty (limiter buckets that as "unknown").
func clientIPFromHeader(h http.Header) string {
	if h == nil {
		return ""
	}
	if xff := h.Get("X-Forwarded-For"); xff != "" {
		if i := strings.LastIndex(xff, ","); i >= 0 {
			return strings.TrimSpace(xff[i+1:])
		}
		return strings.TrimSpace(xff)
	}
	if rip := h.Get("X-Real-Ip"); rip != "" {
		return strings.TrimSpace(rip)
	}
	return ""
}
