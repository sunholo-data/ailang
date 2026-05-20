package managed_agents

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

// stubHTTP is a minimal httpClient stub that returns a canned response body.
type stubHTTP struct {
	resp *http.Response
	err  error
}

func (s *stubHTTP) Do(_ *http.Request) (*http.Response, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.resp, nil
}

func stubToken(_ context.Context) (string, error) {
	return "ya29.test-token", nil
}

// TestRegistration verifies the package's init() registers the executor with
// the global factory so `agent_cli: "managed_agents"` resolves automatically.
func TestRegistration(t *testing.T) {
	factory := executor.GlobalFactory()
	exec, err := factory.GetExecutor("managed_agents")
	if err != nil {
		t.Fatalf("GetExecutor(managed_agents): %v", err)
	}
	if exec.Name() != "managed_agents" {
		t.Errorf("Name()=%q, want %q", exec.Name(), "managed_agents")
	}
}

// TestExecuteWithFixture replays the canonical SSE fixture (a real PONG
// response from ailang-dev captured on 2026-05-20) through the executor and
// asserts the Result fields match.
//
// This is the load-bearing test: if any of the captured event schema changes
// upstream (e.g. event names get renamed, usage payload restructured), this
// test surfaces the drift loudly.
func TestExecuteWithFixture(t *testing.T) {
	fixture, err := os.ReadFile("testdata/sse_pong.txt")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	exec := &Executor{
		agent:    defaultAgent,
		project:  "ailang-dev",
		location: defaultLocation,
		httpClient: &stubHTTP{
			resp: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(fixture)),
				Header:     make(http.Header),
			},
		},
		tokens:         stubToken,
		timeoutSeconds: 30,
	}

	task := &executor.Task{
		ID:         "test-pong",
		Directive:  "reply with exactly: PONG",
		GCPProject: "ailang-dev",
	}

	res, err := exec.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// 1. Success path
	if !res.Success {
		t.Errorf("Success=false, want true; Error=%q", res.Error)
	}
	if res.FinishReason != "stop" {
		t.Errorf("FinishReason=%q, want %q", res.FinishReason, "stop")
	}

	// 2. Output assembled from step.delta events
	if !strings.Contains(res.Output, "PONG") {
		t.Errorf("Output missing PONG: %q", res.Output)
	}

	// 2b. NumTurns mapped from step.start event count.
	// PONG fixture had a single model_output step → NumTurns == 1.
	// (Real benchmark runs produce many steps; agent harness gates
	// agent-mode results on NumTurns > 1 OR ToolCallCount > 0.)
	if res.NumTurns != 1 {
		t.Errorf("NumTurns=%d, want 1 (PONG fixture has one step)", res.NumTurns)
	}

	// 3. Token usage from interaction.completed event
	if res.InputTokens != 6560 {
		t.Errorf("InputTokens=%d, want 6560", res.InputTokens)
	}
	// OutputTokens = candidates (35) + thoughts (748)
	if res.OutputTokens != 35+748 {
		t.Errorf("OutputTokens=%d, want %d", res.OutputTokens, 35+748)
	}

	// 4. Session/interaction ID extracted
	wantID := "ChAxZjhkYWE2YTgwZjA3NmMzEAgaAzJmMioEbWFpbg"
	if res.SessionID != wantID {
		t.Errorf("SessionID=%q, want %q", res.SessionID, wantID)
	}

	// 5. ProviderData contains the multi-turn handles
	pd := res.ProviderData
	if pd == nil {
		t.Fatal("ProviderData=nil")
	}
	if got := pd["managed_agents_interaction_id"]; got != wantID {
		t.Errorf("ProviderData interaction_id=%v, want %v", got, wantID)
	}
	wantEnv := "env_CAEQgICAgIDQxP1sGiBiMDVhYWMyNDc4MWU0ZDZhOWY1ZGZiYzBjMDgxODJiZQ"
	if got := pd["managed_agents_environment_id"]; got != wantEnv {
		t.Errorf("ProviderData environment_id=%v, want %v", got, wantEnv)
	}
	if got := pd["managed_agents_total_thought_tokens"]; got != 748 {
		t.Errorf("ProviderData total_thought_tokens=%v, want 748", got)
	}

	// 6. Cost computed client-side from gemini-3-5-flash rates.
	// Input: 6560 tokens * $1.50/1M = $0.00984
	// Output (incl thoughts): 783 tokens * $9.00/1M = $0.007047
	// Total: $0.016887, but CalculateCost works in cost-per-1K so the math
	// goes input/1000*0.0015 + output/1000*0.009.
	if res.CostUSD <= 0 {
		t.Errorf("CostUSD=%v, want > 0", res.CostUSD)
	}
}

// TestCapabilities sanity-checks the advertised capabilities.
func TestCapabilities(t *testing.T) {
	exec, _ := New(&executor.Config{})
	caps := exec.Capabilities()
	have := map[executor.Capability]bool{}
	for _, c := range caps {
		have[c] = true
	}
	if !have[executor.CapStreaming] {
		t.Error("missing CapStreaming capability")
	}
	if !have[executor.CapSessionResume] {
		t.Error("missing CapSessionResume capability")
	}
}

// TestCostModelPricing pins gemini-3-5-flash Vertex pricing.
func TestCostModelPricing(t *testing.T) {
	exec, _ := New(&executor.Config{})
	cm := exec.CostModel()
	if cm.InputTokenCost != 0.0015 {
		t.Errorf("InputTokenCost=%v, want 0.0015 ($1.50/1M)", cm.InputTokenCost)
	}
	if cm.OutputTokenCost != 0.009 {
		t.Errorf("OutputTokenCost=%v, want 0.009 ($9/1M)", cm.OutputTokenCost)
	}
}

// TestHealthCheckEmptyToken verifies the doctor-style check fails loudly
// when ADC isn't configured.
func TestHealthCheckEmptyToken(t *testing.T) {
	exec, _ := New(&executor.Config{})
	exec.SetTokenSource(func(_ context.Context) (string, error) {
		return "", nil
	})
	if err := exec.HealthCheck(context.Background()); err == nil {
		t.Error("expected error for empty token, got nil")
	}
}

// TestExecuteHTTPError surfaces a non-200 response cleanly with the API's
// error payload embedded in the message.
func TestExecuteHTTPError(t *testing.T) {
	errBody := `{"error":{"message":"Environment configuration is required for Interaction with this Agent.","code":"invalid_request"}}`
	exec := &Executor{
		agent:    defaultAgent,
		project:  "ailang-dev",
		location: defaultLocation,
		httpClient: &stubHTTP{
			resp: &http.Response{
				StatusCode: 400,
				Body:       io.NopCloser(strings.NewReader(errBody)),
				Header:     make(http.Header),
			},
		},
		tokens:         stubToken,
		timeoutSeconds: 30,
	}
	task := &executor.Task{ID: "t-err", Directive: "x", GCPProject: "ailang-dev"}
	res, err := exec.Execute(context.Background(), task)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if res.Success {
		t.Error("Success=true on HTTP 400, want false")
	}
	if !strings.Contains(res.Error, "Environment configuration") {
		t.Errorf("Error message missing API context: %q", res.Error)
	}
}

// TestRequestBodyShape verifies the executor sends the correct body shape
// (background=true, stream=true, store=true, environment.type=remote, input
// as a structured array).
func TestRequestBodyShape(t *testing.T) {
	var captured *http.Request
	exec := &Executor{
		agent:          defaultAgent,
		project:        "ailang-dev",
		location:       defaultLocation,
		httpClient:     capturingHTTP{capture: &captured, resp: emptySSEResponse()},
		tokens:         stubToken,
		timeoutSeconds: 30,
	}
	task := &executor.Task{
		ID:           "t-shape",
		Directive:    "Hello",
		SystemPrompt: "Be brief.",
		GCPProject:   "ailang-dev",
	}
	_, _ = exec.Execute(context.Background(), task)

	if captured == nil {
		t.Fatal("no request captured")
	}
	if got := captured.Header.Get(apiRevisionHeader); got != apiRevision {
		t.Errorf("%s header=%q, want %q", apiRevisionHeader, got, apiRevision)
	}
	if got := captured.Header.Get("Authorization"); !strings.HasPrefix(got, "Bearer ") {
		t.Errorf("Authorization header=%q, want Bearer prefix", got)
	}

	bodyBytes, _ := io.ReadAll(captured.Body)
	var body interactionRequest
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if !body.Stream || !body.Background || !body.Store {
		t.Errorf("expected stream+background+store=true, got %+v", body)
	}
	if body.Agent != defaultAgent {
		t.Errorf("Agent=%q, want %q", body.Agent, defaultAgent)
	}
	// SystemInstruction is passed through verbatim — the executor is
	// policy-free per CapRemoteSandbox design (any bridging is the caller's
	// responsibility, e.g. eval_harness/managed_agents_bridge.go).
	if body.SystemInstruction != "Be brief." {
		t.Errorf("SystemInstruction=%q, want %q (executor should not modify caller's prompt)", body.SystemInstruction, "Be brief.")
	}
	if len(body.Input) != 1 || len(body.Input[0].Content) != 1 {
		t.Fatalf("Input shape unexpected: %+v", body.Input)
	}
	if body.Input[0].Content[0].Text != "Hello" {
		t.Errorf("Input text=%q, want %q", body.Input[0].Content[0].Text, "Hello")
	}
}

// capturingHTTP saves the outgoing request so its shape can be asserted on.
type capturingHTTP struct {
	capture **http.Request
	resp    *http.Response
}

func (c capturingHTTP) Do(req *http.Request) (*http.Response, error) {
	// http.NewRequestWithContext sets req.Body as io.NopCloser around a
	// *bytes.Reader; ReadAll consumes it once, so we need to re-buffer
	// before saving for assertions.
	if req.Body != nil {
		buf, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(buf))
	}
	*c.capture = req
	return c.resp, nil
}

func emptySSEResponse() *http.Response {
	body := "event: done\ndata: [DONE]\n\n"
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

// TestLive_ManagedAgentsEndpoint is a gated end-to-end test against the real
// Vertex Managed Agents API. Skipped unless AILANG_MANAGED_AGENTS_LIVE=1.
//
// Requires:
//   - gcloud auth application-default login
//   - GOOGLE_CLOUD_PROJECT set to a project with the Managed Agents API
//     provisioned (a first call from a fresh project will hit HTTP 400
//     "Provisioning in progress" for a few minutes).
func TestLive_ManagedAgentsEndpoint(t *testing.T) {
	if os.Getenv("AILANG_MANAGED_AGENTS_LIVE") != "1" {
		t.Skip("set AILANG_MANAGED_AGENTS_LIVE=1 to run this test")
	}
	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if project == "" {
		project = "ailang-dev"
	}
	exec, err := New(&executor.Config{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	task := &executor.Task{
		ID:         "live-test",
		Directive:  "Reply with exactly: PONG",
		GCPProject: project,
	}
	res, err := exec.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(res.Output, "PONG") {
		t.Errorf("live response missing PONG: %q", res.Output)
	}
	t.Logf("live PONG took %dms, %d in / %d out tokens, $%.6f",
		res.DurationMS, res.InputTokens, res.OutputTokens, res.CostUSD)
}
