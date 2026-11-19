package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

// M-V1-MEMORY-FOOTPRINT F6: Debug.log used to retain every line of a run and
// apply --log-level only at the final flush, so a logging loop grew without
// bound. The run paths now attach a DebugSink, so each line is written (or,
// below the threshold, dropped) on arrival and nothing is held.
//
// This counts the lines a run HOLDS, which is the property, instead of
// comparing peak RSS: it replaces TestMemprobeDebugLogDoesNotAccumulate
// (cmd/ailang), which failed on macOS CI at +32..+47 MB against a 30 MB
// bound because the same binary measures differently from one runner to the
// next. Two instruments:
//
//  1. Output order. stdout and stderr share one file, and a non-TTY stdout
//     gets each println written directly (run.go; the sink writes each line
//     to stderr directly), so after the program prints marker k, every line it logged
//     before that marker and that passes the level must already be in the
//     file: held(k) = passing lines logged before marker k − lines in the file
//     before marker k, and it must be exactly 0 at every marker. The pre-F6
//     run held every line until the end; a context that no sink reaches holds
//     them forever (#2355).
//
//  2. Retention on the context the program ACTUALLY RAN ON. Lines below the
//     level leave no trace in the output, so (1) cannot see whether they are
//     dropped or kept. And the runner's own context is not the one the
//     program runs on: CallEntrypoint forks the evaluator, and Clone hands the
//     fork a Fresh() context that shares the sink. The test wraps the
//     registered Debug.log operation (effects.Registry, test-only) to capture
//     every context a Debug.log call arrives on, and requires each to hold no
//     entries afterwards. cap(Collect().Logs) is the high-water mark even
//     after the end-of-run flush, because Reset keeps capacity. Review mutant
//     D4b — retain in the cloned context only, still stream every line —
//     passes (1) and fails (2).

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

// debugLogContexts wraps the registered Debug.log operation for the duration
// of the test and returns the effect contexts the program's calls arrived on.
func debugLogContexts(t *testing.T) *[]*effects.EffContext {
	t.Helper()
	orig := effects.Registry["Debug"]["log"]
	if orig == nil {
		t.Fatal("Debug.log is not registered in effects.Registry")
	}
	seen := &[]*effects.EffContext{}
	effects.Registry["Debug"]["log"] = func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		if !slices.Contains(*seen, ctx) {
			*seen = append(*seen, ctx)
		}
		return orig(ctx, args)
	}
	t.Cleanup(func() { effects.Registry["Debug"]["log"] = orig })
	return seen
}

// runDebugFixture runs the fixture through Run with stdout and stderr sharing
// one file and returns that file's lines and the contexts Debug.log ran on.
func runDebugFixture(t *testing.T, opts Options) ([]string, []*effects.EffContext) {
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
	seen := debugLogContexts(t)
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
	return strings.Split(strings.TrimRight(string(b), "\n"), "\n"), *seen
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
			lines, contexts := runDebugFixture(t, c.opts)
			assertDebugLinesStreamed(t, lines, c.items, c.ticksShown)
			assertNothingRetained(t, contexts, c.items)
		})
	}
}

// assertDebugLinesStreamed is instrument (1): held must be exactly 0 at EVERY
// marker — positive is held back, negative means more lines than were logged
// (the instrument is off) — and the output must actually interleave.
func assertDebugLinesStreamed(t *testing.T, lines []string, items int, ticksShown bool) {
	t.Helper()
	perBatch := 1 // the ERROR line
	if ticksShown {
		perBatch += debugLogTicks
	}
	totalMarkers := items * debugLogBatches
	written, markers, betweenMarkers := 0, 0, 0
	var badHeld []string
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "marker "):
			markers++
			if held := markers*perBatch - written; held != 0 {
				badHeld = append(badHeld, fmt.Sprintf("%s: held %d", line, held))
			}
			continue
		case strings.Contains(line, `"severity":"ERROR"`):
		case strings.Contains(line, `"tick `):
			if !ticksShown {
				t.Fatalf("a DEBUG line passed an ERROR threshold: %s", line)
			}
		default:
			continue
		}
		written++
		if markers > 0 && markers < totalMarkers {
			betweenMarkers++
		}
	}
	if markers != totalMarkers {
		t.Fatalf("saw %d markers, want %d (the program did not finish?)\n%.2000s", markers, totalMarkers, strings.Join(lines, "\n"))
	}
	t.Logf("%d Debug.log lines written, %d of them between markers; %d of %d markers with held != 0",
		written, betweenMarkers, len(badHeld), markers)
	if len(badHeld) != 0 {
		t.Errorf("held != 0 at %d of %d markers (first: %s) — each line must be written as it arrives, not held until the run ends (held > 0: Debug lines were held back; held < 0: the markers were)",
			len(badHeld), markers, badHeld[0])
	} else if betweenMarkers == 0 {
		// Unreachable while held is 0 at every marker with lines logged
		// before each, but kept so a future change to the fixture cannot
		// make this check pass without the output interleaving.
		t.Errorf("no Debug.log line landed between the first and the last marker: the output does not interleave, so held measured nothing")
	}
	if want := totalMarkers * perBatch; written != want {
		t.Errorf("%d Debug.log lines reached the output, want %d", written, want)
	}
}

// assertNothingRetained is instrument (2): the contexts the program ran on
// hold no entries, including lines below the level that never reach output.
func assertNothingRetained(t *testing.T, contexts []*effects.EffContext, items int) {
	t.Helper()
	if len(contexts) != items {
		t.Fatalf("Debug.log ran on %d contexts, want %d (one per program run)", len(contexts), items)
	}
	for i, ctx := range contexts {
		if ctx.Debug == nil {
			t.Fatalf("context %d has no Debug context", i)
		}
		if got := ctx.Debug.Collect(); cap(got.Logs) != 0 || len(got.Assertions) != 0 {
			t.Errorf("the context program run %d executed on retained Debug.log entries (log slice capacity %d after %d calls; %d checks); want none — every line must be written or dropped on arrival",
				i, cap(got.Logs), debugLogBatches*(debugLogTicks+1), len(got.Assertions))
		}
	}
}
