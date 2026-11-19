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

// M-V1-MEMORY-FOOTPRINT M1/M2: tracing and logging must cost a bounded amount
// on top of the program's own live data. These run the built binary and compare
// the child's peak RSS between two arms.
//
// Both arms run with GOGC=100. Under the CLI's default GOGC=500 the peak is
// mostly garbage headroom and where the cycles land — the effect-result
// fixture that used to live here read 0.96x on the rig and 1.37x on the ubuntu
// runner for the same binary. That check is now an allocation count in
// internal/effects (TestEffectTraceRenderDoesNotMaterialiseLargeResults), and
// the debuglog fixture that followed it (+34/+38 MB against a 30 MB bound on
// macOS CI) is now a count of the lines a run holds, in internal/runner
// (TestDebugLogDoesNotAccumulateDuringRun). What remains here is LIVE-data
// dominated (cons), which GOGC=100 makes stable. Skipped under -short (it
// builds the CLI) and on Windows (no rusage).
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
	testutil.SkipInFastLoop(t, "memprobe builds the binary and runs the cons fixture twice (~10s)")
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
	base := []string{"AILANG_RELAX_MODULES=1", "AILANG_NO_TRACE=1", "GOGC=100"}
	untraced, out := peakRSS(t, dir, base, bin, "run", "--entry", "main", "--caps", "IO", "cons.ail")
	if !strings.Contains(out, "done") {
		t.Fatalf("untraced run did not finish: %s", out)
	}
	deep, _ := peakRSS(t, dir, []string{"AILANG_RELAX_MODULES=1", "AILANG_TRACE=deep", "GOGC=100"},
		bin, "run", "--entry", "main", "--caps", "IO", "--emit-trace", "jsonl", "cons.ail")
	ratio := float64(deep) / float64(untraced)
	t.Logf("cons: untraced %d MB, deep %d MB, ratio %.2f", untraced>>20, deep>>20, ratio)
	if ratio > 1.25 {
		t.Fatalf("deep trace peak %.2fx untraced; want <= 1.25 (rendering must be bounded, not materialise-then-truncate)", ratio)
	}
}
