package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M-V1-MEMORY-FOOTPRINT F6: Debug.log used to retain every line of a run and
// apply --log-level only at the final flush, so a logging loop grew without
// bound. The run paths now attach a DebugSink, so each line is written (or,
// below the threshold, dropped) on arrival and nothing is held.
//
// This counts the lines a run HOLDS, which is the property, instead of
// comparing peak RSS: it replaces TestMemprobeDebugLogDoesNotAccumulate
// (cmd/ailang), which failed on macOS CI at +34 and +38 MB against a 30 MB
// bound because the same binary measures differently from one runner to the
// next.
//
// The count comes from output order. stdout and stderr share one file and
// stdout is line-flushed, so after the program prints marker k, every line it
// logged before that marker and that passes the level must already be in the
// file. held(k) = lines logged and passing before marker k − lines in the file
// before marker k. A streaming run holds 0 at every marker. The pre-F6 run
// held every line until the end (held = all 40 batches' worth at marker 40);
// a context that no sink reaches holds them forever (#2355).
//
// Lines BELOW the threshold leave no trace in the output either way, so this
// cannot see whether they are dropped or kept; that half is the DebugContext's
// own contract, pinned in internal/effects by
// TestDebugLogFiltersByLevelOnArrival (Collect().Logs stays empty). What this
// test adds is that every run path — single run and each batch item — puts
// its program behind a sink at all; the in-process structure check was blind
// to that, because the entrypoint runs on a cloned context (Fork/Clone).

const (
	debugLogBatches = 40
	debugLogTicks   = 500 // DEBUG lines per batch, below an ERROR threshold
)

var debugLogFixture = fmt.Sprintf(`module debuglog

import std/debug as Debug
import std/io (println)

func inner(i: int) -> int ! {Debug} =
  if i == 0 then 0 else let _ = Debug.log("{\"severity\":\"DEBUG\",\"message\":\"tick ${show(i)}\"}") in inner(i - 1)

func outer(n: int, k: int) -> int ! {IO, Debug} =
  if k > n then 0 else
  let _ = inner(%d) in
  let _ = Debug.log("{\"severity\":\"ERROR\",\"message\":\"batch ${show(k)}\"}") in
  let _ = println("marker ${show(k)}") in
  outer(n, k + 1)

export func main() -> () ! {IO, Debug} {
  let _ = outer(%d, 1);
  println("done")
}
`, debugLogTicks, debugLogBatches)

// runDebugFixture runs the fixture through Run with stdout and stderr sharing
// one file and returns that file's lines.
func runDebugFixture(t *testing.T, opts Options) []string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile("debuglog.ail", []byte(debugLogFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(dir, "output")
	out, err := os.Create(outPath)
	if err != nil {
		t.Fatal(err)
	}
	origOut, origErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = out, out
	opts.Filename, opts.Entry, opts.ArgsJSON, opts.Caps = "debuglog.ail", "main", "null", "IO,Debug"
	opts.Quiet, opts.NoPrint = true, true
	code := func() int {
		defer func() { os.Stdout, os.Stderr = origOut, origErr }()
		return Run(t.Context(), opts)
	}()
	_ = out.Close()
	b, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("run exited %d\n%.3000s", code, b)
	}
	return strings.Split(strings.TrimRight(string(b), "\n"), "\n")
}

func TestDebugLogDoesNotAccumulateDuringRun(t *testing.T) {
	const levelDebug, levelError = 0, 3
	for _, c := range []struct {
		name       string
		opts       Options
		items      int  // program runs (batch inputs)
		ticksShown bool // DEBUG lines pass the level
	}{
		{"run/log-level-error", Options{DebugLogLevel: levelError}, 1, false},
		{"run/log-level-debug", Options{DebugLogLevel: levelDebug}, 1, true},
		{"batch/log-level-error", Options{DebugLogLevel: levelError, Batch: true, ProgramArgs: []string{"in-1", "in-2"}}, 2, false},
		// No strict-bytecode row: the VM does not wire Debug.log (BC001
		// "effectful builtin __debug_log not yet wired").
	} {
		t.Run(c.name, func(t *testing.T) {
			lines := runDebugFixture(t, c.opts)
			perBatch := 1 // the ERROR line
			if c.ticksShown {
				perBatch += debugLogTicks
			}
			written, markers, maxHeld, heldAt := 0, 0, 0, ""
			for _, line := range lines {
				switch {
				case strings.HasPrefix(line, "marker "):
					markers++
					if held := markers*perBatch - written; held > maxHeld {
						maxHeld, heldAt = held, line
					}
				case strings.Contains(line, `"severity":"ERROR"`):
					written++
				case strings.Contains(line, `"tick `):
					if !c.ticksShown {
						t.Fatalf("a DEBUG line passed an ERROR threshold: %s", line)
					}
					written++
				}
			}
			if want := c.items * debugLogBatches; markers != want {
				t.Fatalf("saw %d markers, want %d (the program did not finish?)\n%.2000s", markers, want, strings.Join(lines, "\n"))
			}
			t.Logf("%d Debug.log lines written; at most %d held at any marker", written, maxHeld)
			if maxHeld != 0 {
				t.Errorf("the run held up to %d Debug.log lines (at %q) instead of writing each as it arrived — a logging loop would grow without bound", maxHeld, heldAt)
			}
			if want := c.items * debugLogBatches * perBatch; written != want {
				t.Errorf("%d Debug.log lines reached the output, want %d", written, want)
			}
		})
	}
}
