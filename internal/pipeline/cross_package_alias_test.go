package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo/ailang/internal/types"
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
	pkgAContent := `module pkg_a/types

type Usage = { count: int, pages: int }

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
