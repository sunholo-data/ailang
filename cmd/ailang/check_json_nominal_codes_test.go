package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/types"
)

// Every diagnostic code M-V1-NOMINAL-TYPE-IDENTITY introduced is reported by
// `check --json` as itself, not the generic "ERROR" (review round 2): each
// program below is compiled for real and its error converted exactly as
// `check --json` converts it.
func TestCheckJSON_NominalIdentityCodesAppearAsThemselves(t *testing.T) {
	a := "module q/a\nexport type T = C(int) | D\nexport pure func mka() -> T = C(1)\n"
	b := "module q/b\nexport type T = C(string) | E\nexport pure func mkb() -> T = C(\"s\")\n"
	cases := map[string]string{
		types.TC_NOMINAL_001:        "module q/main\nimport q/a (mka)\nimport q/b (T)\npure func f(t: T) -> int = 0\nexport pure func run() -> int = f(mka())\n",
		types.TC_AMBIGUOUS_TYPE_001: "module q/main\nimport q/a (mka)\nimport q/b (mkb)\npure func f(t: T) -> int = 0\nexport pure func run() -> int = 0\n",
		types.TC_AMBIGUOUS_CTOR_001: "module q/main\nimport q/a (mka)\nimport q/b (mkb)\nexport pure func run() -> int = match C(1) { _ => 0 }\n",
		"MOD015":                    "module q/main\nimport q/a (T)\nimport q/b (T)\nexport pure func run() -> int = 0\n",
		// #2211: a lowercase name applied to arguments that names no type.
		types.TC_UNBOUND_TYPE_001: "module q/main\nimport q/a (mka)\npure func f(t: zz[int]) -> int = 0\nexport pure func run() -> int = 0\n",
		// #2211: a type declaration whose body names a variable that is not its parameter.
		types.TC_UNBOUND_TYPE_003: "module q/main\nimport q/a (mka)\ntype Count = integer\nexport pure func run() -> int = 0\n",
		// A type alias that refers to itself.
		types.TC_TYPE_ALIAS_CYCLE_001: "module q/main\nimport q/a (mka)\ntype L = [L]\nexport pure func run() -> int = 0\n",
	}
	for want, main := range cases {
		t.Run(want, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			for rel, src := range map[string]string{"q/a.ail": a, "q/b.ail": b, "q/main.ail": main} {
				if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(rel, []byte(src), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			_, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, NoCache: true}, pipeline.Source{Filename: "q/main.ail"})
			if err == nil {
				t.Fatalf("expected %s", want)
			}
			if got := errorToCheckJSONError(err, "q/main.ail").Code; got != want {
				t.Fatalf("--json code = %q, want %q (message: %v)", got, want, err)
			}
			if got := jsonErrorCode(err.Error()); want != "MOD015" && got != want {
				t.Fatalf("package/dir --json code = %q, want %q", got, want)
			}
		})
	}
	for _, code := range types.NominalIdentityCodes() {
		if _, ok := cases[code]; !ok {
			t.Errorf("new code %s has no --json case", code)
		}
	}
	// Review round 3: an error whose cause carries another code is coded by
	// its own (first) code.
	nested := "type error in q/main (decl 0): " + types.TC_AMBIGUOUS_CTOR_001 + " at q/main.ail:4:5: …\n  Caused: … " + types.TC_NOMINAL_001 + ": cannot unify T …"
	if got := jsonErrorCode(nested); got != types.TC_AMBIGUOUS_CTOR_001 {
		t.Errorf("nested codes: got %q, want the error's own %s", got, types.TC_AMBIGUOUS_CTOR_001)
	}
}
