package eval_harness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// AICheckResult is the parsed JSON output from `ailang ai-check`
type AICheckResult struct {
	File   string              `json:"file"`
	Check  AICheckCheckResult  `json:"check"`
	Verify AICheckVerifyResult `json:"verify"`
}

// AICheckCheckResult is the type-check portion
type AICheckCheckResult struct {
	Passed     bool `json:"passed"`
	ErrorCount int  `json:"error_count"`
}

// AICheckVerifyResult is the contract verification portion
type AICheckVerifyResult struct {
	Available      bool `json:"available"`
	Verified       int  `json:"verified"`
	Counterexample int  `json:"counterexample"`
	Skipped        int  `json:"skipped"`
	Errors         int  `json:"errors"`
}

// RunAICheck executes `ailang ai-check <file>` and parses the JSON output.
// Returns nil result and error if the command can't be executed.
// Returns parsed result even if verification found counterexamples.
func RunAICheck(ailangPath, filePath string, timeout time.Duration) (*AICheckResult, string, error) {
	if ailangPath == "" {
		ailangPath = "ailang"
	}

	args := []string{"ai-check", "--timeout", timeout.String(), filePath}
	cmd := exec.Command(ailangPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run with a generous timeout (ai-check itself has per-function timeout)
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	cmdTimeout := timeout*3 + 10*time.Second // Allow generous time for full run
	select {
	case <-time.After(cmdTimeout):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		<-done
		return nil, "", fmt.Errorf("ai-check timed out after %v", cmdTimeout)
	case err := <-done:
		// ai-check exits 0 even with counterexamples, exits non-zero on errors
		rawOutput := stdout.String()
		if rawOutput == "" && err != nil {
			return nil, stderr.String(), fmt.Errorf("ai-check failed: %w\nstderr: %s", err, stderr.String())
		}

		var result AICheckResult
		if jsonErr := json.Unmarshal([]byte(rawOutput), &result); jsonErr != nil {
			return nil, rawOutput, fmt.Errorf("failed to parse ai-check JSON: %w\noutput: %s", jsonErr, rawOutput)
		}

		return &result, rawOutput, nil
	}
}

// PopulateVerifyMetrics fills verify fields in RunMetrics from an AICheckResult
func PopulateVerifyMetrics(metrics *RunMetrics, result *AICheckResult, rawJSON string) {
	if result == nil {
		return
	}
	metrics.VerifyVerified = result.Verify.Verified
	metrics.VerifyCounterex = result.Verify.Counterexample
	metrics.VerifySkipped = result.Verify.Skipped
	metrics.VerifyErrors = result.Verify.Errors
	metrics.VerifyOk = result.Verify.Available && result.Verify.Counterexample == 0 && result.Verify.Errors == 0
	metrics.VerifyJSON = rawJSON
}
