package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// TestRunCommand_PipedStdoutFlushesPerLine guards the M-PERF6B regression
// caught while migrating motoko_agent off the AILANG fork:
//
//   - M-PERF6B (perf optimization) wrapped os.Stdout in a 64KB bufio.Writer
//     unconditionally, gating flushes on either buffer-full or process-exit.
//   - Long-running programs that emit small JSON events to a piped stdout
//     (motoko_agent's TS env-server reads ailang's stdout line-by-line via
//     readline.createInterface) accumulate events in the buffer and only
//     flush at process exit. The agent loop never exits during a turn —
//     so the TUI showed zero progress for the entire turn.
//
// Fix (this regression test pins): isStdoutTTY()-gated buffering. When
// stdout is a pipe (not a terminal), bypass the buffer and write
// directly to os.Stdout so each println reaches downstream consumers
// immediately.
//
// The test runs the locally-installed `ailang` binary as a subprocess
// (which forces a piped stdout) and verifies that intermediate println
// output is visible BEFORE the program exits — i.e. proves real-time
// streaming.
func TestRunCommand_PipedStdoutFlushesPerLine(t *testing.T) {
	if os.Getenv("AILANG_SKIP_PIPE_TEST") == "1" {
		t.Skip("AILANG_SKIP_PIPE_TEST=1 — skipping subprocess-based pipe test")
	}
	ailangBin := testutil.FindAilangBinary(t)

	// Create a small AILANG program that emits 3 events with 500ms gaps.
	// 500ms × 3 = 1.5s total runtime. We'll assert we see at least the
	// first 2 events before that 1.5s elapses, proving streaming.
	tmpDir := t.TempDir()
	aiSrc := filepath.Join(tmpDir, "main.ail")
	prog := `module main

import std/io (println)
import std/clock (sleep)

export func main() -> () ! {IO, Clock} {
  println("EVENT_1");
  let _ = sleep(500);
  println("EVENT_2");
  let _ = sleep(500);
  println("EVENT_3")
}
`
	if err := os.WriteFile(aiSrc, []byte(prog), 0o644); err != nil {
		t.Fatalf("write program: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, ailangBin, "run", "--caps", "IO,Clock", "--entry", "main", aiSrc)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	// Capture each event with the wall-clock time we observed it.
	type observed struct {
		line string
		at   time.Duration
	}
	start := time.Now()
	events := make(chan observed, 16)
	go func() {
		defer close(events)
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "EVENT_") {
				events <- observed{line: line, at: time.Since(start)}
			}
		}
	}()

	// We expect EVENT_1 within 1s, EVENT_2 within 1.5s, EVENT_3 within 2.5s.
	// If the buffer is broken (events only flush at exit), they all arrive
	// at ~the same time near the end (~1.5-2s).
	gotByEvent := map[string]time.Duration{}
	deadline := time.After(4 * time.Second)
collect:
	for len(gotByEvent) < 3 {
		select {
		case ev, ok := <-events:
			if !ok {
				break collect
			}
			gotByEvent[ev.line] = ev.at
		case <-deadline:
			break collect
		}
	}

	if _, ok := gotByEvent["EVENT_1"]; !ok {
		t.Fatalf("did not observe EVENT_1 within deadline; got: %+v", gotByEvent)
	}
	if _, ok := gotByEvent["EVENT_2"]; !ok {
		t.Fatalf("did not observe EVENT_2 within deadline; got: %+v", gotByEvent)
	}

	// THE LOAD-BEARING ASSERTION: events stream in real time, not in a
	// single batch at exit. EVENT_1 should arrive at least 200ms before
	// EVENT_2 (sleep is 500ms; allow generous slack for build noise).
	// If buffering is broken, both events arrive within a few ms of each
	// other near the end of the run.
	gap := gotByEvent["EVENT_2"] - gotByEvent["EVENT_1"]
	const minGap = 200 * time.Millisecond
	if gap < minGap {
		t.Errorf("EVENT_1 → EVENT_2 gap = %s, want >= %s. "+
			"Events appear to be batched (buffered until exit). "+
			"This is the M-PERF6B regression — see isStdoutTTY() gate.",
			gap, minGap)
	}

	// Belt-and-suspenders: also assert EVENT_1 arrived before total runtime
	// elapsed (i.e. before all three sleeps would have completed sequentially).
	//
	// On Windows the ailang binary cold-start cost is ~1.7s vs <0.5s on
	// Linux/macOS — runner-VM filesystem + process-launch overhead — so the
	// budget is widened there. The load-bearing assertion is the gap check
	// above (EVENT_1 → EVENT_2 ≥ 200ms); this check is redundant guardrail.
	eventOneBudget := 1500 * time.Millisecond
	if runtime.GOOS == "windows" {
		eventOneBudget = 3500 * time.Millisecond
	}
	if gotByEvent["EVENT_1"] > eventOneBudget {
		t.Errorf("EVENT_1 arrived at %s — too late (budget %s). Expected first println "+
			"to appear before the program had time to call all three sleeps. "+
			"Suggests stdout is buffered until exit.",
			gotByEvent["EVENT_1"], eventOneBudget)
	}

	// Diagnostic output for debugging.
	t.Logf("event timings: EVENT_1=%s, EVENT_2=%s, EVENT_3=%s",
		gotByEvent["EVENT_1"],
		gotByEvent["EVENT_2"],
		gotByEvent["EVENT_3"],
	)
	_ = fmt.Sprint // keep import alive in trimmed builds
}
