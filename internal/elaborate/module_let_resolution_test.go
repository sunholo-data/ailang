package elaborate_test

// Behavior matrix for M-MODULE-LET-FUNC-RESOLUTION (#366): a module-level
// let/letrec value must resolve module-scope names exactly as a func body does.
// This is the 4th member of the "#323/#327 resolution diverges by syntactic
// position" family, at the DECL-CLASS level (module-let vs func) rather than the
// expression-position level that #327 fixed.
//
// The test lives in the external test package (elaborate_test) so it may drive
// the real compile pipeline (internal/pipeline imports internal/elaborate, so an
// internal test here would cycle). Each case is checked through pipeline.Run in
// ModeCheck — exactly what an agent's `ailang check` exercises.
//
// NON-VACUITY: shapes v3/v4/v8/v10 FAIL at pre-fix HEAD (false "undefined
// variable" / silent dup) and PASS/error-correctly here. The sprint's base-binary
// check demonstrates the pre-fix failures on the origin/dev binary.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
)

// checkModule writes src to a temp .ail file whose on-disk path matches modPath
// (so MOD010 does not fire) and runs it through the ModeCheck pipeline. Returns
// the check error (nil on success).
func checkModule(t *testing.T, modPath, src string) error {
	t.Helper()
	dir := t.TempDir()
	// modPath like "bench/solution" -> dir/bench/solution.ail
	full := filepath.Join(dir, filepath.FromSlash(modPath)+".ail")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := pipeline.Run(
		pipeline.Config{Mode: pipeline.ModeCheck, PackageDir: dir},
		pipeline.Source{Code: src, Filename: full},
	)
	return err
}

func TestModuleLetResolutionMatrix(t *testing.T) {
	cases := []struct {
		name    string
		mod     string
		src     string
		wantErr string // "" = must compile clean; else error must contain this
	}{
		{
			// v1: module let -> earlier module let (green pre-fix, stays green)
			name: "v1_let_to_earlier_let",
			mod:  "m/v1",
			src: "module m/v1\n" +
				"let a = 10\n" +
				"let b = a + 5\n" +
				"export func main() -> int = b\n",
		},
		{
			// v2: module func -> module let (M-BUG-MODULE-LET-SCOPE; stays green)
			name: "v2_func_to_let",
			mod:  "m/v2",
			src: "module m/v2\n" +
				"let base = 100\n" +
				"export func getBase() -> int = base\n" +
				"export func main() -> int = getBase()\n",
		},
		{
			// v3: module let -> module func, FUNC DECLARED FIRST (was false error)
			name: "v3_let_to_func_func_first",
			mod:  "m/v3",
			src: "module m/v3\n" +
				"export func double(x: int) -> int = x * 2\n" +
				"let quad = \\y. double(double(y))\n" +
				"export func main() -> int = quad(4)\n",
		},
		{
			// v4: module let -> module func, FUNC DECLARED AFTER (was false error)
			name: "v4_let_to_func_func_after",
			mod:  "m/v4",
			src: "module m/v4\n" +
				"let quad = \\y. double(double(y))\n" +
				"export func double(x: int) -> int = x * 2\n" +
				"export func main() -> int = quad(4)\n",
		},
		{
			// v5: module let -> IMPORTED func (green pre-fix, stays green).
			// std/list (length) is a stable stdlib export.
			name: "v5_let_to_imported_func",
			mod:  "m/v5",
			src: "module m/v5\n" +
				"import std/list (length)\n" +
				"let n = length([1, 2, 3])\n" +
				"export func main() -> int = n\n",
		},
		{
			// v6: module let -> let, both orders in one chain (stays green)
			name: "v6_let_chain",
			mod:  "m/v6",
			src: "module m/v6\n" +
				"let x = 1\n" +
				"let y = x + 1\n" +
				"let z = y + x\n" +
				"export func main() -> int = z\n",
		},
		{
			// v7: module letrec self-ref, RECURSIVE LAMBDA -> compiles (LetRec)
			name: "v7_letrec_recursive_lambda",
			mod:  "m/v7",
			src: "module m/v7\n" +
				"letrec countdown = \\n. if n <= 0 then 0 else countdown(n - 1)\n" +
				"export func main() -> int = countdown(5)\n",
		},
		{
			// v8: module let = IMMEDIATE CALL of a module func (was false error)
			name: "v8_let_immediate_call",
			mod:  "m/v8",
			src: "module m/v8\n" +
				"export func double(x: int) -> int = x * 2\n" +
				"let four = double(2)\n" +
				"export func main() -> int = four\n",
		},
		{
			// v9: export let -> dedicated honest error, unchanged by this work
			name:    "v9_export_let_honest_error",
			mod:     "m/v9",
			src:     "module m/v9\n" + "export let k = 5\n" + "export func main() -> int = k\n",
			wantErr: "EXPORT_LET",
		},
		{
			// v10: let + func SAME NAME -> MOD007 (was silent pre-fix)
			name:    "v10_dup_let_func_MOD007",
			mod:     "m/v10",
			src:     "module m/v10\n" + "let dup = 5\n" + "export func dup() -> int = 10\n" + "export func main() -> int = dup()\n",
			wantErr: "MOD007",
		},
		{
			// v10b: let + let SAME NAME -> MOD007
			name:    "v10b_dup_let_let_MOD007",
			mod:     "m/v10b",
			src:     "module m/v10b\n" + "let dup = 5\n" + "let dup = 6\n" + "export func main() -> int = dup\n",
			wantErr: "MOD007",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkModule(t, tc.mod, tc.src)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected clean compile, got error:\n%v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got clean compile", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got:\n%v", tc.wantErr, err)
			}
		})
	}
}
