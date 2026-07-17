package managed_agents

// Phase-1 contract-discovery spike for M-GEMINI-REPO-MOUNT (mission gap G4).
//
// This is a MANUAL, live, ADC-gated probe. It is NEVER run in default CI: it
// requires working Application Default Credentials, provisions real Google-
// hosted sandboxes (real cost), and hits the live Vertex Managed Agents
// `interactions` endpoint. It exists to RECORD the accepted `environment`
// wire contract (repository/inline source field names, inline content
// encoding, per-file byte ceiling, public-repo behavior, default outbound
// network, and clone depth) before any encoder code is designed — see
// design_docs/planned/v0_30_0/m-gemini-repo-mount.md, Phase 1.
//
// Run it with:
//
//	AILANG_LIVE_MANAGED_AGENTS_MOUNT=1 \
//	  go test ./internal/executor/managed_agents/ -run TestLiveEnvironmentContract -v -timeout 20m
//
// Optional overrides:
//
//	AILANG_LIVE_MA_PROJECT   (default "ailang-dev")
//	AILANG_LIVE_MA_LOCATION  (default "global")
//	AILANG_LIVE_MA_REPO      (default "https://github.com/sunholo-data/ailang.git")
//
// The probe reuses the package's own sendInteraction + parseSSE + foldEvent so
// the request/response handling is byte-identical to production. Interaction
// and environment IDs are redacted from stdout so the transcript can be pasted
// into the (public) design doc without leaking account-scoped identifiers. The
// ADC bearer token is added inside sendInteraction and never printed.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// liveProbe describes one environment-shape experiment.
type liveProbe struct {
	label     string
	envJSON   string // the raw JSON assigned to interactionRequest.Environment
	directive string // the user_input directive; keep it a 1-step observation
}

// probeResult is the sanitized outcome of one live interaction.
type probeResult struct {
	label        string
	status       string
	stepCount    int
	agentText    string
	unknownCount int
	err          error
}

func redactID(id string) string {
	if id == "" {
		return "<none>"
	}
	if len(id) <= 6 {
		return "<redacted>"
	}
	return "<redacted:" + fmt.Sprintf("%d", len(id)) + "chars>"
}

func runLiveProbe(ctx context.Context, project, location string, p liveProbe) probeResult {
	body := &interactionRequest{
		Stream:      true,
		Background:  true,
		Store:       true,
		Agent:       defaultAgent,
		Environment: json.RawMessage(p.envJSON),
		Input: []inputBlock{{
			Type:    "user_input",
			Content: []contentBlock{{Type: "text", Text: p.directive}},
		}},
	}

	reqCtx, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()

	res := probeResult{label: p.label}

	reader, err := sendInteraction(reqCtx, defaultHTTPClient(), defaultTokenSource, project, location, body)
	if err != nil {
		// This is the INFORMATIVE path for a rejected shape: the API's 4xx
		// error body (field-name complaints, "unknown field", size limits)
		// tells us the real contract without provisioning a sandbox.
		res.err = err
		return res
	}
	defer reader.Close()

	state := &streamState{}
	parseErr := parseSSE(reader, func(ev sseEvent) error {
		return foldEvent(state, ev)
	})
	res.status = state.Status
	res.stepCount = state.StepCount
	res.agentText = state.Text.String()
	res.unknownCount = len(state.UnknownEvents)
	if parseErr != nil {
		res.err = parseErr
	}
	return res
}

func TestLiveEnvironmentContract(t *testing.T) {
	if os.Getenv("AILANG_LIVE_MANAGED_AGENTS_MOUNT") != "1" {
		t.Skip("live Managed Agents mount spike disabled; set AILANG_LIVE_MANAGED_AGENTS_MOUNT=1 (needs ADC, provisions real sandboxes)")
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

	// Sentinel that is trivially distinguishable raw-vs-base64: if the mounted
	// file contains this literal string, content is raw UTF-8; if it contains
	// the base64 of it, the API expects base64.
	const inlineSentinel = "AILANG_INLINE_SENTINEL_v1_do_not_edit"
	const inlineTarget = "/workspace/ailang_probe_inline.txt"

	// A repo-observation directive that also folds in the default-network probe
	// and the clone-depth probe (rev-list count + is-shallow) so one paid run
	// covers three Premise Verification Log rows.
	repoDirective := "You are in a Linux sandbox. Run EACH shell command below and paste its raw stdout+stderr verbatim, each under a line '### <n>'. Do not summarize or interpret.\n" +
		"### 1\nls -la /workspace 2>&1\n" +
		"### 2\nls -la /workspace/ailang 2>&1 | head -20\n" +
		"### 3\ngit -C /workspace/ailang rev-parse HEAD 2>&1\n" +
		"### 4\ngit -C /workspace/ailang rev-list --count HEAD 2>&1\n" +
		"### 5\ngit -C /workspace/ailang rev-parse --is-shallow-repository 2>&1\n" +
		"### 6\ngit -C /workspace/ailang log --oneline -3 2>&1\n" +
		"### 7\ncurl -sS -m 10 -o /dev/null -w 'NET_HTTP_CODE=%{http_code}\\n' https://example.com 2>&1\n"

	inlineDirective := "You are in a Linux sandbox. Run EACH shell command below and paste its raw stdout+stderr verbatim, each under a line '### <n>'. Do not summarize.\n" +
		"### 1\nls -la /workspace 2>&1\n" +
		"### 2\ncat " + inlineTarget + " 2>&1\n" +
		"### 3\nwc -c " + inlineTarget + " 2>&1\n" +
		"### 4\nod -c " + inlineTarget + " 2>&1 | head -5\n"

	// The documented shapes (from ai.google.dev/gemini-api/docs/agent-environment).
	// These are the ASSUMED/DOC-ONLY shapes we are here to verify or refute.
	repoSource := fmt.Sprintf(`{"type":"repository","source":%q,"target":"/workspace/ailang"}`, repo)
	inlineSource := fmt.Sprintf(`{"type":"inline","target":%q,"content":%q}`, inlineTarget, inlineSentinel)

	// Boundary payloads: exactly 1<<20 bytes vs 1<<20+1 bytes of inline content.
	atLimit := strings.Repeat("a", 1<<20)
	overLimit := strings.Repeat("a", (1<<20)+1)
	atLimitSource := fmt.Sprintf(`{"type":"inline","target":%q,"content":%q}`, "/workspace/at_limit.txt", atLimit)
	overLimitSource := fmt.Sprintf(`{"type":"inline","target":%q,"content":%q}`, "/workspace/over_limit.txt", overLimit)

	probes := []liveProbe{
		{
			label:     "A_repo_only",
			envJSON:   `{"type":"remote","sources":[` + repoSource + `]}`,
			directive: repoDirective,
		},
		{
			label:     "B_inline_only",
			envJSON:   `{"type":"remote","sources":[` + inlineSource + `]}`,
			directive: inlineDirective,
		},
		{
			label:     "C_combined",
			envJSON:   `{"type":"remote","sources":[` + repoSource + `,` + inlineSource + `]}`,
			directive: repoDirective + "\n### 8\ncat " + inlineTarget + " 2>&1\n",
		},
		{
			label:     "D_over_limit_1MiB_plus_1",
			envJSON:   `{"type":"remote","sources":[` + overLimitSource + `]}`,
			directive: "reply with exactly: OVERLIMIT_ACCEPTED",
		},
		{
			label:     "E_at_limit_1MiB",
			envJSON:   `{"type":"remote","sources":[` + atLimitSource + `]}`,
			directive: "run: wc -c /workspace/at_limit.txt 2>&1 ; reply with its raw output only",
		},
		// The live endpoint rejected repository/inline and named the real
		// supported source types: `gcs` and `skill_registry`. These bare/guess
		// probes elicit the required-field contract for each (validation 400s,
		// no sandbox provisioned) so the redesign is grounded in the real wire
		// shape rather than the refuted documentation.
		{
			label:     "F_gcs_bare",
			envJSON:   `{"type":"remote","sources":[{"type":"gcs"}]}`,
			directive: "reply with exactly: OK",
		},
		{
			label:     "G_gcs_guess_fields",
			envJSON:   `{"type":"remote","sources":[{"type":"gcs","source":"gs://ailang-dev-probe-nonexistent/x.tar","target":"/workspace/x"}]}`,
			directive: "reply with exactly: OK",
		},
		{
			label:     "H_skill_registry_bare",
			envJSON:   `{"type":"remote","sources":[{"type":"skill_registry"}]}`,
			directive: "reply with exactly: OK",
		},
		// Egress is the gate for ALL data sources. These probes guess the
		// enable-egress wire shape; a change away from "Network egress is not
		// enabled" pins the field name. The `config` wrapper is inferred from
		// the F/H error path `environment.config.sources.target`.
		{
			label:     "I_config_network_egress_enabled_bool",
			envJSON:   `{"type":"remote","config":{"network":{"egress_enabled":true},"sources":[{"type":"gcs","source":"gs://ailang-dev-probe-nonexistent/x.tar","target":"/workspace/x"}]}}`,
			directive: "reply with exactly: OK",
		},
		{
			label:     "J_config_network_egress_enum",
			envJSON:   `{"type":"remote","config":{"network":{"egress":"ENABLED"},"sources":[{"type":"gcs","source":"gs://ailang-dev-probe-nonexistent/x.tar","target":"/workspace/x"}]}}`,
			directive: "reply with exactly: OK",
		},
		{
			label:     "K_top_network_egress",
			envJSON:   `{"type":"remote","network":{"egress":"ENABLED"},"sources":[{"type":"gcs","source":"gs://ailang-dev-probe-nonexistent/x.tar","target":"/workspace/x"}]}`,
			directive: "reply with exactly: OK",
		},
		{
			label:     "L_network_enable_egress",
			envJSON:   `{"type":"remote","network":{"enable_egress":true},"sources":[{"type":"gcs","source":"gs://ailang-dev-probe-nonexistent/x.tar","target":"/workspace/x"}]}`,
			directive: "reply with exactly: OK",
		},
		{
			label:     "M_network_enable_internet_access",
			envJSON:   `{"type":"remote","network":{"enable_internet_access":true},"sources":[{"type":"gcs","source":"gs://ailang-dev-probe-nonexistent/x.tar","target":"/workspace/x"}]}`,
			directive: "reply with exactly: OK",
		},
		{
			label:     "N_network_egress_setting_enum",
			envJSON:   `{"type":"remote","network":{"egress_setting":"EGRESS_SETTING_ALL"},"sources":[{"type":"gcs","source":"gs://ailang-dev-probe-nonexistent/x.tar","target":"/workspace/x"}]}`,
			directive: "reply with exactly: OK",
		},
		// Iter-46 (human directive #399, philschmid.de/managed-agents-gh): iter-45's
		// egress guesses (I-N) were all SCALAR enable-flags. The blog demonstrates
		// the real egress shape on the Gemini *Developer* API is a LIST:
		// network.allowlist:[{domain, transform:[{Authorization:...}]}]. Never tried
		// on Vertex. These probe whether the SAME structured shape is the accepted
		// Vertex egress param (a change away from "Network egress is not enabled" or
		// "Unknown parameter environment.network" pins it → G4 unblockable on Vertex,
		// no re-target). gcs source is nonexistent → validation 400, no sandbox.
		{
			label:     "O_top_network_allowlist",
			envJSON:   `{"type":"remote","network":{"allowlist":[{"domain":"api.github.com","transform":[{"Authorization":"Bearer X"}]}]},"sources":[{"type":"gcs","source":"gs://ailang-dev-probe-nonexistent/x.tar","target":"/workspace/x"}]}`,
			directive: "reply with exactly: OK",
		},
		{
			label:     "P_config_network_allowlist",
			envJSON:   `{"type":"remote","config":{"network":{"allowlist":[{"domain":"api.github.com","transform":[{"Authorization":"Bearer X"}]}]},"sources":[{"type":"gcs","source":"gs://ailang-dev-probe-nonexistent/x.tar","target":"/workspace/x"}]}}`,
			directive: "reply with exactly: OK",
		},
		{
			label:     "Q_top_network_allowlist_domain_only",
			envJSON:   `{"type":"remote","network":{"allowlist":[{"domain":"*"}]},"sources":[{"type":"gcs","source":"gs://ailang-dev-probe-nonexistent/x.tar","target":"/workspace/x"}]}`,
			directive: "reply with exactly: OK",
		},
		// Q proved network.allowlist:[{domain:"*"}] enables egress and provisions a
		// sandbox. R is the end-to-end money shot for G4/#399 ("can gemini git clone
		// the codebase?"): egress-only (NO data source at all), agent clones the
		// PUBLIC ailang repo itself over the open egress. If HEAD + file listing come
		// back, the whole repository/inline/gcs MOUNT question is MOOT for public
		// repos — clone-over-egress replaces it. This provisions a real sandbox.
		{
			label:   "R_egress_only_agent_git_clone_public",
			envJSON: `{"type":"remote","network":{"allowlist":[{"domain":"*"}]}}`,
			directive: "You are in a Linux sandbox with outbound network. Run EACH command and paste raw stdout+stderr verbatim under '### <n>'. Do not summarize.\n" +
				"### 1\ncd /tmp && git clone --depth 1 https://github.com/sunholo-data/ailang.git 2>&1 | tail -5\n" +
				"### 2\ngit -C /tmp/ailang rev-parse HEAD 2>&1\n" +
				"### 3\nls /tmp/ailang 2>&1 | head -20\n" +
				"### 4\nsed -n '1,3p' /tmp/ailang/go.mod 2>&1\n",
		},
	}

	// Allow selecting a subset via AILANG_LIVE_MA_PROBES="A,B" to control cost.
	if sel := os.Getenv("AILANG_LIVE_MA_PROBES"); sel != "" {
		want := map[string]bool{}
		for _, s := range strings.Split(sel, ",") {
			want[strings.TrimSpace(strings.ToUpper(s))] = true
		}
		filtered := probes[:0]
		for _, p := range probes {
			if want[strings.ToUpper(p.label[:1])] {
				filtered = append(filtered, p)
			}
		}
		probes = filtered
	}

	fmt.Printf("\n========== LIVE MANAGED-AGENTS ENVIRONMENT CONTRACT SPIKE ==========\n")
	fmt.Printf("project=%s location=%s repo=%s agent=%s revision=%s\n",
		project, location, repo, defaultAgent, apiRevision)
	fmt.Printf("endpoint=%s\n\n", buildInteractionsURL(project, location))

	ctx := context.Background()
	for _, p := range probes {
		fmt.Printf("---------- PROBE %s ----------\n", p.label)
		fmt.Printf("REQUEST environment (verbatim, no credentials):\n%s\n\n", p.envJSON)
		r := runLiveProbe(ctx, project, location, p)
		if r.err != nil {
			fmt.Printf("RESULT: transport/parse error (this is the informative-rejection path):\n%v\n\n", r.err)
			continue
		}
		fmt.Printf("RESULT: status=%q steps=%d unknown_events=%d interaction_id=%s\n",
			r.status, r.stepCount, r.unknownCount, redactID("x"))
		fmt.Printf("AGENT TEXT OUTPUT (verbatim, IDs are not present here):\n%s\n\n", r.agentText)
	}
	fmt.Printf("========== END SPIKE ==========\n")
}
