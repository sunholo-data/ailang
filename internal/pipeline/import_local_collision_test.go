package pipeline

import (
	"strings"
	"testing"
)

// MOD015 (#1467, ruling 2026-10-02): a name a module brings in by an explicit
// selective import — `import M (tick)` or `import M (f as tick)` — and also
// defines at module level is an ambiguous occurrence, as in Haskell/Elm and
// Rust E0255. It used to compile with the import silently winning (row 7 of
// design_docs/implemented/v0_52_0/m-elaborator-lexical-scope.md).

const collideDep = `module dep

export type Color = Red | Green

export pure func tick(n: int) -> int = n + 1

export pure func tock(n: int) -> int = n + 2
`

func wantMOD015(t *testing.T, err error, fragments ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("want MOD015, got no error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "MOD015") {
		t.Fatalf("want MOD015, got: %s", msg)
	}
	for _, f := range fragments {
		if !strings.Contains(msg, f) {
			t.Errorf("MOD015 message missing %q:\n%s", f, msg)
		}
	}
}

func TestMOD015_ImportVsLocalFunc(t *testing.T) {
	err := checkModules(t, map[string]string{
		"dep.ail": collideDep,
		"main.ail": `module main

import dep (tick)

pure func tick(n: int) -> int = n + 100

export pure func main() -> int = tick(1)
`,
	})
	// Both sites (the import at 3:1, the local func's name at 5:6) and both fixes.
	wantMOD015(t, err, "'tick'", "main.ail:3:1", "main.ail:5:6", "a func", "rename the local definition", "import dep (tick as depTick)")
}

func TestMOD015_ImportVsModuleLet(t *testing.T) {
	err := checkModules(t, map[string]string{
		"dep.ail": collideDep,
		"main.ail": `module main

import dep (tick)

let tick = 7

export pure func main() -> int = tick
`,
	})
	wantMOD015(t, err, "'tick'", "main.ail:3:1", "main.ail:5:1", "let")
}

func TestMOD015_AliasedImportVsLocal(t *testing.T) {
	err := checkModules(t, map[string]string{
		"dep.ail": collideDep,
		"main.ail": `module main

import dep (tock as tick)

pure func tick(n: int) -> int = n + 100

export pure func main() -> int = tick(1)
`,
	})
	wantMOD015(t, err, "'tick'", "tock as tick", "import dep (tock as depTick)")
}

// The selective list of an aliased module import binds bare names too.
func TestMOD015_AliasedModuleSelectiveVsLocal(t *testing.T) {
	err := checkModules(t, map[string]string{
		"dep.ail": collideDep,
		"main.ail": `module main

import dep as D (tick)

pure func tick(n: int) -> int = n + 100

export pure func main() -> int = tick(1) + D.tick(1)
`,
	})
	wantMOD015(t, err, "'tick'", "main.ail:3:1", "import dep as D (tick)", "write D.tick")
}

func TestMOD015_ImportedCtorVsLocalCtor(t *testing.T) {
	err := checkModules(t, map[string]string{
		"dep.ail": collideDep,
		"main.ail": `module main

import dep (Color, Red)

type Shade = Red | Blue

export pure func main() -> int = match Blue { Red => 1, Blue => 2 }
`,
	})
	wantMOD015(t, err, "'Red'", "constructor", "constructor of type Shade", "main.ail:3:1", "main.ail:5:1", "import dep (Red as DepRed)")
}

// Legal shapes: these must keep compiling.
func TestMOD015_NoFalsePositives(t *testing.T) {
	cases := map[string]string{
		// Lexical binders shadow an import (#1467, already shipped).
		"param": `module main
import dep (tick)
pure func f(tick: int) -> int = tick + 1
export pure func main() -> int = f(1) + tick(1)
`,
		"let-in": `module main
import dep (tick)
export pure func main() -> int = { let tick = 3; tick }
`,
		// Aliasing the import away removes the collision.
		"aliased-away": `module main
import dep (tick as depTick)
pure func tick(n: int) -> int = n + 100
export pure func main() -> int = tick(1) + depTick(1)
`,
		// A bare module alias is qualified-only: it binds no bare names.
		"module-alias": `module main
import dep as D
pure func tick(n: int) -> int = n + 100
export pure func main() -> int = tick(1) + D.tick(1)
`,
		// The same export imported twice from the same module is one binding.
		"same-export-twice": `module main
import dep (tick)
import dep (tick, tock)
export pure func main() -> int = tick(1) + tock(1)
`,
		// Types are a separate namespace with their own rule (M-TYPE-NAME-SHADOW:
		// the local type declaration wins).
		"local-type-shadows-imported-type": `module main
import dep (Color)
type Color = Cyan | Magenta
export pure func main() -> int = match Cyan { Cyan => 1, Magenta => 2 }
`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			err := checkModules(t, map[string]string{"dep.ail": collideDep, "main.ail": src})
			if err != nil {
				t.Fatalf("legal program rejected: %v", err)
			}
		})
	}
}
