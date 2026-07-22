package ai

import (
	"errors"
	"fmt"
)

// M-AI-REASONING-EFFORT (v0.31.0): cross-provider request-side reasoning control.
//
// This file is the ONE shared resolver that every provider request constructor
// (Generate/Step/StreamStep on openai/gemini/anthropic/openrouter) MUST invoke
// before JSON marshaling and before creating/dispatching an HTTP request. It
// validates the typed Request.ReasoningEffort field plus three legacy Options
// escape hatches with deterministic precedence, conflict, capability, and
// MaxTokens rules — and either produces an exact provider-honorable decision or
// returns a typed, non-retryable schema-validation error.
//
// No silent fallbacks: an explicit reasoning request is honored exactly or the
// call fails loudly before dispatch. The sole compatibility default is the
// all-controls-unset case, which yields ReasoningNone and preserves each
// provider's current wire body byte-for-byte.

// Valid values for Request.ReasoningEffort (and Options["reasoning_effort"]).
const (
	ReasoningEffortOff    = "off"
	ReasoningEffortLow    = "low"
	ReasoningEffortMedium = "medium"
	ReasoningEffortHigh   = "high"
)

// Provider name strings used by the reasoning resolver's capability table.
// These match Provider.Name() for the four in-scope providers (the resolver is
// keyed by the string each client passes, NOT the ProviderType enum in
// config.go — note gemini's Name() is "gemini", distinct from ProviderGoogle).
const (
	reasoningProviderOpenAI     = "openai"
	reasoningProviderGemini     = "gemini"
	reasoningProviderAnthropic  = "anthropic"
	reasoningProviderOpenRouter = "openrouter"
)

// Typed sentinel errors for reasoning-control validation.
//
// They are exposed as sentinel error vars so callers can match with errors.Is,
// while the wire/errors.As shape returned to the provider dispatch path stays a
// non-retryable *AIError with Code == CodeSchemaValidation (see reasoningError).
var (
	// ErrInvalidReasoningEffort — a present effort is not one of "", "off",
	// "low", "medium", "high", or the legacy reasoning_effort option is not a
	// string.
	ErrInvalidReasoningEffort = errors.New("invalid reasoning effort")

	// ErrUnsupportedReasoningEffort — the selected provider/model cannot honor
	// the requested semantic exactly, including exact disablement.
	ErrUnsupportedReasoningEffort = errors.New("unsupported reasoning effort for provider/model")

	// ErrConflictingReasoningConfig — two reasoning controls are present but
	// disagree.
	ErrConflictingReasoningConfig = errors.New("conflicting reasoning configuration")

	// ErrInvalidThinkingBudget — thinking_budget_tokens or reasoning_max_tokens
	// has an invalid Go type or provider-specific range.
	ErrInvalidThinkingBudget = errors.New("invalid thinking budget")

	// ErrReasoningBudgetExceedsMaxTokens — an absolute thinking budget is not
	// strictly below Request.MaxTokens, or MaxTokens is required but unset.
	ErrReasoningBudgetExceedsMaxTokens = errors.New("reasoning budget exceeds max_tokens")
)

// ReasoningKind describes the shape of a resolved reasoning decision.
type ReasoningKind int

const (
	// ReasoningNone: no reasoning control requested. Providers MUST preserve
	// their current wire body exactly (the sole compatibility default).
	ReasoningNone ReasoningKind = iota

	// ReasoningEffortKind: a qualitative effort ("off"/"low"/"medium"/"high")
	// was resolved. Providers that use qualitative controls (OpenAI, OpenRouter)
	// read Effort; providers that use absolute budgets (Gemini, Anthropic) read
	// Budget (mapped from the effort) and BudgetSet.
	ReasoningEffortKind

	// ReasoningBudgetKind: an absolute thinking budget was resolved from
	// Options["thinking_budget_tokens"] (Gemini/Anthropic only).
	ReasoningBudgetKind

	// ReasoningMaxTokensKind: a deprecated OpenRouter-only reasoning.max_tokens
	// budget was resolved from Options["reasoning_max_tokens"].
	ReasoningMaxTokensKind
)

// ReasoningDecision is the resolved, validated outcome the resolver hands back
// to a provider constructor. The provider applies exactly this to its wire body.
type ReasoningDecision struct {
	Kind ReasoningKind

	// Effort is the canonical qualitative value ("off"/"low"/"medium"/"high")
	// when Kind == ReasoningEffortKind. Empty otherwise.
	Effort string

	// Budget is the absolute thinking-token budget when the decision maps to an
	// absolute budget (Gemini/Anthropic effort mapping, or an explicit
	// thinking_budget_tokens). BudgetSet distinguishes a genuine 0 (exact
	// disablement) from "no budget".
	Budget    int
	BudgetSet bool

	// MaxTokensReasoning is the OpenRouter reasoning.max_tokens value when
	// Kind == ReasoningMaxTokensKind.
	MaxTokensReasoning int
}

// IsNone reports whether no reasoning control was requested (preserve body).
func (d ReasoningDecision) IsNone() bool { return d.Kind == ReasoningNone }

// reasoningError wraps a sentinel with a human-readable message and returns it
// as a non-retryable *AIError (CodeSchemaValidation), matching the existing
// provider schema-validation convention. The returned error unwraps to the
// sentinel so callers can errors.Is it, and errors.As to *AIError for the wire
// shape.
func reasoningError(sentinel error, format string, args ...any) *AIError {
	msg := fmt.Sprintf(format, args...)
	return &AIError{
		Code:      CodeSchemaValidation,
		Message:   msg,
		Retryable: false,
		wrapped:   sentinel,
	}
}

// effortToBudget maps a qualitative effort to a provider-specific absolute
// thinking budget. Returns (budget, ok). "off" maps to 0 (exact disablement).
// Only Gemini and Anthropic use absolute budgets.
func effortToBudget(effort string) (int, bool) {
	switch effort {
	case ReasoningEffortOff:
		return 0, true
	case ReasoningEffortLow:
		return 1024, true
	case ReasoningEffortMedium:
		return 4096, true
	case ReasoningEffortHigh:
		return 16384, true
	}
	return 0, false
}

// isValidEffort reports whether s is one of the five valid effort values
// ("" counts as valid — it means "unset").
func isValidEffort(s string) bool {
	switch s {
	case "", ReasoningEffortOff, ReasoningEffortLow, ReasoningEffortMedium, ReasoningEffortHigh:
		return true
	}
	return false
}

// ResolveReasoning validates and resolves all reasoning controls on req for the
// given provider/model. It is the single entry point invoked by every provider
// request constructor before marshaling/dispatch.
//
// Returns ReasoningNone (and nil error) when no reasoning control is present —
// the caller MUST then preserve its current wire body byte-for-byte.
//
// Any validation failure returns a non-retryable *AIError (CodeSchemaValidation)
// that unwraps to one of the five reasoning sentinels.
func ResolveReasoning(req *Request, provider, model string) (ReasoningDecision, error) {
	none := ReasoningDecision{Kind: ReasoningNone}

	// --- Gather raw inputs ------------------------------------------------
	typedEffort := req.ReasoningEffort

	var (
		optEffort    string
		optEffortSet bool
	)
	var (
		budgetTokens    int
		budgetTokensSet bool
	)
	var (
		reasoningMax    int
		reasoningMaxSet bool
	)

	if req.Options != nil {
		if v, present := req.Options["reasoning_effort"]; present {
			s, ok := v.(string)
			if !ok {
				return none, reasoningError(ErrInvalidReasoningEffort,
					"%s: Options[\"reasoning_effort\"] must be a string, got %T", provider, v)
			}
			optEffort = s
			optEffortSet = s != ""
		}
		if v, present := req.Options["thinking_budget_tokens"]; present {
			n, ok := v.(int)
			if !ok {
				return none, reasoningError(ErrInvalidThinkingBudget,
					"%s: Options[\"thinking_budget_tokens\"] must be Go type int, got %T", provider, v)
			}
			budgetTokens = n
			budgetTokensSet = true
		}
		if v, present := req.Options["reasoning_max_tokens"]; present {
			n, ok := v.(int)
			if !ok {
				return none, reasoningError(ErrInvalidThinkingBudget,
					"%s: Options[\"reasoning_max_tokens\"] must be Go type int, got %T", provider, v)
			}
			reasoningMax = n
			reasoningMaxSet = true
		}
	}

	// --- Step 1: validate every present effort input independently --------
	if !isValidEffort(typedEffort) {
		return none, reasoningError(ErrInvalidReasoningEffort,
			"%s: Request.ReasoningEffort %q is not one of \"\"/off/low/medium/high", provider, typedEffort)
	}
	if optEffortSet && !isValidEffort(optEffort) {
		return none, reasoningError(ErrInvalidReasoningEffort,
			"%s: Options[\"reasoning_effort\"] %q is not one of off/low/medium/high", provider, optEffort)
	}

	// --- Cross-cutting: both numeric-budget option names present is ALWAYS a
	// conflict, on every provider, before any per-provider support rule. ---
	if budgetTokensSet && reasoningMaxSet {
		return none, reasoningError(ErrConflictingReasoningConfig,
			"%s: both thinking_budget_tokens and reasoning_max_tokens are set; use exactly one numeric budget", provider)
	}

	// --- Step 2: resolve typed effort vs Options["reasoning_effort"] ------
	var resolvedEffort string
	switch {
	case typedEffort != "" && optEffortSet:
		if typedEffort != optEffort {
			return none, reasoningError(ErrConflictingReasoningConfig,
				"%s: Request.ReasoningEffort %q disagrees with Options[\"reasoning_effort\"] %q",
				provider, typedEffort, optEffort)
		}
		resolvedEffort = typedEffort
	case typedEffort != "":
		resolvedEffort = typedEffort
	case optEffortSet:
		resolvedEffort = optEffort
	}

	// --- Step 4 (checked early where it constrains): reasoning_max_tokens ---
	// It is a deprecated OpenRouter-only input. Presence on any other provider
	// is unsupported; on OpenRouter, combining with a resolved effort conflicts.
	if reasoningMaxSet {
		if provider != reasoningProviderOpenRouter {
			return none, reasoningError(ErrUnsupportedReasoningEffort,
				"%s: Options[\"reasoning_max_tokens\"] is OpenRouter-only and not supported by this provider", provider)
		}
		if reasoningMax < 1 {
			return none, reasoningError(ErrInvalidThinkingBudget,
				"%s: Options[\"reasoning_max_tokens\"] must be an int >= 1, got %d", provider, reasoningMax)
		}
		if resolvedEffort != "" {
			return none, reasoningError(ErrConflictingReasoningConfig,
				"%s: Options[\"reasoning_max_tokens\"] cannot be combined with a reasoning effort (no documented equivalence)", provider)
		}
		// Alone on OpenRouter: preserve today's reasoning.max_tokens body.
		return ReasoningDecision{Kind: ReasoningMaxTokensKind, MaxTokensReasoning: reasoningMax}, nil
	}

	// --- Step 3: resolve Options["thinking_budget_tokens"] ----------------
	// Gemini and Anthropic only. On OpenAI/OpenRouter its presence is rejected.
	if budgetTokensSet {
		switch provider {
		case reasoningProviderGemini, reasoningProviderAnthropic:
			// validated below
		default:
			return none, reasoningError(ErrUnsupportedReasoningEffort,
				"%s: Options[\"thinking_budget_tokens\"] is only supported by gemini and anthropic", provider)
		}
		if budgetTokens < 0 {
			return none, reasoningError(ErrInvalidThinkingBudget,
				"%s: thinking_budget_tokens must be >= 0, got %d", provider, budgetTokens)
		}
		if provider == reasoningProviderAnthropic && budgetTokens >= 1 && budgetTokens < 1024 {
			return none, reasoningError(ErrInvalidThinkingBudget,
				"anthropic: thinking_budget_tokens must be 0 (disabled) or >= 1024, got %d", budgetTokens)
		}
		// If an effort is also resolved, the exact budget must equal that
		// provider's mapped budget — otherwise conflict.
		if resolvedEffort != "" {
			mapped, ok := effortToBudget(resolvedEffort)
			if !ok || mapped != budgetTokens {
				return none, reasoningError(ErrConflictingReasoningConfig,
					"%s: thinking_budget_tokens %d disagrees with effort %q (mapped budget %d)",
					provider, budgetTokens, resolvedEffort, mapped)
			}
		}
		if err := validateBudgetVsMaxTokens(provider, budgetTokens, req.MaxTokens); err != nil {
			return none, err
		}
		if err := checkCapability(provider, model, effortForBudget(provider, budgetTokens)); err != nil {
			return none, err
		}
		return ReasoningDecision{Kind: ReasoningBudgetKind, Budget: budgetTokens, BudgetSet: true}, nil
	}

	// --- No numeric budget; resolve a qualitative effort (if any) ---------
	if resolvedEffort == "" {
		return none, nil
	}

	// Capability-gate the effort for the provider/model.
	if err := checkCapability(provider, model, resolvedEffort); err != nil {
		return none, err
	}

	// For absolute-budget providers, map the effort and validate MaxTokens.
	switch provider {
	case reasoningProviderGemini, reasoningProviderAnthropic:
		budget, ok := effortToBudget(resolvedEffort)
		if !ok {
			return none, reasoningError(ErrInvalidReasoningEffort,
				"%s: effort %q has no budget mapping", provider, resolvedEffort)
		}
		if err := validateBudgetVsMaxTokens(provider, budget, req.MaxTokens); err != nil {
			return none, err
		}
		return ReasoningDecision{Kind: ReasoningEffortKind, Effort: resolvedEffort, Budget: budget, BudgetSet: true}, nil
	default:
		// OpenAI / OpenRouter: qualitative, no absolute-budget MaxTokens check.
		return ReasoningDecision{Kind: ReasoningEffortKind, Effort: resolvedEffort}, nil
	}
}

// effortForBudget picks a representative effort label for a raw budget, used only
// to drive the capability check (which is effort-agnostic today — any non-empty
// control on an unregistered model is rejected). "off" for budget 0, "high"
// otherwise; the exact label does not affect the empty-table reject behavior.
func effortForBudget(provider string, budget int) string {
	if budget == 0 {
		return ReasoningEffortOff
	}
	return ReasoningEffortHigh
}

// validateBudgetVsMaxTokens enforces the Conflict Surface rules for
// absolute-budget providers. B == 0 (exact disablement) is exempt: it consumes
// no output tokens and does not require MaxTokens. For enabled thinking
// (B > 0), MaxTokens MUST be explicitly set and MUST satisfy MaxTokens > B.
func validateBudgetVsMaxTokens(provider string, budget, maxTokens int) *AIError {
	if budget == 0 {
		return nil // exact disablement — no output-token overcommit possible
	}
	if maxTokens == 0 {
		return reasoningError(ErrReasoningBudgetExceedsMaxTokens,
			"%s: enabled thinking budget %d requires Request.MaxTokens to be explicitly set (> budget)", provider, budget)
	}
	if budget >= maxTokens {
		return reasoningError(ErrReasoningBudgetExceedsMaxTokens,
			"%s: thinking budget %d must be strictly less than MaxTokens %d", provider, budget, maxTokens)
	}
	return nil
}
