package pipeline

import (
	"fmt"
	"strings"
	"testing"
)

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
