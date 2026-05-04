package configdriven

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var configDrivenTracer = telemetry.Tracer("ai.configdriven")

// Provider is the generic ai.Provider implementation driven by an
// [[ai_provider]] block in a package's ailang.toml manifest.
//
// One Provider instance corresponds to one [[ai_provider]] block. The
// runtime registers many of these — one per declared block across all
// installed packages — alongside the hardcoded built-in providers
// (openai, anthropic, gemini, ollama, openrouter).
//
// Per the [[ai_provider]] design (see design_docs/planned/v0_15_0/m-ai-provider-config.md):
//   - AIRoutingPolicy on the request is rejected (D11; non-OpenRouter providers
//     must reject non-zero policies — silent routing fallback is forbidden)
//   - Image generation requests are rejected unless capabilities.vision is true
//   - Capability-mismatched requests fail fast with CapabilityNotSupported
//   - Trace spans match the structure of built-in provider spans (so
//     replay/observability tooling treats config-driven and built-in
//     providers identically)
type Provider struct {
	spec       *pkg.AIProviderSpec
	httpClient *http.Client
}

// Option configures a Provider at construction time.
type Option func(*Provider)

// WithHTTPClient overrides the default HTTP client. Used by tests with
// httptest.Server-backed clients.
func WithHTTPClient(c *http.Client) Option {
	return func(p *Provider) { p.httpClient = c }
}

// New creates a Provider for the given [[ai_provider]] spec. The spec is
// retained by reference; callers must not mutate it after construction.
func New(spec *pkg.AIProviderSpec, opts ...Option) *Provider {
	p := &Provider{
		spec:       spec,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Name implements ai.Provider — returns the provider name from the config
// (e.g. "vllm" for an [[ai_provider]] with name = "vllm").
func (p *Provider) Name() string { return p.spec.Name }

// Spec returns the provider config that drives this Provider's behaviour.
// Consumed by the streaming dispatch layer (internal/effects/ai_streaming.go)
// which needs the streaming sub-block, request shape, and auth fields to
// construct an SSE-POST request. Returned by reference; callers must NOT
// mutate it.
func (p *Provider) Spec() *pkg.AIProviderSpec { return p.spec }

// Generate implements ai.Provider.
func (p *Provider) Generate(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	// D11: config-driven providers reject AIRoutingPolicy. Routing belongs
	// to OpenRouter; a static-config provider has no fallback chain to
	// honour the policy with.
	if req.Routing != nil && req.Routing.HasRouting() {
		return nil, ai.NewProviderError(p.spec.Name, 0,
			"this provider does not support AIRoutingPolicy; use openrouter or a built-in routing-capable provider",
			ai.ErrRoutingNotSupported)
	}

	// Image generation gate: refuse unless capabilities.vision declares it.
	// (Vision in v1 schema covers both input and output of images; richer
	// flags can split that in v2 if needed.)
	if ai.RequestsImage(req) && !p.spec.Capabilities.Vision {
		return nil, ai.NewProviderError(p.spec.Name, 0,
			fmt.Sprintf("image generation not supported by config-driven provider %q (capabilities.vision = false)", p.spec.Name),
			nil)
	}

	// Structured output gate: refuse JSON-mode requests if the provider
	// flags it as unsupported.
	if req.ResponseFormat == "json" && !p.spec.Capabilities.JSONMode && !p.spec.Capabilities.StructuredOutputs {
		return nil, ai.NewProviderError(p.spec.Name, 0,
			fmt.Sprintf("JSON mode not supported by config-driven provider %q (capabilities.json_mode and capabilities.structured_outputs both false)", p.spec.Name),
			nil)
	}

	// Trace span — same attribute schema as built-in providers so replay
	// and dashboard tooling treats config-driven calls identically.
	ctx, span := telemetry.StartSpan(ctx, configDrivenTracer, "configdriven.generate",
		trace.WithAttributes(
			attribute.String("ai.provider", p.spec.Name),
			attribute.String("ai.provider_kind", "config_driven"),
			attribute.String("ai.model", req.Model),
			attribute.String("ai.request_shape", p.spec.RequestShape),
			attribute.String("ai.endpoint", safeEndpointForTrace(p.spec)),
			attribute.String("ai.prompt_preview", telemetry.Truncate(req.UserPrompt, 100)),
		),
	)
	defer span.End()

	// Models allow-list enforcement: if declared, requested model must be
	// in the list. Empty list means accept any model under the provider's
	// routing prefix.
	if len(p.spec.Models.Allowed) > 0 {
		allowed := false
		for _, m := range p.spec.Models.Allowed {
			if m == req.Model {
				allowed = true
				break
			}
		}
		if !allowed {
			err := ai.NewProviderError(p.spec.Name, 0,
				fmt.Sprintf("model %q is not in the allowed list for provider %q", req.Model, p.spec.Name),
				nil)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}

	// Build request body using the configured shape.
	body, err := buildRequestBody(p.spec.RequestShape, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, ai.NewProviderError(p.spec.Name, 0, err.Error(), err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.spec.Endpoint, bytes.NewReader(body))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, ai.NewProviderError(p.spec.Name, 0, "build HTTP request: "+err.Error(), err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	if err := applyAuth(httpReq, p.spec); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, ai.NewProviderError(p.spec.Name, 0, err.Error(), err)
	}

	httpResp, err := p.httpClient.Do(httpReq)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, ai.NewProviderError(p.spec.Name, 0, "HTTP request failed: "+err.Error(), err)
	}
	defer httpResp.Body.Close()

	respBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, ai.NewProviderError(p.spec.Name, httpResp.StatusCode, "read response: "+err.Error(), err)
	}

	// Non-2xx: extract error via error_path if declared, fall back to a
	// truncated response body. classifyHTTPError marks 5xx and 429 as
	// retryable per existing provider conventions.
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		errMsg := extractErrorMessage(respBytes, p.spec.ErrorPath)
		err := ai.NewProviderError(p.spec.Name, httpResp.StatusCode, errMsg, nil)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// 2xx: extract response text via response_path.
	var parsed any
	if jerr := json.Unmarshal(respBytes, &parsed); jerr != nil {
		err := ai.NewProviderError(p.spec.Name, httpResp.StatusCode,
			"response is not valid JSON: "+jerr.Error(), jerr)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	text, perr := extractString(parsed, p.spec.ResponsePath)
	if perr != nil {
		err := ai.NewProviderError(p.spec.Name, httpResp.StatusCode,
			fmt.Sprintf("response_path extraction failed: %s", perr.Error()), perr)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	resp := &ai.Response{
		Text:           text,
		Model:          req.Model,
		RequestedModel: req.Model,
	}

	// Best-effort token-usage extraction. Most OpenAI-compatible providers
	// return usage at $.usage.{prompt_tokens,completion_tokens,total_tokens};
	// Anthropic uses $.usage.{input_tokens,output_tokens}. We try both and
	// silently leave zeros if neither shape matches.
	extractUsage(parsed, resp)

	// Cost recording — declared per-token or per-call in the spec; computed
	// from token counts. Stored on the response as a string (provider
	// convention) so callers can pass it into the budget tracker uniformly.
	resp.CostUSD = computeCost(p.spec.Cost, resp.InputTokens, resp.OutputTokens)

	// Trace span: success metrics.
	span.SetAttributes(
		attribute.Int("ai.tokens_in", resp.InputTokens),
		attribute.Int("ai.tokens_out", resp.OutputTokens),
		attribute.Int("ai.tokens_total", resp.TotalTokens),
		attribute.String("ai.cost_usd", resp.CostUSD),
		attribute.String("ai.response_preview", telemetry.Truncate(resp.Text, 100)),
	)
	return resp, nil
}

// extractUsage tries OpenAI-style and Anthropic-style usage paths in turn,
// populating resp's token counts in-place. Best-effort — silent zero on miss.
func extractUsage(parsed any, resp *ai.Response) {
	// OpenAI shape: $.usage.{prompt_tokens, completion_tokens, total_tokens}
	if v, err := extractPath(parsed, "$.usage.prompt_tokens"); err == nil {
		resp.InputTokens = numberToInt(v)
	}
	if v, err := extractPath(parsed, "$.usage.completion_tokens"); err == nil {
		resp.OutputTokens = numberToInt(v)
	}
	if v, err := extractPath(parsed, "$.usage.total_tokens"); err == nil {
		resp.TotalTokens = numberToInt(v)
	}

	// Anthropic shape: $.usage.{input_tokens, output_tokens}
	if resp.InputTokens == 0 {
		if v, err := extractPath(parsed, "$.usage.input_tokens"); err == nil {
			resp.InputTokens = numberToInt(v)
		}
	}
	if resp.OutputTokens == 0 {
		if v, err := extractPath(parsed, "$.usage.output_tokens"); err == nil {
			resp.OutputTokens = numberToInt(v)
		}
	}
	if resp.TotalTokens == 0 {
		resp.TotalTokens = resp.InputTokens + resp.OutputTokens
	}
}

func numberToInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}

// computeCost returns the USD cost as a string with provider-conventional
// precision. Returns "" when no cost data is configured (not 0.0 — empty
// string lets downstream consumers distinguish "no cost data" from "free").
func computeCost(cost pkg.AIProviderCost, inputTok, outputTok int) string {
	hasPerToken := cost.InputPer1MUSD > 0 || cost.OutputPer1MUSD > 0
	hasPerCall := cost.PerCallUSD > 0
	if !hasPerToken && !hasPerCall {
		return ""
	}
	total := cost.PerCallUSD
	if hasPerToken {
		total += cost.InputPer1MUSD * float64(inputTok) / 1_000_000.0
		total += cost.OutputPer1MUSD * float64(outputTok) / 1_000_000.0
	}
	// 6 decimal places matches OpenRouter's reporting precision.
	return fmt.Sprintf("%.6f", total)
}

// extractErrorMessage pulls the error message from a non-2xx response body.
// Tries the configured error_path first; falls back to a truncated raw body
// if extraction fails. We never propagate the underlying parse error here
// because the user's primary problem is the upstream HTTP error, not our
// extraction issue.
func extractErrorMessage(body []byte, errorPath string) string {
	const truncate = 500

	// No error_path configured — return truncated body.
	if errorPath == "" {
		return truncateBody(body, truncate)
	}

	var parsed any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return truncateBody(body, truncate)
	}
	msg, err := extractString(parsed, errorPath)
	if err != nil || msg == "" {
		return truncateBody(body, truncate)
	}
	return msg
}

func truncateBody(body []byte, max int) string {
	s := strings.TrimSpace(string(body))
	if len(s) > max {
		return s[:max] + "...(truncated)"
	}
	return s
}
