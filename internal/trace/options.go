package trace

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Tier controls how much detail the tracing emitter writes.
//
// Precedence (set by callers): CLI flag > AILANG_TRACE env var >
// AILANG_NO_TRACE=1 (legacy) > default (TierStandard).
type Tier int

const (
	// TierOff emits no spans. Equivalent to legacy AILANG_NO_TRACE=1.
	TierOff Tier = iota
	// TierStandard emits module, top-level effect, coordinator, executor,
	// compile, task/chain-linked spans, but NOT per-call function spans or
	// nested effect spans. This is the default.
	TierStandard
	// TierDeep emits everything, including per-call eval.function.* spans
	// and per-op eval.effect.* spans. Opt-in for profiling / training data.
	TierDeep
)

// String returns the canonical tier name ("off", "standard", "deep").
func (t Tier) String() string {
	switch t {
	case TierOff:
		return "off"
	case TierStandard:
		return "standard"
	case TierDeep:
		return "deep"
	default:
		return fmt.Sprintf("unknown(%d)", int(t))
	}
}

// ParseTier parses a tier name. Accepts "off", "standard", "deep"
// (case-insensitive). Returns an error for unrecognized values.
func ParseTier(s string) (Tier, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "off", "none", "disabled":
		return TierOff, nil
	case "standard", "default", "":
		return TierStandard, nil
	case "deep", "full", "all":
		return TierDeep, nil
	default:
		return TierStandard, fmt.Errorf("unknown tracing tier %q (want off|standard|deep)", s)
	}
}

// TracingOptions controls emitter behavior. Threaded from CLI / env
// parsing down into the OTEL emitter.
type TracingOptions struct {
	Tier Tier
	// MaxSpansPerTrace caps spans per trace. 0 = no budget (M2 populates this).
	MaxSpansPerTrace int
}

// DefaultTracingOptions returns the default tier + budget.
func DefaultTracingOptions() TracingOptions {
	opts := TracingOptions{
		Tier:             TierStandard,
		MaxSpansPerTrace: 500,
	}
	if v := os.Getenv("AILANG_TRACE_MAX_SPANS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			opts.MaxSpansPerTrace = n
		}
	}
	return opts
}

// TierFromEnv resolves the tier from env vars (no CLI flag).
// Precedence:
//  1. AILANG_TRACE env var (off|standard|deep)
//  2. AILANG_NO_TRACE=1 legacy alias → TierOff
//  3. TierStandard (default)
//
// Returns the resolved tier and a non-nil error only when AILANG_TRACE
// is set to an unrecognized value.
func TierFromEnv() (Tier, error) {
	if v := os.Getenv("AILANG_TRACE"); v != "" {
		return ParseTier(v)
	}
	if os.Getenv("AILANG_NO_TRACE") == "1" {
		return TierOff, nil
	}
	return TierStandard, nil
}

// ResolveTier applies CLI > env > default precedence.
// cliFlag should be the raw --trace value ("" if unset).
func ResolveTier(cliFlag string) (Tier, error) {
	if cliFlag != "" {
		return ParseTier(cliFlag)
	}
	return TierFromEnv()
}

// DeepTrace reports whether per-call function/effect spans should fire.
func (o TracingOptions) DeepTrace() bool {
	return o.Tier == TierDeep
}

// Enabled reports whether any spans should be emitted.
func (o TracingOptions) Enabled() bool {
	return o.Tier != TierOff
}
