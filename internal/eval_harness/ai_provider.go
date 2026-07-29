// Package eval_harness provides AI code generation benchmarking.
// This file provides the adapter layer between unified ai.Provider and eval harness.
package eval_harness

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/anthropic"
	"github.com/sunholo-data/ailang/internal/ai/gemini"
	"github.com/sunholo-data/ailang/internal/ai/ollama"
	"github.com/sunholo-data/ailang/internal/ai/openai"
	"github.com/sunholo-data/ailang/internal/ai/openrouter"
)

// providerAdapter wraps ai.Provider for eval harness use.
type providerAdapter struct {
	provider ai.Provider
	// providerType is the RESOLVED provider (models.yml is the source of truth,
	// falling back to name inference). Retained so cache decisions can be made
	// without re-guessing from the model string.
	providerType ai.ProviderType
	// expectedCalls is 1 when the caller knows this is a one-shot run, and 0
	// ("undeclared") otherwise. Only 1 disables caching — see cacheBreakpointsFor
	// for why an adapter-local call count is the wrong thing to gate on.
	expectedCalls      int
	model              string
	maxTokens          int             // Max output tokens; 0 means use defaultMaxTokens
	reasoningMaxTokens int             // Cap on hidden thinking tokens; 0 = uncapped
	reasoningEffort    string          // Vendor effort dial ("low"|"medium"|"high"); "" = vendor default
	attribution        *ai.Attribution // OpenRouter app-attribution overrides
}

// defaultMaxTokens is the fallback output budget when a model has no
// max_output_tokens set in models.yml. Pre-reasoning models tolerate this fine;
// reasoning models (Gemini 3.x, GPT-5, Claude 4.x thinking) need more headroom
// because thoughtsTokenCount/reasoning_tokens consume part of the budget
// invisibly. Models that need more should set max_output_tokens in models.yml.
const defaultMaxTokens = 4096

// newProviderAdapter creates a provider adapter for the given model.
// If explicitProvider is non-empty, it overrides name-based inference — this
// honors the provider field set in models.yml (the source of truth) so that
// api_name strings without provider-identifying prefixes (e.g. "gemma4:26b",
// "codellama:7b") still route correctly.
func newProviderAdapter(model string, apiKey string, explicitProvider ai.ProviderType) (*providerAdapter, error) {
	providerType := explicitProvider
	if providerType == "" {
		providerType = ai.GuessProvider(model)
	}

	var provider ai.Provider
	switch providerType {
	case ai.ProviderOpenAI:
		provider = openai.NewClient(apiKey)
	case ai.ProviderAnthropic:
		provider = anthropic.NewClient(apiKey)
	case ai.ProviderGoogle:
		// Gemini uses ADC (Application Default Credentials) via Vertex AI
		client, err := gemini.NewVertexAIClient("")
		if err != nil {
			return nil, fmt.Errorf("failed to create Gemini client: %w", err)
		}
		provider = client
	case ai.ProviderOllama:
		// Ollama is local, strip the "ollama:" prefix for the model name
		client, err := ollama.NewClient()
		if err != nil {
			return nil, fmt.Errorf("failed to create Ollama client: %w", err)
		}
		// Strip ollama: prefix if present
		model = strings.TrimPrefix(model, "ollama:")
		provider = client
	case ai.ProviderOpenRouter:
		// Strip optional explicit "openrouter:" prefix; the model name itself
		// is "vendor/model" (e.g., "anthropic/claude-sonnet-4.5").
		model = strings.TrimPrefix(model, "openrouter:")
		provider = openrouter.NewClient(apiKey)
	default:
		return nil, fmt.Errorf("unsupported provider for model: %s", model)
	}

	return &providerAdapter{
		provider:     provider,
		providerType: providerType,
		model:        model,
		maxTokens:    0, // 0 => defaultMaxTokens; set via setMaxTokens
		attribution:  nil,
	}, nil
}

// setExpectedCalls records how many generate() calls this adapter will receive
// in the current run, so caching can be skipped when a write could never be
// read back. See cacheBreakpointsFor.
func (p *providerAdapter) setExpectedCalls(n int) {
	p.expectedCalls = n
}

// setAttribution sets OpenRouter app-attribution overrides. Retained for the
// app-attribution-override feature (cc726339); the caller that wires it is not
// yet in place, so it is intentionally unused for now.
//
//nolint:unused // setter awaiting its caller (attribution-override feature)
func (p *providerAdapter) setAttribution(attr *ai.Attribution) {
	p.attribution = attr
}

// setMaxTokens overrides the per-request output budget.
// Pass 0 to fall back to defaultMaxTokens.
func (p *providerAdapter) setMaxTokens(n int) {
	p.maxTokens = n
}

// setReasoningMaxTokens caps hidden thinking tokens for reasoning models
// (models.yml reasoning_max_tokens). 0 = provider default / uncapped.
// Currently honored by the OpenRouter adapter only.
func (p *providerAdapter) setReasoningMaxTokens(n int) {
	p.reasoningMaxTokens = n
}

// setReasoningEffort sets the vendor effort dial (models.yml reasoning_effort).
// "" = vendor default. Currently honored by the OpenRouter adapter only.
func (p *providerAdapter) setReasoningEffort(e string) {
	p.reasoningEffort = e
}

// cacheBreakpointsFor decides whether this request declares a prompt-cache
// breakpoint (M-ANTHROPIC-CACHE-HIT-RATE M2, design decision D3).
//
// ON by default rather than behind a flag: opt-in is precisely why our hit rate
// was near zero — the v0.18.4 mechanism shipped and no Go caller ever set it.
//
// Two guards keep it from costing more than it saves:
//   - Only Anthropic. Other providers no-op on hints (OpenAI auto-caches,
//     Gemini needs a different API), and declaring one there just emits a
//     "hint ignored" warning per session for nothing.
//   - Only when the caller has told us this is a genuinely single-shot run.
//     A cache WRITE costs ~1.25x, so a prefix read zero times is a small pure
//     loss; break-even at 5-minute TTL is two requests.
//
// The default when nothing is declared (expectedCalls == 0) is to CACHE.
//
// That default is deliberate and was corrected during implementation. The
// obvious-looking rule — "only cache when this adapter will see >= 2 calls" —
// is wrong here, because Anthropic's cache is SERVER-SIDE and keyed by prompt
// prefix, not by client object. The eval harness constructs a fresh AIAgent per
// benchmark (cmd/ailang/eval_benchmark.go, inside runSingleBenchmark), so every
// adapter sees exactly one generation call, and a >= 2 rule would have disabled
// caching for the entire suite while looking correct. Benchmark N+1 reads the
// entry written by benchmark N precisely because the cache does not live in the
// client. Callers that really are one-shot opt out via SetExpectedCalls(1).
func cacheBreakpointsFor(providerType ai.ProviderType, cachedPrefix string, expectedCalls int) []ai.CacheBreakpoint {
	if providerType != ai.ProviderAnthropic || cachedPrefix == "" {
		return nil
	}
	if expectedCalls == 1 {
		return nil
	}
	return []ai.CacheBreakpoint{{Position: "user_prefix", TTL: "ephemeral"}}
}

// generate calls the unified provider and converts to GenerateResult.
//
// cachedPrefix is the stable leading chunk of the prompt (the teaching prompt);
// prompt is the volatile remainder (the benchmark task). The model always sees
// cachedPrefix+prompt — passing "" for cachedPrefix is exactly the pre-v0.31.0
// behavior.
func (p *providerAdapter) generate(ctx context.Context, cachedPrefix, prompt string) (*GenerateResult, error) {
	systemPrompt := "You are a code generation engine. Output ONLY a complete, runnable program that solves the given task. " +
		"Do NOT output explanations, markdown formatting, or conversational text. " +
		"Do NOT output placeholder or stub code — implement the full solution. " +
		"Output raw code only, no code fences."

	maxTokens := p.maxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}

	req := &ai.Request{
		Model:        p.model,
		SystemPrompt: systemPrompt,
		CachedPrefix: cachedPrefix,
		UserPrompt:   prompt,
		MaxTokens:    maxTokens,
		Attribution:  p.attribution,
		// Note the breakpoint targets the USER prefix, not the system prompt:
		// systemPrompt above is ~70 tokens, far below every model's minimum
		// cacheable prefix, so a system breakpoint here would be a silent no-op.
		// The ~16K-token teaching prompt is what is worth caching, and it lives
		// in the user turn.
		CacheBreakpoints: cacheBreakpointsFor(p.providerType, cachedPrefix, p.expectedCalls),
	}
	if p.reasoningMaxTokens > 0 || p.reasoningEffort != "" {
		req.Options = map[string]any{}
		if p.reasoningMaxTokens > 0 {
			req.Options["reasoning_max_tokens"] = p.reasoningMaxTokens
		}
		if p.reasoningEffort != "" {
			req.Options["reasoning_effort"] = p.reasoningEffort
		}
	}

	resp, err := p.provider.Generate(ctx, req)
	if err != nil {
		return nil, err
	}

	return &GenerateResult{
		Code:         extractCodeFromMarkdown(resp.Text),
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
		ReasonTokens: resp.ReasonTokens,
		TotalTokens:  resp.TotalTokens,
		FinishReason: resp.FinishReason,
		Model:        resp.Model,
	}, nil
}

// getAPIKeyForProvider returns the API key for the given provider.
// Uses ai.EnvVarForProvider as the single source of truth for provider →
// env-var mapping.
func getAPIKeyForProvider(provider string, model string) (string, error) {
	providerType := ai.ProviderFromString(provider)
	switch providerType {
	case ai.ProviderGoogle:
		// Google uses ADC, no API key required
		return "", nil
	case ai.ProviderOllama:
		// Ollama is local, no API key required
		return "", nil
	}

	envVar := ai.EnvVarForProvider(providerType)
	if envVar == "" {
		return "", fmt.Errorf("unsupported provider: %s (model: %s)", provider, model)
	}

	key := os.Getenv(envVar)
	if key == "" {
		return "", fmt.Errorf("%s environment variable not set (required for model: %s)", envVar, model)
	}
	return key, nil
}

// extractCodeFromMarkdown strips markdown code fences if present.
// This is eval-specific logic that doesn't belong in the unified ai package.
func extractCodeFromMarkdown(text string) string {
	// Trim leading/trailing whitespace first
	text = strings.TrimSpace(text)
	lines := []byte(text)

	// Check if starts with ``` (after trimming)
	if len(lines) > 3 && lines[0] == '`' && lines[1] == '`' && lines[2] == '`' {
		// Find first newline (end of opening fence)
		start := 0
		for i, b := range lines {
			if b == '\n' {
				start = i + 1
				break
			}
		}

		// Find last ``` working backwards
		end := len(lines)
		for i := len(lines) - 1; i >= 2; i-- {
			if lines[i] == '`' && lines[i-1] == '`' && lines[i-2] == '`' {
				// Check if this is at start of line or has newline before it
				end = i - 2
				// Trim trailing newline before closing fence
				if end > 0 && lines[end-1] == '\n' {
					end--
				}
				break
			}
		}

		if start < end {
			return string(lines[start:end])
		}
	}

	return text
}
