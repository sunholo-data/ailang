package eval_harness

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// RunResult captures the outcome of running generated code
type RunResult struct {
	Stdout      string
	Stderr      string
	ExitCode    int
	Duration    time.Duration // Total time (startup + compile + execution)
	CompileTime time.Duration // Time spent in compilation/type-checking (if separate)
	ExecuteTime time.Duration // Time spent in actual code execution (if measurable)
	CompileOk   bool
	RuntimeOk   bool
	StdoutOk    bool
	TimedOut    bool
}

// LanguageRunner executes code in a specific language
type LanguageRunner interface {
	Run(code string, timeout time.Duration) (*RunResult, error)
	Language() string
}

// PythonRunner executes Python code
type PythonRunner struct{}

// NewPythonRunner creates a new Python runner
func NewPythonRunner() *PythonRunner {
	return &PythonRunner{}
}

// Language returns "python"
func (r *PythonRunner) Language() string {
	return "python"
}

// Run executes Python code
func (r *PythonRunner) Run(code string, timeout time.Duration) (*RunResult, error) {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "eval_*.py")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write code to file
	if _, err := tmpFile.WriteString(code); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write code: %w", err)
	}
	tmpFile.Close()

	// Execute with timeout
	start := time.Now()
	cmd := exec.Command("python3", tmpFile.Name())

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start command
	if err := cmd.Start(); err != nil {
		return &RunResult{
			Stderr:    err.Error(),
			ExitCode:  -1,
			Duration:  time.Since(start),
			CompileOk: false,
			RuntimeOk: false,
		}, nil
	}

	// Wait with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		// Wait for the goroutine to finish after kill to avoid race
		<-done
		return &RunResult{
			Stdout:    stdout.String(),
			Stderr:    "execution timed out",
			ExitCode:  -1,
			Duration:  timeout,
			CompileOk: true,
			RuntimeOk: false,
			TimedOut:  true,
		}, nil
	case err := <-done:
		duration := time.Since(start)
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = -1
			}
		}

		return &RunResult{
			Stdout:    stdout.String(),
			Stderr:    stderr.String(),
			ExitCode:  exitCode,
			Duration:  duration,
			CompileOk: true, // Python has no separate compile step
			RuntimeOk: exitCode == 0,
		}, nil
	}
}

// AILANGRunner executes AILANG code
type AILANGRunner struct {
	ailangPath string
	caps       []string
}

// NewAILANGRunner creates a new AILANG runner
func NewAILANGRunner(ailangPath string, caps []string) *AILANGRunner {
	if ailangPath == "" {
		ailangPath = "ailang" // Use PATH
	}
	return &AILANGRunner{
		ailangPath: ailangPath,
		caps:       caps,
	}
}

// Language returns "ailang"
func (r *AILANGRunner) Language() string {
	return "ailang"
}

// Run executes AILANG code
func (r *AILANGRunner) Run(code string, timeout time.Duration) (*RunResult, error) {
	// Get current working directory (repo root for stdlib access)
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create benchmark directory in current directory
	benchmarkDir := filepath.Join(cwd, "benchmark")
	if err := os.MkdirAll(benchmarkDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create benchmark dir: %w", err)
	}
	tmpFile := filepath.Join(benchmarkDir, "solution.ail")

	// Write code to file
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("failed to write code: %w", err)
	}
	defer os.Remove(tmpFile) // Clean up after execution

	// Build command with flags BEFORE filename (required by ailang CLI)
	args := []string{"run", "--entry", "main", "--quiet"}

	// Add capabilities if specified
	if len(r.caps) > 0 {
		args = append(args, "--caps", strings.Join(r.caps, ","))
	}

	// Add filename last
	args = append(args, "benchmark/solution.ail")

	// Execute with timeout from current directory (for stdlib access)
	start := time.Now()
	cmd := exec.Command(r.ailangPath, args...)
	cmd.Dir = cwd // Run from current directory

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start command
	if err := cmd.Start(); err != nil {
		return &RunResult{
			Stderr:    err.Error(),
			ExitCode:  -1,
			Duration:  time.Since(start),
			CompileOk: false,
			RuntimeOk: false,
		}, nil
	}

	// Wait with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		// Wait for the goroutine to finish after kill to avoid race
		<-done
		return &RunResult{
			Stdout:    stdout.String(),
			Stderr:    "execution timed out",
			ExitCode:  -1,
			Duration:  timeout,
			CompileOk: true,
			RuntimeOk: false,
			TimedOut:  true,
		}, nil
	case err := <-done:
		duration := time.Since(start)
		exitCode := 0
		compileOk := true
		runtimeOk := true

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = -1
			}
			runtimeOk = false

			// Detect compile errors vs runtime errors
			stderrStr := stderr.String()
			if strings.Contains(stderrStr, "parse error") ||
				strings.Contains(stderrStr, "type error") ||
				strings.Contains(stderrStr, "syntax error") {
				compileOk = false
			}
		}

		return &RunResult{
			Stdout:    stdout.String(),
			Stderr:    stderr.String(),
			ExitCode:  exitCode,
			Duration:  duration,
			CompileOk: compileOk,
			RuntimeOk: runtimeOk,
		}, nil
	}
}

// CompareOutput checks if actual output matches expected output
func CompareOutput(expected, actual string) bool {
	// Normalize whitespace
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)

	// For now, do exact string comparison
	// Could be enhanced with fuzzy matching or line-by-line comparison
	return expected == actual
}

// GetRunner returns a LanguageRunner for the specified language
func GetRunner(lang string, spec *BenchmarkSpec) (LanguageRunner, error) {
	switch lang {
	case "python":
		return NewPythonRunner(), nil
	case "ailang":
		return NewAILANGRunner("", spec.Caps), nil
	default:
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}
}

// FindAILANG attempts to locate the ailang binary
func FindAILANG() (string, error) {
	// Try common locations
	paths := []string{
		"ailang",       // In PATH
		"./bin/ailang", // Local build
		filepath.Join(os.Getenv("GOPATH"), "bin", "ailang"), // GOPATH
	}

	for _, path := range paths {
		if _, err := exec.LookPath(path); err == nil {
			return path, nil
		}
		if _, err := os.Stat(path); err == nil {
			absPath, _ := filepath.Abs(path)
			return absPath, nil
		}
	}

	return "", fmt.Errorf("ailang binary not found in PATH or common locations")
}
