package codegen_golden_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const expectedDifferentialFixtureCount = 8

// strictVMExempt are the show fixtures the strict VM refuses (BC001), with the
// reason. They still run on the evaluator and on --bytecode.
var strictVMExempt = map[string]string{
	"show_differential_array": "array literals are evaluator-only on the VM (stmt.ArrayLit)",
}

// ailangBin is an ailang built from this checkout by TestMain, so these tests
// compare the code under test, not whatever `ailang` is first on PATH.
var ailangBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "golden-codegen-ailang")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ailangBin = filepath.Join(dir, "ailang")
	if runtime.GOOS == "windows" {
		ailangBin += ".exe"
	}
	root, _ := filepath.Abs(filepath.Join("..", "..", ".."))
	build := exec.Command("go", "build", "-o", ailangBin, "./cmd/ailang")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "building ailang: %v\n%s", err, out)
		os.RemoveAll(dir)
		os.Exit(1)
	}
	os.Setenv("AILANG_STDLIB_PATH", filepath.Join(root, "std"))
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// TestShowDifferentialEvaluatorVsVM runs each show fixture on the evaluator
// and on the VM and requires the same stdout byte for byte. (Before the Go
// emitter was removed the second leg was the compiled Go binary.)
func TestShowDifferentialEvaluatorVsVM(t *testing.T) {
	fixtures, err := filepath.Glob("show_differential*.ail")
	if err != nil {
		t.Fatalf("failed to glob differential fixtures: %v", err)
	}
	if len(fixtures) != expectedDifferentialFixtureCount {
		t.Fatalf("differential fixture count = %d, want exactly %d; update the literal deliberately when changing coverage",
			len(fixtures), expectedDifferentialFixtureCount)
	}

	for _, fixture := range fixtures {
		name := strings.TrimSuffix(filepath.Base(fixture), ".ail")
		t.Run(name, func(t *testing.T) {
			absFixture, err := filepath.Abs(fixture)
			if err != nil {
				t.Fatal(err)
			}
			run := func(flags ...string) []byte {
				t.Helper()
				args := append([]string{"run", "--quiet", "--caps", "IO", "--relax-modules"}, flags...)
				cmd := exec.Command(ailangBin, append(args, absFixture)...)
				var stderr bytes.Buffer
				cmd.Stderr = &stderr
				out, err := cmd.Output()
				if err != nil {
					t.Fatalf("ailang %v failed for %s: %v\n%s", flags, fixture, err, stderr.String())
				}
				return out
			}
			evaluated := run()
			if len(bytes.TrimSpace(evaluated)) == 0 {
				t.Fatalf("%s printed nothing on the evaluator", fixture)
			}
			legs := [][]string{{"--bytecode"}}
			if _, exempt := strictVMExempt[name]; !exempt {
				legs = append(legs, []string{"--strict-bytecode"})
			}
			for _, flags := range legs {
				if vm := run(flags...); !bytes.Equal(evaluated, vm) {
					t.Fatalf("evaluator/%v stdout differs for %s\nevaluator:\n%s\nvm:\n%s", flags, fixture, evaluated, vm)
				}
			}
		})
	}
	for name := range strictVMExempt {
		if _, err := os.Stat(name + ".ail"); err != nil {
			t.Errorf("stale strictVMExempt entry %s: %v", name, err)
		}
	}
}
