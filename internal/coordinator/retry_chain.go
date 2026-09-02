package coordinator

import (
	"fmt"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/modelreg"
)

// M-COORDINATOR-EXECUTION-TRUST M3 — the fallback chain, cheapest tier first.
//
// ResolveModel returned chain[0] and discarded the rest, justified by "retry
// chains are a driver concern". That was true when the mission driver was the
// only dispatcher. The coordinator now dispatches unattended, so it needs the
// chain MORE than the driver does, not less: task-c429f9e0 died on
// "pi idle for 3m0.006592232s mid-generation (no output)" with no second lane.
//
// Retry is two-tier, matched to the failure taxonomy (Mark, D4):
//
//	model/provider class — in-container chain walk, inside the existing task
//	                       timeout. Costs nothing extra: no re-clone, no pull.
//	infrastructure class — ONE re-dispatch on the next link. In-container retry
//	                       is impossible by definition here (the container is
//	                       gone), which is exactly why the second tier exists.
//
// Hard cap: 2 Cloud Run executions per task, persisted so a scale-to-zero
// cannot forget it.

// FailureClass says what kind of failure an executor reported, and therefore
// what may be done about it.
type FailureClass string

const (
	// FailureTransport is a provider or connection failure that says nothing
	// about the request. Advancing to the next chain link may well succeed.
	FailureTransport FailureClass = "transport"

	// FailureModel is an outcome: a refusal, a wrong answer, code that does not
	// compile. A second lane produces a second opinion, not a fix, so the chain
	// does not advance.
	FailureModel FailureClass = "model"
)

// transportSignatures are NAMED patterns, deliberately not loose substrings.
//
// The trap this avoids is real and was found in review of a sibling design: an
// isRetryableError that returned true on any "429" substring would retry into a
// spent quota bucket, and would also fire on "benchmark run 429 of 500". Each
// entry below is specific enough to identify the mechanism it names.
var transportSignatures = []string{
	"mid-generation (no output)", // the measured idle stall (design doc V14)
	"status 429",
	"rate_limit",
	"rate limit exceeded",
	"status 500",
	"status 502",
	"status 503",
	"status 504",
	"context deadline exceeded",
	"connection reset",
	"eof",
	"returned zero bytes",
	"empty completion",
}

// ClassifyFailure decides whether an executor's error message describes a
// transport failure worth a second lane. Anything unrecognised is model-class:
// the safe direction is NOT to spend another execution.
func ClassifyFailure(errMsg string) FailureClass {
	lower := strings.ToLower(errMsg)
	for _, sig := range transportSignatures {
		if strings.Contains(lower, sig) {
			return FailureTransport
		}
	}
	return FailureModel
}

// MaxTaskExecutions is the hard cap on Cloud Run executions per task: the
// original dispatch plus at most one infra-class re-dispatch.
const MaxTaskExecutions = 2

// ShouldReDispatch reports whether the stale-task detector may re-dispatch.
//
// The stale-task detector is the SOLE re-dispatcher. Four components can
// already move a task toward a terminal state — this one, the completion
// handler, the stranded-approval sweep and the worktree sweep (design doc V23)
// — and any two of them gaining re-dispatch would breach the cap and duplicate
// work nondeterministically. The other three observe and report.
//
// ageKnown carries getTaskAge's refusal to act on an unknowable age. That
// refusal is load-bearing: time.Since(zero) is ~292 years, which marked tasks
// timed-out about 57s after dispatch and fed the 591-message amplification loop
// e0b12bf5f fixed. An unknowable age is never a reason to spend an execution.
func ShouldReDispatch(task *TaskRecord, age time.Duration, ageKnown bool) bool {
	if task == nil || !ageKnown {
		return false
	}
	if IsTerminalStatus(task.Status) {
		return false // the completion handler owns it; re-dispatch would duplicate
	}
	return task.AttemptCount < MaxTaskExecutions
}

// ResolveModelChain returns every model an agent may run, in order.
//
// An explicit per-agent pin is a deliberate operator choice and outranks the
// table, so it stands alone as a chain of one — the same precedence
// ResolveModel has always had. Chain semantics live in modelreg.ResolveRole;
// nothing here re-creates the routing table that
// M-MODEL-REGISTRY-SINGLE-SOURCE M7 deleted (guarded by
// TestCloudAgents_RegistryMatchesTheDeletedRoutingTable — note the name; the
// comment in model_resolution.go cited a test that never existed).
func ResolveModelChain(agent *AgentConfig) ([]string, error) {
	if agent == nil {
		return nil, nil
	}
	if agent.Model != "" {
		return []string{agent.Model}, nil
	}
	if agent.Role == "" {
		return nil, nil
	}

	if modelreg.GlobalModelsConfig == nil {
		if err := modelreg.InitModelsConfig(); err != nil {
			return nil, fmt.Errorf("agent %q has role %q but the model registry could not be loaded: %w",
				agent.ID, agent.Role, err)
		}
	}

	chain, err := modelreg.GlobalModelsConfig.ResolveRole(agent.Role, modelreg.LaneCloud)
	if err != nil {
		return nil, fmt.Errorf("agent %q: %w (roles the registry knows: %s)",
			agent.ID, err, strings.Join(modelreg.GlobalModelsConfig.ListRoles(), ", "))
	}

	out := make([]string, 0, len(chain))
	for _, entry := range chain {
		out = append(out, entry.ModelName)
	}
	return out, nil
}
