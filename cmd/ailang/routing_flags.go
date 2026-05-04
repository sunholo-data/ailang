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
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/types"
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

// routingFlagValues is the dereferenced snapshot of the routing flags after
// fs.Parse. Used to defer policy construction until after typecheck so the
// safety-gate decision can consult the entry function's declared AI mode
// (M-AI-EFFECT-MODES M2). routingFlagSet.snapshot() builds this.
type routingFlagValues struct {
	fallback     string
	require      string
	prefer       string
	maxPrice     string
	allowRouting bool
}

// snapshot dereferences the flag pointers into a value struct, suitable for
// threading through `runFile` and resolving after typecheck completes.
func (r *routingFlagSet) snapshot() routingFlagValues {
	return routingFlagValues{
		fallback:     *r.fallback,
		require:      *r.require,
		prefer:       *r.prefer,
		maxPrice:     *r.maxPrice,
		allowRouting: *r.allowRouting,
	}
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
//
// M-AI-EFFECT-MODES M2 relaxation: callers that have access to the elaborated
// entry-function effect row should resolve the declared AI mode via
// DetermineDeclaredAIMode and OR its result into allowRouting. A program
// declared !{AI[mode=routeable]} attests routing intent at the type level,
// which is stronger evidence than the runtime --allow-routing flag, so the
// gate is bypassed. Bare !{AI} (which desugars to mode=fixed) and programs
// without an AI effect at all are unaffected — they still require the flag.
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

// resolveRoutingPolicy is the M-AI-EFFECT-MODES M2 entry point used by
// `ailang run` after typecheck completes. It snapshots the runtime
// --allow-routing flag, ORs in the type-level evidence from the entry
// function's declared AI mode, then delegates to buildRoutingPolicy.
//
// The relaxation: when DetermineDeclaredAIMode returns "routeable" or
// "replay-only", the declared signature attests routing intent at the
// type level — stronger evidence than the runtime flag — so the
// safety-gate is bypassed. Bare !{AI} (which desugars to mode=fixed)
// and programs without AI in the entry effect row still require
// --allow-routing to be passed explicitly.
//
// Returns (nil, nil) when no --routing-* flags were set; the caller
// (setupAIHandler) treats a nil policy as "no routing requested".
func resolveRoutingPolicy(values routingFlagValues, programIface *iface.Iface, entry string) (*ai.AIRoutingPolicy, error) {
	declaredMode := DetermineDeclaredAIMode(programIface, entry)
	allowRoutingEffective := values.allowRouting
	switch declaredMode {
	case "routeable", "replay-only":
		// Type-level marker attests intent; runtime gate redundant.
		allowRoutingEffective = true
	}
	return buildRoutingPolicy("",
		values.fallback, values.require, values.prefer, values.maxPrice,
		allowRoutingEffective)
}

// DetermineDeclaredAIMode walks the elaborated entry function's type scheme
// and returns the AI mode declared on its effect row. Returns "" if the
// interface is nil, the entry isn't exported, the entry's type is not a
// function, or its effect row doesn't contain AI.
//
// The result feeds the M-AI-EFFECT-MODES M2 safety-gate relaxation in
// `ailang run`: a program declared !{AI[mode=routeable]} (or replay-only)
// has attested routing intent at the type level, so the runtime
// --allow-routing requirement is bypassed.
//
// "fixed" is returned for bare !{AI} (which desugars to mode=fixed under
// the M-AI-EFFECT-MODES default-mode entry); the safety gate then enforces
// the existing --allow-routing requirement.
//
// We deliberately consult only the entry function's signature: the elaborated
// effect row of the entry attests the program's overall AI usage intent.
// Inner functions are unified against this row by the typechecker.
func DetermineDeclaredAIMode(iface *iface.Iface, entry string) string {
	if iface == nil {
		return ""
	}
	item, ok := iface.GetExport(entry)
	if !ok || item == nil || item.Type == nil {
		return ""
	}
	fn, ok := item.Type.Type.(*types.TFunc2)
	if !ok || fn == nil {
		return ""
	}
	mode, _ := types.EffectModeFor(fn.EffectRow, "AI")
	return mode
}
