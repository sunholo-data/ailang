package pipeline

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestJWT_DecodeExample runs the jwt_decode.ail example through the full pipeline
// and verifies correct output.
func TestJWT_DecodeExample(t *testing.T) {
	// Find ailang binary
	binary, err := exec.LookPath("ailang")
	if err != nil {
		t.Skip("ailang binary not found in PATH; skipping integration test")
	}

	cmd := exec.Command(binary, "run", "--caps", "IO", "--entry", "main",
		"examples/runnable/jwt_decode.ail")
	cmd.Dir = findProjectRoot(t)
	cmd.Env = append(os.Environ(), "OTEL_SDK_DISABLED=true")

	output, err := cmd.CombinedOutput()
	out := string(output)

	if err != nil {
		t.Fatalf("jwt_decode.ail failed: %v\nOutput: %s", err, out)
	}

	// Verify each expected output line
	expected := []string{
		"JWT decoded successfully!",
		"Subject: Some(1234567890)",
		"Name: Some(Test User)",
		"Expired at 1700000000: false",
		"Issuer matches: true",
		"Audience matches: true",
	}
	for _, want := range expected {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

// TestJWT_DecodeInvalidToken verifies that decodeJWT returns Err for malformed tokens.
func TestJWT_DecodeInvalidToken(t *testing.T) {
	binary, err := exec.LookPath("ailang")
	if err != nil {
		t.Skip("ailang binary not found in PATH; skipping integration test")
	}

	// Create a temporary test file that tries to decode an invalid JWT
	tempDir := t.TempDir()
	testFile := tempDir + "/jwt_invalid.ail"
	err = os.WriteFile(testFile, []byte(`module jwt_invalid
import std/jwt (decodeJWT)
import std/result (Result, Ok, Err)

export func main() -> () ! {IO} = {
  match decodeJWT("not-a-jwt") {
    Err(e) => println("Error: ${e}"),
    Ok(_) => println("BUG: should not succeed")
  }
}
`), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cmd := exec.Command(binary, "run", "--caps", "IO", "--entry", "main", testFile)
	cmd.Dir = findProjectRoot(t)
	cmd.Env = append(os.Environ(), "OTEL_SDK_DISABLED=true")

	output, err := cmd.CombinedOutput()
	out := string(output)
	if err != nil {
		t.Fatalf("jwt_invalid.ail failed: %v\nOutput: %s", err, out)
	}

	if !strings.Contains(out, "Error:") {
		t.Errorf("expected error output for invalid JWT, got:\n%s", out)
	}
	if strings.Contains(out, "BUG") {
		t.Errorf("invalid JWT should not decode successfully, got:\n%s", out)
	}
}

// findProjectRoot walks up from the current directory to find the project root.
func findProjectRoot(t *testing.T) string {
	t.Helper()
	// Look for go.mod file starting from common locations
	candidates := []string{
		".",
		"../..",
		"../../..",
		"../../../..",
	}
	for _, dir := range candidates {
		if _, err := os.Stat(dir + "/go.mod"); err == nil {
			abs, _ := os.Getwd()
			if dir == "." {
				return abs
			}
			// Return absolute path
			full := abs + "/" + dir
			return full
		}
	}
	t.Fatal("could not find project root (go.mod)")
	return ""
}
