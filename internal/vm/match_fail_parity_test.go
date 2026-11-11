package vm_test

import (
	"errors"
	"strings"
	"testing"

	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/types"
)

// M-V1-MATCH-EXHAUSTIVENESS M0, in process: a match with no matching arm is
// RT002 on the evaluator and on the strict VM, with the same position and
// shape, reachable with errors.As. Before MATCH_FAIL the VM yielded Unit
// (these entries returned it and "passed"). The binary-level twin is
// cmd/ailang/match_fail_parity_test.go.
//
// Every match here is deliberately non-exhaustive, which `check` now rejects
// (TC_MATCH_002); RT002 is defence in depth for checker bugs. The test turns
// the check's errors off to reach it, with the compile cache off so that no
// module compiled that way is served to a later compile.
func TestMatchFailBothEngines(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	defer types.SetMatchExhaustivenessErrorsForTest(false)()
	cases := []struct {
		name, src, pos, ctor, shape string
	}{
		{
			name: "switch_default",
			src: `module switch_default
type Color = Red | Green | Blue
func code(c: Color) -> int = match c { Red => 1, Green => 2 }
export func main() -> int = code(Blue)
`,
			pos: ":3:30", ctor: "Blue",
		},
		{
			name: "guard_fallback",
			src: `module guard_fallback
type Opt = Has(int) | Nothing
func g(o: Opt) -> int = match o { Has(x) if x > 0 => x, Nothing => 0 }
export func main() -> int = g(Has(0))
`,
			pos: ":3:25", ctor: "Has",
		},
		{
			name: "ifchain_list",
			src: `module ifchain_list
func pair(xs: [int]) -> int = match xs { [a] => a, [a, b] => a + b }
export func main() -> int = pair([1, 2, 3])
`,
			pos: ":2:31", shape: "list of 3 elements",
		},
		{
			name: "ifchain_literal",
			src: `module ifchain_literal
func name(n: int) -> string = match n { 0 => "zero", 1 => "one" }
export func main() -> string = name(7)
`,
			pos: ":2:31", shape: "int",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, evalErr, vmOut, vmErr := runBothEngines(t, tc.name, tc.src)
			var texts []string
			for engine, err := range map[string]error{"evaluator": evalErr, "vm": vmErr} {
				var mf *ailerrors.MatchFailureError
				if !errors.As(err, &mf) {
					t.Fatalf("%s: want RT002 *MatchFailureError, got %v (vm output %q)", engine, err, vmOut)
				}
				if !strings.HasSuffix(mf.Pos, tc.name+".ail"+tc.pos) || mf.Ctor != tc.ctor || mf.Shape != tc.shape {
					t.Errorf("%s: %+v, want pos …%s.ail%s ctor %q shape %q", engine, mf, tc.name, tc.pos, tc.ctor, tc.shape)
				}
				texts = append(texts, mf.Error())
			}
			if texts[0] != texts[1] {
				t.Errorf("engines disagree:\n%s\n%s", texts[0], texts[1])
			}
		})
	}
}
