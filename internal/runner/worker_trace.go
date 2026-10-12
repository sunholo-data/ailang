package runner

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/effects"
	ailtrace "github.com/sunholo-data/ailang/internal/trace"
	"go.opentelemetry.io/otel"
)

// Emit after worker shutdown so the cleanup receipt precedes final flush.
func emitRunTrace(ctx context.Context, effCtx *effects.EffContext, emitTrace string, traceOpts ailtrace.TracingOptions) {
	// M-TRACE-EXPORT: Output semantic execution trace
	if emitTrace != "" && effCtx.Trace != nil {
		events := effCtx.Trace.Events()

		// Phase 1: JSONL output to stdout
		if strings.Contains(emitTrace, "jsonl") && len(events) > 0 {
			if err := ailtrace.WriteJSONL(os.Stdout, events); err != nil {
				fmt.Fprintf(os.Stderr, "%s: trace output: %v\n", red("Error"), err)
			}
		}

		// Phase 2: OTEL span emission
		if (strings.Contains(emitTrace, "otel") || emitTrace == "auto") && len(events) > 0 {
			evalTracer := otel.Tracer("ailang.eval")
			if err := ailtrace.EmitOTELSpansWithOptions(ctx, evalTracer, events, effCtx.Trace.BaseTime(), traceOpts); err != nil {
				fmt.Fprintf(os.Stderr, "%s: OTEL trace emission: %v\n", red("Error"), err)
			}
		}
	}

}
