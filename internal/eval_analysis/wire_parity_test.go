package eval_analysis

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// wireSpan builds the attributes of one Broadcast LLM Generation span: a task
// prompt naming the arm's workdir, and the rawRequest that went on the wire.
func wireSpan(t *testing.T, model string, req map[string]any, host string) map[string]any {
	t.Helper()
	in, _ := json.Marshal(map[string]any{"messages": []map[string]any{
		{"role": "system", "content": "harness prompt"},
		{"role": "user", "content": "work in /tmp/ailang_eval/quine_" + model + "_20261016_101435_48995/x"},
	}})
	req["messages"] = []any{"elided"}
	out, _ := json.Marshal(map[string]any{"rawRequest": req})
	return map[string]any{
		"span.input":  string(in),
		"span.output": string(out),
		"trace.metadata.openrouter.provider_name": host,
		"gen_ai.response.finish_reason":           "tool_calls",
	}
}

func armRows(model string, at time.Time) []*BenchmarkResult {
	return []*BenchmarkResult{{RunMetrics: eval_harness.RunMetrics{ID: "quine", Model: model, Timestamp: at, DurationMs: 60_000}}}
}

// fixedFetch serves the spans on the first chunk of each arm's window, as a
// real window would: the check pages a window in chunks and must not
// double-count. A chunk that does not continue the previous one starts a new
// window.
func fixedFetch(spans []map[string]any) SpanFetcher {
	var lastEnd time.Time
	return func(_ context.Context, _ string, s, e time.Time, _ int) ([]map[string]any, error) {
		first := !s.Equal(lastEnd)
		lastEnd = e
		if first {
			return spans, nil
		}
		return nil, nil
	}
}

func allOpenRouter(string) (string, bool) { return "z-ai/glm-5.3-flash", true }

func TestWireParity_DetectsTheThreeGLMFlashConfounds(t *testing.T) {
	at := time.Date(2026, 10, 16, 16, 0, 0, 0, time.UTC)
	routing := map[string]any{"only": []any{"coreweave"}, "sort": "throughput"}
	spans := []map[string]any{
		wireSpan(t, "motoko-or-glm", map[string]any{"model": "z-ai/glm-5.3-flash", "max_tokens": 32000, "provider": routing, "stream": true}, "CoreWeave"),
		wireSpan(t, "pi-or-glm", map[string]any{"model": "z-ai/glm-5.3-flash", "max_completion_tokens": 943717, "reasoning": map[string]any{"effort": "high"}, "store": false}, "Together"),
	}
	// Called once per 4-minute chunk; return the spans only on the first call.
	calls := 0
	fetch := func(context.Context, string, time.Time, time.Time, int) ([]map[string]any, error) {
		calls++
		if calls == 1 || calls == 3 {
			return spans, nil
		}
		return nil, nil
	}
	p := CheckWireParity(context.Background(), armRows("motoko-or-glm", at), armRows("pi-or-glm", at), allOpenRouter, fetch)
	if p.Status != WireMismatch {
		t.Fatalf("status = %q (%s), want mismatch", p.Status, p.Message)
	}
	got := map[string]bool{}
	for _, d := range p.Diffs {
		got[d.Param] = true
	}
	for _, want := range []string{"output_cap", "provider", "reasoning", "majority_host"} {
		if !got[want] {
			t.Errorf("missing diff %q; got %+v", want, p.Diffs)
		}
	}
	if got["stream"] || got["store"] || got["messages"] {
		t.Errorf("transport fields must not count as differences: %+v", p.Diffs)
	}
	if !strings.HasPrefix(p.Message, "CONFOUNDED") {
		t.Errorf("message must say CONFOUNDED: %q", p.Message)
	}
}

func TestWireParity_MatchingRequestsMatch(t *testing.T) {
	at := time.Date(2026, 10, 16, 16, 0, 0, 0, time.UTC)
	req := func() map[string]any {
		return map[string]any{"model": "z-ai/glm-5.3-flash", "max_tokens": 32000, "reasoning": map[string]any{"effort": "high"}}
	}
	spans := []map[string]any{wireSpan(t, "a-glm", req(), "CoreWeave"), wireSpan(t, "b-glm", req(), "CoreWeave")}
	p := CheckWireParity(context.Background(), armRows("a-glm", at), armRows("b-glm", at), allOpenRouter, fixedFetch(spans))
	if p.Status != WireMatch {
		t.Fatalf("status = %q (%s), want match", p.Status, p.Message)
	}
}

// max_tokens and max_completion_tokens are one knob; the same value under the
// two names must not read as a difference.
func TestWireParity_OutputCapAliasesAreOneKnob(t *testing.T) {
	at := time.Date(2026, 10, 16, 16, 0, 0, 0, time.UTC)
	spans := []map[string]any{
		wireSpan(t, "a-glm", map[string]any{"max_tokens": 32000}, "CoreWeave"),
		wireSpan(t, "b-glm", map[string]any{"max_completion_tokens": 32000}, "CoreWeave"),
	}
	p := CheckWireParity(context.Background(), armRows("a-glm", at), armRows("b-glm", at), allOpenRouter, fixedFetch(spans))
	if p.Status != WireMatch {
		t.Fatalf("status = %q (%s), want match", p.Status, p.Message)
	}
}

// A workdir that appears only in a tool result (an agent listing the shared
// ailang_eval parent) must not put that call in the other arm.
func TestWireParity_ToolResultPathsDoNotClaimACall(t *testing.T) {
	at := time.Date(2026, 10, 16, 16, 0, 0, 0, time.UTC)
	in, _ := json.Marshal(map[string]any{"messages": []map[string]any{
		{"role": "user", "content": "work in /tmp/ailang_eval/quine_a-glm_20261016_1_1"},
		{"role": "assistant", "content": "ls .."},
		{"role": "tool", "content": "/tmp/ailang_eval/quine_b-glm_20261016_101435_48995"},
	}})
	out, _ := json.Marshal(map[string]any{"rawRequest": map[string]any{"max_tokens": 1}})
	leak := map[string]any{"span.input": string(in), "span.output": string(out)}
	own := wireSpan(t, "b-glm", map[string]any{"max_tokens": 2}, "CoreWeave")
	mine := wireSpan(t, "a-glm", map[string]any{"max_tokens": 1}, "CoreWeave")
	p := CheckWireParity(context.Background(), armRows("a-glm", at), armRows("b-glm", at), allOpenRouter, fixedFetch([]map[string]any{leak, own, mine}))
	if p.Off == nil || p.Off.Calls != 1 {
		t.Fatalf("OFF arm claimed %+v calls, want only its own 1", p.Off)
	}
}

func TestWireParity_NoTracesIsUnverifiedNotMatch(t *testing.T) {
	at := time.Date(2026, 10, 16, 16, 0, 0, 0, time.UTC)
	p := CheckWireParity(context.Background(), armRows("a-glm", at), armRows("b-glm", at), allOpenRouter, fixedFetch(nil))
	if p.Status != WireUnverified {
		t.Fatalf("status = %q, want unverified", p.Status)
	}
}

func TestWireParity_NonOpenRouterArmIsNotApplicable(t *testing.T) {
	at := time.Date(2026, 10, 16, 16, 0, 0, 0, time.UTC)
	local := func(string) (string, bool) { return "", false }
	p := CheckWireParity(context.Background(), armRows("a", at), armRows("b", at), local, fixedFetch(nil))
	if p.Status != WireNotApplicable {
		t.Fatalf("status = %q, want not_applicable", p.Status)
	}
}

// A full page means the window may be truncated: it must be split, never read
// as complete.
func TestFetchWindow_SplitsAFullPage(t *testing.T) {
	start := time.Date(2026, 10, 16, 16, 0, 0, 0, time.UTC)
	var widths []time.Duration
	fetch := func(_ context.Context, _ string, s, e time.Time, limit int) ([]map[string]any, error) {
		widths = append(widths, e.Sub(s))
		if e.Sub(s) > time.Minute {
			return make([]map[string]any, limit), nil
		}
		return []map[string]any{{}}, nil
	}
	got, err := fetchWindow(context.Background(), fetch, "m", start, start.Add(4*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Fatalf("got %d spans from 4 one-minute pages, want 4 (widths %v)", len(got), widths)
	}
}
