// Package ai routing types.
//
// AIRoutingPolicy is an optional, additive IR attached to ai.Request that
// expresses constraints and preferences for dynamic model/provider selection.
// In v0.16.0 it is consumed only by the OpenRouter provider — all other
// providers must reject a non-zero policy with ErrRoutingNotSupported so
// callers cannot accidentally mask their intent (no silent fallbacks).
package ai

import "errors"

// RoutePreference is the optimization target for routing.
//
// PreferUnspecified means "no opinion" — let the provider use its default
// routing behaviour. The other constants map onto OpenRouter's `provider.sort`
// field (see openrouter/routing.go for the translation).
type RoutePreference string

const (
	PreferUnspecified  RoutePreference = ""
	PreferCheapest     RoutePreference = "cheapest"
	PreferFastest      RoutePreference = "fastest"
	PreferMostReliable RoutePreference = "most_reliable"
)

// AICapability is a required model capability.
//
// Used in AIRoutingPolicy.Require to constrain which models the router may
// pick. Models that do not advertise a required capability are excluded.
// The string values are stable wire identifiers — keep them aligned with
// OpenRouter's `provider.require_parameters` vocabulary where possible.
type AICapability string

const (
	CapStructuredOutputs AICapability = "structured_outputs"
	CapToolCalling       AICapability = "tool_calling"
	CapVision            AICapability = "vision"
	CapJSONMode          AICapability = "json_mode"
	CapStreaming         AICapability = "streaming"
)

// AIRoutingPolicy expresses constraints and preferences for dynamic
// model/provider selection. Currently consumed only by OpenRouter;
// other providers reject non-nil policies with ErrRoutingNotSupported.
//
// Zero value (all empty) is meaningful: "use the requested model with
// no fallback". A non-nil policy whose HasRouting() returns false is
// equivalent to no policy from the upstream's perspective.
type AIRoutingPolicy struct {
	// Order is the preferred provider sequence (e.g., ["anthropic", "openai", "google"]).
	// OpenRouter routes through this list in order.
	Order []string

	// AllowFallback enables falling through Order on failure.
	AllowFallback bool

	// Require lists hard-required model capabilities.
	// Models that don't advertise these capabilities are excluded.
	Require []AICapability

	// MaxPricePerMTok caps the price per million tokens in USD.
	// Empty string = no cap. Stored as string for precision.
	//
	// NOTE: In M2 this field is currently NOT forwarded to OpenRouter — their
	// per-call max-price filter lives under `transforms` which is explicitly
	// deferred per the v0.16.0 design doc. The field is preserved on the IR
	// so callers can express the intent today and a follow-up milestone can
	// wire it through without breaking the API.
	MaxPricePerMTok string

	// Prefer is the optimization target.
	Prefer RoutePreference
}

// IsZero returns true if the policy expresses no constraints.
// A nil pointer to AIRoutingPolicy and an IsZero policy are semantically
// equivalent — both mean "no routing policy".
func (p *AIRoutingPolicy) IsZero() bool {
	if p == nil {
		return true
	}
	return len(p.Order) == 0 &&
		!p.AllowFallback &&
		len(p.Require) == 0 &&
		p.MaxPricePerMTok == "" &&
		p.Prefer == PreferUnspecified
}

// HasRouting returns true if the policy actually requests provider routing
// (non-empty Order or AllowFallback). Used by non-OpenRouter providers to
// decide whether to reject the request.
//
// A policy that only sets capability requirements or a sort preference but
// does not request a provider order or fallback is not considered "routing"
// for rejection purposes — non-OpenRouter providers may still satisfy those
// constraints natively (or ignore them) without contradicting caller intent.
func (p *AIRoutingPolicy) HasRouting() bool {
	if p == nil {
		return false
	}
	return len(p.Order) > 0 || p.AllowFallback
}

// ErrRoutingNotSupported is returned by providers that do not support routing
// when a non-zero AIRoutingPolicy is present. Callers can use errors.Is to
// detect this case and either remove the policy, switch to OpenRouter, or
// surface the error to the user.
var ErrRoutingNotSupported = errors.New("routing policy not supported by this provider")

// ErrRoutingRequiresRouteableMode is returned when a routing policy is supplied
// for a function declared !{AI[mode=fixed]} (or bare !{AI} which desugars to
// fixed under M-AI-EFFECT-MODES). The fix is either to declare
// !{AI[mode=routeable]} on the function signature (preferred — type-level
// intent), or to remove the routing flags.
//
// This is the runtime sibling to the type-level invariant unification check
// that rejects !{AI[mode=fixed]} unifying with !{AI[mode=routeable]}. The CLI
// safety-gate (cmd/ailang/routing_flags.go) is the load-bearing protection;
// providers may also raise this error as defence-in-depth on any path that
// bypasses the typechecker (manual handler construction, embedded use, etc.).
//
// TODO(M-AI-EFFECT-MODES M3+): wire the declared mode through Request so
// providers can enforce this at handler.Generate time. The current
// AIHandler interface (string -> string) doesn't carry effect-row info;
// extending it requires plumbing a DeclaredAIMode field on Request and
// populating it from the calling function's elaborated effect row at AI
// op invocation. The CLI-side gate is sufficient protection for v0.15.0.
var ErrRoutingRequiresRouteableMode = errors.New("routing requires !{AI[mode=routeable]} on the function signature")
