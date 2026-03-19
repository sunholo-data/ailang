package golang

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestContractViolation_Integration is an end-to-end test that:
// 1. Compiles AILANG code with --verify-contracts
// 2. Compiles the generated Go code
// 3. Runs tests that trigger contract violations
// 4. Verifies panics occur with correct messages
func TestContractViolation_Integration(t *testing.T) {
	// Skip in short mode (these tests compile and run Go code)
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "contract_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Get path to ailang binary
	projectRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	ailangBin := filepath.Join(projectRoot, "bin", "ailang")
	if _, err := os.Stat(ailangBin); os.IsNotExist(err) {
		// Try building it
		cmd := exec.Command("go", "build", "-o", ailangBin, "./cmd/ailang")
		cmd.Dir = projectRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("Skipping: ailang binary not available and failed to build: %v\n%s", err, out)
		}
	}
	// Verify the binary is actually executable
	if _, err := exec.LookPath(ailangBin); err != nil {
		t.Skipf("Skipping: ailang binary not executable: %v", err)
	}

	// Compile the contracts example with --verify-contracts
	sourceFile := filepath.Join(projectRoot, "examples", "runnable", "contracts", "basic.ail")
	outputDir := filepath.Join(tmpDir, "basic")

	cmd := exec.Command(ailangBin, "compile", "--emit-go", "--verify-contracts", "--out", outputDir, sourceFile)
	cmd.Env = append(os.Environ(), "AILANG_RELAX_MODULES=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to compile with contracts: %v\n%s", err, out)
	}

	// Create a test file that calls the functions with invalid inputs
	testCode := `package basic

import (
	"strings"
	"testing"
)

func TestAbsolute_ValidInput(t *testing.T) {
	// Should not panic with valid input
	result := basic__Absolute(5)
	if result != 5 {
		t.Errorf("expected 5, got %d", result)
	}
}

func TestAbsolute_ContractViolation(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for negative input")
		}
		msg := r.(string)
		if !strings.Contains(msg, "contract violation") {
			t.Errorf("expected contract violation message, got: %s", msg)
		}
		if !strings.Contains(msg, "x >= 0") {
			t.Errorf("expected message to mention 'x >= 0', got: %s", msg)
		}
	}()
	// This should panic because -5 violates requires: x >= 0
	basic__Absolute(-5)
}

func TestSafeDivide_ValidInput(t *testing.T) {
	result := basic__SafeDivide(10, 2)
	if result != 5 {
		t.Errorf("expected 5, got %d", result)
	}
}

func TestSafeDivide_DivisionByZero(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for division by zero")
		}
		msg := r.(string)
		if !strings.Contains(msg, "contract violation") {
			t.Errorf("expected contract violation message, got: %s", msg)
		}
		if !strings.Contains(msg, "divisor != 0") {
			t.Errorf("expected message to mention 'divisor != 0', got: %s", msg)
		}
	}()
	// This should panic because divisor=0 violates requires: divisor != 0
	basic__SafeDivide(10, 0)
}

func TestIncrement_ValidInput(t *testing.T) {
	result := basic__Increment(5, 100)
	if result != 6 {
		t.Errorf("expected 6, got %d", result)
	}
}

func TestIncrement_NegativeInput(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for negative input")
		}
		msg := r.(string)
		if !strings.Contains(msg, "contract violation") {
			t.Errorf("expected contract violation message, got: %s", msg)
		}
	}()
	// This should panic because -1 violates requires: x >= 0
	basic__Increment(-1, 100)
}

func TestClamp_ValidInput(t *testing.T) {
	result := basic__Clamp(150, 0, 100)
	if result != 100 {
		t.Errorf("expected 100 (clamped), got %d", result)
	}
}

func TestClamp_InvalidRange(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for invalid range")
		}
		msg := r.(string)
		if !strings.Contains(msg, "contract violation") {
			t.Errorf("expected contract violation message, got: %s", msg)
		}
		if !strings.Contains(msg, "minVal <= maxVal") {
			t.Errorf("expected message to mention 'minVal <= maxVal', got: %s", msg)
		}
	}()
	// This should panic because minVal=100 > maxVal=0 violates requires: minVal <= maxVal
	basic__Clamp(50, 100, 0)
}

func TestMain_Integration(t *testing.T) {
	// Main should work because it uses valid inputs
	result := basic__Main()
	// a=5, b=5, c=6, d=100 => 5+5+6+100 = 116
	if result != 116 {
		t.Errorf("expected 116, got %d", result)
	}
}

// ENSURES VIOLATION TESTS - M-VERIFY Phase 0.5
// These test that ensures (postcondition) checks trigger panics

func TestSafeDivide_EnsuresViolation(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for ensures violation (negative result)")
		}
		msg := r.(string)
		if !strings.Contains(msg, "contract violation") {
			t.Errorf("expected contract violation message, got: %s", msg)
		}
		if !strings.Contains(msg, "ensures") {
			t.Errorf("expected ensures violation, got: %s", msg)
		}
		if !strings.Contains(msg, "result >= 0") {
			t.Errorf("expected message to mention 'result >= 0', got: %s", msg)
		}
	}()
	// dividend=-10, divisor=2 → result=-5, which violates ensures: result >= 0
	basic__SafeDivide(-10, 2)
}

func TestIncrement_EnsuresViolation(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for ensures violation (result not > x)")
		}
		msg := r.(string)
		if !strings.Contains(msg, "contract violation") {
			t.Errorf("expected contract violation message, got: %s", msg)
		}
		if !strings.Contains(msg, "ensures") {
			t.Errorf("expected ensures violation, got: %s", msg)
		}
	}()
	// x=99, max=99 → result=99 (capped), violates ensures: result > x (99 is not > 99)
	basic__Increment(99, 99)
}
`

	testFile := filepath.Join(outputDir, "basic_test.go")
	if err := os.WriteFile(testFile, []byte(testCode), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Run go mod init to make it a valid Go module
	cmd = exec.Command("go", "mod", "init", "basic")
	cmd.Dir = outputDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to init go mod: %v\n%s", err, out)
	}

	// Run the tests
	cmd = exec.Command("go", "test", "-v", "./...")
	cmd.Dir = outputDir
	output, err := cmd.CombinedOutput()

	// go test returns non-zero if any tests fail, but we expect all to pass
	if err != nil {
		// Check if this is a test failure vs compilation error
		if strings.Contains(string(output), "FAIL") && !strings.Contains(string(output), "build") {
			// Test failures - show output for debugging
			t.Logf("Test output:\n%s", output)
			t.Fatalf("Some contract tests failed: %v", err)
		}
		t.Fatalf("Failed to run tests: %v\n%s", err, output)
	}

	t.Logf("Integration test output:\n%s", output)

	// Verify all tests passed
	if !strings.Contains(string(output), "PASS") {
		t.Errorf("Expected tests to pass, output:\n%s", output)
	}
}

// TestContractViolation_NoVerify verifies that without --verify-contracts,
// contract violations do NOT cause panics (contracts are just comments)
func TestContractViolation_NoVerify(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir, err := os.MkdirTemp("", "contract_noverify_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	ailangBin := filepath.Join(projectRoot, "bin", "ailang")
	if _, err := os.Stat(ailangBin); os.IsNotExist(err) {
		// Try building it
		cmd := exec.Command("go", "build", "-o", ailangBin, "./cmd/ailang")
		cmd.Dir = projectRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("Skipping: ailang binary not available and failed to build: %v\n%s", err, out)
		}
	}
	// Verify the binary is actually executable
	if _, err := exec.LookPath(ailangBin); err != nil {
		t.Skipf("Skipping: ailang binary not executable: %v", err)
	}

	// Compile WITHOUT --verify-contracts
	sourceFile := filepath.Join(projectRoot, "examples", "runnable", "contracts", "basic.ail")
	outputDir := filepath.Join(tmpDir, "basic")

	cmd := exec.Command(ailangBin, "compile", "--emit-go", "--out", outputDir, sourceFile)
	cmd.Env = append(os.Environ(), "AILANG_RELAX_MODULES=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to compile: %v\n%s", err, out)
	}

	// Read the generated code - should have comments but no panic checks
	basicGo, err := os.ReadFile(filepath.Join(outputDir, "basic.go"))
	if err != nil {
		t.Fatalf("Failed to read generated code: %v", err)
	}

	// Should have requires comments but NOT panic statements
	if !strings.Contains(string(basicGo), "// Requires:") {
		t.Error("Expected requires comments in generated code")
	}

	// When verify is off, there should be no contract violation panics
	if strings.Contains(string(basicGo), "contract violation") {
		t.Error("Expected NO contract violation panics without --verify-contracts")
	}
}
