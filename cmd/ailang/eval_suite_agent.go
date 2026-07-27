package main

// Agent-mode configuration for `ailang eval-suite`.
//
// Extracted from eval_suite.go, which was at 798/800 lines against the hard
// `make check-file-sizes` CI gate. This is the cohesive "build the
// AgentBenchmarkConfig from the resolved flags" unit, including the two
// condition-triggered prompt loads.

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sunholo-data/ailang/internal/agentprompt"
	"github.com/sunholo-data/ailang/internal/devtoolsprompt"
	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// agentSuiteConfigParams is the resolved flag set the agent config derives from.
type agentSuiteConfigParams struct {
	models             []string
	conditions         []string
	agentModelOverride string
	maxConcurrent      int
	requestsPerSecond  int
	timeoutSeconds     int
	maxTokensPerBench  int
	verify             bool
	verifyTimeout      time.Duration
}

// buildAgentSuiteConfig assembles the AgentBenchmarkConfig for an agent-mode run
// and prints the agent-mode banner. Returns nil for a standard-mode run.
func buildAgentSuiteConfig(agent bool, p agentSuiteConfigParams) *eval_harness.AgentBenchmarkConfig {
	if !agent {
		return nil
	}

	fmt.Println()
	fmt.Printf("%s Agent mode ENABLED\n", cyan("🤖"))
	fmt.Printf("  - Models: %v\n", p.models)
	if p.agentModelOverride != "" {
		fmt.Printf("  - Agent CLI model: %s (override)\n", p.agentModelOverride)
	} else {
		fmt.Printf("  - Agent CLI model: per-model lookup from models.yml\n")
	}
	// Dispatch parallelism is governed by -parallel (the runBenchmarksParallel
	// semaphore). Print THAT value so the banner doesn't mislead the user.
	fmt.Printf("  - Dispatch parallelism: %d (-parallel flag)\n", p.maxConcurrent)
	fmt.Printf("  - Rate limit: %d req/sec\n", p.requestsPerSecond)
	fmt.Printf("  - Timeout: %d seconds\n", p.timeoutSeconds)
	if p.verify {
		fmt.Printf("  - Contract verification: ON (ai-check, %s per-function Z3 timeout)\n", p.verifyTimeout)
	}
	fmt.Println()

	return &eval_harness.AgentBenchmarkConfig{
		MaxTokensPerBench:  p.maxTokensPerBench,
		RequestsPerSecond:  p.requestsPerSecond,
		TimeoutSeconds:     p.timeoutSeconds,
		WorkspaceDir:       filepath.Join(os.TempDir(), "ailang_eval"),
		AllowedTools:       []string{"Bash", "Read", "Write", "Edit", "Grep"},
		ClaudePath:         "claude",             // Use PATH
		ClaudeModel:        p.agentModelOverride, // Empty unless override specified
		Verify:             p.verify,             // M-CONTRACT-EVAL: enable contract verification
		DevtoolsPrompt:     loadDevtoolsPrompt(p.conditions),
		AgentPromptContent: loadAgentCodingPrompt(p.conditions),
		MicroragMode:       evalMicroragMode, // M-BRAIN-MICRORAG: subprocess env mode
		FmtHook:            evalFmtHookMode,  // M-EVAL-FMT-WEAKMODEL-AB: fmt PostToolUse hook A/B toggle
	}
}

// loadDevtoolsPrompt loads the devtools prompt when --devtools-prompt is set or
// the "full" condition is requested (M-CONTRACT-EVAL).
func loadDevtoolsPrompt(conditions []string) string {
	need := evalDevtoolsPromptFlag
	if !need {
		for _, c := range conditions {
			if c == "full" {
				need = true
				break
			}
		}
	}
	if !need {
		return ""
	}
	content, err := devtoolsprompt.LoadPrompt("v0.8.0-compact")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s Failed to load devtools prompt: %v\n", yellow("⚠️"), err)
		return ""
	}
	return content
}

// loadAgentCodingPrompt loads the agent coding prompt when the "agent_prompt"
// condition is requested.
func loadAgentCodingPrompt(conditions []string) string {
	for _, c := range conditions {
		if c != "agent_prompt" {
			continue
		}
		content, err := agentprompt.LoadPrompt("latest")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s Failed to load agent prompt: %v\n", yellow("⚠️"), err)
			return ""
		}
		fmt.Printf("  - Agent prompt loaded (%d bytes)\n", len(content))
		return content
	}
	return ""
}
