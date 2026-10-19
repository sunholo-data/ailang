package coordinator

import (
	"context"
	"strings"
	"testing"
	"time"
)

// laneAgent is a pi agent on the ailang_only lane, the shape the rollout
// shadows first (pkg-sunholo-ailang-parse: pi primary, motoko shadow).
func laneAgent(shadow *ShadowConfig) *AgentConfig {
	return &AgentConfig{
		ID: "pkg-sunholo-ailang-parse", Inbox: "pkg:sunholo/ailang_parse",
		ExecutorVariant: "pi", ToolPolicy: "ailang_only", Shadow: shadow,
	}
}

func motokoShadow() *ShadowConfig {
	return &ShadowConfig{Enabled: true, Provider: "motoko", ExecutorVariant: "motoko", SampleRate: 1}
}

func TestValidateShadowConfig(t *testing.T) {
	cases := []struct {
		name  string
		agent *AgentConfig
		want  string // substring of the single expected error; "" = valid
	}{
		{"no shadow block", laneAgent(nil), ""},
		{"pi primary, motoko shadow", laneAgent(motokoShadow()), ""},
		{"motoko primary, pi shadow", &AgentConfig{
			ID: "ailang-only-executor", ExecutorVariant: "motoko", ToolPolicy: "ailang_only",
			Shadow: &ShadowConfig{Enabled: true, Provider: "pi", ExecutorVariant: "pi", SampleRate: 0.5},
		}, ""},
		{"pkg inbox defaults to ailang_only", &AgentConfig{
			ID: "pkg-x", Inbox: "pkg:sunholo/x", ExecutorVariant: "pi", Shadow: motokoShadow(),
		}, ""},
		{"wrong lane", &AgentConfig{
			ID: "sprint-executor", Inbox: "sprint-executor", ExecutorVariant: "pi", ToolPolicy: "full",
			Shadow: motokoShadow(),
		}, "only ailang_only agents"},
		{"same provider as primary", laneAgent(&ShadowConfig{
			Enabled: true, Provider: "pi", ExecutorVariant: "pi-go", SampleRate: 1,
		}), "is the primary's own"},
		{"variant cannot run provider", laneAgent(&ShadowConfig{
			Enabled: true, Provider: "motoko", ExecutorVariant: "codex", SampleRate: 1,
		}), `runs "codex" and cannot run "motoko"`},
		{"unknown variant", laneAgent(&ShadowConfig{
			Enabled: true, Provider: "motoko", ExecutorVariant: "motoko-next", SampleRate: 1,
		}), "unknown executor_variant"},
		{"retired variant", laneAgent(&ShadowConfig{
			Enabled: true, Provider: "gemini", ExecutorVariant: "gemini", SampleRate: 1,
		}), "is retired"},
		{"missing provider", laneAgent(&ShadowConfig{Enabled: true, ExecutorVariant: "motoko"}),
			"both required"},
		{"sample_rate above 1", laneAgent(&ShadowConfig{
			Enabled: true, Provider: "motoko", ExecutorVariant: "motoko", SampleRate: 1.5,
		}), "outside [0, 1]"},
		{"sample_rate below 0", laneAgent(&ShadowConfig{
			Enabled: true, Provider: "motoko", ExecutorVariant: "motoko", SampleRate: -0.1,
		}), "outside [0, 1]"},
		{"negative cost cap", laneAgent(&ShadowConfig{
			Enabled: true, Provider: "motoko", ExecutorVariant: "motoko", SampleRate: 1, MaxCostUSD: -1,
		}), "max_cost_usd"},
		// A disabled block is validated too: it must not be switchable on broken.
		{"disabled but broken", laneAgent(&ShadowConfig{
			Provider: "pi", ExecutorVariant: "pi", SampleRate: 1,
		}), "is the primary's own"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := ValidateAgentProviders([]*AgentConfig{tc.agent})
			if tc.want == "" {
				if len(errs) != 0 {
					t.Fatalf("want valid, got %v", errs)
				}
				return
			}
			if len(errs) != 1 {
				t.Fatalf("want exactly one error containing %q, got %v", tc.want, errs)
			}
			if msg := errs[0].Error(); !strings.Contains(msg, tc.want) || !strings.Contains(msg, "shadow:") {
				t.Fatalf("error %q does not contain %q", msg, tc.want)
			}
		})
	}
}

// TestSQLiteStoreShadowOfRoundTrip pins ShadowOf through both SQLite read
// paths: GetTask (scanTask) and ListTasks (scanTaskFromRows). A shadow child
// read back with an empty ShadowOf would finalize like a primary: approval
// card, handoff, the lot.
func TestSQLiteStoreShadowOfRoundTrip(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	for _, rec := range []*TaskRecord{
		{ID: "task-p1", Title: "primary", Type: TaskTypeBugFix, Status: TaskStatusPending, CreatedAt: time.Now()},
		{ID: "task-p1-shadow", ShadowOf: "task-p1", Title: "shadow", Type: TaskTypeBugFix,
			Status: TaskStatusPending, CreatedAt: time.Now()},
	} {
		if err := store.CreateTask(ctx, rec); err != nil {
			t.Fatalf("create %s: %v", rec.ID, err)
		}
	}

	got, err := store.GetTask(ctx, "task-p1-shadow")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ShadowOf != "task-p1" {
		t.Errorf("GetTask ShadowOf = %q, want task-p1", got.ShadowOf)
	}

	list, err := store.ListTasks(ctx, &TaskFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	seen := map[string]string{}
	for _, r := range list {
		seen[r.ID] = r.ShadowOf
	}
	if seen["task-p1-shadow"] != "task-p1" {
		t.Errorf("ListTasks ShadowOf = %q, want task-p1", seen["task-p1-shadow"])
	}
	if v, ok := seen["task-p1"]; !ok || v != "" {
		t.Errorf("primary ShadowOf = %q (present=%v), want empty", v, ok)
	}
}
