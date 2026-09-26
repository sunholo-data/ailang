package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/mission/dispatch"
)

func TestIterationReceiptProgressCrashAndTruncatedTail(t *testing.T) {
	dir, canonicalErr := filepath.EvalSymlinks(t.TempDir())
	if canonicalErr != nil {
		t.Fatal(canonicalErr)
	}
	path := filepath.Join(dir, "review-"+strings.Repeat("a", 32)+".jsonl")
	event := dispatch.Event{Version: 1, Kind: "progress", RequestDigest: "digest", Report: &dispatch.Report{RequestDigest: "digest", Progress: &dispatch.Progress{ToolCalls: 4, RepeatedCalls: 2, LastCompletedTool: "read"}}}
	b, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	for _, tail := range []string{"", "{broken", "{broken}\n"} {
		if err = os.WriteFile(path, append(append(b, '\n'), []byte(tail)...), 0600); err != nil {
			t.Fatal(err)
		}
		p, state := readIterationReceiptProgress(dir, "review", "digest")
		if p == nil || p.ToolCalls != 4 {
			t.Fatalf("lost crash progress: %+v %s", p, state)
		}
		if tail != "" && !strings.Contains(state, "incomplete") {
			t.Fatalf("tail misrepresented: %s", state)
		}
	}
	if p, _ := readIterationReceiptProgress(dir, "review", "different"); p != nil {
		t.Fatal("unrelated digest exposed")
	}
}

func TestIterationReceiptProgressRejectsUnsafeOrOversizedEvidence(t *testing.T) {
	dir, canonicalErr := filepath.EvalSymlinks(t.TempDir())
	if canonicalErr != nil {
		t.Fatal(canonicalErr)
	}
	path := filepath.Join(dir, "review-"+strings.Repeat("a", 32)+".jsonl")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 4*1024*1024+1)), 0600); err != nil {
		t.Fatal(err)
	}
	if p, state := readIterationReceiptProgress(dir, "review", "digest"); p != nil || !strings.Contains(state, "bound") {
		t.Fatalf("unbounded evidence: %+v %s", p, state)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "private")
	if err := os.WriteFile(target, []byte("SECRET"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if p, state := readIterationReceiptProgress(dir, "review", "digest"); p != nil || strings.Contains(state, "SECRET") {
		t.Fatalf("unsafe evidence: %+v %s", p, state)
	}
}
