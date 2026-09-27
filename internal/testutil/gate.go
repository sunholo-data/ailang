package testutil

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"
)

// LiveNetworkDecision describes how a test should handle live network access.
type LiveNetworkDecision uint8

const (
	// LiveNetworkSkip means the test has not explicitly opted in to live access.
	LiveNetworkSkip LiveNetworkDecision = iota
	// LiveNetworkFatal means the live lane is enabled but misconfigured.
	LiveNetworkFatal
	// LiveNetworkRun means the test may perform live network operations.
	LiveNetworkRun
)

func (d LiveNetworkDecision) String() string {
	switch d {
	case LiveNetworkSkip:
		return "skip"
	case LiveNetworkFatal:
		return "fatal"
	case LiveNetworkRun:
		return "run"
	default:
		return fmt.Sprintf("LiveNetworkDecision(%d)", d)
	}
}

var proxyEnvironmentVariables = []string{
	"HTTP_PROXY",
	"HTTPS_PROXY",
	"http_proxy",
	"https_proxy",
}

// LiveNetworkStatus returns the live-network decision without acting on a
// testing.T, allowing all three branches to be tested directly.
func LiveNetworkStatus() (LiveNetworkDecision, string) {
	if os.Getenv("AILANG_LIVE_NET") != "1" {
		return LiveNetworkSkip, "AILANG_LIVE_NET is not 1; live network tests require explicit opt-in"
	}

	for _, name := range proxyEnvironmentVariables {
		if proxyPointsAtPoison(os.Getenv(name)) {
			return LiveNetworkFatal, fmt.Sprintf("%s points at the poison proxy 127.0.0.1:9 in the live network lane", name)
		}
	}
	return LiveNetworkRun, ""
}

func proxyPointsAtPoison(value string) bool {
	if value == "" {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		parsed, err = url.Parse("http://" + value)
	}
	return err == nil && parsed.Hostname() == "127.0.0.1" && parsed.Port() == "9"
}

// RequiresLiveNetwork skips tests outside the live lane and fails tests when
// that lane still carries the poison proxy configuration.
func RequiresLiveNetwork(t *testing.T) {
	t.Helper()
	decision, reason := LiveNetworkStatus()
	if decision == LiveNetworkSkip {
		t.Skip(reason)
	}
	if decision == LiveNetworkFatal {
		// Do not unset proxy variables here: Go caches proxy configuration
		// process-wide on first use, so changing the environment after an
		// earlier request may silently retain the poisoned proxy.
		t.Fatalf("live network lane is misconfigured: %s", reason)
	}
}

// HangGuard returns an operation timeout capped by both cap and the test's
// remaining deadline, with time reserved for reporting and cleanup.
//
// When the test binary's -timeout deadline is nearly spent, HangGuard fails the
// test and names the package budget. It used to hand out a 1s floor instead, so
// a subprocess that had already printed its correct answer was killed at 1s and
// the test read "exit 1" with an empty stderr (Windows cmd/ailang, 2026-09-27:
// the package ran 406s against -timeout 416s, and six unrelated tests went red).
func HangGuard(t *testing.T, cap time.Duration) time.Duration {
	t.Helper()
	deadline, ok := t.Deadline()
	if !ok {
		return cap
	}
	bound, exhausted := hangGuardBound(cap, time.Until(deadline))
	if exhausted {
		t.Fatalf("hang guard: the test binary's -timeout deadline is %s away; this PACKAGE has outgrown its go test -timeout budget (not a failure of this test's logic)",
			time.Until(deadline).Round(time.Millisecond))
	}
	return bound
}

// hangGuardBound is HangGuard's arithmetic: the operation bound for a caller cap
// and the time left before the test deadline, and whether that deadline is too
// close (under reserve + 1s) to run anything honestly.
func hangGuardBound(cap, untilDeadline time.Duration) (time.Duration, bool) {
	const reserve = 20 * time.Second
	remaining := untilDeadline - reserve
	if remaining < time.Second {
		return 0, true
	}
	bound := min(cap, remaining)
	if bound < time.Second {
		return time.Second, false
	}
	return bound, false
}

// HangGuardContext returns a background context bounded by HangGuard.
func HangGuardContext(t *testing.T, cap time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), HangGuard(t, cap))
}
