package eval_harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractCode_FencedBlock(t *testing.T) {
	resp := `Here's my solution:

` + "```ailang" + `
module benchmark/solution
import std/io (println)
export func main() -> () ! {IO} = println("hello")
` + "```" + `

That should work.`
	got := extractCode(resp, "/x/benchmark/solution.ail")
	if !strings.Contains(got, "module benchmark/solution") {
		t.Errorf("extractCode missed module decl: %q", got)
	}
	if !strings.Contains(got, "println(\"hello\")") {
		t.Errorf("extractCode missed body: %q", got)
	}
	if strings.Contains(got, "```") {
		t.Errorf("extractCode kept backticks: %q", got)
	}
}

func TestExtractCode_LastBlockWins(t *testing.T) {
	resp := "First attempt:\n```\nbroken\n```\n\nFinal:\n```ailang\nmodule benchmark/solution\nexport func main() -> () = ()\n```\n"
	got := extractCode(resp, "/x/benchmark/solution.ail")
	if strings.Contains(got, "broken") {
		t.Errorf("extractCode should pick LAST block, not first: %q", got)
	}
	if !strings.Contains(got, "module benchmark/solution") {
		t.Errorf("extractCode missed last block: %q", got)
	}
}

func TestExtractCode_RawAILANGNoFence(t *testing.T) {
	// Agent dumps raw code without fences — accept it if it looks like a module.
	resp := "module benchmark/solution\n\nexport func main() -> () = ()\n"
	got := extractCode(resp, "/x/benchmark/solution.ail")
	if !strings.Contains(got, "module benchmark/solution") {
		t.Errorf("extractCode raw-AILANG path failed: %q", got)
	}
}

func TestExtractCode_NoCode(t *testing.T) {
	resp := "Sorry, I cannot complete this task."
	if got := extractCode(resp, "/x/benchmark/solution.ail"); got != "" {
		t.Errorf("extractCode should return empty for non-code response, got: %q", got)
	}
}

// TestExtractCode_ChainOfThoughtNoFences pins the regression where chain-of-
// thought commentary that MENTIONS the word "module" (e.g. "I will view the
// module file") was misclassified as raw code because the fallback strategy
// matched on `strings.Contains(response, "module ")`. After the fix, raw-code
// detection requires the FIRST non-comment line to START with `module`.
//
// Captured from a live gemini-3-5-flash run on adt_option (smoke v3) where
// the agent narrated its tool plan but never fenced its final code.
func TestExtractCode_ChainOfThoughtNoFences(t *testing.T) {
	resp := `I will start by viewing the target file to understand if there is any template or existing code that I should build upon.

I will list the workspace directory to get an overview of the module structure.

Let me think about what an Option module would look like in AILANG...`
	got := extractCode(resp, "/x/benchmark/solution.ail")
	if got != "" {
		t.Errorf("chain-of-thought commentary should NOT be extracted as code, got %d chars: %q", len(got), got[:min(200, len(got))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestWriteSolutionFromResponse_HappyPath(t *testing.T) {
	dir := t.TempDir()
	bench := filepath.Join(dir, "benchmark")
	if err := os.MkdirAll(bench, 0755); err != nil {
		t.Fatal(err)
	}
	placeholder := "module benchmark/solution\n// TODO\n"
	solutionPath := filepath.Join(bench, "solution.ail")
	if err := os.WriteFile(solutionPath, []byte(placeholder), 0644); err != nil {
		t.Fatal(err)
	}

	resp := "Here's the solution:\n```ailang\nmodule benchmark/solution\nexport func main() -> () = ()\n```\n"
	path, n, err := writeSolutionFromResponse(dir, resp)
	if err != nil {
		t.Fatalf("writeSolutionFromResponse: %v", err)
	}
	if path != solutionPath {
		t.Errorf("path=%q, want %q", path, solutionPath)
	}
	if n == 0 {
		t.Error("bytesWritten=0, want >0")
	}

	got, _ := os.ReadFile(solutionPath)
	if !strings.Contains(string(got), "export func main") {
		t.Errorf("file content not updated: %q", string(got))
	}
}

func TestWriteSolutionFromResponse_NoPlaceholder(t *testing.T) {
	// Empty workspace — no placeholder file present. Should be a no-op,
	// not an error.
	dir := t.TempDir()
	path, n, err := writeSolutionFromResponse(dir, "module foo\n```ailang\nmodule x\n```")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "" || n != 0 {
		t.Errorf("expected no-op (path=\"\", n=0), got path=%q n=%d", path, n)
	}
}

func TestWriteSolutionFromResponse_EmptyResponse(t *testing.T) {
	dir := t.TempDir()
	bench := filepath.Join(dir, "benchmark")
	_ = os.MkdirAll(bench, 0755)
	_ = os.WriteFile(filepath.Join(bench, "solution.ail"), []byte("placeholder"), 0644)

	path, n, err := writeSolutionFromResponse(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if path != "" || n != 0 {
		t.Errorf("empty response should no-op, got path=%q n=%d", path, n)
	}
}
