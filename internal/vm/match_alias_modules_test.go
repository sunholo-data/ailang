package vm_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #2244 review: constructors matched BY NAME (#323) — their type is loaded
// only through another module, so the elaborator cannot stamp them — against
// a value whose type is an imported alias. The VM must dispatch on the alias's
// ADT, which the type checker stamps on each such pattern; before, it took the
// written type (`unknown ADT "Hue" in switch`, so --strict-bytecode refused
// the program), or, for a pattern with no type at all (nested, or an if-chain
// test whose type was the alias), the first loaded ADT declaring the tag —
// lib/light's Light declares Green and Red in the other order, which made the
// VM print [2, 1, 3, …] where the evaluator printed [1, 2, 3, …].
//
// The last case has no alias at all: a nested constructor whose name two
// types in one module declare. The VM read its tag ordinal from the first of
// them (Light) and missed every arm (RT002), or took the wrong one.

func runModulesBothEngines(t *testing.T, files map[string]string) (evalOut string, evalErr error, vmOut string, vmErr error) {
	t.Helper()
	root := t.TempDir()
	for path, src := range files {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(root)
	t.Setenv("AILANG_NO_CACHE", "1")
	evalOut, evalErr = runMapModule(t, "main.ail", "", false)
	vmOut, vmErr = runMapModule(t, "main.ail", "", true)
	return strings.TrimSpace(evalOut), evalErr, strings.TrimSpace(vmOut), vmErr
}

const byNameLibs = `module lib/light
export type Light = Green | Red | Blue | Amber
export pure func amber() -> Light = Amber
export pure func lcode(l: Light) -> int = match l { Amber => 0, Red => 1, Green => 2, Blue => 3 }
`

func TestMatchByNameOnImportedAliasBothEngines(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{
			name: "alias of a user ADT, switch and if-chain",
			files: map[string]string{
				"lib/light.ail": byNameLibs,
				"lib/hold.ail":  "module lib/hold\nimport lib/light (Light, lcode, amber)\nexport pure func one() -> int = lcode(amber()) + 1\n",
				"lib/color.ail": "module lib/color\nexport type Color = Red | Green | Blue\nexport pure func pick(i: int) -> Color = if i == 0 then Red else if i == 1 then Green else Blue\n",
				"lib/b.ail":     "module lib/b\nimport lib/color (Color, pick)\nexport type Hue = Color\nexport pure func mk(i: int) -> Hue = pick(i)\n",
				"main.ail": `module main
import lib/hold (one)
import lib/b (Hue, mk)

pure func sw(h: Hue) -> int = match h { Red => 1, Green => 2, Blue => 3 }

pure func chain(h: Hue, n: int) -> int = match h {
  Red if n > 100 => 100,
  Red => 1,
  Green => 2,
  Blue => 3
}

export func main() -> string = show([sw(mk(0)), sw(mk(1)), sw(mk(2)), chain(mk(0), 0), chain(mk(1), 0), chain(mk(2), 0), one()])
`,
			},
			want: "[1, 2, 3, 1, 2, 3, 1]",
		},
		{
			name: "alias of an applied prelude type",
			files: map[string]string{
				"lib/al.ail": "module lib/al\nimport std/option (Option, Some, None)\nexport type MaybeInt = Option[int]\nexport pure func mi(i: int) -> MaybeInt = if i == 0 then None else Some(i)\n",
				"main.ail": `module main
import lib/al (MaybeInt, mi)

pure func o(m: MaybeInt) -> int = match m { Some(n) => n * 10, None => 0 }

export func main() -> string = show([o(mi(0)), o(mi(4))])
`,
			},
			want: "[0, 40]",
		},
		{
			name: "nested constructor two local types declare, no alias",
			files: map[string]string{
				"main.ail": `module main
import std/option (Option, Some, None)

type Light = Amber | Red | Green
type Color = Red | Green | Blue

pure func f(o: Option[Color]) -> int = match o {
  Some(Red) => 10,
  Some(Green) => 20,
  Some(Blue) => 30,
  None => 0
}

export func main() -> string = show([f(Some(Red)), f(Some(Green)), f(Some(Blue)), f(None)])
`,
			},
			want: "[10, 20, 30, 0]",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e, eerr, v, verr := runModulesBothEngines(t, c.files)
			if eerr != nil || verr != nil {
				t.Fatalf("eval err %v, strict vm err %v", eerr, verr)
			}
			if e != c.want {
				t.Fatalf("evaluator: got %q, want %q", e, c.want)
			}
			if v != e {
				t.Fatalf("engines differ: eval %q, strict vm %q", e, v)
			}
		})
	}
}
