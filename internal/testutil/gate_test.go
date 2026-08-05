package testutil

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func cleanLiveNetworkEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("AILANG_LIVE_NET", "1")
	for _, name := range proxyEnvironmentVariables {
		t.Setenv(name, "")
	}
}

func TestLiveNetworkStatus_OptInSetRuns(t *testing.T) {
	cleanLiveNetworkEnvironment(t)

	status, reason := LiveNetworkStatus()
	if status != LiveNetworkRun {
		t.Fatalf("LiveNetworkStatus() status = %v, want %v (reason: %q)", status, LiveNetworkRun, reason)
	}
	if reason != "" {
		t.Fatalf("LiveNetworkStatus() reason = %q, want empty reason", reason)
	}
}

func TestLiveNetworkStatus_UnsetSkips(t *testing.T) {
	cleanLiveNetworkEnvironment(t)
	t.Setenv("AILANG_LIVE_NET", "")

	status, reason := LiveNetworkStatus()
	if status != LiveNetworkSkip {
		t.Fatalf("LiveNetworkStatus() status = %v, want %v (reason: %q)", status, LiveNetworkSkip, reason)
	}
	if !strings.Contains(reason, "AILANG_LIVE_NET") {
		t.Fatalf("LiveNetworkStatus() reason = %q, want it to name AILANG_LIVE_NET", reason)
	}
}

func TestLiveNetworkStatus_PoisonedProxyFatal(t *testing.T) {
	for _, poisoned := range proxyEnvironmentVariables {
		t.Run(poisoned, func(t *testing.T) {
			cleanLiveNetworkEnvironment(t)
			t.Setenv(poisoned, "http://127.0.0.1:9")

			status, reason := LiveNetworkStatus()
			if status != LiveNetworkFatal {
				t.Fatalf("LiveNetworkStatus() status = %v, want %v (reason: %q)", status, LiveNetworkFatal, reason)
			}
			if !strings.Contains(reason, poisoned) {
				t.Fatalf("LiveNetworkStatus() reason = %q, want it to name %s", reason, poisoned)
			}
		})
	}
}

func TestLiveNetworkStatus_DoesNotConfuseAnotherPortForPoison(t *testing.T) {
	cleanLiveNetworkEnvironment(t)
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:90")

	status, reason := LiveNetworkStatus()
	if status != LiveNetworkRun {
		t.Fatalf("LiveNetworkStatus() status = %v, want %v (reason: %q)", status, LiveNetworkRun, reason)
	}
}

func TestRequiresLiveNetwork_PoisonedLiveLaneFatal(t *testing.T) {
	if os.Getenv("TESTUTIL_POISONED_LIVE_LANE_HELPER") == "1" {
		RequiresLiveNetwork(t)
		return
	}

	cleanLiveNetworkEnvironment(t)
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:9")
	t.Setenv("TESTUTIL_POISONED_LIVE_LANE_HELPER", "1")
	cmd := exec.Command(os.Args[0], "-test.run=^TestRequiresLiveNetwork_PoisonedLiveLaneFatal$")
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("poisoned live-lane helper succeeded, want fatal failure; output:\n%s", output)
	}
	if !strings.Contains(string(output), "HTTPS_PROXY") {
		t.Fatalf("fatal output does not name HTTPS_PROXY:\n%s", output)
	}
}

func TestHangGuard_FloorsAtOneSecond(t *testing.T) {
	if got := HangGuard(t, 0); got != time.Second {
		t.Fatalf("HangGuard(t, 0) = %v, want %v", got, time.Second)
	}
}

func TestHangGuard_UsesCap(t *testing.T) {
	const cap = 2 * time.Second
	if got := HangGuard(t, cap); got != cap {
		t.Fatalf("HangGuard(t, %v) = %v, want cap unchanged", cap, got)
	}
}

func TestHangGuard_NoDeadlineReturnsCap(t *testing.T) {
	if _, ok := t.Deadline(); ok {
		t.Skip("requires go test -timeout 0 to exercise testing.T with no deadline")
	}

	const cap = 7 * time.Second
	if got := HangGuard(t, cap); got != cap {
		t.Fatalf("HangGuard(t, %v) = %v, want cap unchanged with no test deadline", cap, got)
	}
}

func TestHangGuardContext_UsesGuardedDeadline(t *testing.T) {
	ctx, cancel := HangGuardContext(t, 2*time.Second)
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("HangGuardContext() returned a context without a deadline")
	}
	remaining := time.Until(deadline)
	if remaining <= 0 || remaining > 2*time.Second {
		t.Fatalf("context deadline remaining = %v, want in (0, 2s]", remaining)
	}
}
