package config

import "strconv"

// AILANG's own execution tracing (internal/trace) — distinct from OTLP
// export, which telemetry.go covers.
const (
	EnvTrace          = "AILANG_TRACE"
	EnvNoTrace        = "AILANG_NO_TRACE"
	EnvTraceMaxSpans  = "AILANG_TRACE_MAX_SPANS"
	EnvTraceValues    = "AILANG_TRACE_VALUES"
	EnvTraceRecording = "AILANG_TRACE_RECORDING"
)

var traceVars = []Var{
	{EnvTrace, "standard", AreaTrace, "Tracing tier: off, standard or deep (~2x overhead); an unknown value is an error."},
	{EnvNoTrace, "0", AreaTrace, "1 selects the off tier when AILANG_TRACE is unset."},
	{EnvTraceMaxSpans, "500", AreaTrace, "Cap on spans kept per trace; a non-integer or negative value keeps the default."},
	{EnvTraceValues, "", AreaTrace, "Value-recording mode when the --trace-values flag is empty; an unknown value is an error, never a widening."},
	{EnvTraceRecording, "0", AreaTrace, "1 records span names into the TraceRegistry at start-up."},
}

// TraceTier returns AILANG_TRACE verbatim, "" when unset; internal/trace
// parses it and applies the NoTrace and default rules.
func TraceTier() string { return get(EnvTrace) }

// NoTrace reports AILANG_NO_TRACE=1.
func NoTrace() bool { return getOr(EnvNoTrace) == "1" }

// TraceMaxSpans returns AILANG_TRACE_MAX_SPANS as a non-negative integer,
// falling back to the registered default when unset or invalid.
func TraceMaxSpans() int {
	if v := get(EnvTraceMaxSpans); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	n, _ := strconv.Atoi(defaultOf(EnvTraceMaxSpans))
	return n
}

// TraceValues returns AILANG_TRACE_VALUES verbatim, "" when unset.
func TraceValues() string { return get(EnvTraceValues) }

// TraceRecording reports AILANG_TRACE_RECORDING=1.
func TraceRecording() bool { return getOr(EnvTraceRecording) == "1" }
