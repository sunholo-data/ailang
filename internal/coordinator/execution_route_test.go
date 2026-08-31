package coordinator

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveExecutionRoute_AllowedMatrixAndClaudeLegacyNormalization(t *testing.T) {
	tests := []struct {
		provider string
		variant  string
		want     string
	}{
		{"claude", "", "default"},
		{"claude", "default", "default"},
		{"claude", "go", "go"},
		{"gemini", "gemini", "gemini"},
		{"gemini", "gemini-go", "gemini-go"},
		{"codex", "codex", "codex"},
		{"codex", "codex-go", "codex-go"},
		{"opencode", "opencode", "opencode"},
		{"pi", "pi", "pi"},
		{"motoko", "motoko", "motoko"},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"/"+tt.variant, func(t *testing.T) {
			agent := &AgentConfig{
				ID:              "agent-a",
				Provider:        tt.provider,
				ExecutorVariant: tt.variant,
				Model:           "model-a",
			}
			route, err := ResolveExecutionRoute(agent.ID, agent)
			if err != nil {
				t.Fatalf("ResolveExecutionRoute: %v", err)
			}
			if route.AgentID() != agent.ID || route.Provider() != tt.provider || route.ExecutorVariant() != tt.want || route.Model() != agent.Model {
				t.Fatalf("route = %q/%q/%q/%q, want %q/%q/%q/%q",
					route.AgentID(), route.Provider(), route.ExecutorVariant(), route.Model(),
					agent.ID, tt.provider, tt.want, agent.Model)
			}
		})
	}
}

func TestResolveExecutionRoute_RejectsEveryOffDiagonalAndUnknownAsPermanent(t *testing.T) {
	providers := []string{"claude", "gemini", "codex", "opencode", "pi", "motoko"}
	variants := []string{"default", "go", "gemini", "gemini-go", "codex", "codex-go", "opencode", "pi", "motoko"}
	allowed := map[string]map[string]bool{
		"claude":   {"default": true, "go": true},
		"gemini":   {"gemini": true, "gemini-go": true},
		"codex":    {"codex": true, "codex-go": true},
		"opencode": {"opencode": true},
		"pi":       {"pi": true},
		"motoko":   {"motoko": true},
	}

	for _, provider := range providers {
		for _, variant := range variants {
			if allowed[provider][variant] {
				continue
			}
			name := provider + "/" + variant
			t.Run(name, func(t *testing.T) {
				_, err := ResolveExecutionRoute("agent-b", &AgentConfig{
					ID:              "agent-b",
					Provider:        provider,
					ExecutorVariant: variant,
					Model:           "model-b",
				})
				assertPermanentRouteError(t, err, "agent-b", provider, variant)
			})
		}
	}

	for _, tc := range []struct{ provider, variant string }{
		{"unknown", "codex"},
		{"codex", "unknown"},
		{"", "default"},
		{"opencode", ""},
		{"eval", "eval"},
		{"claude", "eval-go"},
	} {
		t.Run("unknown/"+tc.provider+"/"+tc.variant, func(t *testing.T) {
			_, err := ResolveExecutionRoute("agent-c", &AgentConfig{
				ID:              "agent-c",
				Provider:        tc.provider,
				ExecutorVariant: tc.variant,
			})
			assertPermanentRouteError(t, err, "agent-c", tc.provider, tc.variant)
		})
	}
}

func TestResolveExecutionRoute_RejectsMissingOrDifferentAgent(t *testing.T) {
	for _, tc := range []struct {
		name  string
		id    string
		agent *AgentConfig
	}{
		{name: "missing", id: "missing", agent: nil},
		{name: "different", id: "expected", agent: &AgentConfig{ID: "actual", Provider: "claude", ExecutorVariant: "default"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ResolveExecutionRoute(tc.id, tc.agent)
			assertPermanentRouteError(t, err, tc.id, "", "")
		})
	}
}

func TestValidateCloudExecutionRoutes_LocalLaneExempt(t *testing.T) {
	cfg := &CoordinatorConfig{Agents: []*AgentConfig{
		{
			ID:              "local-agent",
			Workspace:       "/local/checkout",
			ExecutionLane:   LaneLocal,
			Provider:        "local-provider",
			ExecutorVariant: "local-variant",
		},
	}}
	if err := ValidateCloudExecutionRoutes(cfg); err != nil {
		t.Fatalf("local lane must be exempt from the Cloud Run matrix: %v", err)
	}
}

func TestDispatchTasksCloud_InvalidRouteHasNoReservationOrDispatchEffect(t *testing.T) {
	store := NewMockStore()
	store.tasks["task-bad-route"] = &TaskRecord{
		ID:      "task-bad-route",
		AgentID: "bad-agent",
		Status:  TaskStatusPending,
	}
	registry := NewAgentRegistry()
	if err := registry.Register(&AgentConfig{
		ID:              "bad-agent",
		Inbox:           "bad-agent",
		Workspace:       "org/repo",
		Provider:        "opencode",
		ExecutorVariant: "codex",
	}); err != nil {
		t.Fatal(err)
	}
	dispatcher := &recordingCloudDispatcher{}
	d := &Daemon{
		ctx:             t.Context(),
		logger:          log.New(io.Discard, "", 0),
		taskStore:       store,
		agentRegistry:   registry,
		cloudDispatcher: dispatcher,
	}

	if err := d.dispatchTasksCloud(); err != nil {
		t.Fatalf("dispatchTasksCloud: %v", err)
	}
	if got := store.calls["MarkTaskQueued"]; got != 0 {
		t.Fatalf("route failure must precede reservation/queued state; MarkTaskQueued calls = %d", got)
	}
	if dispatcher.calls != 0 {
		t.Fatalf("route failure must precede external dispatch; calls = %d", dispatcher.calls)
	}
}

func TestDispatchTasksCloud_LocalLaneHasNoCloudEffects(t *testing.T) {
	store := NewMockStore()
	store.tasks["task-local"] = &TaskRecord{
		ID:      "task-local",
		AgentID: "local-agent",
		Status:  TaskStatusPending,
	}
	registry := NewAgentRegistry()
	if err := registry.Register(&AgentConfig{
		ID:              "local-agent",
		Inbox:           "local-agent",
		Workspace:       "/srv/ailang",
		ExecutionLane:   LaneLocal,
		Provider:        "ollama",
		ExecutorVariant: "local-ollama",
	}); err != nil {
		t.Fatal(err)
	}
	publisher := &recordingTaskPublisher{}
	dispatcher := &recordingCloudDispatcher{}
	d := &Daemon{
		ctx:             t.Context(),
		logger:          log.New(io.Discard, "", 0),
		taskStore:       store,
		agentRegistry:   registry,
		taskPublisher:   publisher,
		cloudDispatcher: dispatcher,
	}

	if err := d.dispatchTasksCloud(); err != nil {
		t.Fatalf("dispatchTasksCloud: %v", err)
	}
	if got := store.calls["MarkTaskQueued"]; got != 0 {
		t.Fatalf("local lane must not be queued for cloud; MarkTaskQueued calls = %d", got)
	}
	if publisher.calls != 0 || dispatcher.calls != 0 {
		t.Fatalf("local lane cloud calls = publisher %d, dispatcher %d; want zero", publisher.calls, dispatcher.calls)
	}
}

func TestTaskMaxCostForProvider_UsesResolvedRouteProvider(t *testing.T) {
	cfg := &BudgetsConfig{
		Global: &GlobalBudget{TaskMaxCost: 5},
		Providers: map[string]*ProviderLimit{
			"opencode": {TaskMaxCost: 99},
			"codex":    {TaskMaxCost: 17},
		},
	}
	if got := taskMaxCostForProvider(cfg, "codex"); got != 17 {
		t.Fatalf("codex route selected task max cost %v, want 17 (opencode default must be irrelevant)", got)
	}
}

func TestDispatchTasksCloud_HistoricalDefaultProviderCannotSplitRoute(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte(`budgets:
  global:
    task_max_cost: 5
  providers:
    opencode:
      task_max_cost: 99
    codex:
      task_max_cost: 17
`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_CONFIG", configPath)

	store := NewMockStore()
	store.tasks["task-historical"] = &TaskRecord{
		ID:        "task-historical",
		AgentID:   "planner",
		Workspace: "org/repo",
		Status:    TaskStatusPending,
	}
	registry := NewAgentRegistry()
	if err := registry.Register(&AgentConfig{
		ID:              "planner",
		Inbox:           "planner",
		Workspace:       "org/repo",
		Provider:        "codex",
		ExecutorVariant: "codex",
		Model:           "openai/codex-model",
	}); err != nil {
		t.Fatal(err)
	}
	publisher := &recordingTaskPublisher{}
	dispatcher := &recordingCloudDispatcher{}
	d := &Daemon{
		ctx:             t.Context(),
		logger:          log.New(io.Discard, "", 0),
		taskStore:       store,
		agentRegistry:   registry,
		coordConfig:     &CoordinatorConfig{DefaultProvider: "opencode"},
		taskPublisher:   publisher,
		cloudDispatcher: dispatcher,
	}

	if err := d.dispatchTasksCloud(); err != nil {
		t.Fatalf("dispatchTasksCloud: %v", err)
	}
	if publisher.calls != 1 || publisher.provider != "codex" {
		t.Fatalf("audit publish = %d calls with provider %q, want 1 call with codex", publisher.calls, publisher.provider)
	}
	if dispatcher.calls != 1 {
		t.Fatalf("dispatcher calls = %d, want 1", dispatcher.calls)
	}
	params := dispatcher.params
	if params.Provider != "codex" || params.ExecutorVariant != "codex" || params.Model != "openai/codex-model" {
		t.Fatalf("dispatch route = provider %q, variant %q, model %q", params.Provider, params.ExecutorVariant, params.Model)
	}
	if params.MaxCostUSD != 17 {
		t.Fatalf("MaxCostUSD = %v, want codex-specific 17 (global default provider is opencode)", params.MaxCostUSD)
	}
}

type recordingCloudDispatcher struct {
	calls  int
	params DispatchParams
}

type recordingTaskPublisher struct {
	calls    int
	provider string
}

func (p *recordingTaskPublisher) PublishTask(_ context.Context, _, _, _, provider string) error {
	p.calls++
	p.provider = provider
	return nil
}

func (d *recordingCloudDispatcher) Dispatch(_ context.Context, params DispatchParams) error {
	d.calls++
	d.params = params
	return nil
}

func assertPermanentRouteError(t *testing.T, err error, agentID, provider, variant string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected route error")
	}
	var permanent *PermanentDispatchError
	if !errors.As(err, &permanent) {
		t.Fatalf("error type = %T, want *PermanentDispatchError: %v", err, err)
	}
	for _, want := range []string{agentID, provider, variant} {
		if want != "" && !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	}
}
