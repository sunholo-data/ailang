package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExpectedFail_PureWithEffectRow pins examples/expected_fail/pure_with_effect_row.ail
// (#1443): the issue's `sneaky` repro must stay rejected with the actionable diagnostic,
// and its accepted twin examples/runnable/pure_means_empty_row.ail must stay clean with
// interface purity derived from each export's row.
func TestExpectedFail_PureWithEffectRow(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatal("repository go.mod not found")
		}
		root = parent
	}
	t.Chdir(root)

	_, err = Run(Config{Mode: ModeCheck}, Source{Filename: "examples/expected_fail/pure_with_effect_row.ail"})
	if err == nil {
		t.Fatal("pure func with ! {IO} accepted: #1443 regressed")
	}
	for _, part := range []string{"'sneaky'", "pure keyword contradicts", "! {IO}", "remove pure"} {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("diagnostic missing %q: %v", part, err)
		}
	}

	result, err := Run(Config{Mode: ModeCheck}, Source{Filename: "examples/runnable/pure_means_empty_row.ail"})
	if err != nil {
		t.Fatalf("accepted spellings rejected: %v", err)
	}
	want := map[string]bool{"double": true, "clampPct": true, "redact": false, "report": false, "main": false}
	for name, pure := range want {
		item := result.Interface.Exports[name]
		if item == nil {
			t.Fatalf("export %s missing from interface", name)
		}
		if item.Purity != pure {
			t.Errorf("iface purity of %s = %v, want %v", name, item.Purity, pure)
		}
	}
}

func TestPureSignature_DeclaredRows(t *testing.T) {
	for _, tc := range []struct {
		name, row string
		reject    bool
	}{
		{"IO", " ! {IO}", true}, {"Debug", " ! {Debug}", true},
		{"Declassify", " ! {Declassify}", true}, {"scoped", " ! {Declassify[label=x]}", true},
		{"variable", " ! {e}", true}, {"mixed", " ! {IO, e}", true},
		{"empty", " ! {}", false}, {"absent", "", false},
	} {
		for _, pure := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/pure=%v", tc.name, pure), func(t *testing.T) {
				keyword := ""
				if pure {
					keyword = "pure "
				}
				src := "module main\nexport " + keyword + "func probe[e](x: int) -> int" + tc.row + " { x }"
				err := checkModules(t, map[string]string{"main.ail": src})
				if pure && tc.reject {
					if err == nil {
						t.Fatal("contradictory pure declaration accepted")
					}
					for _, part := range []string{"probe", "pure", "contradict", "remove", "! {}"} {
						if !strings.Contains(err.Error(), part) {
							t.Errorf("diagnostic missing %q: %v", part, err)
						}
					}
				} else if err != nil {
					t.Fatalf("valid declaration rejected: %v", err)
				}
			})
		}
	}
}
