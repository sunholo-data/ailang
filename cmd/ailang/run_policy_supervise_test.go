package main

import (
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// The copier is slow to start; the writer has already exited (its write end
// is closed). drainOutput must wait for EOF instead of closing the read end
// under the copier. The old order (cmd.Wait closing StdoutPipe, then
// wg.Wait) dropped exactly this output.
func TestDrainOutput_KeepsOutputWrittenBeforeExit(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Repeat("line\n", 2000)
	go func() { _, _ = io.WriteString(w, want); _ = w.Close() }()

	var got strings.Builder
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(50 * time.Millisecond) // the worker exits before we read
		_, _ = io.Copy(&got, r)
	}()
	drainOutput(&wg, 5*time.Second, r)
	if got.String() != want {
		t.Fatalf("lost output: got %d bytes, want %d", got.Len(), len(want))
	}
}

// A descendant that keeps the write end open must not hang the supervisor:
// after the grace period the read end is closed and the copier returns.
func TestDrainOutput_BoundedWhenAWriteEndLingers(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); _, _ = io.Copy(io.Discard, r) }()
	start := time.Now()
	drainOutput(&wg, 100*time.Millisecond, r)
	if d := time.Since(start); d > 3*time.Second {
		t.Fatalf("drainOutput blocked %s on a lingering write end", d)
	}
}

// End to end: every line a policy-run program prints reaches stdout. On
// linux CI the supervisor used to drop the tail (or all) of it.
func TestRunPolicy_SupervisorKeepsAllProgramOutput(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	pol := writePolicy(t, dir, "allowed_caps = [\"IO\"]\nentry = \"main\"\ntimeout_ms = 30000\n")
	f := writeAil(t, dir, "prog.ail", `module prog
func loop(i: int) -> () ! {IO} = if i > 300 then () else {
  println("L${show(i)}");
  loop(i + 1)
}
export func main() -> () ! {IO} = loop(1)
`)
	for i := range 5 {
		stdout, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, f)
		if code != 0 {
			t.Fatalf("run %d: exit %d\n%s%s", i, code, stdout, stderr)
		}
		if !strings.Contains(stdout, "L1\n") || !strings.Contains(stdout, "L300\n") {
			t.Fatalf("run %d: program output lost (%d bytes of stdout)\nstderr: %s", i, len(stdout), stderr)
		}
	}
}
