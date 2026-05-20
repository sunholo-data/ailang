package managed_agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var managedAgentsTracer = telemetry.Tracer("executor.managed_agents")

// Executor implements executor.Executor for the Vertex AI Managed Agents API.
type Executor struct {
	agent          string      // Agent name (default: "antigravity-preview-05-2026")
	project        string      // Default GCP project (overridden per-task via Task.GCPProject)
	location       string      // Default GCP location (overridden per-task)
	timeoutSeconds int         // Default hard ceiling for one interaction
	httpClient     httpClient  // Swappable for tests
	tokens         tokenSource // Swappable for tests
}

// New constructs the executor from the shared executor.Config.
//
// Config plumbing through the existing executor.Config struct is intentionally
// minimal — most callers pass project/location per-Task via Task.GCPProject /
// Task.GCPLocation (set by the eval harness from models.yml). The executor
// just keeps defaults so a bare HealthCheck call still has somewhere to go.
func New(cfg *executor.Config) (*Executor, error) {
	timeout := cfg.TimeoutSeconds
	if timeout == 0 {
		timeout = 300
	}
	return &Executor{
		agent:          defaultAgent,
		project:        "", // Filled per-task; left empty by default
		location:       defaultLocation,
		timeoutSeconds: timeout,
		httpClient:     defaultHTTPClient(),
		tokens:         defaultTokenSource,
	}, nil
}

// Name returns the executor identifier.
func (e *Executor) Name() string { return "managed_agents" }

// Capabilities advertises supported features.
//
// CapRemoteSandbox is the load-bearing flag: the Managed Agents API runs the
// agent in a Google-hosted Linux sandbox, so the agent's file edits do NOT
// touch the caller's filesystem. Callers that need to receive file artifacts
// (e.g. the eval harness reading solution.ail) must arrange their own bridge
// — typically by instructing the agent to dump artifacts in its text
// response and parsing them from Result.Output. See the eval harness's
// managed_agents_bridge.go for that bridge.
func (e *Executor) Capabilities() []executor.Capability {
	return []executor.Capability{
		executor.CapStreaming,
		executor.CapSessionResume, // Multi-turn via interaction.id + environment_id
		executor.CapRemoteSandbox, // Sandbox runs server-side; no shared filesystem with caller
	}
}

// CostModel returns gemini-3-5-flash Vertex pricing (the default agent's
// underlying model as of 2026-05-20). Thought tokens are billed at output
// rate — added together client-side because the API reports them separately
// in usage.
func (e *Executor) CostModel() *executor.CostModel {
	return &executor.CostModel{
		ProviderName:    "google-managed-agents",
		InputTokenCost:  0.0015, // $1.50 per 1M tokens
		OutputTokenCost: 0.009,  // $9.00 per 1M (includes reasoning/thought tokens)
	}
}

// HealthCheck verifies ADC is configured. We don't make a real interaction
// call (it would cost real money on a service that provisions a sandbox);
// validating the token source is sufficient signal for the harness doctor.
func (e *Executor) HealthCheck(ctx context.Context) error {
	if e.tokens == nil {
		return errors.New("managed_agents: token source not configured")
	}
	tok, err := e.tokens(ctx)
	if err != nil {
		return err
	}
	if tok == "" {
		return errors.New("managed_agents: empty ADC token; run `gcloud auth application-default login`")
	}
	return nil
}

// Close releases any held resources. The Managed Agents executor holds no
// long-lived state (no subprocess, no persistent connection), so this is a
// no-op.
func (e *Executor) Close() error { return nil }

// Execute runs a task without streaming events to a handler.
func (e *Executor) Execute(ctx context.Context, task *executor.Task) (*executor.Result, error) {
	return e.ExecuteStreaming(ctx, task, &executor.NoOpEventHandler{})
}

// ExecuteStreaming runs a task and forwards step.delta text to the handler as
// it arrives. The handler receives one OnTurnStart + a single accumulating
// OnText (per step.delta) + one OnTurnEnd. There's no tool-call surfacing in
// this first cut — see managedAgentsEvents in ProviderData for any unknown
// event types the API may grow.
func (e *Executor) ExecuteStreaming(
	ctx context.Context,
	task *executor.Task,
	handler executor.EventHandler,
) (*executor.Result, error) {
	ctx, span := telemetry.StartSpan(ctx, managedAgentsTracer, "managed_agents.execute",
		trace.WithAttributes(
			attribute.String("executor.name", "managed_agents"),
			attribute.String("task.id", task.ID),
			attribute.String("task.parent_task_id", task.ParentTaskID),
			attribute.String("task.directive", telemetry.Truncate(task.Directive, 500)),
		),
	)
	defer span.End()

	if ctxHandler, ok := handler.(executor.ContextAwareHandler); ok {
		ctxHandler.SetContext(ctx)
	}

	// Resolve project + location: Task fields win over executor defaults.
	project := task.GCPProject
	if project == "" {
		project = e.project
	}
	location := task.GCPLocation
	if location == "" {
		location = e.location
	}

	// Resolve agent name. Task.Model carries the per-model override from
	// models.yml::agent_model_name (e.g. "antigravity-preview-05-2026" or a
	// future agent identifier).
	agent := task.Model
	if agent == "" {
		agent = e.agent
	}

	// Build the interaction body. We do a fresh sandbox every call here — see
	// the design doc M2.5 for the multi-turn / sandbox-reuse follow-up.
	//
	// Note on cross-environment file bridging: the agent runs in a Google-
	// hosted sandbox (CapRemoteSandbox), so any file edits the agent makes
	// don't touch the caller's filesystem. This is intentional for backend
	// callers that just want chat/reasoning responses. Callers that NEED
	// file artifacts (e.g. the eval harness reading solution.ail) are
	// expected to (a) instruct the agent to dump artifacts in its text
	// response via task.SystemPrompt, and (b) parse them from Result.Output
	// after the call returns. The executor itself stays policy-free.
	envRaw := json.RawMessage(`{"type":"remote"}`)
	body := &interactionRequest{
		Stream:      true,
		Background:  true,
		Store:       true,
		Agent:       agent,
		Environment: envRaw,
		Input: []inputBlock{{
			Type:    "user_input",
			Content: []contentBlock{{Type: "text", Text: task.Directive}},
		}},
		SystemInstruction: task.SystemPrompt,
	}

	// Hard ceiling: per-task Timeout wins, then executor default.
	timeout := task.Timeout
	if timeout == 0 {
		timeout = time.Duration(e.timeoutSeconds) * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	bodyReader, err := sendInteraction(reqCtx, e.httpClient, e.tokens, project, location, body)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		handler.OnError(err)
		return &executor.Result{
			Success:      false,
			Error:        err.Error(),
			DurationMS:   int(time.Since(start).Milliseconds()),
			FinishReason: "error",
		}, err
	}
	defer bodyReader.Close()

	handler.OnTurnStart(1)

	state := &streamState{}
	parseErr := parseSSE(bodyReader, func(ev sseEvent) error {
		if foldErr := foldEvent(state, ev); foldErr != nil {
			return foldErr
		}
		// Surface text deltas to the handler as they arrive.
		if ev.Name == "step.delta" && len(ev.RawData) > 0 {
			var p stepDeltaPayload
			if err := json.Unmarshal(ev.RawData, &p); err == nil {
				if p.Delta.Type == "text" || p.Delta.Type == "" {
					handler.OnText(p.Delta.Text)
				}
			}
		}
		return nil
	})

	handler.OnTurnEnd(1)
	duration := time.Since(start)

	// Compose the Result.
	//
	// NumTurns maps to the number of agent steps the server-side harness ran.
	// Each step.start event is one agentic iteration (model_output, tool_call,
	// tool_result, etc.). A trivial "say PONG" probe produces exactly 1 step;
	// a real benchmark with file edits + code execution produces many. The
	// eval harness rejects results with NumTurns <= 1 AND ToolCallCount == 0
	// as "0-shot generation, not agent mode" — so accurate step counting is
	// load-bearing for ever passing agent-mode gates.
	turns := state.StepCount
	if turns < 1 {
		turns = 1
	}
	res := &executor.Result{
		Output:                   state.Text.String(),
		DurationMS:               int(duration.Milliseconds()),
		NumTurns:                 turns,
		SessionID:                state.InteractionID,
		InputTokens:              state.Usage.TotalInputTokens,
		OutputTokens:             state.Usage.TotalOutputTokens + state.Usage.TotalThoughtTokens,
		CacheReadInputTokens:     0, // Not reported by Managed Agents API
		CacheCreationInputTokens: 0,
	}

	// Cost from the API-reported usage. We bill thought tokens at the output
	// rate because Vertex's gemini-3-5-flash pricing model doesn't separate
	// them.
	cm := e.CostModel()
	res.CostUSD = cm.CalculateCost(executor.TokenUsage{
		InputTokens:  res.InputTokens,
		OutputTokens: res.OutputTokens,
	})

	// Stash multi-turn handles + unknown events into ProviderData for the
	// eval harness / future multi-turn integration. The eval harness uses
	// the CapRemoteSandbox capability + agent's text Output to bridge files
	// across to the local workspace (see eval_harness/managed_agents_bridge.go).
	pd := map[string]any{
		"managed_agents_interaction_id":       state.InteractionID,
		"managed_agents_environment_id":       state.EnvironmentID,
		"managed_agents_step_count":           state.StepCount,
		"managed_agents_status":               state.Status,
		"managed_agents_total_thought_tokens": state.Usage.TotalThoughtTokens,
	}
	if len(state.UnknownEvents) > 0 {
		pd["managed_agents_unknown_events"] = state.UnknownEvents
	}
	res.ProviderData = pd

	// Determine Success + FinishReason.
	switch state.Status {
	case "completed":
		res.Success = true
		res.FinishReason = "stop"
	case "failed", "interaction.failed":
		res.Success = false
		res.FinishReason = "error"
		res.Error = "managed_agents: interaction status=failed"
	case "cancelled", "interaction.cancelled":
		res.Success = false
		res.FinishReason = "timeout"
		res.Error = "managed_agents: interaction cancelled"
	default:
		// If we got a parse error AND no terminal status, surface the parser
		// error; otherwise the stream just closed mid-event.
		if parseErr != nil {
			res.Success = false
			res.FinishReason = "error"
			res.Error = parseErr.Error()
		} else if state.Status == "" {
			res.Success = false
			res.FinishReason = "error"
			res.Error = "managed_agents: stream closed without interaction.completed"
		} else {
			res.Success = false
			res.FinishReason = "error"
			res.Error = fmt.Sprintf("managed_agents: unexpected status=%q", state.Status)
		}
	}

	if parseErr != nil && res.Success {
		// Successful interaction but parser warning — keep success, log the
		// parser error to ProviderData for postmortem.
		pd["managed_agents_parse_warning"] = parseErr.Error()
	}

	if !res.Success {
		span.SetStatus(codes.Error, res.Error)
		handler.OnError(errors.New(res.Error))
	}

	return res, nil
}

// SetHTTPClient overrides the HTTP client. For tests.
func (e *Executor) SetHTTPClient(c httpClient) { e.httpClient = c }

// SetTokenSource overrides the token source. For tests.
func (e *Executor) SetTokenSource(t tokenSource) { e.tokens = t }

// Register registers the managed_agents executor with the global factory.
func Register() {
	executor.GlobalFactory().Register("managed_agents", func(cfg *executor.Config) (executor.Executor, error) {
		return New(cfg)
	})
}

func init() {
	Register()
}

// Ensure the package depends on net/http for the test build path even when
// tests don't reference it directly. (Removing this would let the import get
// elided by goimports.)
var _ = http.StatusOK
