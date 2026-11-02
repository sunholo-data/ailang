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

func fakeFS(files map[string]string) func(string) ([]byte, error) {
	return func(p string) ([]byte, error) {
		if v, ok := files[p]; ok {
			return []byte(v), nil
		}
		return nil, os.ErrNotExist
	}
}

// The cap follows the instance limit: a hardcoded 384 MiB outlived the move to
// a 1 GiB instance and refused sunholo/celestial@0.6.0 (2026-11-02).
func TestInstanceRSSCap_FollowsLimit(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  uint64
		src   string
	}{
		{"cgroup v2 1GiB", map[string]string{"/sys/fs/cgroup/memory.max": "1073741824\n"}, 896 << 20, "memory.max"},
		{"cgroup v2 512MiB keeps the old cap", map[string]string{"/sys/fs/cgroup/memory.max": "536870912\n"}, 384 << 20, "memory.max"},
		{"v2 unlimited falls to v1", map[string]string{
			"/sys/fs/cgroup/memory.max":                   "max\n",
			"/sys/fs/cgroup/memory/memory.limit_in_bytes": "2147483648\n",
		}, 1920 << 20, "limit_in_bytes"},
		{"v1 unlimited falls to meminfo", map[string]string{
			"/sys/fs/cgroup/memory/memory.limit_in_bytes": "9223372036854771712\n",
			"/proc/meminfo": "MemTotal:        1048576 kB\nMemFree: 1 kB\n",
		}, 896 << 20, "meminfo"},
		{"gvisor meminfo only", map[string]string{"/proc/meminfo": "MemTotal:        1048576 kB\n"}, 896 << 20, "meminfo"},
		{"nothing readable", map[string]string{}, fallbackRSSCap, "fallback"},
		{"tiny instance keeps a quarter", map[string]string{"/sys/fs/cgroup/memory.max": "268435456"}, 192 << 20, "memory.max"},
	}
	for _, c := range cases {
		got, src := instanceRSSCap(fakeFS(c.files))
		if got != c.want {
			t.Errorf("%s: cap = %s, want %s", c.name, mib(got), mib(c.want))
		}
		if !strings.Contains(src, c.src) {
			t.Errorf("%s: source %q does not name %q", c.name, src, c.src)
		}
	}
}
