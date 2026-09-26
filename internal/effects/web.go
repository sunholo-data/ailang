package effects

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/eval"
)

// std/web — web search and page fetch as a program capability whose ONLY
// visible effect is {Net}. M-DANEEL-AILANG-EXECUTOR M2.
//
// The API key is read here, in Go, from OLLAMA_API_KEY: a program never holds
// it, never passes it, and never sees it in a value, an error or a trace. That
// is the lending boundary an ailang_only agent needs — the operator grants
// {Net} + net_allow ["ollama.com"], and the program can search but cannot
// name a key or a host. The endpoints are FIXED: webFetch(url) fetches through
// the backend's fetch API, never from the supplied URL directly, so a caller
// cannot smuggle an arbitrary host past the allowlist.
//
// Every request still goes through buildSecureRequest: the policy's
// net_allow, the DNS/IP checks and the response size limit apply unchanged.
// The backend is a seam (webBackend) so a second provider can sit behind the
// same typed contract without touching the builtins or std/web.

// WebSearchMaxResults bounds `max`: Ollama's API caps at 10; a larger value
// is a programmer error, refused before any request is made.
const WebSearchMaxResults = 10

// webBackend is one provider of search + fetch. Implementations build the
// request for a fixed endpoint and decode the provider's response shape.
type webBackend interface {
	name() string
	searchRequest(query string, max int) (url string, payload []byte)
	decodeSearch(body []byte) ([]webSearchResult, error)
	fetchRequest(pageURL string) (url string, payload []byte)
	decodeFetch(body []byte) (webFetchResult, error)
	apiKey() string
}

type webSearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

type webFetchResult struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Links   []string `json:"links"`
}

// ollamaWebBaseURL is the fixed endpoint base. Tests point it at an httptest
// server; nothing else changes it — there is deliberately no env var.
var ollamaWebBaseURL = "https://ollama.com"

// WebCredentialVars names the environment variables std/web reads in Go when
// a policy's net_allow admits the backend's host — nil otherwise. A
// restricted worker's environment is an allowlist, so the supervisor asks
// here rather than knowing the backend: one credential, for one granted
// host (M-EXECUTOR-POLICY-HARDENING M7 posture). An empty net_allow admits
// nothing here, unlike isAllowedDomain's "no list = any host".
func WebCredentialVars(netAllow []string) []string {
	if len(netAllow) == 0 {
		return nil
	}
	u, err := url.Parse(ollamaWebBaseURL)
	if err != nil || !isAllowedDomain(u.Hostname(), netAllow) {
		return nil
	}
	return []string{config.EnvOllamaAPIKey}
}

type ollamaWebBackend struct{}

func (ollamaWebBackend) name() string   { return "ollama" }
func (ollamaWebBackend) apiKey() string { return config.OllamaAPIKey() }

func (ollamaWebBackend) searchRequest(query string, max int) (string, []byte) {
	payload, _ := json.Marshal(map[string]any{"query": query, "max_results": max})
	return ollamaWebBaseURL + "/api/web_search", payload
}

func (ollamaWebBackend) decodeSearch(body []byte) ([]webSearchResult, error) {
	var out struct {
		Results []webSearchResult `json:"results"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("malformed search response: %v", err)
	}
	if out.Results == nil {
		return nil, fmt.Errorf("malformed search response: no results field")
	}
	return out.Results, nil
}

func (ollamaWebBackend) fetchRequest(pageURL string) (string, []byte) {
	payload, _ := json.Marshal(map[string]any{"url": pageURL})
	return ollamaWebBaseURL + "/api/web_fetch", payload
}

func (ollamaWebBackend) decodeFetch(body []byte) (webFetchResult, error) {
	var out webFetchResult
	if err := json.Unmarshal(body, &out); err != nil {
		return out, fmt.Errorf("malformed fetch response: %v", err)
	}
	if out.Links == nil {
		out.Links = []string{}
	}
	return out, nil
}

// activeWebBackend is the provider std/web talks to. One implementation
// today; a second (e.g. Gemini grounding) is a follow-up behind this seam.
var activeWebBackend webBackend = ollamaWebBackend{}

// WebSearch implements std/web.webSearch(query, max).
func WebSearch(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if !ctx.HasCap("Net") {
		return nil, NewCapabilityError("Net")
	}
	if len(args) != 2 {
		return nil, fmt.Errorf("E_WEB_TYPE_ERROR: webSearch: expected 2 arguments (query, max), got %d", len(args))
	}
	query, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_WEB_TYPE_ERROR: webSearch: query must be String, got %T", args[0])
	}
	max, ok := args[1].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("E_WEB_TYPE_ERROR: webSearch: max must be Int, got %T", args[1])
	}
	if query.Value == "" {
		return nil, fmt.Errorf("E_WEB_INVALID_INPUT: webSearch: query must not be empty")
	}
	if max.Value < 1 || max.Value > WebSearchMaxResults {
		return nil, fmt.Errorf("E_WEB_INVALID_INPUT: webSearch: max must be 1..%d, got %d", WebSearchMaxResults, max.Value)
	}
	b := activeWebBackend
	url, payload := b.searchRequest(query.Value, max.Value)
	body, errVal := webPost(ctx, b, "webSearch", url, payload)
	if errVal != nil {
		return errVal, nil
	}
	results, err := b.decodeSearch(body)
	if err != nil {
		return makeResultErr("Transport", "webSearch: "+err.Error()), nil
	}
	elems := make([]eval.Value, 0, len(results))
	for _, r := range results {
		elems = append(elems, &eval.RecordValue{Fields: map[string]eval.Value{
			"title":   &eval.StringValue{Value: r.Title},
			"url":     &eval.StringValue{Value: r.URL},
			"content": &eval.StringValue{Value: r.Content},
		}})
	}
	return makeResultOk(&eval.ListValue{Elements: elems}), nil
}

// WebFetch implements std/web.webFetch(url).
func WebFetch(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if !ctx.HasCap("Net") {
		return nil, NewCapabilityError("Net")
	}
	if len(args) != 1 {
		return nil, fmt.Errorf("E_WEB_TYPE_ERROR: webFetch: expected 1 argument (url), got %d", len(args))
	}
	pageURL, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_WEB_TYPE_ERROR: webFetch: url must be String, got %T", args[0])
	}
	if pageURL.Value == "" {
		return nil, fmt.Errorf("E_WEB_INVALID_INPUT: webFetch: url must not be empty")
	}
	b := activeWebBackend
	url, payload := b.fetchRequest(pageURL.Value)
	body, errVal := webPost(ctx, b, "webFetch", url, payload)
	if errVal != nil {
		return errVal, nil
	}
	page, err := b.decodeFetch(body)
	if err != nil {
		return makeResultErr("Transport", "webFetch: "+err.Error()), nil
	}
	links := make([]eval.Value, 0, len(page.Links))
	for _, l := range page.Links {
		links = append(links, &eval.StringValue{Value: l})
	}
	return makeResultOk(&eval.RecordValue{Fields: map[string]eval.Value{
		"title":   &eval.StringValue{Value: page.Title},
		"content": &eval.StringValue{Value: page.Content},
		"links":   &eval.ListValue{Elements: links},
	}}), nil
}

// webPost sends one JSON request to a fixed backend endpoint through the Net
// security path and returns the raw body, or a Result Err value the caller
// returns as-is. The key goes into the Authorization header here and nowhere
// else; error text never includes it.
func webPost(ctx *EffContext, b webBackend, op, url string, payload []byte) ([]byte, eval.Value) {
	key := b.apiKey()
	if key == "" {
		return nil, makeResultErr("Transport", fmt.Sprintf("%s: %s is not set for the %s backend", op, config.EnvOllamaAPIKey, b.name()))
	}
	headers := &eval.ListValue{Elements: []eval.Value{
		&eval.RecordValue{Fields: map[string]eval.Value{"name": &eval.StringValue{Value: "Authorization"}, "value": &eval.StringValue{Value: "Bearer " + key}}},
		&eval.RecordValue{Fields: map[string]eval.Value{"name": &eval.StringValue{Value: "Content-Type"}, "value": &eval.StringValue{Value: "application/json"}}},
	}}
	client, req, errVal := buildSecureRequest(ctx, "POST", url, headers, bytes.NewReader(payload))
	if errVal != nil {
		return nil, errVal
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, makeResultErr("Transport", op+": "+transportMessage(err))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, ctx.Net.MaxBytes))
	if err != nil {
		return nil, makeResultErr("Transport", fmt.Sprintf("%s: failed to read response: %v", op, err))
	}
	if int64(len(body)) >= ctx.Net.MaxBytes {
		return nil, makeResultErr("BodyTooLarge", fmt.Sprintf("%d", ctx.Net.MaxBytes))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, makeResultErr("Transport", fmt.Sprintf("%s: %s backend returned HTTP %d", op, b.name(), resp.StatusCode))
	}
	return body, nil
}

// makeResultOk wraps a value in Result's Ok constructor.
func makeResultOk(v eval.Value) eval.Value {
	return &eval.TaggedValue{ModulePath: "std/result", TypeName: "Result", CtorName: "Ok", Fields: []eval.Value{v}}
}
