package codegen_golden_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const expectedDifferentialFixtureCount = 1

// TestGoldenCompile verifies that each .ail golden test file compiles to Go
// that passes `go build`. This is the primary acceptance test for the codegen
// pipeline — the generated Go must be valid, not necessarily identical to baselines.
func TestGoldenCompile(t *testing.T) {
	// Find all .ail files in this directory
	ailFiles, err := filepath.Glob("*.ail")
	if err != nil {
		t.Fatalf("failed to glob .ail files: %v", err)
	}
	if len(ailFiles) == 0 {
		t.Fatal("no .ail golden test files found")
	}

	for _, ailFile := range ailFiles {
		name := strings.TrimSuffix(ailFile, ".ail")
		t.Run(name, func(t *testing.T) {
			// Create temp dir for this test's output
			outDir := t.TempDir()

			// Compile .ail to Go using ailang CLI
			absAil, _ := filepath.Abs(ailFile)
			cmd := exec.Command("ailang", "compile", "--emit-go",
				"--out", outDir,
				"--package-name", "golden",
				"--relax-modules",
				"--no-verify-go",
				absAil)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("ailang compile failed for %s:\n%s", ailFile, string(out))
			}

			// Verify go build succeeds on generated code
			pkgDir := filepath.Join(outDir, "golden")
			if _, err := os.Stat(pkgDir); os.IsNotExist(err) {
				t.Fatalf("expected output directory %s does not exist", pkgDir)
			}

			// Create go.mod for the generated package
			goMod := filepath.Join(outDir, "go.mod")
			if err := os.WriteFile(goMod, []byte("module golden\n\ngo 1.21\n"), 0644); err != nil {
				t.Fatalf("failed to write go.mod: %v", err)
			}

			buildCmd := exec.Command("go", "build", "./golden/")
			buildCmd.Dir = outDir
			buildOut, err := buildCmd.CombinedOutput()
			if err != nil {
				t.Fatalf("go build failed for %s:\n%s", ailFile, string(buildOut))
			}

			// Verify go vet succeeds
			vetCmd := exec.Command("go", "vet", "./golden/")
			vetCmd.Dir = outDir
			vetOut, err := vetCmd.CombinedOutput()
			if err != nil {
				t.Fatalf("go vet failed for %s:\n%s", ailFile, string(vetOut))
			}
		})
	}
}

// TestInterpreterCompiledDifferential executes the same fixtures through both
// runtimes. This catches semantic drift that compile-only golden tests cannot.
func TestInterpreterCompiledDifferential(t *testing.T) {
	fixtures, err := filepath.Glob("show_differential*.ail")
	if err != nil {
		t.Fatalf("failed to glob differential fixtures: %v", err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no run-vs-compiled differential fixtures found")
	}
	if len(fixtures) != expectedDifferentialFixtureCount {
		t.Fatalf("differential fixture count = %d, want exactly %d; update the literal deliberately when changing coverage",
			len(fixtures), expectedDifferentialFixtureCount)
	}

	for _, fixture := range fixtures {
		fixture := fixture
		name := strings.TrimSuffix(filepath.Base(fixture), ".ail")
		t.Run(name, func(t *testing.T) {
			absFixture, err := filepath.Abs(fixture)
			if err != nil {
				t.Fatal(err)
			}

			interpreted := exec.Command("ailang", "run", "--quiet", "--caps", "IO", "--relax-modules", absFixture)
			interpretedOut, err := interpreted.Output()
			if err != nil {
				t.Fatalf("ailang run failed for %s: %v", fixture, err)
			}

			outDir := t.TempDir()
			compile := exec.Command("ailang", "compile", "--emit-go", "--out", outDir,
				"--package-name", "main", "--relax-modules", "--no-verify-go", absFixture)
			if output, err := compile.CombinedOutput(); err != nil {
				t.Fatalf("ailang compile failed for %s: %v\n%s", fixture, err, output)
			}
			pkgDir := filepath.Join(outDir, "main")
			driver := fmt.Sprintf("package main\n\nfunc main() { %s__Main() }\n", name)
			if err := os.WriteFile(filepath.Join(pkgDir, "driver.go"), []byte(driver), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(outDir, "go.mod"), []byte("module differential\n\ngo 1.21\n"), 0644); err != nil {
				t.Fatal(err)
			}
			binary := filepath.Join(outDir, "fixture-bin")
			build := exec.Command("go", "build", "-o", binary, "./main/")
			build.Dir = outDir
			if output, err := build.CombinedOutput(); err != nil {
				t.Fatalf("go build failed for %s: %v\n%s", fixture, err, output)
			}
			compiledOut, err := exec.Command(binary).Output()
			if err != nil {
				t.Fatalf("compiled fixture failed for %s: %v", fixture, err)
			}
			if !bytes.Equal(interpretedOut, compiledOut) {
				t.Fatalf("interpreter/compiled stdout differs for %s\ninterpreter:\n%s\ncompiled:\n%s",
					fixture, interpretedOut, compiledOut)
			}
		})
	}
}
