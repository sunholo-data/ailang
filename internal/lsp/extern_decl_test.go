package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// EXT004 (#2390) reaches the editor as a positioned diagnostic at the extern
// declaration, though nothing calls it.
func TestExternDeclDiagnostic(t *testing.T) {
	mainPath := filepath.Join("testdata", "externdecl", "main.ail")
	src, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatal(err)
	}
	diags := runPipelineForDiagnostics(mainPath, string(src))
	for _, d := range diags {
		if code, _ := d.Code.(string); code == "EXT004" {
			if d.Range.Start.Line != 2 { // 0-based: `extern func hostAdd` is line 3
				t.Errorf("EXT004 placed at line %d, want 2 (the declaration)", d.Range.Start.Line)
			}
			if !strings.Contains(d.Message, "'hostAdd'") {
				t.Errorf("EXT004 message does not name the function: %q", d.Message)
			}
			return
		}
	}
	t.Fatalf("no EXT004 diagnostic; got %+v", diags)
}
