package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
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
func NewEffectSpanWrapper() effects.SpanWrapperFunc {
	if !IsEnabled() && !IsGoogleCloudEnabled() {
		return nil
	}
	tracer := Tracer("ailang-effects")

	return func(
		goCtx context.Context,
		effectName, opName string,
		args []eval.Value,
		fn func() (eval.Value, error),
	) (eval.Value, error) {
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
