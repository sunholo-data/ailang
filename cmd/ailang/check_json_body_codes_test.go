package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/pipeline"
)

// #2232 / #2248: an ill-typed test body or contract reaches `check --json`
// under its own code, with a line, in file mode and in directory mode.
func TestCheckJSON_BodyCheckCodes(t *testing.T) {
	cases := map[string]string{
		elaborate.CodeTestBody: "module m\nexport pure func k(x: int) -> int = x\ntest \"t\" { assert k(\"x\") == 1 }\n",
		pipeline.CodeContract:  "module m\nexport func total(n: int) -> int ! {}\nrequires { forall i: 0..2 => i + \"x\" }\n{ 1 }\n",
	}
	for want, src := range cases {
		t.Run(want, func(t *testing.T) {
			t.Chdir(t.TempDir())
			if err := os.WriteFile("m.ail", []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, NoCache: true}, pipeline.Source{Filename: "m.ail"})
			if err == nil {
				t.Fatalf("expected %s", want)
			}
			got := errorToCheckJSONError(err, "m.ail")
			if got.Code != want || got.Line == 0 {
				t.Fatalf("--json = %+v, want code %s with a line (error: %v)", got, want, err)
			}
			if dir := jsonErrorCode(err.Error()); dir != want {
				t.Fatalf("directory --json code = %q, want %q", dir, want)
			}
		})
	}
}

// The missing-Eq message names the declaration the user must edit (with
// forward slashes on every OS, whether the file is the root or was loaded), through
// the real pipeline: the declaration sites are recorded per loaded module
// (pipeline.recordTypeDeclSites), in this module and in another module of the
// same package. The contract note appears only for a contract clause.
func TestCheckJSON_MissingEqNamesTheDeclaration(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	files := map[string]string{
		"q/types.ail": "module q/types\nexport type Grade = A | B\nexport type Pt = {x: int, y: int}\n",
		"q/main.ail":  "module q/main\nimport q/types (Grade, A, B)\nexport pure func g(n: int) -> Grade\n  ensures { result == A }\n{ A }\n",
		"q/rec.ail":   "module q/rec\nimport q/types (Pt)\nexport pure func same(p: Pt, r: Pt) -> bool = p == r\n",
		"q/local.ail": "module q/local\ntype Color = Red | Green\nexport pure func same(c: Color) -> bool = c == Red\n",
		"q/plain.ail": "module q/plain\nexport pure func f(n: int) -> int\n  requires { n + \"x\" > 0 }\n{ n }\n",
	}
	for rel, src := range files {
		if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(rel, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		file, code string
		want       []string
		avoid      []string
	}{
		{"q/main.ail", pipeline.CodeContract, []string{"Add `deriving (Eq)` to type Grade (declared at q/types.ail:2)", "type Grade = A | B deriving (Eq)", "requires/ensures clause exactly as in a function body"}, nil},
		{"q/rec.ail", "ERROR", []string{"Add `deriving (Eq)` to type Pt (declared at q/types.ail:3)", "type Pt = {x: int, y: int} deriving (Eq)"}, []string{"A | B", "requires/ensures"}},
		{"q/local.ail", "ERROR", []string{"Add `deriving (Eq)` to type Color (declared at q/local.ail:2)"}, []string{"requires/ensures"}},
		{"q/plain.ail", pipeline.CodeContract, []string{"cannot unify"}, []string{"== needs Eq"}},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			_, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, NoCache: true}, pipeline.Source{Filename: c.file})
			if err == nil {
				t.Fatal("want a missing-Eq error")
			}
			got := errorToCheckJSONError(err, c.file)
			if got.Code != c.code {
				t.Errorf("code = %q, want %q", got.Code, c.code)
			}
			for _, w := range c.want {
				if !strings.Contains(got.Message, w) {
					t.Errorf("--json message lacks %q: %s", w, got.Message)
				}
			}
			for _, a := range c.avoid {
				if strings.Contains(got.Message, a) {
					t.Errorf("--json message must not say %q: %s", a, got.Message)
				}
			}
		})
	}
}
