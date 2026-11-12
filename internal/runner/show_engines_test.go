package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestShowShapesAllEngines runs testdata/show_shapes.ail — one line per value
// shape `show` renders, plus interpolation and showPreview — on the evaluator,
// the bytecode VM and the strict bytecode VM. Every engine must print
// show_shapes.golden byte for byte (M-V1-SHOW-TOTAL). The value-level half of
// the differential (spec table, generated values, cost) is
// internal/vm/show_engines_test.go.
func TestShowShapesAllEngines(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("testdata", "show_shapes.ail"))
	if err != nil {
		t.Fatal(err)
	}
	golden, err := os.ReadFile(filepath.Join("testdata", "show_shapes.golden"))
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	file := filepath.Join(t.TempDir(), "show_shapes.ail")
	for _, engine := range allEngines {
		t.Run(engine, func(t *testing.T) {
			if got := runOnEngine(t, file, string(src), engine); got != string(golden) {
				t.Fatalf("%s differs from show_shapes.golden: %s", engine, firstLineDiff(got, string(golden)))
			}
		})
	}
}

func firstLineDiff(got, want string) string {
	g, w := strings.Split(got, "\n"), strings.Split(want, "\n")
	for i := 0; i < len(g) || i < len(w); i++ {
		var gl, wl string
		if i < len(g) {
			gl = g[i]
		}
		if i < len(w) {
			wl = w[i]
		}
		if gl != wl {
			return fmt.Sprintf("line %d\n got %q\nwant %q", i+1, gl, wl)
		}
	}
	return "(no line differs)"
}
