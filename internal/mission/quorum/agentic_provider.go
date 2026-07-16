package quorum

import (
	"context"
	"fmt"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// NewCoordinatorAgenticRunner builds a production AgenticRunner over the shipped
// coordinator executor layer (internal/coordinator/provider_executor.go),
// REUSING it rather than rebuilding any plumbing (Critical Principle 1):
//   - read-only tool mode via Kind=="question" (provider_executor.go:122-124
//     sets AllowedTools=[Read,Grep,Glob,WebFetch,WebSearch]),
//   - bounded turn/time cap via opts.Timeout + opts.IdleTimeout,
//   - cancellation via Execute(ctx, ...),
//   - observed cost via result.Cost (= execResult.CostUSD),
//   - read-only worktree via opts.Workspace.
//
// executorName is the executor registered in the global factory ("codex" =
// OpenAI, "claude", "managed_agents" = Gemini). workspace is a READ-ONLY
// worktree of the repo at the pinned review SHA (concurrent-agent safety — the
// reviewer must never write the shared tree; Kind=="question" tool restriction
// enforces no write tool, and the worktree itself should be checked out
// read-only by the caller). model is an optional provider-specific model id.
//
// The returned runner maps coordinator.ExecuteResult into an AgenticRun; a
// non-nil executor error or Success=false becomes an AgenticRun with
// Success=false so RunAgenticReviewer records a named absence, never a silent
// pass.
func NewCoordinatorAgenticRunner(executorName, workspace, model string, idleTimeout time.Duration) (AgenticRunner, error) {
	provider, err := coordinator.NewExecutorProvider(executorName)
	if err != nil {
		return nil, fmt.Errorf("resolve %q executor for agentic reviewer: %w", executorName, err)
	}
	return func(ctx context.Context, systemPrompt, userPrompt string) (*AgenticRun, error) {
		// A Kind=="question" task => read-only AllowedTools in provider_executor.go.
		// The directive (userPrompt) carries the doc + verification instruction;
		// the systemPrompt is supplied through the agent config's meta-prompt path.
		task := &coordinator.AnalyzedTask{
			Task: &coordinator.Task{
				ID:      "quorum-agentic-review",
				Content: systemPrompt + "\n\n" + userPrompt,
				Kind:    "question", // read-only tools (no writes to the shared tree)
			},
			Type: coordinator.TaskTypeResearch,
		}
		// Bound the wall clock from the ctx deadline (RunAgenticReviewer's caller
		// sets it) and the idle timeout so a hung run is killed.
		timeout := time.Duration(0)
		if dl, ok := ctx.Deadline(); ok {
			timeout = time.Until(dl)
		}
		opts := &coordinator.ExecuteOptions{
			Workspace:   workspace, // read-only worktree
			Model:       model,
			Timeout:     timeout,
			IdleTimeout: idleTimeout,
		}
		res, execErr := provider.Execute(ctx, task, opts)
		if execErr != nil {
			return &AgenticRun{Success: false, Err: execErr.Error()}, nil
		}
		if res == nil {
			return &AgenticRun{Success: false, Err: "executor returned nil result"}, nil
		}
		return &AgenticRun{
			Output:  res.Output,
			Success: res.Success,
			Err:     res.Error,
			CostUSD: res.Cost,
		}, nil
	}, nil
}
