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
func HangGuard(t *testing.T, cap time.Duration) time.Duration {
	t.Helper()
	deadline, ok := t.Deadline()
	if !ok {
		return cap
	}

	bound := min(cap, time.Until(deadline)-20*time.Second)
	if bound < time.Second {
		return time.Second
	}
	return bound
}

// HangGuardContext returns a background context bounded by HangGuard.
func HangGuardContext(t *testing.T, cap time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), HangGuard(t, cap))
}
