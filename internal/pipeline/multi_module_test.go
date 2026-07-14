package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMultiModuleImports tests that importing from subdirectory modules works correctly.
// This is a regression test for the bug where imports like "sim/protocol" from "sim/world.ail"
// would fail with "LDR001: module not found" because the elaborator created its own loader
// with basePath = filepath.Dir(modID) instead of using the pipeline's loader with basePath ".".
func TestMultiModuleImports(t *testing.T) {
	// Create a temporary directory structure
	tempDir, err := os.MkdirTemp("", "ailang-multi-module-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create subdirectory "sim"
	simDir := filepath.Join(tempDir, "sim")
	if err := os.MkdirAll(simDir, 0755); err != nil {
		t.Fatalf("failed to create sim dir: %v", err)
	}

	// Create sim/protocol.ail - a module with a record type
	protocolContent := `module sim/protocol

type Coord = { x: int, y: int }

export pure func origin() -> Coord {
    { x: 0, y: 0 }
}
`
	if err := os.WriteFile(filepath.Join(simDir, "protocol.ail"), []byte(protocolContent), 0644); err != nil {
		t.Fatalf("failed to write protocol.ail: %v", err)
	}

	// Create sim/world.ail - a module that imports from sim/protocol
	worldContent := `module sim/world

import sim/protocol (origin)

export func main() -> int {
    let c = origin();
    c.x
}
`
	if err := os.WriteFile(filepath.Join(simDir, "world.ail"), []byte(worldContent), 0644); err != nil {
		t.Fatalf("failed to write world.ail: %v", err)
	}

	// Change to the temp directory (simulating running from project root)
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer os.Chdir(originalDir)

	// Compile the module - this should succeed
	src := Source{
		Filename: "sim/world.ail",
	}
	cfg := Config{
		Mode: ModeCheck,
	}

	result, err := Run(cfg, src)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify both user modules were compiled.
	// M-PRELUDE-OPTION-RESULT: entry modules now implicitly import std/option +
	// std/result, so the compiled set is 2 user modules + 2 implicit prelude = 4.
	if len(result.Modules) != 4 {
		t.Errorf("expected 4 modules (2 user + std/option + std/result), got %d", len(result.Modules))
	}
	if _, ok := result.Modules["std/option"]; !ok {
		t.Error("std/option not implicitly loaded for entry module")
	}
	if _, ok := result.Modules["std/result"]; !ok {
		t.Error("std/result not implicitly loaded for entry module")
	}

	// Check that sim/protocol was loaded
	if _, ok := result.Modules["sim/protocol"]; !ok {
		t.Error("sim/protocol module not found in result.Modules")
	}

	// Check that sim/world was loaded
	if _, ok := result.Modules["sim/world"]; !ok {
		t.Error("sim/world module not found in result.Modules")
	}

	// Check that the interface exports main
	if result.Interface == nil {
		t.Fatal("result.Interface is nil")
	}
	if _, ok := result.Interface.Exports["main"]; !ok {
		t.Error("main not found in exports")
	}
}

// TestNestedDirectoryImports tests deeply nested directory structures
func TestNestedDirectoryImports(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ailang-nested-module-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create nested structure: lib/math/utils
	utilsDir := filepath.Join(tempDir, "lib", "math", "utils")
	if err := os.MkdirAll(utilsDir, 0755); err != nil {
		t.Fatalf("failed to create utils dir: %v", err)
	}

	// Create lib/math/utils/helpers.ail
	helpersContent := `module lib/math/utils/helpers

export pure func double(n: int) -> int {
    n + n
}
`
	if err := os.WriteFile(filepath.Join(utilsDir, "helpers.ail"), []byte(helpersContent), 0644); err != nil {
		t.Fatalf("failed to write helpers.ail: %v", err)
	}

	// Create app directory
	appDir := filepath.Join(tempDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}

	// Create app/main.ail that imports from lib/math/utils/helpers
	mainContent := `module app/main

import lib/math/utils/helpers (double)

export func main() -> int {
    double(21)
}
`
	if err := os.WriteFile(filepath.Join(appDir, "main.ail"), []byte(mainContent), 0644); err != nil {
		t.Fatalf("failed to write main.ail: %v", err)
	}

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer os.Chdir(originalDir)

	// Compile
	src := Source{
		Filename: "app/main.ail",
	}
	cfg := Config{
		Mode: ModeCheck,
	}

	result, err := Run(cfg, src)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify both user modules were compiled.
	// M-PRELUDE-OPTION-RESULT: entry modules now implicitly import std/option +
	// std/result, so the compiled set is 2 user modules + 2 implicit prelude = 4.
	if len(result.Modules) != 4 {
		t.Errorf("expected 4 modules (2 user + std/option + std/result), got %d", len(result.Modules))
	}
	if _, ok := result.Modules["std/option"]; !ok {
		t.Error("std/option not implicitly loaded for entry module")
	}
	if _, ok := result.Modules["std/result"]; !ok {
		t.Error("std/result not implicitly loaded for entry module")
	}

	if _, ok := result.Modules["lib/math/utils/helpers"]; !ok {
		t.Error("lib/math/utils/helpers module not found in result.Modules")
	}
}

// TestCrossDirectoryImports tests importing between sibling directories
func TestCrossDirectoryImports(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ailang-cross-module-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create two sibling directories: lib/types and lib/impl
	// (Note: "pkg/" is reserved for external package imports in AILANG)
	typesDir := filepath.Join(tempDir, "lib", "types")
	implDir := filepath.Join(tempDir, "lib", "impl")
	if err := os.MkdirAll(typesDir, 0755); err != nil {
		t.Fatalf("failed to create types dir: %v", err)
	}
	if err := os.MkdirAll(implDir, 0755); err != nil {
		t.Fatalf("failed to create impl dir: %v", err)
	}

	// Create lib/types/defs.ail
	defsContent := `module lib/types/defs

type Point = { x: int, y: int }

export pure func newPoint(x: int, y: int) -> Point {
    { x: x, y: y }
}
`
	if err := os.WriteFile(filepath.Join(typesDir, "defs.ail"), []byte(defsContent), 0644); err != nil {
		t.Fatalf("failed to write defs.ail: %v", err)
	}

	// Create lib/impl/main.ail that imports from sibling
	mainContent := `module lib/impl/main

import lib/types/defs (newPoint)

export func main() -> int {
    let p = newPoint(10, 20);
    p.x
}
`
	if err := os.WriteFile(filepath.Join(implDir, "main.ail"), []byte(mainContent), 0644); err != nil {
		t.Fatalf("failed to write main.ail: %v", err)
	}

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer os.Chdir(originalDir)

	// Compile
	src := Source{
		Filename: "lib/impl/main.ail",
	}
	cfg := Config{
		Mode: ModeCheck,
	}

	result, err := Run(cfg, src)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify both user modules were compiled.
	// M-PRELUDE-OPTION-RESULT: entry modules now implicitly import std/option +
	// std/result, so the compiled set is 2 user modules + 2 implicit prelude = 4.
	if len(result.Modules) != 4 {
		t.Errorf("expected 4 modules (2 user + std/option + std/result), got %d", len(result.Modules))
	}
	if _, ok := result.Modules["std/option"]; !ok {
		t.Error("std/option not implicitly loaded for entry module")
	}
	if _, ok := result.Modules["std/result"]; !ok {
		t.Error("std/result not implicitly loaded for entry module")
	}
}

// TestInterFunctionReferences tests that a consumer module can import and compile
// against a dependency module whose exported functions reference non-exported helpers.
// This is a regression test for M-PKG-INTERREF: the resolver must accumulate Let
// bindings in the evaluator env so sequential function declarations can reference
// earlier ones when the module is evaluated on-demand.
func TestInterFunctionReferences(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ailang-interref-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	libDir := filepath.Join(tempDir, "lib")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		t.Fatalf("failed to create lib dir: %v", err)
	}

	// Create lib/helpers.ail — module with non-exported helpers
	helpersContent := `module lib/helpers

pure func double(x: int) -> int {
  x + x
}

pure func quadruple(x: int) -> int {
  double(double(x))
}

export pure func scale(x: int) -> int {
  quadruple(x) + double(x)
}
`
	if err := os.WriteFile(filepath.Join(libDir, "helpers.ail"), []byte(helpersContent), 0644); err != nil {
		t.Fatalf("failed to write helpers.ail: %v", err)
	}

	// Create consumer.ail — imports scale which depends on non-exported helpers
	consumerContent := `module consumer

import lib/helpers (scale)

export func main() -> int {
  scale(5)
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "consumer.ail"), []byte(consumerContent), 0644); err != nil {
		t.Fatalf("failed to write consumer.ail: %v", err)
	}

	originalDir, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer os.Chdir(originalDir)

	// Compile the consumer — this triggers loading lib/helpers as a dependency.
	// Before the fix, this would succeed at compile time but fail at eval time
	// with "undefined variable: double" inside the scale function.
	src := Source{
		Filename: "consumer.ail",
	}
	cfg := Config{
		Mode: ModeCheck,
	}

	result, err := Run(cfg, src)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify both user modules compiled.
	// M-PRELUDE-OPTION-RESULT: `consumer` is an entry module, so std/option +
	// std/result are implicitly imported: 2 user modules + 2 prelude = 4.
	if len(result.Modules) != 4 {
		t.Errorf("expected 4 modules (2 user + std/option + std/result), got %d", len(result.Modules))
	}

	if _, ok := result.Modules["lib/helpers"]; !ok {
		t.Error("lib/helpers module not found in result.Modules")
	}

	// Verify scale is exported
	if result.Interface == nil {
		t.Fatal("result.Interface is nil")
	}
	if _, ok := result.Interface.Exports["main"]; !ok {
		t.Error("main not found in consumer exports")
	}
}
