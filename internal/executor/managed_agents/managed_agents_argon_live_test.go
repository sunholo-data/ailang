package managed_agents

// Live probe W series: can agent_config pin gemini-4-argon, and on which surface?
//
// Re-asks the 2026-09-02 (Vertex, S series) and 2026-10-13 (Developer API)
// questions against the current docs, which show
//
//	agent="antigravity-preview-09-2026", agent_config={"type":"antigravity","model":"gemini-4-argon"}
//
// with a bare genai.Client() — i.e. the Developer API. Every pinned probe has a
// bogus-model twin: if the bogus one is ALSO accepted, acceptance proves
// nothing and only the billing export (SKU per project/day) can say what ran.
// W*_wrong_type asks whether `type` is a validated discriminator.
//
//	AILANG_LIVE_MA_FEATURES=1 GEMINI_AGENTS_API_KEY=... \
//	  go test ./internal/executor/managed_agents/ -run TestLiveArgonPin -v -timeout 30m
//
// Vertex probes need ADC (project AILANG_LIVE_MA_PROJECT, default ailang-dev);
// Developer API probes need GEMINI_AGENTS_API_KEY. Either half skips alone.
// Every probe directive is "reply with exactly: OK" — one model step each.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

const argonModel = "gemini-4-argon"

func argonBody(agent, agentConfig string) string {
	b := fmt.Sprintf(`{"stream":true,"background":true,"store":true,"agent":%q,`+
		`"environment":{"type":"remote"},`+
		`"input":[{"type":"user_input","content":[{"type":"text","text":"reply with exactly: OK"}]}]`, agent)
	if agentConfig != "" {
		b += `,"agent_config":` + agentConfig
	}
	return b + "}"
}

func argonProbes(agent, prefix string) []featureProbe {
	return []featureProbe{
		{prefix + "1_no_config", "Is agent " + agent + " served on this surface at all (404 = not listed)?",
			argonBody(agent, "")},
		{prefix + "2_pin_argon", "Is agent_config{type:antigravity, model:" + argonModel + "} accepted?",
			argonBody(agent, `{"type":"antigravity","model":"`+argonModel+`"}`)},
		{prefix + "3_pin_bogus", "CONTROL: is a nonexistent model rejected? Accepted here means W2's acceptance carries no information.",
			argonBody(agent, `{"type":"antigravity","model":"gemini-9-does-not-exist"}`)},
		{prefix + "4_wrong_type", "Is `type` a validated discriminator (\"antigravity\" required) or free text?",
			argonBody(agent, `{"type":"ailang-probe-nonsense","model":"`+argonModel+`"}`)},
	}
}

func TestLiveArgonPin(t *testing.T) {
	if os.Getenv("AILANG_LIVE_MA_FEATURES") != "1" {
		t.Skip("live probe disabled; set AILANG_LIVE_MA_FEATURES=1 (provisions real, billed sandboxes)")
	}
	ctx := context.Background()

	if _, err := defaultTokenSource(ctx); err != nil {
		fmt.Printf("VERTEX half SKIPPED (ADC unavailable: %v)\n\n", err)
	} else {
		project := os.Getenv("AILANG_LIVE_MA_PROJECT")
		if project == "" {
			project = "ailang-dev"
		}
		header(project, defaultLocation, "ARGON PIN — VERTEX (W_vertex)")
		var probes []featureProbe
		probes = append(probes, argonProbes(devAPIDefaultAgent, "W_vertex_09_")...)
		probes = append(probes, argonProbes(defaultAgent, "W_vertex_05_")[1:]...)
		reportFeatureProbes(ctx, project, defaultLocation, selectedFeatureProbes(probes))
	}

	key := os.Getenv(DevAPIKeyEnv)
	if key == "" {
		fmt.Printf("DEVELOPER API half SKIPPED (%s unset)\n", DevAPIKeyEnv)
		return
	}
	url := fmt.Sprintf("%s/%s/interactions", devAPIHost, devAPIVersion)
	fmt.Printf("\n========== ARGON PIN — DEVELOPER API (W_dev) ==========\nendpoint=%s revision=%s\n\n", url, apiRevision)
	for _, p := range selectedFeatureProbes(argonProbes(devAPIDefaultAgent, "W_dev_09_")) {
		fmt.Printf("---------- PROBE %s ----------\nQUESTION: %s\nREQUEST body:\n%s\n\n", p.label, p.question, p.bodyJSON)
		fmt.Printf("RESULT: %s\n\n", devStreamSummary(ctx, url, key, p.bodyJSON))
	}
}

var (
	reModelField = regexp.MustCompile(`"[a-z_]*model[a-z_]*"\s*:\s*"[^"]*"`)
	reUsage      = regexp.MustCompile(`"usage"\s*:\s*\{[^}]*\}`)
	reStatus     = regexp.MustCompile(`"status"\s*:\s*"[^"]*"`)
)

// devStreamSummary POSTs one streamed interaction and reports the HTTP status,
// every model-named field the server put on the wire (the only non-billing
// evidence of what ran), the last status and the last usage block. The key is
// sent as a header and never printed.
func devStreamSummary(ctx context.Context, url, key, body string) string {
	reqCtx, cancel := context.WithTimeout(ctx, 6*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return "build request: " + err.Error()
	}
	req.Header.Set(devAPIKeyHeader, key)
	req.Header.Set(apiRevisionHeader, apiRevision)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "transport error: " + err.Error()
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	s := string(raw)
	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("REJECTED HTTP %d\n%s", resp.StatusCode, strings.TrimSpace(truncate(s, 2048)))
	}
	seen := map[string]bool{}
	var models []string
	for _, m := range reModelField.FindAllString(s, -1) {
		if !seen[m] {
			seen[m] = true
			models = append(models, m)
		}
	}
	last := func(re *regexp.Regexp) string {
		all := re.FindAllString(s, -1)
		if len(all) == 0 {
			return "(none)"
		}
		return all[len(all)-1]
	}
	return fmt.Sprintf("ACCEPTED HTTP 200 (%d bytes)\n  model fields on wire: %v\n  last %s\n  last %s",
		len(raw), models, last(reStatus), last(reUsage))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
