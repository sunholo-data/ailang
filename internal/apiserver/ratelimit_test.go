package apiserver

import (
	"net/http"
	"sync"
	"testing"
	"time"
)

// fakeClock is an injectable clock used by the limiter under test.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func newTestLimiter(t *testing.T, rpm, burst int) (*IPRateLimiter, *fakeClock) {
	t.Helper()
	clock := &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	l := NewIPRateLimiter(rpm, burst)
	if l == nil {
		t.Fatalf("expected non-nil limiter for rpm=%d burst=%d", rpm, burst)
	}
	l.now = clock.now
	return l, clock
}

func TestIPRateLimiter_NilWhenDisabled(t *testing.T) {
	if l := NewIPRateLimiter(0, 5); l != nil {
		t.Fatalf("rpm=0 must return nil, got %#v", l)
	}
	if l := NewIPRateLimiter(-1, 5); l != nil {
		t.Fatalf("rpm<0 must return nil, got %#v", l)
	}
}

func TestIPRateLimiter_NilSafeAllow(t *testing.T) {
	var l *IPRateLimiter
	if !l.Allow("1.2.3.4") {
		t.Fatal("nil limiter must allow all traffic")
	}
}

func TestIPRateLimiter_AllowsBurst(t *testing.T) {
	l, _ := newTestLimiter(t, 60, 3)
	for i := 0; i < 3; i++ {
		if !l.Allow("1.2.3.4") {
			t.Fatalf("call %d: burst should allow", i+1)
		}
	}
}

func TestIPRateLimiter_DeniesAfterBurst(t *testing.T) {
	l, _ := newTestLimiter(t, 60, 3) // 1 token/sec, 3 burst
	for i := 0; i < 3; i++ {
		_ = l.Allow("1.2.3.4")
	}
	if l.Allow("1.2.3.4") {
		t.Fatal("4th immediate call must be denied")
	}
}

func TestIPRateLimiter_RefillsOverTime(t *testing.T) {
	l, clock := newTestLimiter(t, 60, 1) // 1 token/sec, burst 1
	if !l.Allow("ip") {
		t.Fatal("first call must allow")
	}
	if l.Allow("ip") {
		t.Fatal("second immediate call must deny")
	}
	clock.advance(2 * time.Second) // refills 2 tokens, capped at burst=1
	if !l.Allow("ip") {
		t.Fatal("after 2s the bucket should have refilled")
	}
	if l.Allow("ip") {
		t.Fatal("burst is 1; the second post-refill call must deny")
	}
}

func TestIPRateLimiter_PerIPIsolation(t *testing.T) {
	l, _ := newTestLimiter(t, 60, 2)
	for i := 0; i < 2; i++ {
		if !l.Allow("a") {
			t.Fatalf("ip a call %d must allow", i+1)
		}
	}
	if l.Allow("a") {
		t.Fatal("ip a 3rd call must deny")
	}
	for i := 0; i < 2; i++ {
		if !l.Allow("b") {
			t.Fatalf("ip b call %d must allow (separate bucket)", i+1)
		}
	}
}

func TestIPRateLimiter_EmptyIPBucketed(t *testing.T) {
	l, _ := newTestLimiter(t, 60, 2)
	if !l.Allow("") {
		t.Fatal("empty ip should still allow first request (bucketed as 'unknown')")
	}
	if !l.Allow("") {
		t.Fatal("empty ip should allow up to burst")
	}
	if l.Allow("") {
		t.Fatal("empty ip past burst must deny — never an unbucketed bypass")
	}
}

func TestIPRateLimiter_BucketGCDoesNotEvictActive(t *testing.T) {
	l, clock := newTestLimiter(t, 60, 3)
	// Use ip "active" recently
	if !l.Allow("active") {
		t.Fatal("active first call must allow")
	}
	// Stale ip — no recent activity
	_ = l.Allow("stale")

	// Fast-forward past the 1h cutoff and 10m sweep gap
	clock.advance(2 * time.Hour)
	// Triggering Allow() runs sweepLocked
	_ = l.Allow("trigger")

	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.buckets["stale"]; ok {
		t.Error("stale bucket should have been swept")
	}
	if _, ok := l.buckets["trigger"]; !ok {
		t.Error("trigger bucket (just used) must remain")
	}
}

func TestClientIPFromHeader_RightmostXFF(t *testing.T) {
	tests := []struct {
		name string
		hdr  http.Header
		want string
	}{
		{
			name: "single XFF entry",
			hdr:  http.Header{"X-Forwarded-For": []string{"1.2.3.4"}},
			want: "1.2.3.4",
		},
		{
			name: "two-entry XFF — rightmost is the trusted GFE-appended source",
			hdr:  http.Header{"X-Forwarded-For": []string{"client-claimed, 9.9.9.9"}},
			want: "9.9.9.9",
		},
		{
			name: "spaced XFF",
			hdr:  http.Header{"X-Forwarded-For": []string{"1.1.1.1,  9.9.9.9"}},
			want: "9.9.9.9",
		},
		{
			name: "X-Real-Ip fallback",
			hdr:  http.Header{"X-Real-Ip": []string{"5.5.5.5"}},
			want: "5.5.5.5",
		},
		{
			name: "neither header — empty (caller buckets as 'unknown')",
			hdr:  http.Header{},
			want: "",
		},
		{
			name: "nil header — empty",
			hdr:  nil,
			want: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := clientIPFromHeader(tc.hdr); got != tc.want {
				t.Errorf("clientIPFromHeader = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIPRateLimiter_ConcurrentAccessIsRaceFree(t *testing.T) {
	l, _ := newTestLimiter(t, 600, 100)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ip := "ip"
			if i%2 == 0 {
				ip = "ip-other"
			}
			for j := 0; j < 20; j++ {
				_ = l.Allow(ip)
			}
		}(i)
	}
	wg.Wait()
	// Pass = no race detector failure under `go test -race`.
}
