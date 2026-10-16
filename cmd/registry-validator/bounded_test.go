package main

import (
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func fixedRSS(b uint64) rssReader { return func(int) (uint64, bool) { return b, true } }

func TestRunBounded_UnderBudgetReturnsOutput(t *testing.T) {
	out, err := runBounded(t.TempDir(), 1<<30, 10*time.Second, fixedRSS(1<<20), "echo", "ok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(string(out)) != "ok" {
		t.Fatalf("output = %q", out)
	}
}

// The #1811 shape: the child grows past the cap. It must be killed and
// reported as a memory budget refusal, not left to take the instance down.
func TestRunBounded_OverMemoryIsKilledAndNamed(t *testing.T) {
	start := time.Now()
	_, err := runBounded(t.TempDir(), 384<<20, 30*time.Second, fixedRSS(600<<20), "sleep", "20")
	be, ok := asBudgetError(err)
	if !ok {
		t.Fatalf("want budgetError, got %v", err)
	}
	if be.resource != "memory" || be.observed != "600 MiB" || be.limit != "384 MiB" {
		t.Fatalf("budgetError = %+v", be)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatalf("child was not killed promptly (%v)", time.Since(start))
	}
	if !strings.Contains(err.Error(), "memory budget") {
		t.Fatalf("message does not name the budget: %s", err)
	}
}

func TestRunBounded_OverTimeIsNamed(t *testing.T) {
	_, err := runBounded(t.TempDir(), 1<<30, 300*time.Millisecond, fixedRSS(1<<20), "sleep", "20")
	be, ok := asBudgetError(err)
	if !ok || be.resource != "time" {
		t.Fatalf("want time budgetError, got %v", err)
	}
}

// A compile failure is the package's fault and must stay a plain error, never
// a budget refusal.
func TestRunBounded_ChildFailureIsNotBudget(t *testing.T) {
	_, err := runBounded(t.TempDir(), 1<<30, 10*time.Second, fixedRSS(1<<20), "false")
	if err == nil {
		t.Fatal("want error from failing child")
	}
	if _, ok := asBudgetError(err); ok {
		t.Fatalf("plain failure reported as budget: %v", err)
	}
}

func TestProcRSS_UnreadablePidIsFalse(t *testing.T) {
	if _, ok := procRSS(-1); ok {
		t.Fatal("procRSS(-1) reported a value")
	}
}

// The watchdog is only as good as the reader Cloud Run uses: on Linux it must
// see this very process's RSS, or the memory cap silently never fires.
func TestProcRSS_ReadsSelfOnLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("/proc is Linux-only; the validator runs on Linux")
	}
	rss, ok := procRSS(os.Getpid())
	if !ok || rss < 1<<20 {
		t.Fatalf("procRSS(self) = %d, %v", rss, ok)
	}
}
