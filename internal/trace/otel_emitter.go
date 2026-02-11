package trace

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// spanEntry tracks an in-progress OTEL span during event replay.
type spanEntry struct {
	ctx  context.Context
	span oteltrace.Span
}

// EmitOTELSpans converts collected trace events into OTEL spans.
// Called after program execution completes (batch emission).
// The parentCtx should carry the root "ailang run" span so program
// spans appear as children of the run command.
//
// Algorithm:
//  1. Walk events sequentially
//  2. function_enter → start a new span, push onto stack
//  3. effect → create a short-lived child span of current function
//  4. contract_check → add as span event on current stack top
//  5. budget_delta → add as attributes on current span
//  6. function_exit → end current span, pop stack
//  7. module_start/end → root span wrapping everything
//  8. error → span event with Error status
func EmitOTELSpans(parentCtx context.Context, tracer oteltrace.Tracer, events []TraceEvent, baseTime time.Time) error {
	if len(events) == 0 {
		return nil
	}

	// Stack of in-progress spans for nesting
	stack := make([]spanEntry, 0, 16)

	// currentCtx returns the context from the top of the stack,
	// or parentCtx if the stack is empty.
	currentCtx := func() context.Context {
		if len(stack) > 0 {
			return stack[len(stack)-1].ctx
		}
		return parentCtx
	}

	for _, evt := range events {
		ts := baseTime.Add(time.Duration(evt.TimestampNS))

		switch evt.Event {
		case EventModuleStart:
			if evt.Module == nil {
				continue
			}
			name := "eval.module." + evt.Module.Name
			ctx, span := tracer.Start(currentCtx(), name,
				oteltrace.WithTimestamp(ts),
				oteltrace.WithAttributes(
					attribute.String("ailang.module.name", evt.Module.Name),
					attribute.StringSlice("ailang.module.caps", evt.Module.Caps),
				),
			)
			stack = append(stack, spanEntry{ctx: ctx, span: span})

		case EventModuleEnd:
			if evt.Module == nil {
				continue
			}
			endTs := baseTime.Add(time.Duration(evt.TimestampNS))
			if len(stack) > 0 {
				top := stack[len(stack)-1]
				if evt.Module.DurationNS > 0 {
					top.span.SetAttributes(
						attribute.Int64("ailang.module.duration_ns", evt.Module.DurationNS),
					)
				}
				top.span.End(oteltrace.WithTimestamp(endTs))
				stack = stack[:len(stack)-1]
			}

		case EventFunctionEnter:
			if evt.Function == nil {
				continue
			}
			name := "eval.function." + evt.Function.Name
			attrs := []attribute.KeyValue{
				attribute.String("ailang.function.name", evt.Function.Name),
				attribute.Int("ailang.function.depth", evt.Depth),
			}
			if len(evt.Function.Args) > 0 {
				attrs = append(attrs, attribute.StringSlice("ailang.function.args", evt.Function.Args))
			}
			ctx, span := tracer.Start(currentCtx(), name,
				oteltrace.WithTimestamp(ts),
				oteltrace.WithAttributes(attrs...),
			)
			stack = append(stack, spanEntry{ctx: ctx, span: span})

		case EventFunctionExit:
			if evt.Function == nil {
				continue
			}
			endTs := baseTime.Add(time.Duration(evt.TimestampNS))
			if len(stack) > 0 {
				top := stack[len(stack)-1]
				if evt.Function.Result != "" {
					top.span.SetAttributes(
						attribute.String("ailang.function.result", evt.Function.Result),
					)
				}
				if evt.Function.DurationNS > 0 {
					top.span.SetAttributes(
						attribute.Int64("ailang.function.duration_ns", evt.Function.DurationNS),
					)
				}
				top.span.End(oteltrace.WithTimestamp(endTs))
				stack = stack[:len(stack)-1]
			}

		case EventEffect:
			if evt.Effect == nil {
				continue
			}
			name := fmt.Sprintf("eval.effect.%s.%s", evt.Effect.EffectName, evt.Effect.OpName)
			attrs := []attribute.KeyValue{
				attribute.String("ailang.effect.name", evt.Effect.EffectName),
				attribute.String("ailang.effect.op", evt.Effect.OpName),
			}
			if len(evt.Effect.Args) > 0 {
				attrs = append(attrs, attribute.StringSlice("ailang.effect.args", evt.Effect.Args))
			}
			if evt.Effect.Result != "" {
				attrs = append(attrs, attribute.String("ailang.effect.result", evt.Effect.Result))
			}
			// Effect spans are short-lived children: start and end at same timestamp
			_, span := tracer.Start(currentCtx(), name,
				oteltrace.WithTimestamp(ts),
				oteltrace.WithAttributes(attrs...),
			)
			span.End(oteltrace.WithTimestamp(ts))

		case EventContractCheck:
			if evt.Contract == nil {
				continue
			}
			// Contract checks are span events on the current function span
			if len(stack) > 0 {
				top := stack[len(stack)-1]
				attrs := []attribute.KeyValue{
					attribute.String("ailang.contract.kind", evt.Contract.Kind),
					attribute.Bool("ailang.contract.passed", evt.Contract.Passed),
				}
				if evt.Contract.Message != "" {
					attrs = append(attrs, attribute.String("ailang.contract.message", evt.Contract.Message))
				}
				if evt.Contract.Location != "" {
					attrs = append(attrs, attribute.String("ailang.contract.location", evt.Contract.Location))
				}
				if evt.Contract.Function != "" {
					attrs = append(attrs, attribute.String("ailang.contract.function", evt.Contract.Function))
				}
				eventName := "contract." + evt.Contract.Kind
				if !evt.Contract.Passed {
					eventName += ".failed"
				}
				top.span.AddEvent(eventName, oteltrace.WithTimestamp(ts), oteltrace.WithAttributes(attrs...))
			}

		case EventBudgetDelta:
			if evt.Budget == nil {
				continue
			}
			// Budget deltas are attributes on the current span
			if len(stack) > 0 {
				top := stack[len(stack)-1]
				top.span.SetAttributes(
					attribute.String("ailang.budget.effect", evt.Budget.Effect),
					attribute.Int("ailang.budget.used", evt.Budget.Used),
					attribute.Int("ailang.budget.limit", evt.Budget.Limit),
					attribute.Int("ailang.budget.remaining", evt.Budget.Remaining),
					attribute.Int("ailang.budget.physical", evt.Budget.Physical),
				)
			}

		case EventError:
			if evt.Error == nil {
				continue
			}
			// Errors set error status and add an event
			if len(stack) > 0 {
				top := stack[len(stack)-1]
				top.span.SetStatus(codes.Error, evt.Error.Message)
				attrs := []attribute.KeyValue{
					attribute.String("ailang.error.message", evt.Error.Message),
				}
				if evt.Error.Location != "" {
					attrs = append(attrs, attribute.String("ailang.error.location", evt.Error.Location))
				}
				top.span.AddEvent("error", oteltrace.WithTimestamp(ts), oteltrace.WithAttributes(attrs...))
			}
		}
	}

	// Close any unclosed spans (safety net for malformed event streams)
	for i := len(stack) - 1; i >= 0; i-- {
		stack[i].span.End()
	}

	return nil
}
