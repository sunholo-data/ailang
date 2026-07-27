package eval_harness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// DefaultVerifyTimeout is the per-function Z3 timeout for contract verification.
//
// SINGLE source of the number: the `--verify-timeout` flag default, the
// package-level eval config default, DefaultAgentConfig, and the agent-mode
// verifier's zero-value fallback all reference it. It used to be four separate
// literal `5 * time.Second`s, which is how the agent path came to hardcode its
// own (BF-3) while the cohort manifest recorded the flag's — an artifact that
// would have documented a timeout the run never used.
const DefaultVerifyTimeout = 5 * time.Second

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

	// M-EVAL-MEM-GUARD: ai-check runs the type checker + Z3 over MODEL source,
	// so guard it like the execution lanes — own process group (kills any
	// solver children with it) and the memory watchdog.
	SetProcessGroup(cmd)
	maxRSS, err := evalMaxRSS()
	if err != nil {
		return nil, "", err
	}
	if startErr := cmd.Start(); startErr != nil {
		return nil, "", fmt.Errorf("ai-check failed to start: %w", startErr)
	}

	cmdTimeout := timeout*3 + 10*time.Second // Allow generous time for full run (ai-check has per-function timeouts)
	g := waitWithGuards(cmd, cmdTimeout, maxRSS)
	if g.timedOut {
		return nil, "", fmt.Errorf("ai-check timed out after %v", cmdTimeout)
	}
	if g.memKilled {
		return nil, "", fmt.Errorf("ai-check killed: %s process group exceeded the %s memory cap", MemKillMarker, EnvEvalMaxRSS)
	}

	// ai-check exits 0 even with counterexamples, exits non-zero on errors
	rawOutput := stdout.String()
	if rawOutput == "" && g.waitErr != nil {
		return nil, stderr.String(), fmt.Errorf("ai-check failed: %w\nstderr: %s", g.waitErr, stderr.String())
	}

	var result AICheckResult
	if jsonErr := json.Unmarshal([]byte(rawOutput), &result); jsonErr != nil {
		return nil, rawOutput, fmt.Errorf("failed to parse ai-check JSON: %w\noutput: %s", jsonErr, rawOutput)
	}

	return &result, rawOutput, nil
}

// agentAICheck is the ai-check invocation used by the AGENT-mode verifier.
// A package-level var so tests can inject a fake verifier and exercise the
// wiring with no subprocess, no Z3 and no benchmark execution.
var agentAICheck = RunAICheck

// applyAgentVerification is THE agent-mode contract-verification step
// (M-CONTRACT-EVAL + M-COST-PER-SUCCESS-KPI M4a-3).
//
// BF-1: this logic previously existed ONLY inside RunAgentBenchmark(), the legacy
// Claude-hardcoded runner, which had no caller — cmd/ailang/eval_benchmark.go
// always uses RunAgentBenchmarkWithExecutor. So no agent run could ever bank
// verify_verified > 0, isVerifiedSuccess() was permanently false, and
// `cost-per-verified-success` returned zero_denominator forever. Per CLAUDE.md §3
// the fix was to MOVE verification onto the live path and delete the dead one —
// one funnel, not two divergent copies. Every agent-mode caller goes through here.
//
// The gate is preserved exactly: Verify && ContractSpec != "" && ailang && compiled.
// When any condition fails, the verify block stays ALL-ZERO, so the run
// classifies as verification_missing (an unverified pass) and can never count as
// a verified success.
func applyAgentVerification(out *AgentBenchmarkResult, config AgentBenchmarkConfig, spec *BenchmarkSpec, language, solutionPath string, compileOk bool) {
	if out == nil || spec == nil {
		return
	}
	if !config.Verify || spec.ContractSpec == "" || language != "ailang" || !compileOk {
		return
	}

	result, rawJSON, err := agentAICheck("", solutionPath, config.ResolvedVerifyTimeout())
	if err != nil {
		// Do NOT swallow this (the dead path discarded it with `_`). A verification
		// that could not run is not a verification that passed: the block stays
		// zero so the run is banked as an unverified pass, and the reason is
		// visible to whoever reads the run log (CLAUDE.md §2).
		fmt.Fprintf(os.Stderr, "[eval] contract verification did not run for %s (%s): %v\n",
			spec.ID, solutionPath, err)
		return
	}
	if result == nil {
		fmt.Fprintf(os.Stderr, "[eval] contract verification returned no result for %s\n", spec.ID)
		return
	}

	out.VerifyVerified = result.Verify.Verified
	out.VerifyCounterex = result.Verify.Counterexample
	out.VerifySkipped = result.Verify.Skipped
	out.VerifyErrors = result.Verify.Errors
	out.VerifyOk = result.Verify.Available && result.Verify.Counterexample == 0 && result.Verify.Errors == 0
	out.VerifyJSON = rawJSON
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
