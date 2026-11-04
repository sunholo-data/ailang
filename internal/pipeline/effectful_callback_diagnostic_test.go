package pipeline

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/stdlibindex"
	"github.com/sunholo-data/ailang/internal/types"
)

func TestEffectfulCallbackDiagnostic(t *testing.T) {
	oldImport, oldVariant := types.ImportSuggester, types.EffectfulVariantSuggester
	t.Cleanup(func() { types.ImportSuggester, types.EffectfulVariantSuggester = oldImport, oldVariant })
	types.ImportSuggester, types.EffectfulVariantSuggester = stdlibindex.Modules, stdlibindex.EffectVariants
	for _, tc := range []struct {
		name, callee, arg, list string
		accept, effect          bool
	}{
		{"map_lambda", "map", `\x. println(x)`, `["a", "b"]`, false, true},
		{"map_named", "map", "println", `["a"]`, false, true},
		{"filter_lambda", "filter", `\x. { println(x); true }`, `["a"]`, false, true},
		{"map_type", "map", `\x. x`, `"abc"`, false, false},
		{"map_remedy", "mapE", `\x. println(x)`, `["a", "b"]`, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := fmt.Sprintf("module %s\nimport std/list (%s)\nimport std/io (println)\nexport func main() -> () ! {IO} { let _ = %s(%s, %s); () }", tc.name, tc.callee, tc.callee, tc.arg, tc.list)
			err := check386(t, tc.name, src)
			if tc.accept {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("invalid callback accepted")
			}
			msg := err.Error()
			if !tc.effect {
				if !strings.Contains(msg, "also exported by") {
					t.Fatal(msg)
				}
				return
			}
			for _, want := range []string{"`" + tc.callee + "` requires parameter 0", "(pure)", "IO", "std/list." + tc.callee + "E", "(solver: type unification failed", "failed to unify parameter 0: failed to unify effect rows:"} {
				if !strings.Contains(msg, want) {
					t.Errorf("%s missing %s", msg, want)
				}
			}
			if strings.Contains(msg, "also exported by") {
				t.Fatal(msg)
			}
			if strings.Contains(src, `\x.`) && !strings.Contains(msg, "this lambda (at ") {
				t.Fatal(msg)
			}
			if tc.arg == "println" && !strings.Contains(msg, "this argument (at ") {
				t.Fatal(msg)
			}
		})
	}
}
