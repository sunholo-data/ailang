package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// examples/runnable/web_search.ail is skipped by verify-examples (it needs
// OLLAMA_API_KEY and the network), so its type-check is asserted here — the
// example must never rot silently. Runs from the file's directory with the
// bare name, exactly as `ailang check web_search.ail` does.
func TestWebSearchExample_TypeChecks(t *testing.T) {
	dir, _ := filepath.Abs(filepath.Join("..", "..", "examples", "runnable"))
	t.Chdir(dir)
	src, err := os.ReadFile("web_search.ail")
	if err != nil {
		t.Fatal(err)
	}
	res, err := RunWithContext(context.Background(), Config{DryLink: true}, Source{Code: string(src), Filename: "web_search.ail"})
	if err != nil {
		t.Fatalf("web_search.ail must type-check: %v", err)
	}
	if res.Artifacts.Core == nil {
		t.Fatal("no core artifact — the example did not compile")
	}
}
