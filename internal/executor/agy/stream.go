package agy

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

// streamState accumulates one run's stream. observe runs on the reader
// goroutine; verdict/fill run after it has finished (done is received only
// after the reader loop ends), but the mutex keeps a timeout path that reads
// mid-stream honest.
type streamState struct {
	mu sync.Mutex

	initModel      string
	permissionMode string
	conversationID string

	transcript strings.Builder
	stderr     strings.Builder

	toolSteps     map[int]bool
	toolCalls     map[string]int
	toolCallCount int
	modelSteps    int

	// Per-step deltas (used for live budget tallies only).
	stepIn, stepOut, stepThink, stepCache int

	result        *agyResult
	unknownEvents map[string]int
	unparsed      int
	firstEditAt   time.Time
	firstTextAt   time.Time
	costKilled    bool
}

// observe folds one stream line in. It returns true when a budget breach
// means the caller must kill the process.
func (s *streamState) observe(line []byte, task *executor.Task, handler executor.EventHandler) bool {
	ev, err := parseAgyEvent(line)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.unparsed++
		return false
	}
	switch ev.Event {
	case "init":
		if ev.Init != nil {
			s.initModel = ev.Init.Model
			s.permissionMode = ev.Init.PermissionMode
		}
	case "step_update":
		st := ev.StepUpdate
		if st == nil {
			return false
		}
		if s.conversationID == "" {
			s.conversationID = st.ConversationID
		}
		switch st.StepType {
		case "agent_response":
			if st.TextDelta != "" {
				if s.firstTextAt.IsZero() {
					s.firstTextAt = time.Now()
				}
				s.transcript.WriteString(st.TextDelta)
				handler.OnText(st.TextDelta)
			}
			if st.State == "DONE" {
				s.modelSteps++
				handler.OnTurnEnd(s.modelSteps)
				if u := st.Usage; u != nil {
					s.stepIn += u.InputTokens
					s.stepOut += u.OutputTokens
					s.stepThink += u.ThinkingTokens
					s.stepCache += u.CacheReadTokens
					if task.Budget != nil && (u.InputTokens > 0 || u.OutputTokens > 0) {
						if _, exceeded := task.Budget.Add(u.InputTokens, u.OutputTokens+u.ThinkingTokens); exceeded {
							s.costKilled = true
							return true
						}
					}
				}
			}
		case "tool":
			// A tool step is reported ACTIVE then DONE under one step_index;
			// count it once, on first sight.
			if !s.toolSteps[st.StepIndex] {
				s.toolSteps[st.StepIndex] = true
				s.toolCallCount++
				name := st.ToolName
				if name == "" {
					name = "unknown"
				}
				s.toolCalls[name]++
				args := string(st.ToolInfo)
				if args == "" {
					args = "{}"
				}
				handler.OnToolUse(name, args)
				if s.firstEditAt.IsZero() && isEditTool(name) {
					s.firstEditAt = time.Now()
				}
			}
			if st.State == "DONE" {
				handler.OnToolResult(st.ToolName, "")
			}
		case "user_input":
			// The directive itself; no work.
		default:
			s.unknownEvents["step:"+st.StepType]++
		}
	case "result":
		s.result = ev.Result
	default:
		s.unknownEvents[ev.Event]++
	}
	return false
}

func isEditTool(name string) bool {
	switch name {
	case "write_file", "write_to_file", "replace_file_content", "multi_replace_file_content", "sed_file", "notebook_edit":
		return true
	}
	return false
}

func (s *streamState) firstAttemptMs(start time.Time) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.firstEditAt
	if t.IsZero() {
		t = s.firstTextAt
	}
	if t.IsZero() {
		return -1
	}
	return t.Sub(start).Milliseconds()
}

// verdict decides success from the process exit and the result event. The
// exit code alone is never trusted: agy exits 0 on a soft denial.
func (s *streamState) verdict(waitErr error, route string) *executor.Result {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := &executor.Result{}
	stderrTail := strings.TrimSpace(s.stderr.String())
	exitCode := 0
	var ee *exec.ExitError
	if errors.As(waitErr, &ee) {
		exitCode = ee.ExitCode()
	} else if waitErr != nil {
		exitCode = -1
	}

	switch {
	case s.costKilled:
		r.FinishReason = executor.FinishCostExhausted
		r.Error = "agy: cost budget exceeded"
	case s.result == nil:
		r.FinishReason = executor.FinishWireDrift
		r.Error = fmt.Sprintf("agy exited (code %d) without a result event — pinned agy %s; stderr: %s",
			exitCode, ExpectedVersion, tailOf(stderrTail, 600))
	case s.result.Status != "SUCCESS":
		r.FinishReason = executor.FinishError
		msg := s.result.Error
		if msg == "" {
			msg = tailOf(stderrTail, 600)
		}
		if isAuthFailure(msg) {
			msg = signinHint(route) + " — agy said: " + msg
		}
		r.Error = "agy status " + s.result.Status + ": " + msg
	case len(s.result.DeniedActions) > 0:
		// Soft denial: exit 0, status SUCCESS, but the agent was refused a tool
		// it asked for. Banked as a failure so an under-permitted run is never
		// read as a model that finished its work.
		r.FinishReason = executor.FinishError
		r.Error = fmt.Sprintf("agy soft-denied %d tool permission(s) [%s]; the run did not get to do what it asked (permission posture: internal/executor/agy/route.go)",
			len(s.result.DeniedActions), deniedSummary(s.result.DeniedActions))
	case exitCode == 3:
		r.FinishReason = executor.FinishTimeout
		r.Error = "agy --print-timeout cut the turn short (exit 3)"
	case exitCode != 0:
		r.FinishReason = executor.FinishError
		r.Error = fmt.Sprintf("agy exited with code %d despite status SUCCESS; stderr: %s", exitCode, tailOf(stderrTail, 600))
	case s.result.FinishedReason != "":
		r.FinishReason = executor.FinishError
		r.Error = "agy finished_reason=" + s.result.FinishedReason
	default:
		r.Success = true
		r.FinishReason = executor.FinishStop
	}
	return r
}

// fill copies the accounting onto a Result (every result shape, including
// timeouts, so a killed run still banks what it spent).
func (s *streamState) fill(r *executor.Result, task *executor.Task, model, route string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	in, out, think, cache := s.stepIn, s.stepOut, s.stepThink, s.stepCache
	if s.result != nil && s.result.Usage != nil {
		u := s.result.Usage
		in, out, think, cache = u.InputTokens, u.OutputTokens, u.ThinkingTokens, u.CacheReadTokens
	}
	r.InputTokens = in
	r.OutputTokens = out
	r.ReasonTokens = think
	r.CacheReadInputTokens = cache
	r.NumTurns = s.modelSteps
	r.ToolCallCount = s.toolCallCount
	r.ToolCalls = s.toolCalls
	r.SessionID = s.conversationID
	output := s.transcript.String()
	if s.result != nil && s.result.Response != "" {
		output = s.result.Response
	}
	r.Output = output
	r.Transcript = output

	lane := executor.AuthLaneBilled
	if route == routeSubscription {
		lane = executor.AuthLaneSubscription
	}
	var fallback *executor.CostModel
	if task == nil || task.Pricing == nil {
		fallback = executor.HarnessCostModel("agy", model)
	}
	r.CostUSD, r.CostProvenance = executor.PriceTokens(task, fallback, lane, executor.TokenUsage{
		InputTokens:          in,
		OutputTokens:         out + think,
		CacheReadInputTokens: cache,
	})

	pd := map[string]any{
		"agy_route":           route,
		"agy_init_model":      s.initModel,
		"agy_permission_mode": s.permissionMode,
	}
	if s.result != nil {
		pd["agy_status"] = s.result.Status
		if len(s.result.DeniedActions) > 0 {
			pd["agy_denied_actions"] = deniedSummary(s.result.DeniedActions)
		}
		if s.result.FinishedReason != "" {
			pd["agy_finished_reason"] = s.result.FinishedReason
		}
	}
	if len(s.unknownEvents) > 0 {
		pd["agy_unknown_events"] = s.unknownEvents
	}
	if s.unparsed > 0 {
		pd["agy_unparsed_lines"] = s.unparsed
	}
	r.ProviderData = pd
}

func tailOf(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}
