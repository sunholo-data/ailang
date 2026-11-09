package effects

import (
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/trace"
)

// Trace-recording methods of EffContext (split from context.go to keep it
// under the 800-line gate; no behaviour change).

// HasTraceCollector reports whether an enabled trace collector is attached.
func (ctx *EffContext) HasTraceCollector() bool {
	return ctx.Trace != nil && ctx.Trace.Enabled()
}

// RecordsFunctionCalls reports whether the active collector's tier admits
// per-call function events.
//
// The evaluator consults this BEFORE rendering arguments (M-TRACE-TIER-NOT-ENFORCED):
// rendering a String() per argument per call is where the superlinear memory cost
// is paid, so discovering inside the collector that the event is unwanted would be
// too late.
func (ctx *EffContext) RecordsFunctionCalls() bool {
	return ctx.Trace != nil && ctx.Trace.Enabled() && ctx.Trace.RecordsFunctionCalls()
}

// RenderTraceValue renders a value for the trace under the collector's value
// policy: bounded to the per-value budget, or a byte-count descriptor in
// redacted mode. The whole value is never materialised (M-V1-MEMORY-FOOTPRINT
// M1) — this is the render every trace site must use in place of v.String().
// With no collector it renders unbounded, which callers never reach because
// they gate on HasTraceCollector first.
func (ctx *EffContext) RenderTraceValue(v eval.Value) string {
	if v == nil {
		return ""
	}
	if ctx.Trace == nil {
		return eval.ShowTraceBounded(v, 0)
	}
	// M-TRACE-LABEL-AWARE: inside a labelled call the value is never
	// rendered at all, so it is never materialised as a trace string.
	if label := ctx.Trace.LabelScope(); label != "" {
		return trace.LabelRedactedMarker(label)
	}
	budget, redacted := ctx.Trace.ValueBudget()
	if redacted {
		return trace.RedactedDescriptor(eval.RenderedLen(v))
	}
	// Credentials are withheld at every tier (M-SERVEAPI-WS-BRIDGE G7):
	// this is the one renderer effect, builtin and function-call trace
	// sites share.
	return eval.ShowTraceBounded(v, budget)
}

// PushTraceLabelScope opens an IFC label scope on the active collector for
// the dynamic extent of a call the static IFC pass marked
// (M-TRACE-LABEL-AWARE; eval.TraceLabelScoper). Returns nil when no enabled
// collector is attached.
func (ctx *EffContext) PushTraceLabelScope(label string) func() {
	if ctx == nil || !ctx.HasTraceCollector() {
		return nil
	}
	return ctx.Trace.PushLabelScope(label)
}

// traceSpanArgs is what an effect span wrapper may inspect for enrichment
// (fs.path, net.url, process.command): the real args, or — inside a label
// scope — markers of the same arity, so a labelled URL or path never becomes
// a span attribute either.
func (ctx *EffContext) traceSpanArgs(args []eval.Value) []eval.Value {
	if ctx.Trace == nil {
		return args
	}
	label := ctx.Trace.LabelScope()
	if label == "" {
		return args
	}
	out := make([]eval.Value, len(args))
	for i := range args {
		out[i] = &eval.StringValue{Value: trace.LabelRedactedMarker(label)}
	}
	return out
}

// RecordFunctionEnter delegates to trace collector if present.
func (ctx *EffContext) RecordFunctionEnter(name string, args []string) {
	if ctx.Trace != nil && ctx.Trace.Enabled() {
		ctx.Trace.RecordFunctionEnter(name, args)
	}
}

// RecordFunctionExit delegates to trace collector if present.
func (ctx *EffContext) RecordFunctionExit(name string, result string) {
	if ctx.Trace != nil && ctx.Trace.Enabled() {
		ctx.Trace.RecordFunctionExit(name, result)
	}
}

// RecordEffect delegates to trace collector if present.
func (ctx *EffContext) RecordEffect(effectName, opName string, args []string, result string) {
	if ctx.Trace != nil && ctx.Trace.Enabled() {
		ctx.Trace.RecordEffect(effectName, opName, args, result)
	}
}

// RecordModedEffect delegates to the trace collector, attaching a
// parameterised-effect mode and its replay-contract label
// (M-EFFECT-REPLAY-CONTRACTS). No-op when no trace collector is active.
func (ctx *EffContext) RecordModedEffect(effectName, opName string, args []string, result, mode, contract string) {
	if ctx.Trace != nil && ctx.Trace.Enabled() {
		ctx.Trace.RecordModedEffect(effectName, opName, args, result, mode, contract)
	}
}

// RecordAIEffect delegates to the trace collector with optional routing metadata.
// Effect name is fixed to "AI". Route may be nil for non-routed AI calls.
func (ctx *EffContext) RecordAIEffect(opName string, args []string, result string, route *trace.ResolvedRoute) {
	if ctx.Trace != nil && ctx.Trace.Enabled() {
		ctx.Trace.RecordAIEffect(opName, args, result, route)
	}
}
