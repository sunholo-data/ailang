package eval_analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// WHY WIRE PARITY
//
// An A/B between two harnesses is only an A/B if both arms send the provider
// the same request. Every pi-vs-motoko comparison on glm-5.3-flash through
// 2026-10-16 failed that, each time in a knob nobody had compared: first the
// host (price routing put motoko on 10 tok/s hosts that also capped reasoning
// at 8193), then the output cap (motoko 32000, pi 943717 — pi's installed
// config never carried the model, so pi used its own catalogue), then reasoning
// effort (pi "high", motoko unset) and routing (pi no provider block at all).
// The configs said the arms matched; the wire said otherwise.
//
// So the check reads the wire: OpenRouter Broadcast traces carry the full
// rawRequest of every call. This compares the request parameters of the two
// arms and reports CONFOUNDED when they differ. It cannot prove the arms are
// equal beyond what the request shows (tools and prompts differ by design
// between harnesses and are not compared).

// SpanFetcher returns the attributes of the LLM Generation spans for one model
// (its OpenRouter api_name) that started in [start, end). It returns at most
// limit spans; a result of exactly limit means the window may be truncated.
type SpanFetcher func(ctx context.Context, apiName string, start, end time.Time, limit int) ([]map[string]any, error)

// WireArm is what one arm actually sent.
type WireArm struct {
	Model   string `json:"model"`
	APIName string `json:"api_name"`
	Calls   int    `json:"calls"`
	// Params maps a request parameter to the values seen and how often.
	Params         map[string]map[string]int `json:"params"`
	Hosts          map[string]int            `json:"hosts"`
	LengthFinishes int                       `json:"length_finishes"`
}

// WireDiff is one parameter the arms disagree on.
type WireDiff struct {
	Param string   `json:"param"`
	On    []string `json:"on"`
	Off   []string `json:"off"`
}

// WireParity statuses.
const (
	WireMatch         = "match"
	WireMismatch      = "mismatch"
	WireUnverified    = "unverified"
	WireNotApplicable = "not_applicable"
)

// WireParity is the verdict on whether the arms sent comparable requests.
type WireParity struct {
	Status  string     `json:"status"`
	Message string     `json:"message"`
	On      *WireArm   `json:"on,omitempty"`
	Off     *WireArm   `json:"off,omitempty"`
	Diffs   []WireDiff `json:"diffs,omitempty"`
}

// transportKeys are request fields that carry the conversation or the
// transport, not a behavioural knob. Everything else is compared, so a knob
// nobody has thought of yet still shows up as a difference.
var transportKeys = map[string]bool{
	"messages": true, "tools": true, "stream": true, "stream_options": true,
	"store": true, "_skin": true, "user": true, "metadata": true,
	"session_id": true, "prompt_cache_key": true, "usage": true,
	"include_reasoning":   true, // response shape only; reasoning still runs
	"tool_choice":         true, // harness design, not a model knob
	"parallel_tool_calls": true,
}

const (
	spanPageLimit = 1000
	spanChunk     = 4 * time.Minute
	minSpanChunk  = 15 * time.Second
	windowSlack   = 2 * time.Minute
)

// CheckWireParity compares what the two arms sent. apiNameFor resolves a
// registry model name to its OpenRouter api_name; ok=false means the model is
// not routed through OpenRouter, which Broadcast cannot see.
func CheckWireParity(ctx context.Context, on, off []*BenchmarkResult,
	apiNameFor func(model string) (apiName string, ok bool), fetch SpanFetcher) WireParity {

	onArm, err := collectArm(ctx, on, apiNameFor, fetch)
	if err != nil {
		return WireParity{Status: wireStatusFor(err), Message: "ON arm: " + err.Error()}
	}
	offArm, err := collectArm(ctx, off, apiNameFor, fetch)
	if err != nil {
		return WireParity{Status: wireStatusFor(err), Message: "OFF arm: " + err.Error(), On: onArm}
	}
	p := WireParity{On: onArm, Off: offArm}

	params := map[string]bool{}
	for k := range onArm.Params {
		params[k] = true
	}
	for k := range offArm.Params {
		params[k] = true
	}
	for _, k := range sortedKeys(params) {
		a, b := valueSet(onArm.Params[k]), valueSet(offArm.Params[k])
		if strings.Join(a, "\x00") != strings.Join(b, "\x00") {
			p.Diffs = append(p.Diffs, WireDiff{Param: k, On: a, Off: b})
		}
	}
	if ha, hb := majorityHost(onArm.Hosts), majorityHost(offArm.Hosts); ha != hb {
		p.Diffs = append(p.Diffs, WireDiff{Param: "majority_host", On: []string{ha}, Off: []string{hb}})
	}

	if len(p.Diffs) == 0 {
		p.Status = WireMatch
		p.Message = fmt.Sprintf("both arms sent the same request parameters (%d and %d calls)", onArm.Calls, offArm.Calls)
		return p
	}
	p.Status = WireMismatch
	parts := make([]string, 0, len(p.Diffs))
	for _, d := range p.Diffs {
		parts = append(parts, fmt.Sprintf("%s on=%v off=%v", d.Param, d.On, d.Off))
	}
	p.Message = "CONFOUNDED: the arms sent different requests — " + strings.Join(parts, "; ")
	return p
}

type notApplicableError struct{ msg string }

func (e notApplicableError) Error() string { return e.msg }

func wireStatusFor(err error) string {
	if _, ok := err.(notApplicableError); ok {
		return WireNotApplicable
	}
	return WireUnverified
}

func collectArm(ctx context.Context, rows []*BenchmarkResult,
	apiNameFor func(string) (string, bool), fetch SpanFetcher) (*WireArm, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("no rows")
	}
	model := rows[0].Model
	for _, r := range rows {
		if r.Model != model {
			return nil, fmt.Errorf("arm mixes models %q and %q; compare one model per arm", model, r.Model)
		}
	}
	apiName, ok := apiNameFor(model)
	if !ok {
		return nil, notApplicableError{fmt.Sprintf("%s is not routed through OpenRouter, so no provider-side trace exists", model)}
	}

	var start, end time.Time
	for _, r := range rows {
		if r.Timestamp.IsZero() {
			continue
		}
		s := r.Timestamp.Add(-time.Duration(r.DurationMs) * time.Millisecond)
		if start.IsZero() || s.Before(start) {
			start = s
		}
		if r.Timestamp.After(end) {
			end = r.Timestamp
		}
	}
	if start.IsZero() {
		return nil, fmt.Errorf("rows carry no timestamps")
	}
	start, end = start.Add(-windowSlack), end.Add(windowSlack)

	spans, err := fetchWindow(ctx, fetch, apiName, start, end)
	if err != nil {
		return nil, err
	}
	arm := &WireArm{Model: model, APIName: apiName, Params: map[string]map[string]int{}, Hosts: map[string]int{}}
	marker := "_" + model + "_"
	for _, a := range spans {
		if !promptNamesArm(a, marker) {
			continue
		}
		raw := rawRequest(a)
		if raw == nil {
			continue
		}
		arm.Calls++
		for k, v := range normaliseRequest(raw) {
			if arm.Params[k] == nil {
				arm.Params[k] = map[string]int{}
			}
			arm.Params[k][v]++
		}
		if h, _ := a["trace.metadata.openrouter.provider_name"].(string); h != "" {
			arm.Hosts[h]++
		}
		if fr, _ := a["gen_ai.response.finish_reason"].(string); fr == "length" {
			arm.LengthFinishes++
		}
	}
	if arm.Calls == 0 {
		return nil, fmt.Errorf("no provider traces matched %s between %s and %s (Broadcast off, or the traces aged out)",
			model, start.Format(time.RFC3339), end.Format(time.RFC3339))
	}
	return arm, nil
}

// promptNamesArm reports whether the call's task prompt — the messages before
// the first assistant turn — names one of this arm's eval workdirs
// (<tmp>/ailang_eval/<bench>_<model>_<date>_<time>_<pid>). Only the prompt
// counts: the workdirs share one parent, and an agent that lists it carries
// every other run's directory name into its tool results (seen 2026-10-16:
// motoko `ls` output naming pi's workdirs).
func promptNamesArm(attrs map[string]any, marker string) bool {
	in, _ := attrs["span.input"].(string)
	if !strings.Contains(in, marker) {
		return false
	}
	var v struct {
		Messages []struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
	}
	if json.Unmarshal([]byte(in), &v) != nil {
		return false
	}
	for _, m := range v.Messages {
		if m.Role == "assistant" {
			return false
		}
		c := string(m.Content)
		if strings.Contains(c, "ailang_eval/") && strings.Contains(c, marker) {
			return true
		}
	}
	return false
}

// fetchWindow pages through [start, end) in chunks, halving a chunk whose
// result fills the page so a busy minute is never silently truncated.
func fetchWindow(ctx context.Context, fetch SpanFetcher, apiName string, start, end time.Time) ([]map[string]any, error) {
	var out []map[string]any
	for s := start; s.Before(end); {
		e := s.Add(spanChunk)
		if e.After(end) {
			e = end
		}
		got, err := fetchChunk(ctx, fetch, apiName, s, e)
		if err != nil {
			return nil, err
		}
		out = append(out, got...)
		s = e
	}
	return out, nil
}

func fetchChunk(ctx context.Context, fetch SpanFetcher, apiName string, s, e time.Time) ([]map[string]any, error) {
	got, err := fetch(ctx, apiName, s, e, spanPageLimit)
	if err != nil {
		return nil, fmt.Errorf("fetching traces %s..%s: %w", s.Format(time.RFC3339), e.Format(time.RFC3339), err)
	}
	if len(got) < spanPageLimit {
		return got, nil
	}
	if e.Sub(s) <= minSpanChunk {
		return nil, fmt.Errorf("more than %d traces in %s from %s; cannot page further", spanPageLimit, e.Sub(s), s.Format(time.RFC3339))
	}
	mid := s.Add(e.Sub(s) / 2)
	a, err := fetchChunk(ctx, fetch, apiName, s, mid)
	if err != nil {
		return nil, err
	}
	b, err := fetchChunk(ctx, fetch, apiName, mid, e)
	if err != nil {
		return nil, err
	}
	return append(a, b...), nil
}

// rawRequest extracts the request body OpenRouter recorded for the call.
func rawRequest(attrs map[string]any) map[string]any {
	out, _ := attrs["span.output"].(string)
	if out == "" {
		return nil
	}
	var v struct {
		RawRequest map[string]any `json:"rawRequest"`
	}
	if json.Unmarshal([]byte(out), &v) != nil {
		return nil
	}
	return v.RawRequest
}

// normaliseRequest flattens a request to comparable parameter strings.
// max_tokens and max_completion_tokens are one knob under two names, and an
// absent knob is recorded as "(unset)" so "one arm sets it" is a difference.
func normaliseRequest(raw map[string]any) map[string]string {
	p := map[string]string{}
	for k, v := range raw {
		if transportKeys[k] {
			continue
		}
		if k == "max_tokens" || k == "max_completion_tokens" {
			k = "output_cap"
		}
		b, _ := json.Marshal(v) // encoding/json sorts map keys: canonical
		p[k] = string(b)
	}
	for _, k := range []string{"output_cap", "reasoning", "provider", "temperature", "top_p"} {
		if _, ok := p[k]; !ok {
			p[k] = "(unset)"
		}
	}
	return p
}

// valueSet lists the distinct values of a parameter. A numeric parameter with
// many values (pi sizes its output cap to the context left, so it differs on
// every call) is collapsed to its range so the report stays readable; the
// range still differs from any single fixed value.
func valueSet(m map[string]int) []string {
	if len(m) == 0 {
		return []string{"(unset)"}
	}
	keys := make([]string, 0, len(m))
	var nums []float64
	for k := range m {
		keys = append(keys, k)
		if f, err := strconv.ParseFloat(k, 64); err == nil {
			nums = append(nums, f)
		}
	}
	sort.Strings(keys)
	if len(keys) > 3 && len(nums) == len(keys) {
		sort.Float64s(nums)
		return []string{fmt.Sprintf("%g..%g (%d values)", nums[0], nums[len(nums)-1], len(nums))}
	}
	return keys
}

func majorityHost(h map[string]int) string {
	best, n := "", -1
	for _, k := range sortedKeys(h) {
		if h[k] > n {
			best, n = k, h[k]
		}
	}
	return best
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
