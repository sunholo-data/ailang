package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// TestCollectTConNames verifies the helper that extracts type constructor names from types.
func TestCollectTConNames(t *testing.T) {
	tests := []struct {
		name     string
		typ      types.Type
		expected []string
	}{
		{
			name:     "simple TCon",
			typ:      &types.TCon{Name: "Usage"},
			expected: []string{"Usage"},
		},
		{
			name: "TApp with nested TCons (Result[Usage, string])",
			typ: &types.TApp{
				Constructor: &types.TCon{Name: "Result"},
				Args: []types.Type{
					&types.TCon{Name: "Usage"},
					&types.TCon{Name: "string"},
				},
			},
			expected: []string{"Result", "Usage", "string"},
		},
		{
			name: "TFunc2 with TCon params and return",
			typ: &types.TFunc2{
				Params: []types.Type{
					&types.TCon{Name: "Usage"},
					&types.TCon{Name: "UsageDelta"},
				},
				Return: &types.TCon{Name: "Usage"},
			},
			expected: []string{"Usage", "UsageDelta"},
		},
		{
			name: "TRecord with TCon fields",
			typ: &types.TRecord{
				Fields: map[string]types.Type{
					"count": &types.TCon{Name: "int"},
					"data":  &types.TCon{Name: "Payload"},
				},
			},
			expected: []string{"int", "Payload"},
		},
		{
			name: "TList with TCon element",
			typ: &types.TList{
				Element: &types.TCon{Name: "Item"},
			},
			expected: []string{"Item"},
		},
		{
			name:     "TVar2 has no TCon names",
			typ:      &types.TVar2{Name: "a"},
			expected: []string{},
		},
		{
			name:     "nil type",
			typ:      nil,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := make(map[string]bool)
			collectTConNames(tt.typ, names)

			// Check all expected names are present
			for _, exp := range tt.expected {
				if !names[exp] {
					t.Errorf("expected TCon name %q not found in %v", exp, names)
				}
			}

			// Check no unexpected names
			if len(names) != len(tt.expected) {
				// Deduplicate expected for comparison
				unique := make(map[string]bool)
				for _, e := range tt.expected {
					unique[e] = true
				}
				if len(names) != len(unique) {
					t.Errorf("got %d TCon names %v, expected %d", len(names), names, len(unique))
				}
			}
		})
	}
}

// TestCrossPackageTypeAliasUnification tests that type aliases propagate across packages
// when importing functions (not just types). This is the core fix from M-TYPE-ALIAS.
//
// Scenario: Three modules simulating three packages:
//
//	Package A: defines type Usage = { count: int } and func applyDelta(Usage) -> Usage
//	Package B: imports Usage from A, exports func getUsage() -> Usage
//	Package C: imports getUsage from B and applyDelta from A, uses both without type annotations
func TestCrossPackageTypeAliasUnification(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ailang-cross-pkg-alias-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create package directories
	pkgA := filepath.Join(tempDir, "pkg_a")
	pkgB := filepath.Join(tempDir, "pkg_b")
	if err := os.MkdirAll(pkgA, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pkgB, 0755); err != nil {
		t.Fatal(err)
	}

	// Package A: defines Usage type and applyDelta function
	// Usage must be exported so its type alias is available to importers
	pkgAContent := `module pkg_a/types

export type Usage = { count: int, pages: int }

export pure func emptyUsage() -> Usage {
    { count: 0, pages: 0 }
}

export pure func applyDelta(current: Usage, delta: int) -> Usage {
    { count: current.count + delta, pages: current.pages + 1 }
}
`
	if err := os.WriteFile(filepath.Join(pkgA, "types.ail"), []byte(pkgAContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Package B: imports Usage from A, exports getUsage that returns Usage
	pkgBContent := `module pkg_b/repo

import pkg_a/types (emptyUsage)

export pure func getUsage() -> { count: int, pages: int } {
    emptyUsage()
}
`
	if err := os.WriteFile(filepath.Join(pkgB, "repo.ail"), []byte(pkgBContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Package C (main): imports from both A and B, uses functions without type annotations
	mainContent := `module main

import pkg_a/types (applyDelta, emptyUsage)
import pkg_b/repo (getUsage)

export func main() -> int {
    let current = getUsage();
    let updated = applyDelta(current, 5);
    updated.count
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "main.ail"), []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile — this should succeed without "cannot unify type constructor Usage with *types.TRecord"
	originalDir, err2 := os.Getwd()
	if err2 != nil {
		t.Fatal(err2)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalDir)

	src := Source{Filename: "main.ail"}
	cfg := Config{Mode: ModeCheck}
	result, compileErr := Run(cfg, src)
	if compileErr != nil {
		t.Fatalf("Cross-package type alias unification failed: %v", compileErr)
	}
	if len(result.Modules) == 0 {
		t.Fatal("Expected compiled modules, got none")
	}
}

// TestTransitiveTypeAliasPropagation tests that type aliases are available
// transitively through module interfaces. If Package B uses a type from Package A
// in its function signatures, Package C should be able to use those types even
// if it only imports from Package B.
func TestTransitiveTypeAliasPropagation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ailang-transitive-alias-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create package directories
	pkgA := filepath.Join(tempDir, "pkg_a")
	pkgB := filepath.Join(tempDir, "pkg_b")
	if err := os.MkdirAll(pkgA, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pkgB, 0755); err != nil {
		t.Fatal(err)
	}

	// Package A: defines Coord type
	pkgAContent := `module pkg_a/geo

type Coord = { x: int, y: int }

export pure func origin() -> Coord {
    { x: 0, y: 0 }
}
`
	if err := os.WriteFile(filepath.Join(pkgA, "geo.ail"), []byte(pkgAContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Package B: imports from A, exports function using Coord in its signature
	pkgBContent := `module pkg_b/world

import pkg_a/geo (origin)

export pure func startPos() -> { x: int, y: int } {
    origin()
}

export pure func moveRight(pos: { x: int, y: int }) -> { x: int, y: int } {
    { x: pos.x + 1, y: pos.y }
}
`
	if err := os.WriteFile(filepath.Join(pkgB, "world.ail"), []byte(pkgBContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Package C: imports from B only, uses Coord transitively
	mainContent := `module main

import pkg_b/world (startPos, moveRight)

export func main() -> int {
    let pos = startPos();
    let moved = moveRight(pos);
    moved.x
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "main.ail"), []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	originalDir, err2 := os.Getwd()
	if err2 != nil {
		t.Fatal(err2)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalDir)

	src := Source{Filename: "main.ail"}
	cfg := Config{Mode: ModeCheck}
	result, compileErr := Run(cfg, src)
	if compileErr != nil {
		t.Fatalf("Transitive type alias propagation failed: %v", compileErr)
	}
	if len(result.Modules) == 0 {
		t.Fatal("Expected compiled modules, got none")
	}
}

// TestCrossModuleNestedRecordAlias is the regression test for
// M-TRANSITIVE-ALIAS-ENV-IMPORT.
//
// Scenario from the motoko_agent bug report (msg_20260522_170317_f850eba4):
//
//	Package A: defines `type Inner = { name: string }`
//	Package B: imports Inner from A, exports `type Outer = { items: [Inner] }`
//	           plus `build() -> Outer` and `use_outer(o: Outer) -> int`
//	Package C: imports only build + use_outer from B, calls `use_outer(build())`
//
// Pre-fix the call-site unification fails with
//
//	cannot unify type constructor Inner with *types.TRecord
//
// because C's Unifier.aliasEnv lacks `Inner` (B's iface only carries `Outer`,
// and C never directly imports A). Post-fix `resolveModuleImports` walks the
// linker's full closure of loaded ifaces, so `Inner` reaches C's aliasEnv and
// `expandAlias(TCon("Inner"))` resolves correctly.
//
// This is distinct from TestTransitiveTypeAliasPropagation above: that test
// sidesteps the bug by having package B inline the structural record type
// in its signatures. This test forces B to use A's alias nominally,
// exercising the exact failure mode the bug describes.
func TestCrossModuleNestedRecordAlias(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ailang-nested-alias-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	pkgA := filepath.Join(tempDir, "pkg_a")
	pkgB := filepath.Join(tempDir, "pkg_b")
	if err := os.MkdirAll(pkgA, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pkgB, 0755); err != nil {
		t.Fatal(err)
	}

	// Package A declares the leaf alias.
	pkgAContent := `module pkg_a/types

export type Inner = { name: string }
`
	if err := os.WriteFile(filepath.Join(pkgA, "types.ail"), []byte(pkgAContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Package B imports Inner and nests it inside its own exported record alias.
	pkgBContent := `module pkg_b/lib

import pkg_a/types (Inner)

export type Outer = { items: [Inner] }

export pure func build() -> Outer {
    { items: [{name: "a"}, {name: "b"}] }
}

export pure func use_outer(o: Outer) -> int {
    match o.items {
        [] => 0,
        _ :: _ => 1
    }
}
`
	if err := os.WriteFile(filepath.Join(pkgB, "lib.ail"), []byte(pkgBContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Package C imports B's functions but NOT pkg_a/types — yet Inner must
	// still be reachable to the unifier through B's transitive iface.
	mainContent := `module main

import pkg_b/lib (build, use_outer)

export func main() -> int {
    let v = build();
    use_outer(v)
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "main.ail"), []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	originalDir, err2 := os.Getwd()
	if err2 != nil {
		t.Fatal(err2)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalDir)

	src := Source{Filename: "main.ail"}
	cfg := Config{Mode: ModeCheck}
	result, compileErr := Run(cfg, src)
	if compileErr != nil {
		t.Fatalf("Cross-module nested record alias failed to type-check: %v", compileErr)
	}
	if len(result.Modules) == 0 {
		t.Fatal("Expected compiled modules, got none")
	}
}

// TestCrossModuleAliasCollisionPrecedence verifies that when two modules
// export same-named aliases with different bodies, the direct-import
// precedence wins for the importer.
//
// Pre-existing behavior (preserved by M-TRANSITIVE-ALIAS-ENV-IMPORT's
// first-wins guard): direct-import aliases populate first; the transitive
// closure pass only fills gaps. Local aliases beat both.
func TestCrossModuleAliasCollisionPrecedence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ailang-alias-collision-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	pkgA := filepath.Join(tempDir, "pkg_a")
	pkgB := filepath.Join(tempDir, "pkg_b")
	if err := os.MkdirAll(pkgA, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pkgB, 0755); err != nil {
		t.Fatal(err)
	}

	// pkg_a.Status has a `code` field of type int.
	pkgAContent := `module pkg_a/status

export type Status = { code: int }

export pure func ok() -> Status {
    { code: 200 }
}
`
	if err := os.WriteFile(filepath.Join(pkgA, "status.ail"), []byte(pkgAContent), 0644); err != nil {
		t.Fatal(err)
	}

	// pkg_b.Status has a `text` field of type string — incompatible with pkg_a.Status.
	// Plus a helper function so main has something else to import from pkg_b.
	pkgBContent := `module pkg_b/status

export type Status = { text: string }

export pure func describe(s: Status) -> string {
    s.text
}
`
	if err := os.WriteFile(filepath.Join(pkgB, "status.ail"), []byte(pkgBContent), 0644); err != nil {
		t.Fatal(err)
	}

	// main directly imports Status from pkg_a only. Importing pkg_b's
	// `describe` function pulls pkg_b's iface into the linker's loaded set
	// — but main's direct import of pkg_a.Status must win (first-wins
	// guard) so `ok()` typechecks against pkg_a.Status, not pkg_b.Status.
	mainContent := `module main

import pkg_a/status (Status, ok)
import pkg_b/status (describe)

export func main() -> int {
    let s = ok();
    s.code
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "main.ail"), []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	originalDir, err2 := os.Getwd()
	if err2 != nil {
		t.Fatal(err2)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalDir)

	src := Source{Filename: "main.ail"}
	cfg := Config{Mode: ModeCheck}
	_, compileErr := Run(cfg, src)
	if compileErr != nil {
		t.Fatalf("Alias-collision precedence broke direct-import resolution: %v", compileErr)
	}
}
