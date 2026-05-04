// Routing flags shared between `ailang exec --api-only` and `ailang run`.
//
// The two CLI entry points need the same routing flag set with the same
// safety-gate semantics: any --routing-* flag without --allow-routing must
// fail loudly because routing introduces dynamic provider selection. This
// file is the single source of truth so `exec` and `run` cannot drift.
package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
)

// routingFlagSet holds pointers to the routing CLI flags. Returned by
// registerRoutingFlags so callers can read them after fs.Parse.
type routingFlagSet struct {
	fallback     *string
	require      *string
	prefer       *string
	maxPrice     *string
	allowRouting *bool
}

// registerRoutingFlags registers the standard --routing-* flags on the given
// FlagSet. Returns a routingFlagSet containing the parsed values (read after
// fs.Parse).
//
// providerHint is used in flag descriptions only — it does not constrain
// behaviour. Pass "openrouter" for clarity in help text on `exec`, or
// leave empty for `run` (which doesn't have a positional provider arg).
func registerRoutingFlags(fs *flag.FlagSet) *routingFlagSet {
	return &routingFlagSet{
		fallback: fs.String("routing-fallback", "",
			"Comma-separated provider order for OpenRouter (e.g. \"anthropic,openai,google\"); also enables fallback"),
		require: fs.String("routing-require", "",
			"Comma-separated required model capabilities (e.g. \"structured_outputs,tool_calling\")"),
		prefer: fs.String("routing-prefer", "",
			"Routing preference: cheapest|fastest|most_reliable"),
		maxPrice: fs.String("routing-max-price", "",
			"Max price per million tokens in USD (currently parsed but not forwarded; reserved for v0.17.0)"),
		allowRouting: fs.Bool("allow-routing", false,
			"Required acknowledgment when --routing-* flags are set; routing introduces dynamic provider selection so explicit opt-in is required"),
	}
}

// routingFlagNames returns the bare flag names for normalizeArgsForFlags.
// Kept in this file so callers don't have to repeat the list.
func routingFlagNames() []string {
	return []string{
		"routing-fallback", "routing-require", "routing-prefer",
		"routing-max-price", "allow-routing",
	}
}

// buildRoutingPolicy assembles an *ai.AIRoutingPolicy from the four routing
// CLI flags. Returns (nil, nil) when all flags are empty (no policy).
//
// Validation:
//   - routing-prefer must be one of "cheapest", "fastest", "most_reliable" (or empty).
//   - routing-fallback / routing-require parse comma-separated lists, trimming whitespace.
//   - routing-max-price is forwarded as a string (preserving precision); not validated
//     for numeric format here — OpenRouter is not yet receiving it (see translatePolicy).
//   - When the provider isn't openrouter and any routing flag is set, we return an
//     error early rather than relying on the provider to reject it, so the CLI gives
//     a fast, friendly diagnostic. Pass providerCheck=false from `ailang run` where
//     the provider is inferred from the model string at handler-setup time.
//   - Safety gate: any routing flag set without allowRouting=true returns a typed
//     error before submitting the request. Routing introduces dynamic provider
//     selection, so explicit opt-in is required. This is the runtime equivalent
//     of the design doc's AI[Routeable] type-level marker.
func buildRoutingPolicy(provider, fallback, require, prefer, maxPrice string, allowRouting bool) (*ai.AIRoutingPolicy, error) {
	if fallback == "" && require == "" && prefer == "" && maxPrice == "" {
		return nil, nil
	}
	if !allowRouting {
		return nil, fmt.Errorf("routing flags set but --allow-routing not enabled.\n" +
			"Routing introduces dynamic provider selection; pass --allow-routing\n" +
			"to acknowledge this is intentional")
	}
	// providerCheck is enforced only when caller supplies a non-empty provider.
	// `ailang run` infers the provider from --ai <model>, so it passes "" here
	// and skips the strict provider check. The handler dispatch then surfaces
	// any "non-openrouter provider with routing" mismatch via ErrRoutingNotSupported.
	if provider != "" && provider != "openrouter" {
		return nil, fmt.Errorf("--routing-* flags require --provider openrouter (got %q)", provider)
	}

	p := &ai.AIRoutingPolicy{}

	if fallback != "" {
		for _, s := range strings.Split(fallback, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				p.Order = append(p.Order, s)
			}
		}
		// Specifying a fallback list implies AllowFallback=true.
		p.AllowFallback = true
	}

	if require != "" {
		for _, s := range strings.Split(require, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				p.Require = append(p.Require, ai.AICapability(s))
			}
		}
	}

	switch prefer {
	case "":
		// no-op
	case "cheapest":
		p.Prefer = ai.PreferCheapest
	case "fastest":
		p.Prefer = ai.PreferFastest
	case "most_reliable":
		p.Prefer = ai.PreferMostReliable
	default:
		return nil, fmt.Errorf("--routing-prefer must be one of cheapest|fastest|most_reliable (got %q)", prefer)
	}

	p.MaxPricePerMTok = maxPrice

	return p, nil
}
