package pipeline

import "testing"

// `import M (Row as R)` must expand R to Row's definition. The imported alias
// was registered under its original name, so R stayed an opaque constructor:
// passing an R where a Row is expected failed with "cannot unify record with
// unexpandable type constructor R", and `run --args-json` could not decode an
// R parameter (stapledons_godot, 2026-10-01).
func TestRenamedTypeImport_Expands(t *testing.T) {
	err := checkModules(t, map[string]string{
		"pkg_a/types.ail": `module pkg_a/types

export type Row = { id: string, x: float }
export pure func getX(r: Row) -> float = r.x
`,
		"main.ail": `module main

import pkg_a/types (Row as R, getX)

export pure func viaR(r: R) -> float = getX(r) + r.x

export pure func mk(x: float) -> R = { id: "a", x: x }
`,
	})
	if err != nil {
		t.Fatalf("renamed type import should type-check: %v", err)
	}
}
