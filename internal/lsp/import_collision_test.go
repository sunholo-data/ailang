package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MOD015 (#1467) reaches the editor as a positioned Error diagnostic at the
// local definition — not the position-less "ERROR" fallback at 0:0. Module
// paths resolve from the package directory (the test's working directory).
func TestImportLocalCollisionDiagnostic(t *testing.T) {
	mainPath := filepath.Join("testdata", "importcollide", "main.ail")
	src, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatal(err)
	}

	diags := runPipelineForDiagnostics(mainPath, string(src))
	for _, d := range diags {
		if code, _ := d.Code.(string); code == "MOD015" {
			if d.Range.Start.Line != 4 { // 0-based: `pure func tick` is line 5
				t.Errorf("MOD015 placed at line %d, want 4 (the local definition)", d.Range.Start.Line)
			}
			if !strings.Contains(d.Message, "'tick'") {
				t.Errorf("MOD015 message does not name the symbol: %q", d.Message)
			}
			return
		}
	}
	t.Fatalf("no MOD015 diagnostic; got %+v", diags)
}
