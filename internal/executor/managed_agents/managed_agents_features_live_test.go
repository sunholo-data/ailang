package managed_agents

// Live contract probe for the 2026-07 Managed Agents feature drop
// (ai.google.dev/gemini-api/docs/agent-hooks + the "3.6 Flash, hooks and more"
// announcement): agent_config model selection, agent_config.max_total_tokens,
// environment hooks, and the Environments API.
//
// WHY a probe and not an implementation: those docs describe the Gemini
// *Developer* API surface (generativelanguage.googleapis.com, API-key auth).
// Our executor runs on **Vertex** (aiplatform.googleapis.com, ADC). The two
// have already diverged once — probes A-E in managed_agents_live_test.go
// recorded Vertex REJECTING the documented `repository` / `inline` source
// types and naming `gcs` + `skill_registry` instead. So every field below is
// DOC-ONLY until this probe says otherwise, on the surface we actually use.
//
// Probe letters continue the A-R series in managed_agents_live_test.go.
//
//	S* — agent_config: model selection + max_total_tokens (Vertex)
//	T* — hooks delivery + behaviour (Vertex)
//	U* — Environments API list/get/delete (both surfaces)
//	V* — the Gemini Developer API surface itself (is it usable by us at all?)
//
// Cost discipline: probes whose only question is "is this field accepted?"
// are paired with a directive of "reply with exactly: OK" (one model step) or,
// better, a deliberately tiny max_total_tokens so an ACCEPTED field halts the
// run immediately. A REJECTED field 400s at validation and provisions no
// sandbox at all — the rejection path is the cheap path.
//
// Run it with:
//
//	AILANG_LIVE_MA_FEATURES=1 \
//	  go test ./internal/executor/managed_agents/ -run 'TestLiveFeature' -v -timeout 30m
//
// Optional overrides:
//
//	AILANG_LIVE_MA_PROJECT   (default "ailang-dev")
//	AILANG_LIVE_MA_LOCATION  (default "global")
//	AILANG_LIVE_MA_FPROBES   comma-separated label prefixes, e.g. "S,U"
//	GOOGLE_API_KEY           required only for the V* / Gemini-dev-surface probes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	// geminiDevHost is the Gemini *Developer* API host the new feature docs
	// are written against. Distinct from vertexAIHost in client.go.
	geminiDevHost = "https://generativelanguage.googleapis.com"
	// geminiDevVersion is the version segment used by the documented curl
	// examples (note: v1beta, NOT Vertex's v1beta1).
	geminiDevVersion = "v1beta"
)

// featureProbe is one raw-body experiment against the interactions endpoint.
// bodyJSON is the COMPLETE request body, verbatim, so the transcript pasted
// into a design doc is exactly what went on the wire.
type featureProbe struct {
	label    string
	question string // what a pass/fail actually tells us
	bodyJSON string
}

// runFeatureProbe POSTs a verbatim body through the production auth/header
// path and folds the SSE stream. A transport error IS a result here: the
// API's 4xx body names unknown fields and required discriminators.
func runFeatureProbe(ctx context.Context, project, location string, p featureProbe) probeResult {
	reqCtx, cancel := context.WithTimeout(ctx, 6*time.Minute)
	defer cancel()

	res := probeResult{label: p.label}

	reader, err := sendInteraction(reqCtx, defaultHTTPClient(), defaultTokenSource,
		project, location, json.RawMessage(p.bodyJSON))
	if err != nil {
		res.err = err
		return res
	}
	defer reader.Close()

	state := &streamState{}
	parseErr := parseSSE(reader, func(ev sseEvent) error { return foldEvent(state, ev) })
	res.status = state.Status
	res.stepCount = state.StepCount
	res.agentText = state.Text.String()
	res.unknownCount = len(state.UnknownEvents)
	res.usage = state.Usage
	if parseErr != nil {
		res.err = parseErr
	}
	return res
}

// vertexBody assembles a minimal-but-valid Vertex interaction body with an
// arbitrary extra field block spliced in, so each probe differs ONLY by the
// field under test.
func vertexBody(extra, directive string) string {
	b := fmt.Sprintf(`{"stream":true,"background":true,"store":true,"agent":%q,`+
		`"environment":{"type":"remote"},`+
		`"input":[{"type":"user_input","content":[{"type":"text","text":%q}]}]`,
		defaultAgent, directive)
	if extra != "" {
		b += "," + extra
	}
	return b + "}"
}

func selectedFeatureProbes(probes []featureProbe) []featureProbe {
	sel := os.Getenv("AILANG_LIVE_MA_FPROBES")
	if sel == "" {
		return probes
	}
	want := map[string]bool{}
	for s := range strings.SplitSeq(sel, ",") {
		if s = strings.TrimSpace(strings.ToUpper(s)); s != "" {
			want[s] = true
		}
	}
	out := probes[:0]
	for _, p := range probes {
		up := strings.ToUpper(p.label)
		for w := range want {
			if strings.HasPrefix(up, w) {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

func reportFeatureProbes(ctx context.Context, project, location string, probes []featureProbe) {
	for _, p := range probes {
		fmt.Printf("---------- PROBE %s ----------\n", p.label)
		fmt.Printf("QUESTION: %s\n", p.question)
		fmt.Printf("REQUEST body (verbatim, no credentials):\n%s\n\n", p.bodyJSON)
		r := runFeatureProbe(ctx, project, location, p)
		if r.err != nil {
			fmt.Printf("RESULT: REJECTED / error (the informative path):\n%v\n\n", r.err)
			continue
		}
		fmt.Printf("RESULT: ACCEPTED status=%q steps=%d unknown_events=%d\n",
			r.status, r.stepCount, r.unknownCount)
		// Usage is how we tell an ENFORCED cap from an accepted-and-ignored
		// one: a cap that bound shows total_tokens at/below the requested
		// ceiling, and a status other than "completed".
		fmt.Printf("USAGE: total=%d in=%d out=%d thought=%d\n",
			r.usage.TotalTokens, r.usage.TotalInputTokens,
			r.usage.TotalOutputTokens, r.usage.TotalThoughtTokens)
		fmt.Printf("AGENT TEXT (verbatim):\n%s\n\n", r.agentText)
	}
}

// TestLiveFeatureAgentConfig probes the S series: does Vertex accept the
// `agent_config` container documented for the Gemini Developer API, and with
// it explicit model selection + the max_total_tokens cost cap?
//
// This matters twice over:
//  1. max_total_tokens would close the gap README.md documents as IMPOSSIBLE
//     ("mid-stream kill-on-cost"), because Vertex only reports usage in the
//     terminal event.
//  2. The announcement says gemini-3.6-flash is now the DEFAULT for
//     antigravity-preview-05-2026 — the agent string we pin. Our CostUSD is
//     hardcoded to gemini-3-5-flash rates, so if the default moved under us
//     every banked managed_agents cost figure is mislabelled. S6 asks the
//     sandbox which model it is actually running.
func TestLiveFeatureAgentConfig(t *testing.T) {
	project, location := requireFeatureLive(t)

	const okDirective = "reply with exactly: OK"

	probes := []featureProbe{
		{
			label:    "S1_agent_config_typed_bare",
			question: "Is `agent_config` accepted AT ALL on Vertex once the `type` discriminator is present? (the earlier spike saw 'unknown field agent_config' — likely a oneof with an unresolved discriminator, not an absent feature)",
			bodyJSON: vertexBody(`"agent_config":{"type":"antigravity"}`, okDirective),
		},
		{
			label:    "S2_agent_config_model_36_flash",
			question: "Can we pin the reasoning model to gemini-3.6-flash?",
			bodyJSON: vertexBody(`"agent_config":{"type":"antigravity","model":"gemini-3.6-flash"}`, okDirective),
		},
		{
			label:    "S3_agent_config_model_flash_lite",
			question: "Is the cheap tier (gemini-3.5-flash-lite) selectable? Directly sets the floor of eval cost-per-run.",
			bodyJSON: vertexBody(`"agent_config":{"type":"antigravity","model":"gemini-3.5-flash-lite"}`, okDirective),
		},
		{
			label:    "S4_agent_config_model_bogus",
			question: "Is `model` VALIDATED or silently ignored? A bogus id that is ACCEPTED means the field is decorative and any model pinning we report would be a lie.",
			bodyJSON: vertexBody(`"agent_config":{"type":"antigravity","model":"gemini-9.9-does-not-exist"}`, okDirective),
		},
		{
			label:    "S5_max_total_tokens_in_agent_config",
			question: "Is max_total_tokens accepted inside agent_config? (documented placement; string-typed in the doc example)",
			bodyJSON: vertexBody(`"agent_config":{"type":"antigravity","max_total_tokens":"200000"}`, okDirective),
		},
		{
			label:    "S6_max_total_tokens_tiny_behavioral",
			question: "Does a TINY cap actually halt the run with status 'incomplete' (resumable), rather than being accepted-and-ignored? Deliberately cheap: if the cap works, the run dies almost immediately.",
			bodyJSON: vertexBody(`"agent_config":{"type":"antigravity","max_total_tokens":"64"}`,
				"Count slowly from 1 to 500, writing out every number in full words on its own line. Do not stop early."),
		},
		{
			label:    "S7_max_total_tokens_top_level",
			question: "Alternate placement: does Vertex take max_total_tokens at the TOP level instead? (pins which of the two shapes is real)",
			bodyJSON: vertexBody(`"max_total_tokens":"200000"`, okDirective),
		},
		{
			label:    "S9_max_total_tokens_garbage_value",
			question: "Is agent_config even PARSED? A non-numeric value that is ACCEPTED proves the sub-fields are discarded wholesale (so S6's ignored cap is not a units/typing mistake on our side). A 400 proves it IS parsed and the ignoring happens at enforcement.",
			bodyJSON: vertexBody(`"agent_config":{"type":"antigravity","max_total_tokens":"not-a-number"}`, okDirective),
		},
		{
			label:    "S10_max_total_tokens_int_form",
			question: "The doc example quotes the value as a STRING; maybe Vertex wants an int64. Same tiny cap, integer form, on a prompt that would blow through it — does the int form enforce where the string form did not?",
			bodyJSON: vertexBody(`"agent_config":{"type":"antigravity","max_total_tokens":64}`,
				"Count slowly from 1 to 200, writing out every number in full words on its own line. Do not stop early."),
		},
		{
			label:    "S11_unknown_subfield_in_agent_config",
			question: "Control for S9: does agent_config reject a field that exists in NO version of the API? If a pure nonsense key is also accepted, the container is a bag we can put anything in — i.e. acceptance carries zero information about support.",
			bodyJSON: vertexBody(`"agent_config":{"type":"antigravity","ailang_probe_nonsense_key":"xyzzy"}`, okDirective),
		},
		{
			label:    "S8_which_model_am_i",
			question: "GROUND TRUTH for the cost-label risk: ask the running sandbox to self-report the backing model, with NO agent_config set (i.e. today's production shape). Our cost model assumes gemini-3-5-flash.",
			bodyJSON: vertexBody("", "Do not use any tools. Reply with exactly one line: MODEL=<the exact model identifier you are running as>"),
		},
	}

	header(project, location, "AGENT_CONFIG / COST-CAP CONTRACT (S series)")
	reportFeatureProbes(context.Background(), project, location, selectedFeatureProbes(probes))
}

// TestLiveFeatureHooks probes the T series: can we get a pre_tool_execution
// hook to run inside a Vertex sandbox?
//
// Why we care: a pre-`write_file` hook can DENY with a `reason`, and the docs
// say the model then adapts. That is a feedback channel — it would let us
// reject a proposed .ail edit that does not type-check, with the compiler
// diagnostic as the reason. Per the docx convergence work, compile-preserving
// incremental edits are the difference between converging and spiralling, so
// this is the highest-value item in the whole feature drop for evals.
//
// Two delivery routes, because Vertex rejected `inline` sources before:
//
//	T1 — mount .agents/hooks.json via an inline source  (cheap: 400s if refused)
//	T2 — agent WRITES the hook config itself in an egress sandbox, then trips
//	     it. Tests whether discovery is boot-time-only or dynamic. The docs'
//	     own warning that "agents with write permissions can modify local hook
//	     configs" implies dynamic — T2 checks that.
//
// T2 also checks the FAIL-OPEN trap: a hook that crashes/times out is treated
// as `allow`, so a broken gate is indistinguishable from "the gate didn't
// help" unless we assert the deny path positively.
func TestLiveFeatureHooks(t *testing.T) {
	project, location := requireFeatureLive(t)

	const gateScript = `import json,sys
d=json.load(sys.stdin)
code=json.dumps(d.get("tool_call",{}).get("args",{}))
if "AILANG_FORBIDDEN_SENTINEL" in code:
    print(json.dumps({"decision":"deny","reason":"AILANG_HOOK_DENIED: blocked by probe gate"}))
else:
    print(json.dumps({"decision":"allow"}))
`
	hooksJSON := `{"ailang-probe-gate":{"enabled":true,"pre_tool_execution":[{"matcher":"code_execution","hooks":[{"type":"command","command":"python3 /workspace/hooks/gate.py","timeout":10}]}]}}`

	inlineHooks := fmt.Sprintf(`{"type":"inline","target":"/workspace/.agents/hooks.json","content":%s}`,
		mustJSONString(hooksJSON))
	inlineGate := fmt.Sprintf(`{"type":"inline","target":"/workspace/hooks/gate.py","content":%s}`,
		mustJSONString(gateScript))

	t1Body := fmt.Sprintf(`{"stream":true,"background":true,"store":true,"agent":%q,`+
		`"environment":{"type":"remote","sources":[%s,%s]},`+
		`"input":[{"type":"user_input","content":[{"type":"text","text":%q}]}]}`,
		defaultAgent, inlineHooks, inlineGate,
		"Run this shell command and paste its raw stdout+stderr verbatim: echo AILANG_FORBIDDEN_SENTINEL. "+
			"Then state on its own line either HOOK_FIRED (if the command was blocked/denied) or HOOK_ABSENT (if it ran normally).")

	// T2: self-installed hooks in an egress sandbox.
	t2Directive := "You are in a Linux sandbox. Do EXACTLY these steps and paste raw stdout+stderr verbatim under '### <n>' headings. Do not summarize.\n" +
		"### 1\nmkdir -p /workspace/.agents /workspace/hooks && cat > /workspace/hooks/gate.py <<'PYEOF'\n" + gateScript + "PYEOF\n" +
		"cat > /workspace/.agents/hooks.json <<'JSONEOF'\n" + hooksJSON + "\nJSONEOF\n" +
		"ls -la /workspace/.agents /workspace/hooks 2>&1\n" +
		"### 2\necho '{\"tool_call\":{\"name\":\"code_execution\",\"args\":{\"code\":\"AILANG_FORBIDDEN_SENTINEL\"}}}' | python3 /workspace/hooks/gate.py 2>&1\n" +
		"### 3\necho AILANG_FORBIDDEN_SENTINEL\n" +
		"### 4\nFinally, output on its own line HOOK_FIRED if step 3 was blocked or denied by a hook, or HOOK_ABSENT if step 3 executed normally and printed the sentinel.\n"

	t2Body := fmt.Sprintf(`{"stream":true,"background":true,"store":true,"agent":%q,`+
		`"environment":{"type":"remote","network":{"allowlist":[{"domain":"*"}]}},`+
		`"input":[{"type":"user_input","content":[{"type":"text","text":%q}]}]}`,
		defaultAgent, t2Directive)

	// T3 closes the gap T2 leaves: the docs name TWO discovery paths,
	// `.agents/hooks.json` (workspace-relative, which T2 covered) and
	// `/.agents/hooks.json` (filesystem root, which it did not). Concluding
	// "hooks are inert on Vertex" from T2 alone would be reading one probe as
	// if it covered both paths.
	t3Directive := "You are in a Linux sandbox. Do EXACTLY these steps and paste raw stdout+stderr verbatim under '### <n>' headings. Do not summarize.\n" +
		"### 1\nls -la / 2>&1 | head -30 ; echo '--- existing agent config, if any ---' ; ls -la /.agents 2>&1 ; cat /.agents/hooks.json 2>&1\n" +
		"### 2\n(mkdir -p /.agents 2>&1 || sudo mkdir -p /.agents 2>&1) ; " +
		"mkdir -p /workspace/hooks && cat > /workspace/hooks/gate.py <<'PYEOF'\n" + gateScript + "PYEOF\n" +
		"cat > /.agents/hooks.json <<'JSONEOF'\n" + hooksJSON + "\nJSONEOF\n" +
		"ls -la /.agents 2>&1 ; echo '--- HOME ---' ; echo $HOME ; mkdir -p $HOME/.agents && cp /.agents/hooks.json $HOME/.agents/hooks.json 2>&1 ; ls -la $HOME/.agents 2>&1\n" +
		"### 3\necho AILANG_FORBIDDEN_SENTINEL\n" +
		"### 4\nOutput on its own line HOOK_FIRED if step 3 was blocked or denied by a hook, or HOOK_ABSENT if step 3 executed normally and printed the sentinel.\n"

	t3Body := fmt.Sprintf(`{"stream":true,"background":true,"store":true,"agent":%q,`+
		`"environment":{"type":"remote","network":{"allowlist":[{"domain":"*"}]}},`+
		`"input":[{"type":"user_input","content":[{"type":"text","text":%q}]}]}`,
		defaultAgent, t3Directive)

	probes := []featureProbe{
		{
			label:    "T3_hooks_at_filesystem_root",
			question: "Does the OTHER documented discovery path, /.agents/hooks.json (plus $HOME/.agents), activate a hook? Also dumps any PRE-EXISTING /.agents the runtime ships, which would show hooks are wired here at all.",
			bodyJSON: t3Body,
		},
		{
			label:    "T1_hooks_via_inline_source",
			question: "Can .agents/hooks.json be MOUNTED via an inline source on Vertex? (probe B previously proved Vertex rejects `inline` — re-checked because the docs changed)",
			bodyJSON: t1Body,
		},
		{
			label:    "T2_hooks_self_installed_in_sandbox",
			question: "Is hook discovery DYNAMIC (re-read after boot)? Step 2 proves the gate script itself works; step 3 proves whether the runtime actually consults it. Step2-denies + step3-runs => discovery is boot-time-only => hooks need a mount path we do not have on Vertex.",
			bodyJSON: t2Body,
		},
	}

	header(project, location, "HOOKS DELIVERY + BEHAVIOUR (T series)")
	reportFeatureProbes(context.Background(), project, location, selectedFeatureProbes(probes))
}

// TestLiveFeatureEnvironmentsAPI probes the U series: is there a
// list/get/delete surface for sandbox environments?
//
// We POST every Execute() with store:true and never clean up, so today this is
// an unbounded sandbox leak; it is also the missing half of the "no multi-turn
// yet" gap in README.md. These are GETs — no sandbox is provisioned, no cost.
func TestLiveFeatureEnvironmentsAPI(t *testing.T) {
	project, location := requireFeatureLive(t)

	header(project, location, "ENVIRONMENTS API (U series)")
	ctx := context.Background()

	// U1/U2: Vertex, both plausible path spellings.
	for _, path := range []string{"environments", "agentEnvironments"} {
		url := fmt.Sprintf("%s/%s/projects/%s/locations/%s/%s?pageSize=5",
			vertexAIHost, apiVersion, project, location, path)
		fmt.Printf("---------- PROBE U_vertex_%s ----------\nGET %s\n", path, url)
		tok, err := defaultTokenSource(ctx)
		if err != nil {
			fmt.Printf("RESULT: ADC unavailable: %v\n\n", err)
			continue
		}
		fmt.Printf("RESULT: %s\n\n", httpGetSummary(ctx, url, map[string]string{
			"Authorization":   "Bearer " + tok,
			apiRevisionHeader: apiRevision,
		}))
	}

	// U3: the documented Gemini Developer API path (API-key auth).
	if key := os.Getenv("GOOGLE_API_KEY"); key != "" {
		url := fmt.Sprintf("%s/%s/environments?pageSize=5", geminiDevHost, geminiDevVersion)
		fmt.Printf("---------- PROBE U_geminidev_environments ----------\nGET %s\n", url)
		fmt.Printf("RESULT: %s\n\n", httpGetSummary(ctx, url, map[string]string{"x-goog-api-key": key}))
	} else {
		fmt.Printf("---------- PROBE U_geminidev_environments ---------- SKIPPED (GOOGLE_API_KEY unset)\n\n")
	}
}

// TestLiveFeatureGeminiDevSurface probes the V series: is the Gemini
// *Developer* API surface — the one every new feature doc is written against —
// actually usable by us?
//
// If Vertex refuses agent_config/hooks but this surface accepts them, the
// unblock is a SURFACE SWITCH (API key, no ADC, free tier for experimentation),
// not a wait on a Vertex rollout. That is a materially different plan, so it is
// worth one cheap probe before designing anything.
func TestLiveFeatureGeminiDevSurface(t *testing.T) {
	if os.Getenv("AILANG_LIVE_MA_FEATURES") != "1" {
		t.Skip("live Managed Agents feature probe disabled; set AILANG_LIVE_MA_FEATURES=1")
	}
	key := os.Getenv("GOOGLE_API_KEY")
	if key == "" {
		t.Skip("GOOGLE_API_KEY unset — cannot probe the Gemini Developer API surface")
	}

	fmt.Printf("\n========== GEMINI DEVELOPER API SURFACE (V series) ==========\n")
	fmt.Printf("host=%s version=%s agent=%s\n\n", geminiDevHost, geminiDevVersion, defaultAgent)

	ctx := context.Background()
	url := fmt.Sprintf("%s/%s/interactions", geminiDevHost, geminiDevVersion)

	bodies := []featureProbe{
		{
			label:    "V1_minimal_model_interaction",
			question: "Does the interactions endpoint answer AT ALL for this API key (is the API enabled on this project/key)?",
			bodyJSON: `{"model":"gemini-3.6-flash","input":"reply with exactly: OK","stream":false,"store":false,"background":false}`,
		},
		{
			label:    "V2_antigravity_agent_config",
			question: "Does the agent surface accept agent_config{type,model,max_total_tokens} here — i.e. is the doc true on THIS host even if false on Vertex?",
			bodyJSON: fmt.Sprintf(`{"agent":%q,"input":"reply with exactly: OK","stream":false,"store":true,"background":true,`+
				`"environment":{"type":"remote"},"agent_config":{"type":"antigravity","model":"gemini-3.6-flash","max_total_tokens":"200000"}}`, defaultAgent),
		},
		{
			label:    "V3_repository_source",
			question: "Does the documented `repository` source type work here? Vertex refuted it (probe A) and we fell back to clone-over-egress; if it works here, a direct repo mount of the ailang tree is available on this surface.",
			bodyJSON: fmt.Sprintf(`{"agent":%q,"input":"run: git -C /workspace/ailang rev-parse HEAD ; reply with its raw output","stream":false,"store":true,"background":true,`+
				`"environment":{"type":"remote","sources":[{"type":"repository","source":"https://github.com/sunholo-data/ailang","target":"/workspace/ailang"}]}}`, defaultAgent),
		},
	}

	for _, p := range selectedFeatureProbes(bodies) {
		fmt.Printf("---------- PROBE %s ----------\n", p.label)
		fmt.Printf("QUESTION: %s\n", p.question)
		fmt.Printf("REQUEST body (verbatim):\n%s\n\n", p.bodyJSON)
		fmt.Printf("RESULT: %s\n\n", httpPostSummary(ctx, url,
			map[string]string{"x-goog-api-key": key, "Content-Type": "application/json"}, p.bodyJSON))
	}
	fmt.Printf("========== END V SERIES ==========\n")
}

// TestLiveFeatureRawStream dumps the RAW SSE bytes for one interaction.
//
// Needed because probe S4 proved `agent_config.model` accepts a nonexistent
// model id without complaint: on this API, ACCEPTED does not mean EFFECTIVE.
// Asking the agent which model it is runs into models confidently
// hallucinating their own identity, so it is not evidence either. The only
// trustworthy source is what the service itself puts on the wire — this dump
// exists to find any `model` echo in the event payloads (and to expose fields
// our typed structs silently drop, e.g. a per-step usage or a cap/limit
// status).
//
//	AILANG_LIVE_MA_FEATURES=1 AILANG_LIVE_MA_RAW_MODEL=gemini-3.5-flash-lite \
//	  go test ./internal/executor/managed_agents/ -run TestLiveFeatureRawStream -v -timeout 10m
func TestLiveFeatureRawStream(t *testing.T) {
	project, location := requireFeatureLive(t)

	extra := `"agent_config":{"type":"antigravity"}`
	if m := os.Getenv("AILANG_LIVE_MA_RAW_MODEL"); m != "" {
		extra = fmt.Sprintf(`"agent_config":{"type":"antigravity","model":%q}`, m)
	}
	body := vertexBody(extra, "reply with exactly: OK")

	fmt.Printf("\n========== RAW SSE DUMP ==========\nbody: %s\n\n", body)

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	reader, err := sendInteraction(ctx, defaultHTTPClient(), defaultTokenSource,
		project, location, json.RawMessage(body))
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	defer reader.Close()

	raw, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	// IDs in a raw dump are account-scoped; the caller decides what to paste.
	fmt.Printf("%s\n", raw)
	fmt.Printf("========== END RAW DUMP (%d bytes) ==========\n", len(raw))

	for _, needle := range []string{"model", "max_total", "limit", "incomplete", "usage"} {
		fmt.Printf("contains %-12q: %v\n", needle, strings.Contains(string(raw), needle))
	}
}

// --- shared probe plumbing ------------------------------------------------

func requireFeatureLive(t *testing.T) (project, location string) {
	t.Helper()
	if os.Getenv("AILANG_LIVE_MA_FEATURES") != "1" {
		t.Skip("live Managed Agents feature probe disabled; set AILANG_LIVE_MA_FEATURES=1 (needs ADC, provisions real sandboxes)")
	}
	if _, err := defaultTokenSource(context.Background()); err != nil {
		t.Skipf("ADC unavailable — run `gcloud auth application-default login`: %v", err)
	}
	project = os.Getenv("AILANG_LIVE_MA_PROJECT")
	if project == "" {
		project = "ailang-dev"
	}
	location = os.Getenv("AILANG_LIVE_MA_LOCATION")
	if location == "" {
		location = "global"
	}
	return project, location
}

func header(project, location, title string) {
	fmt.Printf("\n========== %s ==========\n", title)
	fmt.Printf("project=%s location=%s agent=%s revision=%s\n", project, location, defaultAgent, apiRevision)
	fmt.Printf("endpoint=%s\n\n", buildInteractionsURL(project, location))
}

func mustJSONString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func httpGetSummary(ctx context.Context, url string, headers map[string]string) string {
	return httpSummary(ctx, http.MethodGet, url, headers, "")
}

func httpPostSummary(ctx context.Context, url string, headers map[string]string, body string) string {
	return httpSummary(ctx, http.MethodPost, url, headers, body)
}

// httpSummary performs a one-shot request and renders status + a bounded body
// preview. Credentials live only in the request headers and are never printed.
func httpSummary(ctx context.Context, method, url string, headers map[string]string, body string) string {
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(reqCtx, method, url, rdr)
	if err != nil {
		return fmt.Sprintf("build request: %v", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Sprintf("transport error: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Sprintf("HTTP %d\n%s", resp.StatusCode, strings.TrimSpace(string(b)))
}
