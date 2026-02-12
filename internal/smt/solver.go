package smt

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SolverStatus represents the result of an SMT solver invocation.
type SolverStatus int

const (
	StatusVerified       SolverStatus = iota // unsat — postconditions hold for all inputs
	StatusCounterexample                     // sat — found inputs that violate postconditions
	StatusUnknown                            // unknown — solver could not determine
	StatusError                              // error — solver error or not found
)

func (s SolverStatus) String() string {
	switch s {
	case StatusVerified:
		return "verified"
	case StatusCounterexample:
		return "counterexample"
	case StatusUnknown:
		return "unknown"
	case StatusError:
		return "error"
	default:
		return fmt.Sprintf("status(%d)", int(s))
	}
}

// ModelBinding represents a variable assignment from a counterexample.
type ModelBinding struct {
	Name  string `json:"name"`
	Sort  string `json:"sort"`
	Value string `json:"value"`
}

// SolverResult holds the result of running Z3 on an SMT-LIB program.
type SolverResult struct {
	Status    SolverStatus
	Model     []ModelBinding // Populated when Status == StatusCounterexample
	Duration  time.Duration
	SMTLib    string // The input SMT-LIB program (for --verbose)
	RawOutput string // Raw Z3 stdout
	Error     string // Error message if Status == StatusError
}

// SolverConfig holds configuration for the Z3 solver.
type SolverConfig struct {
	Z3Path  string        // Path to z3 binary (auto-discovered if empty)
	Timeout time.Duration // Per-function timeout (default 5s)
}

// DefaultSolverConfig returns a SolverConfig with sensible defaults.
func DefaultSolverConfig() SolverConfig {
	return SolverConfig{
		Timeout: 5 * time.Second,
	}
}

// FindZ3 discovers the Z3 binary location.
// Search order: AILANG_Z3_PATH env, PATH, common locations.
func FindZ3() (string, error) {
	// 1. Check AILANG_Z3_PATH environment variable
	if envPath := os.Getenv("AILANG_Z3_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
		return "", fmt.Errorf("AILANG_Z3_PATH set to %q but file not found", envPath)
	}

	// 2. Check PATH
	if path, err := exec.LookPath("z3"); err == nil {
		return path, nil
	}

	// 3. Check common locations
	commonPaths := []string{
		"/opt/homebrew/bin/z3", // macOS Homebrew (Apple Silicon)
		"/usr/local/bin/z3",    // macOS Homebrew (Intel) / Linux
		"/usr/bin/z3",          // Linux package manager
		"/snap/bin/z3",         // Ubuntu snap
	}
	for _, p := range commonPaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("Z3 solver not found. Install with: brew install z3 (macOS) or apt install z3 (Linux). Or set AILANG_Z3_PATH")
}

// Solve runs Z3 on an SMT-LIB program and parses the result.
func Solve(smtlib string, config SolverConfig) (*SolverResult, error) {
	start := time.Now()
	result := &SolverResult{
		SMTLib: smtlib,
	}

	// Find Z3
	z3Path := config.Z3Path
	if z3Path == "" {
		var err error
		z3Path, err = FindZ3()
		if err != nil {
			result.Status = StatusError
			result.Error = err.Error()
			result.Duration = time.Since(start)
			return result, nil
		}
	}

	// Write SMT-LIB to temp file
	tmpFile, err := os.CreateTemp("", "ailang-smt-*.smt2")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(smtlib); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("writing SMT-LIB: %w", err)
	}
	tmpFile.Close()

	// Build Z3 command with timeout
	timeoutSecs := int(config.Timeout.Seconds())
	if timeoutSecs < 1 {
		timeoutSecs = 5
	}
	args := []string{
		"-smt2",
		fmt.Sprintf("-T:%d", timeoutSecs),
		tmpPath,
	}

	cmd := exec.Command(z3Path, args...)
	output, err := cmd.CombinedOutput()
	result.Duration = time.Since(start)
	result.RawOutput = string(output)

	// Parse result - Z3 exits 0 for sat/unsat, non-zero for errors
	// But also exits non-zero when model is unavailable (unsat + get-model)
	outputStr := strings.TrimSpace(string(output))

	if strings.HasPrefix(outputStr, "unsat") {
		result.Status = StatusVerified
		return result, nil
	}

	if strings.HasPrefix(outputStr, "sat") {
		result.Status = StatusCounterexample
		result.Model = parseModel(outputStr)
		return result, nil
	}

	if strings.HasPrefix(outputStr, "unknown") {
		result.Status = StatusUnknown
		return result, nil
	}

	// Check for timeout
	if strings.Contains(outputStr, "timeout") {
		result.Status = StatusUnknown
		result.Error = "solver timeout"
		return result, nil
	}

	// Other error
	if err != nil {
		result.Status = StatusError
		result.Error = fmt.Sprintf("Z3 error (exit %v): %s", err, outputStr)
		return result, nil
	}

	result.Status = StatusError
	result.Error = fmt.Sprintf("unexpected Z3 output: %s", outputStr)
	return result, nil
}

// parseModel extracts variable bindings from Z3 sat output.
// Z3 model format:
//
//	sat
//	(
//	  (define-fun x () Int
//	    0)
//	  (define-fun result () Int
//	    5)
//	)
func parseModel(output string) []ModelBinding {
	var bindings []ModelBinding
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Look for (define-fun name () Type
		if strings.HasPrefix(line, "(define-fun ") {
			binding := parseDefineFun(line, scanner)
			if binding != nil {
				bindings = append(bindings, *binding)
			}
		}
	}
	return bindings
}

// parseDefineFun parses a single define-fun from the model.
// Format: (define-fun name () Type\n    value)
func parseDefineFun(firstLine string, scanner *bufio.Scanner) *ModelBinding {
	// Remove "(define-fun " prefix
	rest := strings.TrimPrefix(firstLine, "(define-fun ")

	// Parse: name () Type
	parts := strings.Fields(rest)
	if len(parts) < 3 {
		return nil
	}

	name := parts[0]
	// parts[1] should be "()"
	sort := parts[2]

	// Check if value is on the same line
	remaining := strings.Join(parts[3:], " ")
	remaining = strings.TrimSuffix(remaining, ")")

	if remaining != "" {
		return &ModelBinding{
			Name:  name,
			Sort:  sort,
			Value: strings.TrimSpace(remaining),
		}
	}

	// Value is on the next line
	if scanner.Scan() {
		valueLine := strings.TrimSpace(scanner.Text())
		valueLine = strings.TrimSuffix(valueLine, ")")
		return &ModelBinding{
			Name:  name,
			Sort:  sort,
			Value: strings.TrimSpace(valueLine),
		}
	}

	return nil
}

// Z3Available returns true if Z3 is installed and accessible.
func Z3Available() bool {
	_, err := FindZ3()
	return err == nil
}

// Z3Version returns the installed Z3 version string, or empty if not found.
func Z3Version() string {
	z3Path, err := FindZ3()
	if err != nil {
		return ""
	}
	out, err := exec.Command(z3Path, "--version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// SolveFile runs Z3 on an SMT-LIB file.
func SolveFile(path string, config SolverConfig) (*SolverResult, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("reading SMT-LIB file: %w", err)
	}
	return Solve(string(data), config)
}
