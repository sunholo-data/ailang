//go:build !windows

// syscall.Rusage.Maxrss exists on Unix only; the Windows CI runner cannot
// compile this file (measured 2026-09-17: test-windows red on every dev push
// since 2491ab45e). The probe measures a Unix RSS; there is nothing to port.

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// M-V1-MEMORY-FOOTPRINT M1: tracing must cost a bounded amount on top of the
// program's own live data. These run the built binary and compare the child's
// peak RSS traced vs untraced. Skipped under -short (they build the CLI) and
// on Windows (no rusage).
//
// Measured before the fix (2026-09-16, cons depth 9000): untraced 772 MB,
// deep 2282 MB (2.96x). After: 814 MB (1.06x).

func peakRSS(t *testing.T, dir string, env []string, args ...string) (int64, string) {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v failed: %v\n%s", args, err, out)
	}
	ru, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage)
	if !ok {
		t.Skip("no rusage on this platform")
	}
	rss := int64(ru.Maxrss)
	if runtime.GOOS == "linux" {
		rss *= 1024 // linux reports KB, darwin bytes
	}
	return rss, string(out)
}

func memprobeSetup(t *testing.T) (bin, dir string) {
	testutil.SkipInFastLoop(t, "memprobe builds the binary and runs two ~300 MB fixtures (~10s)")
	bin = buildAilang(t)
	src, err := filepath.Abs(filepath.Join("testdata", "memprobe"))
	if err != nil {
		t.Fatal(err)
	}
	dir = t.TempDir()
	entries, _ := os.ReadDir(src)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".ail") {
			continue // `ailang check` leaves a .ailang/ state dir behind
		}
		b, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, e.Name()), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return bin, dir
}

func TestMemprobeDeepTraceIsBoundedOnConsRecursion(t *testing.T) {
	bin, dir := memprobeSetup(t)
	base := []string{"AILANG_RELAX_MODULES=1", "AILANG_NO_TRACE=1"}
	untraced, out := peakRSS(t, dir, base, bin, "run", "--entry", "main", "--caps", "IO", "cons.ail")
	if !strings.Contains(out, "done") {
		t.Fatalf("untraced run did not finish: %s", out)
	}
	deep, _ := peakRSS(t, dir, []string{"AILANG_RELAX_MODULES=1", "AILANG_TRACE=deep"},
		bin, "run", "--entry", "main", "--caps", "IO", "--emit-trace", "jsonl", "cons.ail")
	ratio := float64(deep) / float64(untraced)
	t.Logf("cons: untraced %d MB, deep %d MB, ratio %.2f", untraced>>20, deep>>20, ratio)
	if ratio > 1.25 {
		t.Fatalf("deep trace peak %.2fx untraced; want <= 1.25 (rendering must be bounded, not materialise-then-truncate)", ratio)
	}
}

func TestMemprobeStandardTierEffectResultsAreBounded(t *testing.T) {
	bin, dir := memprobeSetup(t)
	big := strings.Repeat("q", 20<<20)
	if err := os.WriteFile(filepath.Join(dir, "big.txt"), []byte(big), 0o644); err != nil {
		t.Fatal(err)
	}
	untraced, out := peakRSS(t, dir, []string{"AILANG_RELAX_MODULES=1", "AILANG_NO_TRACE=1"},
		bin, "run", "--entry", "main", "--caps", "IO,FS", "effect_result.ail")
	if !strings.Contains(out, "n=") {
		t.Fatalf("untraced run did not finish: %s", out)
	}
	traced, _ := peakRSS(t, dir, []string{"AILANG_RELAX_MODULES=1"},
		bin, "run", "--entry", "main", "--caps", "IO,FS", "--emit-trace", "jsonl", "effect_result.ail")
	ratio := float64(traced) / float64(untraced)
	t.Logf("effect_result: untraced %d MB, standard+emit %d MB, ratio %.2f", untraced>>20, traced>>20, ratio)
	if ratio > 1.10 {
		t.Fatalf("standard-tier trace peak %.2fx untraced; want <= 1.10 (effect results must not be rendered in full)", ratio)
	}
}

// Debug.log used to retain every line for the whole run and apply --log-level
// only at the final flush (M-V1-MEMORY-FOOTPRINT F6). 200k structured DEBUG
// lines below an ERROR threshold must now cost nothing: they are dropped on
// arrival. The comparison is against the same program logging 1 line.
func TestMemprobeDebugLogDoesNotAccumulate(t *testing.T) {
	bin, dir := memprobeSetup(t)
	src, err := os.ReadFile(filepath.Join(dir, "debuglog.ail"))
	if err != nil {
		t.Fatal(err)
	}
	one := strings.Replace(string(src), "outer(400)", "outer(1)", 1)
	if err := os.WriteFile(filepath.Join(dir, "debuglog_one.ail"), []byte(strings.Replace(one, "module debuglog", "module debuglog_one", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	// GOGC=100 so the comparison measures RETENTION, not how much garbage the
	// CLI's default GOGC=500 lets pile up before a cycle (measured: 200k
	// lines old binary +64 MB retained; new +12 MB of transient garbage).
	env := []string{"AILANG_RELAX_MODULES=1", "AILANG_NO_TRACE=1", "GOGC=100"}
	floor, _ := peakRSS(t, dir, env, bin, "run", "--entry", "main", "--caps", "IO,Debug", "--log-level", "error", "debuglog_one.ail")
	many, out := peakRSS(t, dir, env, bin, "run", "--entry", "main", "--caps", "IO,Debug", "--log-level", "error", "debuglog.ail")
	if !strings.Contains(out, "done") {
		t.Fatalf("run did not finish: %s", out)
	}
	if strings.Contains(out, "tick") {
		t.Fatalf("DEBUG lines leaked past --log-level error")
	}
	delta := (many - floor) >> 20
	t.Logf("debuglog: 1 line %d MB, 200k lines %d MB, delta %d MB", floor>>20, many>>20, delta)
	if delta > 30 {
		t.Fatalf("200k filtered Debug.log lines grew RSS by %d MB; want <= 30 (lines must be dropped on arrival, not retained; the old binary measured +64)", delta)
	}
}
