package types_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
)

// Keep the invalid reporter fixture embedded: runnable examples must typecheck.
func TestSwappedMapLookupDiagnostic(t *testing.T) {
	const source = `module swapped
import std/map (fromList, lookup)
export func main() -> int {
  let m = fromList([("a", 1), ("b", 2)]);
  match lookup("b", m) { Some(v) => v, None => 0 }
}
`
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", filepath.Join(root, "std"))
	file := filepath.Join(t.TempDir(), "swapped.ail")
	if err := os.WriteFile(file, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, NoCache: true}, pipeline.Source{Filename: file})
	if err == nil {
		t.Fatal("swapped std/map.lookup arguments accepted")
	}
	message := filepath.ToSlash(err.Error())
	for _, want := range []string{"application of lookup", "swapped.ail:5:"} {
		if !strings.Contains(message, want) {
			t.Errorf("diagnostic %q missing %q", message, want)
		}
	}
}
