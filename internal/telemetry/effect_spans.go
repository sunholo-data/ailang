package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	ailtrace "github.com/sunholo-data/ailang/internal/trace"
)

// NewEffectSpanWrapper returns a SpanWrapperFunc that wraps each effect
// operation with an OTEL span. Returns nil if telemetry is not configured,
// ensuring zero overhead when OTEL is disabled.
//
// The returned wrapper:
//  1. Creates a span named "effect.<effectName>.<opName>"
//  2. Sets pre-call attributes (effect name, op, arg count)
//  3. Adds per-effect enrichment (e.g., Process command, FS path, Net URL)
//  4. Executes the effect operation
//  5. Sets post-call status and per-effect result attributes
//  6. Ends the span
//
// The tier comes from the process environment (AILANG_TRACE), the same
// source the run path resolves before a --trace flag; a CLI --trace=deep
// therefore does not reach this wrapper, and AILANG_TRACE=deep is the switch.
func NewEffectSpanWrapper() effects.SpanWrapperFunc {
	if !IsEnabled() && !IsGoogleCloudEnabled() {
		return nil
	}
	tier, err := ailtrace.TierFromEnv()
	if err != nil {
		tier = ailtrace.TierStandard
	}
	return newEffectSpanWrapper(Tracer("ailang-effects"), tier)
}

// consoleEffectSpan reports whether an effect op is console chatter that a
// standard-tier span records nothing about: no enrichment, zero duration, one
// span per call. One coordinator-run program wrote 2,044 effect.Debug.log spans
// into the prod observatory in a single trace on 2026-09-21 (85% of the newest
// spans), and the local DB carried 3,818 effect.IO.println rows; each of these
// is also doubled by the trace stream's eval.effect.* span. Deep tier keeps
// them, since that is the profiling / training-data opt-in.
func consoleEffectSpan(effectName, opName string) bool {
	switch effectName {
	case "Debug":
		return true
	case "IO":
		switch opName {
		case "print", "println", "eprint", "eprintln", "flush":
			return true
		}
	}
	return false
}

func newEffectSpanWrapper(tracer trace.Tracer, tier ailtrace.Tier) effects.SpanWrapperFunc {
	return func(
		goCtx context.Context,
		effectName, opName string,
		args []eval.Value,
		fn func() (eval.Value, error),
	) (eval.Value, error) {
		if tier != ailtrace.TierDeep && consoleEffectSpan(effectName, opName) {
			return fn()
		}
		spanName := "effect." + effectName + "." + opName
		_, span := StartSpan(goCtx, tracer, spanName)
		defer span.End()

		// Pre-call attributes
		span.SetAttributes(
			attribute.String("effect.name", effectName),
			attribute.String("effect.op", opName),
			attribute.Int("effect.arg_count", len(args)),
		)

		// Per-effect pre-call enrichment
		enrichPreCall(span, effectName, opName, args)

		// Execute the effect operation
		result, err := fn()

		// Post-call: status + enrichment
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		} else {
			span.SetStatus(codes.Ok, "")
			enrichPostCall(span, effectName, opName, result)
		}

		return result, err
	}
}

// enrichPreCall adds per-effect attributes before the operation executes.
func enrichPreCall(span interface{ SetAttributes(...attribute.KeyValue) }, effectName, opName string, args []eval.Value) {
	switch effectName {
	case "Process":
		if opName == "exec" && len(args) >= 1 {
			if cmd, ok := args[0].(*eval.StringValue); ok {
				span.SetAttributes(attribute.String("process.command", cmd.Value))
			}
			if len(args) >= 2 {
				if listVal, ok := args[1].(*eval.ListValue); ok {
					span.SetAttributes(attribute.Int("process.arg_count", len(listVal.Elements)))
				}
			}
		}

	case "FS":
		if len(args) >= 1 {
			if path, ok := args[0].(*eval.StringValue); ok {
				span.SetAttributes(attribute.String("fs.path", path.Value))
			}
		}

	case "Net":
		if len(args) >= 1 {
			if url, ok := args[0].(*eval.StringValue); ok {
				span.SetAttributes(attribute.String("net.url", url.Value))
			}
		}

	case "Stream":
		if opName == "connect" && len(args) >= 1 {
			if url, ok := args[0].(*eval.StringValue); ok {
				span.SetAttributes(attribute.String("stream.url", url.Value))
			}
			if len(args) >= 2 {
				if proto, ok := args[1].(*eval.StringValue); ok {
					span.SetAttributes(attribute.String("stream.protocol", proto.Value))
				}
			}
		}
	}
}

// enrichPostCall adds per-effect attributes after the operation completes successfully.
func enrichPostCall(span interface{ SetAttributes(...attribute.KeyValue) }, effectName, opName string, result eval.Value) {
	if result == nil {
		return
	}

	switch effectName {
	case "Process":
		if opName == "exec" {
			enrichProcessResult(span, result)
		}
	}
}

// enrichProcessResult extracts exit code, stdout size, and error type from
// a Process.exec Result[ProcessOutput, ProcessError] return value.
func enrichProcessResult(span interface{ SetAttributes(...attribute.KeyValue) }, result eval.Value) {
	tagged, ok := result.(*eval.TaggedValue)
	if !ok {
		return
	}

	if tagged.CtorName == "Ok" && len(tagged.Fields) > 0 {
		if record, ok := tagged.Fields[0].(*eval.RecordValue); ok {
			if exitCode, ok := record.Fields["exitCode"].(*eval.IntValue); ok {
				span.SetAttributes(attribute.Int("process.exit_code", exitCode.Value))
			}
			if stdout, ok := record.Fields["stdout"].(*eval.BytesValue); ok {
				span.SetAttributes(attribute.Int("process.stdout_bytes", len(stdout.Value)))
			}
			if stderr, ok := record.Fields["stderr"].(*eval.BytesValue); ok {
				span.SetAttributes(attribute.Int("process.stderr_bytes", len(stderr.Value)))
			}
		}
	} else if tagged.CtorName == "Err" && len(tagged.Fields) > 0 {
		if errTagged, ok := tagged.Fields[0].(*eval.TaggedValue); ok {
			span.SetAttributes(attribute.String("process.error_type", errTagged.CtorName))
		}
	}
}
