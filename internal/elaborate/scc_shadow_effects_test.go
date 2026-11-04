package elaborate_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Check the original report and every design-matrix control through ModeCheck.
// Contracts/inline tests are unnecessary here: the Go assertions check compiler
// acceptance and diagnostics, including intentionally unsound effect rows.
func TestSCCShadowEffects(t *testing.T) {
	each := `func each[e](f: (int) -> () ! {e}, xs: [int]) -> () ! {e} = match xs { [] => (), h :: t => { f(h); each(f, t) } }`
	caller := `export func t() -> () ! {IO} = each(\x. println("x"), [1, 2])`
	cases := []struct {
		name, code string
		reject     bool
	}{
		{"01_issue", each + "\n" + caller, false},
		{"02_alone", each, false},
		{"03_renamed_caller", each + "\n" + strings.Replace(caller, "func t()", "func u()", 1), false},
		{"04_renamed_pattern", strings.ReplaceAll(strings.ReplaceAll(each, "h :: t", "h :: rest"), "each(f, t)", "each(f, rest)") + "\n" + caller, false},
		{"05_lambda_shadow", `func each[e](f: (int) -> () ! {e}, xs: [int]) -> () ! {e} = match xs { [] => (), h :: r => { f(h); each(\t. f(t), r) } }` + "\n" + caller, false},
		{"06_let_shadow", `func each[e](f: (int) -> () ! {e}, xs: [int]) -> () ! {e} = match xs { [] => (), h :: r => { f(h); let t = r in each(f, t) } }` + "\n" + caller, false},
		{"07_pure_caller", each + "\n" + `export func t() -> () = each(\x. (), [1, 2])`, false},
		{"08_noncalling_sibling", each + "\n" + `export func t() -> () ! {IO} = println("x")`, false},
		{"09_mutual", `func even[e](f: (int) -> () ! {e}, n: int) -> () ! {e} = if n == 0 then () else { f(n); odd(f, n - 1) }
func odd[e](f: (int) -> () ! {e}, n: int) -> () ! {e} = if n == 0 then () else { f(n); even(f, n - 1) }`, false},
		{"10_true_scc_closed_row", `func a[e](f: (int) -> () ! {e}) -> () ! {e} = { f(1); b(\x. println("boom")) }
func b[e](f: (int) -> () ! {e}) -> () ! {e} = { f(1); a(f) }`, true},
		{"11_self_closed_row", `func d[e](f: (int) -> () ! {e}) -> () ! {e} = { f(1); d(\x. println("boom")) }`, true},
		{"12_let_discard", strings.Replace(each, "f(h);", "let _ = f(h);", 1) + "\n" + caller, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			mod := "scc/" + "r" + tt.name
			src := fmt.Sprintf("module %s\nimport std/io (println)\n%s\n", mod, tt.code)
			// Optional fixture export supports the base/candidate CLI differential check.
			if dir := os.Getenv("SCC_FIXTURE_DIR"); dir != "" {
				full := filepath.Join(dir, filepath.FromSlash(mod)+".ail")
				if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(full, []byte(src), 0644); err != nil {
					t.Fatal(err)
				}
			}
			err := checkModule(t, mod, src)
			if tt.reject {
				if err == nil || !strings.Contains(err.Error(), "Missing effects") || !strings.Contains(err.Error(), "IO") {
					t.Fatalf("want missing IO effects, got %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}
