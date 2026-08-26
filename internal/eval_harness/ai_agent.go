package eval_harness

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
)

// AIAgent generates code using LLM APIs.
// Uses the unified internal/ai/ providers via providerAdapter.
type AIAgent struct {
	friendlyName string           // Friendly name (e.g., "claude-sonnet-4-5") - used for cost lookups
	model        string           // API model name (e.g., "claude-sonnet-4-5-20250929") - used for API calls
	adapter      *providerAdapter // Unified provider adapter
	seed         int64
	attribution  *ai.Attribution // OpenRouter app-attribution overrides (nil = use defaults)
}

// NewAIAgent creates a new AI agent using unified providers.
func NewAIAgent(model string, seed int64) (*AIAgent, error) {
	// Resolve model name to API name and provider
	apiName, provider, err := ResolveModelName(model)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve model: %w", err)
	}

	// Get API key for provider
	apiKey, err := getAPIKeyForProvider(provider, model)
	if err != nil {
		return nil, err
	}

	// Create unified provider adapter. Pass the explicit provider from models.yml
	// so api_names without provider-identifying prefixes (e.g. "gemma4:26b") route correctly.
	adapter, err := newProviderAdapter(apiName, apiKey, ai.ProviderFromString(provider))
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Wire max_output_tokens from models.yml. Reasoning models (Gemini 3.x,
	// GPT-5, Claude 4.x thinking) need this to avoid burning the whole 4K
	// default budget on hidden thoughts and returning empty content.
	if GlobalModelsConfig != nil {
		if cfg, lookupErr := GlobalModelsConfig.GetModel(model); lookupErr == nil {
			if cfg.MaxOutputTokens > 0 {
				adapter.setMaxTokens(cfg.MaxOutputTokens)
			}
			if cfg.ReasoningMaxTokens > 0 {
				adapter.setReasoningMaxTokens(cfg.ReasoningMaxTokens)
			}
			if cfg.ReasoningEffort != "" {
				adapter.setReasoningEffort(cfg.ReasoningEffort)
			}
		}
	}

	return &AIAgent{
		friendlyName: model,
		model:        apiName,
		adapter:      adapter,
		seed:         seed,
		attribution:  nil,
	}, nil
}

// SetExpectedCalls declares how many generation calls this run will make
// against the model (M-ANTHROPIC-CACHE-HIT-RATE M2).
//
// Pass 1 for genuinely one-shot work to opt OUT of prompt caching — a cache
// write that is never read costs ~1.25x for nothing. Leave it unset for suite
// runs: Anthropic's cache is server-side and shared across agent instances, so
// a per-agent call count says nothing about whether the entry gets reused.
func (a *AIAgent) SetExpectedCalls(n int) {
	a.adapter.setExpectedCalls(n)
}

// WithAttribution sets OpenRouter app-attribution overrides for this agent.
func (a *AIAgent) WithAttribution(attr *ai.Attribution) *AIAgent {
	a.attribution = attr
	return a
}

// GenerateCode generates code using the unified provider.
//
// The whole prompt is treated as volatile, so nothing is cached. Prefer
// GenerateCodeSplit when the prompt has a stable teaching-prompt prefix.
func (a *AIAgent) GenerateCode(ctx context.Context, prompt string) (*GenerateResult, error) {
	return checkReasoningStall(a.adapter.generate(ctx, "", prompt))
}

// ErrReasoningStall is returned when the model spent its output budget thinking
// and returned no code. See ErrorCategoryReasoningStall for the measurement.
//
// It is an ERROR rather than an empty-code success on purpose: the previous
// behaviour handed empty code to the compiler, which banked the run as
// compile_error and attributed a harness/provider event to the model's ability
// to write AILANG.
var ErrReasoningStall = errors.New("reasoning stall: model returned reasoning but no content")

// checkReasoningStall converts the stall signature into a typed error.
//
// Deliberately wrapped around GenerateCode and GenerateCodeSplit INDIVIDUALLY
// rather than inside adapter.generate, because GenerateCodeWarmup caps output at
// one token by design — every warm-up call matches the stall signature and must
// not be reported as one.
func checkReasoningStall(res *GenerateResult, err error) (*GenerateResult, error) {
	if err != nil || res == nil {
		return res, err
	}
	if IsReasoningStall(res.Code, res.OutputTokens, res.ReasonTokens) {
		return res, fmt.Errorf("%w (reasoning_tokens=%d output_tokens=%d finish_reason=%q)",
			ErrReasoningStall, res.ReasonTokens, res.OutputTokens, res.FinishReason)
	}
	return res, nil
}

// GenerateCodeSplit generates code from a prompt already split into a stable
// cacheable prefix and a per-benchmark task (M-ANTHROPIC-CACHE-HIT-RATE M2).
//
// The model sees cachedPrefix+task, identical to passing the joined string to
// GenerateCode. On Anthropic the split additionally lets a cache breakpoint sit
// at the boundary, so repeat calls in a suite re-read the teaching prompt at
// ~10% of input price instead of re-paying it in full.
func (a *AIAgent) GenerateCodeSplit(ctx context.Context, cachedPrefix, task string) (*GenerateResult, error) {
	return checkReasoningStall(a.adapter.generate(ctx, cachedPrefix, task))
}

// GenerateCodeWarmup issues a deliberately tiny call whose only purpose is to
// make the provider prefill — and therefore cache — cachedPrefix
// (M-ANTHROPIC-CACHE-HIT-RATE, D4).
//
// maxTokens caps the output; 1 is enough, because the prefill that writes the
// cache happens regardless of how much the model is then allowed to say. The
// result is normally discarded — it is returned so a caller can assert on
// cache-creation tokens to verify the warm-up actually landed.
//
// The adapter's previous budget is restored on return so a warmed agent is not
// left crippled for real work.
func (a *AIAgent) GenerateCodeWarmup(ctx context.Context, cachedPrefix, task string, maxTokens int) (*GenerateResult, error) {
	prev := a.adapter.maxTokens
	a.adapter.setMaxTokens(maxTokens)
	defer a.adapter.setMaxTokens(prev)
	return a.adapter.generate(ctx, cachedPrefix, task)
}

// GenerateResult contains the result of code generation
type GenerateResult struct {
	Code         string
	InputTokens  int // Prompt tokens (input to LLM)
	OutputTokens int // Completion tokens (generated code, reasoning excluded)
	ReasonTokens int // Hidden reasoning/thinking tokens (billed as output)
	// Prompt-cache activity, when the provider reports it (Anthropic, OpenAI,
	// Gemini). Zero means "not reported", not "no cache".
	CacheReadInputTokens     int
	CacheCreationInputTokens int
	TotalTokens              int    // Total tokens (for billing; includes reasoning)
	FinishReason             string // Normalized stop reason ("stop", "length", ...); "" if unreported
	Model                    string
}

// RetryConfig configures retry behavior
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
}

// GenerateWithRetry generates code with retry logic
func (a *AIAgent) GenerateWithRetry(ctx context.Context, prompt string, cfg RetryConfig) (*GenerateResult, error) {
	var lastErr error
	delay := cfg.BaseDelay

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait with exponential backoff
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			delay *= 2
		}

		result, err := a.GenerateCode(ctx, prompt)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// AC8 (M-OLLAMA-CLOUD). Quota exhaustion must be checked BEFORE the 429
	// rule below, because it ARRIVES AS A 429 and would otherwise be retried.
	// An Ollama Cloud session limit does not clear until the 5-hour window
	// rolls, and a weekly limit takes days — so retrying is not merely
	// unhelpful, it burns the remaining run against a bucket that cannot
	// recover. Shares one definition with the error categoriser so the two
	// cannot disagree about the same error.
	if isQuotaExhaustion(strings.ToLower(errStr)) {
		return false
	}

	// Rate limiting errors — transient, genuinely worth retrying.
	if strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "429") {
		return true
	}

	// Temporary network errors
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection") {
		return true
	}

	// Server errors
	if strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") {
		return true
	}

	return false
}

// MockAIAgent is a mock implementation for testing
type MockAIAgent struct {
	model string
	code  string
}

// NewMockAIAgent creates a mock AI agent
func NewMockAIAgent(model, code string) *MockAIAgent {
	return &MockAIAgent{
		model: model,
		code:  code,
	}
}

// GenerateCode returns the pre-configured mock code
func (m *MockAIAgent) GenerateCode(ctx context.Context, prompt string) (*GenerateResult, error) {
	outputTokens := len(m.code) / 4 // Rough estimate: ~4 chars per token
	inputTokens := len(prompt) / 4  // Rough estimate: ~4 chars per token
	return &GenerateResult{
		Code:         m.code,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		Model:        m.model,
	}, nil
}

// SetCorrelation attaches OpenRouter Broadcast correlation identifiers so the
// trace OpenRouter pushes for each request can be joined back to the eval run
// that caused it (M-OPENROUTER-BROADCAST-INGEST M3).
//
// Nil, or a run with no chain, leaves every request wire-identical to before.
func (a *AIAgent) SetCorrelation(chainID, benchmarkID, tier string) {
	if chainID == "" && benchmarkID == "" {
		return
	}
	trace := map[string]any{}
	if benchmarkID != "" {
		trace["trace_name"] = "eval:" + benchmarkID
		trace["benchmark"] = benchmarkID
	}
	if tier != "" {
		trace["tier"] = tier
	}
	a.adapter.setCorrelation(&ai.Correlation{
		SessionID: chainID,
		Trace:     trace,
	})
}
