// Live experiment: does ENVIRONMENT REUSE actually cut review-run cost?
//
// Mark 2026-07-18 ("lets see if environment reuse helps at all — I think
// surely it must be just like running it locally?"): reuse persists the
// sandbox FILESYSTEM, not the model conversation — a new interaction on a
// reused env starts with a fresh context but the repo already in place, so
// the expected saving is every clone/setup step and its transcript replay.
// This test measures that delta with two paid interactions:
//
//	A: fresh env + egress allowlist → shallow-clone the repo → rev-parse HEAD
//	B: REUSED env (raw "env_<id>") → rev-parse HEAD + ls (no clone, no egress)
//
// Guarded (provisions real sandboxes, costs real money — ~$1 for A, cents
// for B if the hypothesis holds): AILANG_LIVE_MA_REUSE=1 + ADC.
package managed_agents

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func runReuseInteraction(ctx context.Context, t *testing.T, project, location, envJSON, directive string) *streamState {
	t.Helper()
	body := &interactionRequest{
		Stream:      true,
		Background:  true,
		Store:       true,
		Agent:       defaultAgent,
		Environment: json.RawMessage(envJSON),
		Input: []inputBlock{{
			Type:    "user_input",
			Content: []contentBlock{{Type: "text", Text: directive}},
		}},
	}
	reqCtx, cancel := context.WithTimeout(ctx, 6*time.Minute)
	defer cancel()

	reader, err := sendInteraction(reqCtx, defaultHTTPClient(), defaultTokenSource, project, location, body)
	if err != nil {
		t.Fatalf("sendInteraction: %v", err)
	}
	defer reader.Close()

	state := &streamState{}
	if err := parseSSE(reader, func(ev sseEvent) error { return foldEvent(state, ev) }); err != nil {
		t.Fatalf("parseSSE: %v", err)
	}
	return state
}

func costUSD(u Usage) float64 {
	// gemini-3-5-flash Vertex rates, thought tokens at output rate — must
	// match Executor.CostModel() ($1.50/M in, $9.00/M out).
	in := float64(u.TotalInputTokens) * 0.0015 / 1000.0
	out := float64(u.TotalOutputTokens+u.TotalThoughtTokens) * 0.009 / 1000.0
	return in + out
}

func TestLiveEnvironmentReuseEconomics(t *testing.T) {
	if os.Getenv("AILANG_LIVE_MA_REUSE") != "1" {
		t.Skip("live env-reuse economics test disabled; set AILANG_LIVE_MA_REUSE=1 (needs ADC, spends ~$1)")
	}
	project := os.Getenv("AILANG_LIVE_MA_PROJECT")
	if project == "" {
		project = "ailang-dev"
	}
	location := os.Getenv("AILANG_LIVE_MA_LOCATION")
	if location == "" {
		location = "global"
	}
	repo := os.Getenv("AILANG_LIVE_MA_REPO")
	if repo == "" {
		repo = "https://github.com/sunholo-data/ailang.git"
	}
	ctx := context.Background()

	// --- Interaction A: fresh env, egress on, clone + observe ---------------
	envA := `{"type":"remote","network":{"allowlist":[{"domain":"*"}]}}`
	directiveA := "You are in a Linux sandbox. Run exactly these commands, paste raw output, do not summarize, do not explore further:\n" +
		"git clone --depth 1 " + repo + " /workspace/ailang 2>&1 | tail -2\n" +
		"git -C /workspace/ailang rev-parse HEAD\n" +
		"Then print the single line CLONE_OK and stop."
	stateA := runReuseInteraction(ctx, t, project, location, envA, directiveA)

	if stateA.Status != "completed" || !strings.Contains(stateA.Text.String(), "CLONE_OK") {
		t.Fatalf("interaction A did not complete cleanly: status=%q text=%q", stateA.Status, stateA.Text.String())
	}
	if stateA.EnvironmentID == "" {
		t.Fatalf("interaction A returned no environment_id — cannot test reuse")
	}
	shaLine := ""
	for l := range strings.SplitSeq(stateA.Text.String(), "\n") {
		l = strings.TrimSpace(l)
		if len(l) == 40 && !strings.ContainsAny(l, " \t") {
			shaLine = l
			break
		}
	}

	// --- Interaction B: REUSED env, no egress needed, tiny observation ------
	envB := fmt.Sprintf("%q", stateA.EnvironmentID) // raw JSON string "env_…"
	directiveB := "You are in a Linux sandbox with files already present. Run exactly these commands, paste raw output, do not summarize:\n" +
		"git -C /workspace/ailang rev-parse HEAD\n" +
		"ls /workspace/ailang | head -3\n" +
		"Then print the single line REUSE_OK and stop."
	stateB := runReuseInteraction(ctx, t, project, location, envB, directiveB)

	if stateB.Status != "completed" || !strings.Contains(stateB.Text.String(), "REUSE_OK") {
		t.Fatalf("interaction B did not complete cleanly: status=%q text=%q", stateB.Status, stateB.Text.String())
	}
	// The filesystem must have PERSISTED: same repo, same HEAD SHA.
	if shaLine != "" && !strings.Contains(stateB.Text.String(), shaLine) {
		t.Fatalf("reused env did NOT retain the clone: SHA %s absent from B's output:\n%s", shaLine, stateB.Text.String())
	}

	costA, costB := costUSD(stateA.Usage), costUSD(stateB.Usage)
	t.Logf("REUSE ECONOMICS RESULT")
	t.Logf("A (fresh+clone): steps=%d in=%d out=%d thought=%d cost=$%.4f",
		stateA.StepCount, stateA.Usage.TotalInputTokens, stateA.Usage.TotalOutputTokens, stateA.Usage.TotalThoughtTokens, costA)
	t.Logf("B (reused env):  steps=%d in=%d out=%d thought=%d cost=$%.4f",
		stateB.StepCount, stateB.Usage.TotalInputTokens, stateB.Usage.TotalOutputTokens, stateB.Usage.TotalThoughtTokens, costB)
	if costA > 0 {
		t.Logf("B/A cost ratio: %.1f%% (saving %.1f%%)", 100*costB/costA, 100*(1-costB/costA))
	}
}
