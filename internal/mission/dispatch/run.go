package dispatch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

type Factory interface {
	GetExecutor(string) (executor.Executor, error)
}

type Runner struct {
	Models    *modelreg.ModelsConfig
	Executors Factory
	Record    func(Event) error
	LookupEnv func(string) string
	Admit     func(context.Context, Candidate) (Admission, error)
}

type Attempt struct {
	Candidate Candidate `json:"candidate"`
	Status    string    `json:"status"`
	Reason    string    `json:"reason,omitempty"`
}

type Report struct {
	Version            int              `json:"version"`
	AttemptID          string           `json:"attempt_id"`
	RequestDigest      string           `json:"request_digest"`
	Status             string           `json:"status"`
	ArtifactVerified   bool             `json:"artifact_verified"`
	IdentityProvenance string           `json:"identity_provenance"`
	Attempts           []Attempt        `json:"attempts"`
	Progress           *Progress        `json:"progress,omitempty"`
	Result             *executor.Result `json:"execution,omitempty"`
}

// Run falls through only before ExecuteStreaming. Once an executor starts,
// failure can leave artifacts behind and must never trigger a blind rerun.
func (r *Runner) Run(ctx context.Context, req Request) (*Report, error) {
	p, err := Resolve(req, r.Models)
	if err != nil {
		return nil, err
	}
	if r.Executors == nil || r.Record == nil || r.Admit == nil {
		return nil, fmt.Errorf("executor factory, admission policy and durable receipt recorder are required")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds)*time.Second)
	defer cancel()
	report := &Report{Version: 1, AttemptID: req.AttemptID, RequestDigest: p.RequestDigest, Status: "blocked", IdentityProvenance: p.IdentityProvenance}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	if err := r.Record(Event{Kind: "started", Plan: p, Request: &req, RequestDigest: p.RequestDigest}); err != nil {
		return report, err
	}
	for _, c := range p.Candidates {
		if err := ctx.Err(); err != nil {
			return r.finish(report, err)
		}
		e, reason := r.preflight(ctx, c)
		a := Attempt{Candidate: c, Status: "skipped", Reason: reason}
		if reason != "" {
			report.Attempts = append(report.Attempts, a)
			if err := r.Record(Event{Kind: "candidate_skipped", RequestDigest: p.RequestDigest, Attempt: &a}); err != nil {
				return report, err
			}
			continue
		}
		if err := ctx.Err(); err != nil {
			return r.finish(report, err)
		}
		if reason := r.admission(ctx, c); reason != "" {
			a.Reason = reason
			report.Attempts = append(report.Attempts, a)
			if err := r.Record(Event{Kind: "candidate_skipped", RequestDigest: p.RequestDigest, Attempt: &a}); err != nil {
				return report, err
			}
			continue
		}
		a.Status = "executing"
		report.Attempts = append(report.Attempts, a)
		if err := r.Record(Event{Kind: "dispatching", RequestDigest: p.RequestDigest, Attempt: &a}); err != nil {
			return report, err
		}
		if err := ctx.Err(); err != nil {
			return r.finish(report, err)
		}
		observer := newProgressObserver(p.RequestDigest, r.Record)
		result, runErr := e.ExecuteStreaming(ctx, taskFor(req, c), observer)
		progress := observer.snapshot()
		report.Progress = &progress
		runErr = errors.Join(runErr, observer.failure())
		report.Result = result
		if result != nil && result.CostProvenance == "" {
			result.CostProvenance = executor.CostProvenanceUnknown
		}
		runErr = executionError(ctx, req, result, runErr)
		i := len(report.Attempts) - 1
		if runErr != nil {
			report.Status = "execution_failed"
			report.Attempts[i].Status = "failed"
			report.Attempts[i].Reason = runErr.Error()
		} else {
			report.Status = "execution_completed"
			report.Attempts[i].Status = "completed"
		}
		return r.finish(report, runErr)
	}
	return r.finish(report, fmt.Errorf("no compatible, healthy role candidate; inspect receipt attempts"))
}

func (r *Runner) finish(report *Report, runErr error) (*Report, error) {
	if err := r.Record(Event{Kind: "finished", RequestDigest: report.RequestDigest, Report: report}); err != nil {
		return report, fmt.Errorf("completion receipt failed (execution status %s): %w", report.Status, err)
	}
	return report, runErr
}

func (r *Runner) preflight(ctx context.Context, c Candidate) (executor.Executor, string) {
	if c.SkipReason != "" {
		return nil, c.SkipReason
	}
	if reason := r.admission(ctx, c); reason != "" {
		return nil, reason
	}
	e, err := r.Executors.GetExecutor(c.Executor)
	if err != nil {
		return nil, err.Error()
	}
	if e == nil || e.Name() != c.Executor {
		return nil, "factory returned a mismatched executor"
	}
	local := false
	for _, cap := range e.Capabilities() {
		if cap == executor.CapLocalWorkspace {
			local = true
		}
	}
	if !local {
		return nil, "executor cannot access the supplied local workspace"
	}
	lookup := r.LookupEnv
	if lookup == nil {
		lookup = os.Getenv
	}
	var keys []string
	switch c.Executor {
	case "claude":
		keys = []string{"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN"}
	case "codex":
		keys = []string{"OPENAI_API_KEY"}
	}
	for _, key := range keys {
		if lookup(key) != "" {
			return nil, "subscription route refused with ambient " + key
		}
	}
	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := e.HealthCheck(healthCtx); err != nil {
		return nil, "executor health: " + err.Error()
	}
	if err := healthCtx.Err(); err != nil {
		return nil, "executor health: " + err.Error()
	}
	return e, ""
}

// evaluatorTools is what an evaluator may hold: everything readable, nothing that mutates.
//
// pi's defaults are read, bash, edit and write. An evaluator was therefore handed edit and
// write while its own contract told it, in prose, to "preserve HEAD and all tracked files",
// "do not repair the candidate" and "do not add review docs in the workspace" — three
// mutation prohibitions, none enforced. A judge that can silently rewrite the artifact it is
// judging invalidates the acceptance evidence, and no amount of instruction text makes that
// safe. internal/mission/quorum already bounds question-kind tasks this way; the stage path
// simply never did.
//
// bash STAYS. The evaluator contract requires it — the bound verification commands are run
// through it — so a read-only allowlist would break the role rather than protect it.
// grep/find/ls are OFF by default in pi and are added here deliberately: they strictly
// increase what an evaluator can read, so this allowlist cannot break an evaluator that
// worked before, while removing every tool that can write.
var evaluatorTools = []string{"read", "bash", "grep", "find", "ls"}

func taskFor(r Request, c Candidate) *executor.Task {
	m := c.config
	var allowed []string
	isEvaluator := strings.EqualFold(r.Role, "evaluator")
	if isEvaluator {
		allowed = evaluatorTools
	}
	return &executor.Task{
		AllowedTools: allowed,
		// SCOPED TO THE EVALUATOR DELIBERATELY. Its contract states that the bound packet
		// is the evidence, so ambient repo instructions are pure contamination. Author
		// roles are left alone pending a decision: they arguably SHOULD follow the repo's
		// conventions while writing code, and silently changing that could regress every
		// executor with no evidence either way. The determinism argument applies to them
		// too — a frozen work item whose behaviour depends on today's AGENTS.md is not
		// frozen — but that is a ruling, not a refactor.
		IsolateFromAmbientContext: isEvaluator,
		ID:                        strings.Join([]string{r.MissionID, r.WorkItemID, r.StageID, r.AttemptID}, "/"),
		ParentTaskID:              r.WorkItemID, Directive: r.Instructions,
		SystemPrompt: fmt.Sprintf("Mission role contract v1. Role: %s. Input revision declared by caller: %s. Request digest: %s. Produce the requested artifact; execution success is not acceptance or permission to publish.", r.Role, r.InputRevision, r.Digest()),
		Workspace:    r.Workspace, Model: c.WireModel, Timeout: time.Duration(r.TimeoutSeconds) * time.Second,
		MaxTokensPerBench: r.MaxTokens, MaxOutputTokens: m.MaxOutputTokens, ReasoningEffort: m.ReasoningEffort,
		TTFTTimeout: time.Duration(m.TTFTTimeoutSeconds) * time.Second, IdleTimeout: time.Duration(m.GenerationTimeoutSeconds) * time.Second,
		GCPProject: m.GCPProject, GCPLocation: m.GCPLocation,
		Budget:  executor.NewCostBudget(r.MaxCostUSD, m.Pricing.InputPer1K, m.Pricing.OutputPer1K),
		Pricing: &executor.CostModel{InputTokenCost: m.Pricing.InputPer1K, OutputTokenCost: m.Pricing.OutputPer1K, CacheReadCost: m.Pricing.CacheReadPer1K},
		ExtraEnv: map[string]string{
			"AILANG_MESSAGES_STORE": "gcp", "AILANG_MESSAGES_PROJECT": "ailang-multivac",
			// Marks this process as FROZEN STAGE EXECUTION so the repo's Claude Code hooks
			// inject nothing into it. Applies to EVERY role, unlike the AGENTS.md isolation
			// above: repo conventions are arguably an author's business, but prompt-matched
			// brain resolutions and an inbox banner are neither conventions nor contract —
			// they are per-run-variable content, and a work item whose input varies run to
			// run is not frozen. Read by scripts/hooks/{brain_on_prompt,session_start}.sh.
			"AILANG_MISSION_STAGE": "1",
		},
		Metadata: map[string]string{"chain_id": r.WorkItemID, "stage_id": r.StageID, "mission_id": r.MissionID, "request_digest": r.Digest()},
	}
}

func executionError(ctx context.Context, r Request, result *executor.Result, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		return err
	}
	if result == nil {
		return fmt.Errorf("executor returned no result")
	}
	if !result.Success || result.FinishReason != executor.FinishStop || result.CostKilledAt > 0 || result.ThrashKilledAt > 0 {
		return fmt.Errorf("executor did not finish successfully (finish=%q): %s", result.FinishReason, result.Error)
	}
	if result.InputTokens+result.OutputTokens > r.MaxTokens {
		return fmt.Errorf("execution exceeded max_tokens")
	}
	if result.CostProvenance == executor.CostMetered && result.CostUSD > r.MaxCostUSD {
		return fmt.Errorf("execution exceeded max_cost_usd")
	}
	if strings.TrimSpace(result.Output) == "" {
		return fmt.Errorf("executor returned empty output; artifact remains unverified")
	}
	return nil
}
