package openrouter

import (
	"strings"
	"sync"

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

	// MaxPrice caps the per-million-token price OpenRouter will pay when
	// selecting an upstream provider. nil when no cap is set.
	MaxPrice *maxPriceField `json:"max_price,omitempty"`

	// Ignore lists upstream provider slugs OpenRouter must not route to.
	// Filled from the model registry's openrouter_ignore (see routeFor).
	Ignore []string `json:"ignore,omitempty"`
}

// maxPriceField is the JSON shape OpenRouter accepts under `provider.max_price`.
// Values are USD per million tokens; providers priced above the cap are
// excluded. See https://openrouter.ai/docs/features/provider-routing.
//
// AIRoutingPolicy carries a single MaxPricePerMTok cap, which we apply to both
// the prompt and completion legs (the two OpenRouter enforces per token).
type maxPriceField struct {
	Prompt     float64 `json:"prompt"`
	Completion float64 `json:"completion"`
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
//   - policy.MaxPricePerMTok → MaxPrice.{Prompt,Completion} (USD per million
//     tokens). The single cap is applied to both legs; providers priced above
//     it are excluded by OpenRouter. Returns an error when the string is set
//     but not a valid non-negative number (fail loud — never silently drop a
//     cost guard the caller believes is enforced).
func translatePolicy(p *ai.AIRoutingPolicy) (*providerField, error) {
	if p.IsZero() {
		return nil, nil
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

	price, ok, err := p.ParsedMaxPrice()
	if err != nil {
		return nil, err
	}
	if ok {
		pf.MaxPrice = &maxPriceField{Prompt: price, Completion: price}
	}

	return pf, nil
}

// providerIgnore maps an OpenRouter wire model ("z-ai/glm-5.3-flash") to the
// upstream provider slugs it must not be routed to. Registered once by the
// command layer from the model registry (modelreg openrouter_ignore), so this
// core package never reads models.yml itself.
var (
	providerIgnoreMu sync.RWMutex
	providerIgnore   map[string][]string
)

// SetProviderIgnore installs the per-model provider ignore lists. Keys are
// wire model names without the "openrouter/" prefix.
func SetProviderIgnore(m map[string][]string) {
	providerIgnoreMu.Lock()
	defer providerIgnoreMu.Unlock()
	providerIgnore = m
}

// ProviderIgnoreFor reports the installed exclusions for a model (diagnostics, tests).
func ProviderIgnoreFor(model string) []string { return ignoreFor(model) }

func ignoreFor(model string) []string {
	providerIgnoreMu.RLock()
	defer providerIgnoreMu.RUnlock()
	return providerIgnore[strings.TrimPrefix(model, "openrouter/")]
}

// routeFor is the request's `provider` block: the caller's routing policy
// plus the registry's ignore list for this model. nil when neither applies.
//
// Why a per-model ignore: OpenRouter load-balances a model over every host
// that serves it, and hosts differ. Measured 2026-10-12, GLM-5.3-Flash:
// inference-net/fp4 (4-bit weights) cached 8% of input against 91-97% on
// Friendli, and its tool calls contradicted the model's own reasoning
// ({"op":"tree","module":"x"} for 25+ steps while the reasoning said "use
// path"). A routing policy cannot express that per model; the registry can.
func routeFor(req *ai.Request) (*providerField, error) {
	pf, err := translatePolicy(req.Routing)
	if err != nil {
		return nil, err
	}
	ignore := ignoreFor(req.Model)
	if len(ignore) == 0 {
		return pf, nil
	}
	if pf == nil {
		pf = &providerField{}
	}
	pf.Ignore = append(append([]string(nil), pf.Ignore...), ignore...)
	return pf, nil
}
