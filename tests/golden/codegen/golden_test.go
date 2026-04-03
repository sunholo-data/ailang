package codegen_golden_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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
