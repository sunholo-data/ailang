// Package main is a small smoke-test runner for the motoko executor adapter.
//
// Usage:
//
//	OPENROUTER_API_KEY=... go run ./cmd/smoke-motoko -model openrouter/anthropic/claude-haiku-4-5 -task "say hello"
//
// Spawns motoko via the adapter against a fresh tmpdir, parses the resulting
// session JSONL, and prints a compact human-readable summary of the Result.
//
// This is M5 of M-MOTOKO-EXECUTOR-ADAPTER (threshold-measurement) at the
// single-task granularity — just enough to validate the parser + cost rollup
// against a real Anthropic-backed run before we wire eval-suite paired
// comparisons.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
	_ "github.com/sunholo-data/ailang/internal/executor/motoko" // init() registration
)

func main() {
	var (
		model    = flag.String("model", "openrouter/anthropic/claude-haiku-4-5", "motoko model id (openrouter/* form)")
		task     = flag.String("task", "Print the number 42 to stdout. Just the number, nothing else. Use python3 -c if you need a shell.", "task directive")
		profile  = flag.String("profile", "dogfood", "MOTOKO_CONFIG profile name")
		motoPath = flag.String("motoko", "motoko", "path to motoko binary (or just 'motoko' if on PATH)")
		timeout  = flag.Duration("timeout", 5*time.Minute, "hard ceiling for the subprocess")
	)
	flag.Parse()

	if os.Getenv("OPENROUTER_API_KEY") == "" {
		fmt.Fprintln(os.Stderr, "ERROR: OPENROUTER_API_KEY is not set; motoko routes via OpenRouter")
		os.Exit(2)
	}

	// motoko's wrapper does `cd "$MOTOKO_REPO"` before exec'ing — JSONL
	// lands there, NOT in the task workspace. The adapter's
	// findSessionJSONL has a MOTOKO_REPO env-var fallback for exactly this;
	// default it to the wrapper's hardcoded fallback if the user hasn't
	// set it explicitly.
	if os.Getenv("MOTOKO_REPO") == "" {
		_ = os.Setenv("MOTOKO_REPO", "/Users/mark/dev/sunholo/motoko_agent")
	}

	// Mutate the global factory's config in place — the motoko package's
	// init() already registered the builder against this factory, so we
	// must NOT replace it (that would orphan all registrations).
	executor.GlobalFactory().UpdateConfig(func(c *executor.Config) {
		c.MotokoPath = *motoPath
		c.MotokoModel = *model
		c.MotokoProfile = *profile
	})

	exec, err := executor.GlobalFactory().GetExecutor("motoko")
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetExecutor: %v\n", err)
		os.Exit(2)
	}

	wsDir, err := os.MkdirTemp("", "motoko-smoke-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "mkdir temp: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = os.RemoveAll(wsDir) }()

	fmt.Printf("=== motoko smoke test ===\n")
	fmt.Printf("Model:     %s\n", *model)
	fmt.Printf("Profile:   %s\n", *profile)
	fmt.Printf("Workspace: %s\n", wsDir)
	fmt.Printf("Task:      %s\n", *task)
	fmt.Println()

	if err := exec.HealthCheck(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "HealthCheck failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("HealthCheck: OK")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	startTime := time.Now()
	res, err := exec.Execute(ctx, &executor.Task{
		Workspace: wsDir,
		Directive: *task,
		Model:     *model,
	})
	elapsed := time.Since(startTime)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Execute returned hard error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Result ===")
	fmt.Printf("Success:      %v\n", res.Success)
	if res.Error != "" {
		fmt.Printf("Error:        %s\n", res.Error)
	}
	fmt.Printf("SessionID:    %s\n", res.SessionID)
	fmt.Printf("Duration:     %v (wall-clock %v)\n", time.Duration(res.DurationMS)*time.Millisecond, elapsed)
	fmt.Printf("Turns:        %d\n", res.NumTurns)
	fmt.Printf("Tool calls:   %d\n", res.ToolCallCount)
	fmt.Printf("Input tok:    %d\n", res.InputTokens)
	fmt.Printf("Output tok:   %d\n", res.OutputTokens)
	if res.CacheReadInputTokens > 0 {
		fmt.Printf("Cache read:   %d\n", res.CacheReadInputTokens)
	}
	if res.CacheCreationInputTokens > 0 {
		fmt.Printf("Cache write:  %d\n", res.CacheCreationInputTokens)
	}
	fmt.Printf("Cost USD:     $%.6f\n", res.CostUSD)
	if commit, ok := res.ProviderData["motoko_commit"]; ok {
		fmt.Printf("motoko commit: %v\n", commit)
	}
	if dp7, ok := res.ProviderData["dp7_rejections"]; ok {
		fmt.Printf("DP7 rejections: %v\n", dp7)
	}
	if fr, ok := res.ProviderData["motoko_finish_reason"]; ok {
		fmt.Printf("Finish reason: %v\n", fr)
	}
	if res.Output != "" {
		preview := res.Output
		if len(preview) > 300 {
			preview = preview[:300] + "..."
		}
		fmt.Printf("\nOutput preview:\n%s\n", preview)
	}

	// Surface where the JSONL is so the user can inspect it post-run
	// (we deliberately don't auto-delete the workspace until program exit,
	// so this is informational only — user can copy it before exit).
	if res.SessionID != "" {
		jsonlPath := filepath.Join(wsDir, ".motoko", "logfile", res.SessionID+".jsonl")
		if _, err := os.Stat(jsonlPath); err == nil {
			fmt.Printf("\nSession JSONL (will be deleted on exit): %s\n", jsonlPath)
		}
	}

	if !res.Success {
		os.Exit(1)
	}
}
