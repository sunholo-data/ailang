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
	return EmitOTELSpansWithOptions(parentCtx, tracer, events, baseTime, TracingOptions{Tier: TierDeep})
}

// EmitOTELSpansWithOptions is like EmitOTELSpans but honors TracingOptions
// to filter out per-call function/effect spans in non-deep tiers.
//
// Tier behavior:
//   - TierOff:      nothing is emitted
//   - TierStandard: module/contract/budget/error spans emitted; EventFunctionEnter
//     skipped entirely; EventEffect only emitted when evt.Depth <= 1
//     (top-level, not inside a user function call).
//   - TierDeep:     everything emitted (same as legacy EmitOTELSpans).
func EmitOTELSpansWithOptions(parentCtx context.Context, tracer oteltrace.Tracer, events []TraceEvent, baseTime time.Time, opts TracingOptions) error {
	if len(events) == 0 || opts.Tier == TierOff {
		return nil
	}
	deep := opts.Tier == TierDeep

	// Budget tracking (M-OBS-TRACE-TRIAGE M2).
	// Every time we'd call tracer.Start(...) we first check the counter.
	// When exceeded, we emit exactly one "trace.truncated" rollup span and
	// silently drop subsequent spans. Function entries whose span was dropped
	// get a nil-sentinel on the stack so the matching Exit still pops cleanly.
	budget := opts.MaxSpansPerTrace // 0 = unlimited
	spanCount := 0
	truncated := false
	droppedCount := 0
	firstDroppedName := ""
	lastKeptTS := baseTime

	// tryStart wraps tracer.Start with budget enforcement.
	// Returns (ctx, span, ok). ok=false means the span was dropped.
	tryStart := func(ctx context.Context, name string, ts time.Time, attrs []attribute.KeyValue) (context.Context, oteltrace.Span, bool) {
		if budget > 0 && spanCount >= budget {
			if !truncated {
				// Emit the single rollup span. Counted separately from the
				// budget so "budget=N → N+1 total spans" (N real + 1 rollup).
				truncated = true
				firstDroppedName = name
			}
			droppedCount++
			return ctx, nil, false
		}
		spanCount++
		lastKeptTS = ts
		newCtx, span := tracer.Start(ctx, name,
			oteltrace.WithTimestamp(ts),
			oteltrace.WithAttributes(attrs...),
		)
		return newCtx, span, true
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

	// topSpan walks backwards through the stack to find the nearest
	// entry with a live OTEL span (non-nil). Returns nil if none exist.
	// Used by contract/budget/error attachments so they still land on
	// a real span even when skipped function entries are on the stack.
	topSpan := func() oteltrace.Span {
		for i := len(stack) - 1; i >= 0; i-- {
			if stack[i].span != nil {
				return stack[i].span
			}
		}
		return nil
	}

	for _, evt := range events {
		ts := baseTime.Add(time.Duration(evt.TimestampNS))

		switch evt.Event {
		case EventModuleStart:
			if evt.Module == nil {
				continue
			}
			name := "eval.module." + evt.Module.Name
			ctx, span, _ := tryStart(currentCtx(), name, ts, []attribute.KeyValue{
				attribute.String("ailang.module.name", evt.Module.Name),
				attribute.StringSlice("ailang.module.caps", evt.Module.Caps),
			})
			stack = append(stack, spanEntry{ctx: ctx, span: span})

		case EventModuleEnd:
			if evt.Module == nil {
				continue
			}
			endTs := baseTime.Add(time.Duration(evt.TimestampNS))
			if len(stack) > 0 {
				top := stack[len(stack)-1]
				if top.span != nil {
					if evt.Module.DurationNS > 0 {
						top.span.SetAttributes(
							attribute.Int64("ailang.module.duration_ns", evt.Module.DurationNS),
						)
					}
					top.span.End(oteltrace.WithTimestamp(endTs))
				}
				stack = stack[:len(stack)-1]
			}

		case EventFunctionEnter:
			if evt.Function == nil {
				continue
			}
			if !deep {
				// Skip per-call function spans in non-deep tiers, but
				// push a sentinel so the matching EventFunctionExit
				// doesn't pop a real span underneath us.
				stack = append(stack, spanEntry{ctx: currentCtx(), span: nil})
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
			ctx, span, _ := tryStart(currentCtx(), name, ts, attrs)
			stack = append(stack, spanEntry{ctx: ctx, span: span})

		case EventFunctionExit:
			if evt.Function == nil {
				continue
			}
			endTs := baseTime.Add(time.Duration(evt.TimestampNS))
			if len(stack) > 0 {
				top := stack[len(stack)-1]
				if top.span != nil {
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
				}
				stack = stack[:len(stack)-1]
			}

		case EventEffect:
			if evt.Effect == nil {
				continue
			}
			// In non-deep tiers, emit only "top-level" effects: those whose
			// parent in the event stack is a module/run root, not another
			// user function call. Heuristic: evt.Depth <= 1.
			if !deep && evt.Depth > 1 {
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
			if _, span, ok := tryStart(currentCtx(), name, ts, attrs); ok {
				span.End(oteltrace.WithTimestamp(ts))
			}

		case EventContractCheck:
			if evt.Contract == nil {
				continue
			}
			// Contract checks are span events on the current function span
			if sp := topSpan(); sp != nil {
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
				sp.AddEvent(eventName, oteltrace.WithTimestamp(ts), oteltrace.WithAttributes(attrs...))
			}

		case EventBudgetDelta:
			if evt.Budget == nil {
				continue
			}
			// Budget deltas are attributes on the current span
			if sp := topSpan(); sp != nil {
				sp.SetAttributes(
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
			if sp := topSpan(); sp != nil {
				sp.SetStatus(codes.Error, evt.Error.Message)
				attrs := []attribute.KeyValue{
					attribute.String("ailang.error.message", evt.Error.Message),
				}
				if evt.Error.Location != "" {
					attrs = append(attrs, attribute.String("ailang.error.location", evt.Error.Location))
				}
				sp.AddEvent("error", oteltrace.WithTimestamp(ts), oteltrace.WithAttributes(attrs...))
			}
		}
	}

	// Close any unclosed spans (safety net for malformed event streams)
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i].span != nil {
			stack[i].span.End()
		}
	}

	// M-OBS-TRACE-TRIAGE M2: If budget was exceeded, emit exactly one
	// "trace.truncated" rollup span summarizing the drop. Counted outside
	// the budget (so callers observe budget + 1 spans total in the export).
	if truncated {
		_, rollup := tracer.Start(parentCtx, "trace.truncated",
			oteltrace.WithTimestamp(lastKeptTS),
			oteltrace.WithAttributes(
				attribute.Int("ailang.trace.budget", budget),
				attribute.Int("ailang.trace.kept_count", spanCount),
				attribute.Int("ailang.trace.dropped_count", droppedCount),
				attribute.String("ailang.trace.first_dropped_name", firstDroppedName),
			),
		)
		rollup.End(oteltrace.WithTimestamp(lastKeptTS))
	}

	return nil
}
