package elaborate_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
)

// Two modules both offering constructors C (one field) and N (none), never
// imported by name: the elaborator marks them ambiguous (ambiguous_ctor.go).
const (
	ambA = "module m/a\nexport type T = C(int) | N\nexport pure func mka() -> T = C(1)\n"
	ambB = "module m/b\nexport type U = C(int) | N | K(int)\nexport pure func mkb() -> U = C(41)\n"
)

func checkAmbiguous(t *testing.T, imports, body string) error {
	t.Helper()
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", filepath.Join(repo, "std"))
	root := t.TempDir()
	t.Chdir(root)
	for rel, src := range map[string]string{"m/a.ail": ambA, "m/b.ail": ambB, "m/main.ail": "module m/main\n" + imports + body} {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, err = pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, NoCache: true}, pipeline.Source{Filename: "m/main.ail"})
	return err
}

func TestAmbiguousConstructor_Elaboration(t *testing.T) {
	auto := "import m/a (mka)\nimport m/b (mkb)\n"
	for _, c := range []struct {
		name, imports, body string
		want                []string // empty: must check
	}{
		{"a bare nullary pattern is left to the typechecker", auto,
			"export pure func run() -> int = match mkb() { N => 1, _ => 0 }\n", nil},
		{"a constructor offered by one module only is not ambiguous", auto,
			"export pure func run() -> int = match mkb() { K(n) => n, _ => 0 }\n", nil},
		{"a bare pattern name no candidate can take without arguments", auto,
			"export pure func run() -> int = match mkb() { C => 1, _ => 0 }\n", []string{"expects argument(s)"}},
		{"a constructor call", auto,
			"export pure func run() -> int = match C(1) { _ => 0 }\n", []string{"TC_AMBIGUOUS_CTOR_001", "m/main.ail:", "m/a's T and m/b's U"}},
		{"a nullary constructor value", auto,
			"export pure func run() -> int = match N { _ => 0 }\n", []string{"TC_AMBIGUOUS_CTOR_001", "m/main.ail:"}},
		{"importing it by name decides", "import m/a (mka)\nimport m/b (mkb, C)\n",
			"export pure func run() -> int = match C(1) { C(n) => n, _ => 0 }\n", nil},
		{"two explicit imports of one type", "import m/a (T)\nimport m/b (U as T)\n",
			"export pure func run() -> int = 0\n", []string{"MOD015", "imported twice", "from m/a", "from m/b", "alias one of them"}},
		{"two explicit imports of one constructor", "import m/a (C)\nimport m/b (C)\n",
			"export pure func run() -> int = 0\n", []string{"MOD015", "constructor 'C'", "import m/b (C as BC)"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := checkAmbiguous(t, c.imports, c.body)
			if len(c.want) == 0 {
				if err != nil {
					t.Fatalf("must check: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("must fail with %v", c.want)
			}
			for _, w := range c.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("diagnostic missing %q: %v", w, err)
				}
			}
		})
	}
}
