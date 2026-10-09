package lsp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInlineRowDiagnosticLocation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad_row.ail")
	const src = `module bad_row
export func id(x: int) -> int ! {}
tests [ (1 + 2, 3) ]
{ x }
`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	ds := runPipelineForDiagnostics(path, src)
	if len(ds) != 1 || ds[0].Code != "TST002" || ds[0].Range.Start.Line != 2 {
		t.Fatalf("%+v", ds)
	}
}
