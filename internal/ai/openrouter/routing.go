package openrouter

import (
	"github.com/sunholo-data/ailang/internal/ai"
)

// providerField is the JSON shape OpenRouter accepts on the chat-completions
// request body under the top-level `provider` key. It controls dynamic
// provider routing (order, fallback) and capability requirements.
//
// See https://openrouter.ai/docs#provider-routing for the full schema. We
// only model the subset needed by AIRoutingPolicy.
type providerField struct {
	// Order is the preferred upstream-vendor sequence (e.g. "anthropic", "openai").
	Order []string `json:"order,omitempty"`

	// AllowFallbacks enables falling through Order on failure. Pointer so we
	// can serialise an explicit `false` (otherwise omitempty would drop it).
	AllowFallbacks *bool `json:"allow_fallbacks,omitempty"`

	// RequireParameters lists hard-required model capabilities. Models that
	// don't advertise these capabilities are excluded by OpenRouter.
	RequireParameters []string `json:"require_parameters,omitempty"`

	// Sort is OpenRouter's optimisation target — maps from RoutePreference.
	// One of "price", "throughput", "latency", or empty.
	Sort string `json:"sort,omitempty"`
}

// translatePolicy converts an AIRoutingPolicy into an OpenRouter provider
// field. Returns nil when the policy is nil or zero-valued so the chat
// request omits the `provider` block entirely.
//
// Mapping notes:
//   - policy.Order → Order (vendor names like "anthropic" pass through as-is;
//     OpenRouter accepts lowercase identifiers).
//   - policy.AllowFallback → AllowFallbacks (pointer so explicit false ships).
//   - policy.Require → RequireParameters (AICapability values are already
//     stable strings aligned with OpenRouter's vocabulary).
//   - policy.Prefer → Sort:
//     PreferCheapest → "price",
//     PreferFastest → "throughput",
//     PreferMostReliable → "latency" (OpenRouter has no true "reliability"
//     sort; latency is the closest reasonable proxy),
//     PreferUnspecified → empty.
//   - policy.MaxPricePerMTok is currently NOT forwarded — OpenRouter's
//     per-call max-price filter lives under `transforms`, which the v0.16.0
//     design doc explicitly defers. We silently ignore the field in M2 and
//     a follow-up milestone can wire it without an API change.
func translatePolicy(p *ai.AIRoutingPolicy) *providerField {
	if p.IsZero() {
		return nil
	}

	pf := &providerField{}

	if len(p.Order) > 0 {
		pf.Order = append(pf.Order, p.Order...)
	}

	// AllowFallback only meaningful with Order; serialise the bool either way
	// when the caller has opted in to a routing policy. We only emit the
	// pointer when the caller set Order or explicitly set AllowFallback=true,
	// to avoid a noisy `allow_fallbacks: false` on capability-only policies.
	if p.AllowFallback || len(p.Order) > 0 {
		v := p.AllowFallback
		pf.AllowFallbacks = &v
	}

	if len(p.Require) > 0 {
		pf.RequireParameters = make([]string, 0, len(p.Require))
		for _, c := range p.Require {
			pf.RequireParameters = append(pf.RequireParameters, string(c))
		}
	}

	switch p.Prefer {
	case ai.PreferCheapest:
		pf.Sort = "price"
	case ai.PreferFastest:
		pf.Sort = "throughput"
	case ai.PreferMostReliable:
		pf.Sort = "latency"
	}

	// TODO(v0.17.0): forward p.MaxPricePerMTok via OpenRouter `transforms`
	// once that surface area is in scope.

	return pf
}
