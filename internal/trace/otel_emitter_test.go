package trace

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// setupTestTracer creates a tracer with an in-memory exporter for testing.
func setupTestTracer() (*tracetest.InMemoryExporter, *sdktrace.TracerProvider) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	return exporter, tp
}

func TestEmitOTELSpans_Empty(t *testing.T) {
	exporter, tp := setupTestTracer()
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")

	err := EmitOTELSpans(context.Background(), tracer, nil, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exporter.GetSpans()) != 0 {
		t.Errorf("expected 0 spans, got %d", len(exporter.GetSpans()))
	}
}

func TestEmitOTELSpans_FunctionSpans(t *testing.T) {
	exporter, tp := setupTestTracer()
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")

	base := time.Now()
	events := []TraceEvent{
		{Version: "1.0", Event: EventFunctionEnter, TimestampNS: 1000, Depth: 1,
			Function: &FunctionEvent{Name: "factorial", Args: []string{"5"}}},
		{Version: "1.0", Event: EventFunctionEnter, TimestampNS: 2000, Depth: 2,
			Function: &FunctionEvent{Name: "factorial", Args: []string{"4"}}},
		{Version: "1.0", Event: EventFunctionExit, TimestampNS: 3000, Depth: 2,
			Function: &FunctionEvent{Name: "factorial", Result: "24", DurationNS: 1000}},
		{Version: "1.0", Event: EventFunctionExit, TimestampNS: 4000, Depth: 1,
			Function: &FunctionEvent{Name: "factorial", Result: "120", DurationNS: 3000}},
	}

	err := EmitOTELSpans(context.Background(), tracer, events, base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(spans))
	}

	// Spans are recorded in order of End() calls: inner first, then outer
	inner := spans[0]
	outer := spans[1]

	if inner.Name != "eval.function.factorial" {
		t.Errorf("inner span name: expected eval.function.factorial, got %s", inner.Name)
	}
	if outer.Name != "eval.function.factorial" {
		t.Errorf("outer span name: expected eval.function.factorial, got %s", outer.Name)
	}

	// Inner should be child of outer
	if inner.Parent.SpanID() != outer.SpanContext.SpanID() {
		t.Error("inner span should be child of outer span")
	}

	// Check attributes on inner span
	assertAttribute(t, inner.Attributes, "ailang.function.result", "24")
	assertAttribute(t, inner.Attributes, "ailang.function.name", "factorial")
}

func TestEmitOTELSpans_ModuleSpan(t *testing.T) {
	exporter, tp := setupTestTracer()
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")

	base := time.Now()
	events := []TraceEvent{
		{Version: "1.0", Event: EventModuleStart, TimestampNS: 0,
			Module: &ModuleEvent{Name: "hello", Caps: []string{"IO"}}},
		{Version: "1.0", Event: EventModuleEnd, TimestampNS: 5000,
			Module: &ModuleEvent{Name: "hello", DurationNS: 5000}},
	}

	err := EmitOTELSpans(context.Background(), tracer, events, base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	if spans[0].Name != "eval.module.hello" {
		t.Errorf("expected eval.module.hello, got %s", spans[0].Name)
	}
	assertAttribute(t, spans[0].Attributes, "ailang.module.name", "hello")
}

func TestEmitOTELSpans_EffectSpan(t *testing.T) {
	exporter, tp := setupTestTracer()
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")

	base := time.Now()
	events := []TraceEvent{
		{Version: "1.0", Event: EventFunctionEnter, TimestampNS: 1000, Depth: 1,
			Function: &FunctionEvent{Name: "main"}},
		{Version: "1.0", Event: EventEffect, TimestampNS: 2000, Depth: 1,
			Effect: &EffectEvent{EffectName: "IO", OpName: "println", Args: []string{"\"hello\""}, Result: "()"}},
		{Version: "1.0", Event: EventFunctionExit, TimestampNS: 3000, Depth: 1,
			Function: &FunctionEvent{Name: "main", Result: "()"}},
	}

	err := EmitOTELSpans(context.Background(), tracer, events, base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("expected 2 spans (effect + function), got %d", len(spans))
	}

	// Effect span ends first (instantaneous), then function span
	effect := spans[0]
	fn := spans[1]

	if effect.Name != "eval.effect.IO.println" {
		t.Errorf("expected eval.effect.IO.println, got %s", effect.Name)
	}

	// Effect should be child of function
	if effect.Parent.SpanID() != fn.SpanContext.SpanID() {
		t.Error("effect span should be child of function span")
	}

	assertAttribute(t, effect.Attributes, "ailang.effect.name", "IO")
	assertAttribute(t, effect.Attributes, "ailang.effect.op", "println")
	assertAttribute(t, effect.Attributes, "ailang.effect.result", "()")
}

func TestEmitOTELSpans_ContractEvent(t *testing.T) {
	exporter, tp := setupTestTracer()
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")

	base := time.Now()
	events := []TraceEvent{
		{Version: "1.0", Event: EventFunctionEnter, TimestampNS: 1000, Depth: 1,
			Function: &FunctionEvent{Name: "factorial"}},
		{Version: "1.0", Event: EventContractCheck, TimestampNS: 1500, Depth: 1,
			Contract: &ContractEvent{Kind: "requires", Passed: true, Message: "x > 0", Location: "main.ail:5:3", Function: "factorial"}},
		{Version: "1.0", Event: EventFunctionExit, TimestampNS: 2000, Depth: 1,
			Function: &FunctionEvent{Name: "factorial", Result: "120"}},
	}

	err := EmitOTELSpans(context.Background(), tracer, events, base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	// Contract should appear as an event on the function span
	if len(spans[0].Events) != 1 {
		t.Fatalf("expected 1 span event, got %d", len(spans[0].Events))
	}

	spanEvt := spans[0].Events[0]
	if spanEvt.Name != "contract.requires" {
		t.Errorf("expected event name contract.requires, got %s", spanEvt.Name)
	}
}

func TestEmitOTELSpans_ContractFailed(t *testing.T) {
	exporter, tp := setupTestTracer()
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")

	base := time.Now()
	events := []TraceEvent{
		{Version: "1.0", Event: EventFunctionEnter, TimestampNS: 1000, Depth: 1,
			Function: &FunctionEvent{Name: "factorial"}},
		{Version: "1.0", Event: EventContractCheck, TimestampNS: 1500, Depth: 1,
			Contract: &ContractEvent{Kind: "ensures", Passed: false, Message: "result > 0"}},
		{Version: "1.0", Event: EventFunctionExit, TimestampNS: 2000, Depth: 1,
			Function: &FunctionEvent{Name: "factorial"}},
	}

	err := EmitOTELSpans(context.Background(), tracer, events, base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	if len(spans[0].Events) != 1 {
		t.Fatalf("expected 1 span event, got %d", len(spans[0].Events))
	}

	if spans[0].Events[0].Name != "contract.ensures.failed" {
		t.Errorf("expected contract.ensures.failed, got %s", spans[0].Events[0].Name)
	}
}

func TestEmitOTELSpans_BudgetDelta(t *testing.T) {
	exporter, tp := setupTestTracer()
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")

	base := time.Now()
	events := []TraceEvent{
		{Version: "1.0", Event: EventFunctionEnter, TimestampNS: 1000, Depth: 1,
			Function: &FunctionEvent{Name: "main"}},
		{Version: "1.0", Event: EventBudgetDelta, TimestampNS: 1500, Depth: 1,
			Budget: &BudgetEvent{Effect: "IO", Used: 3, Limit: 5, Remaining: 2, Physical: 3}},
		{Version: "1.0", Event: EventFunctionExit, TimestampNS: 2000, Depth: 1,
			Function: &FunctionEvent{Name: "main"}},
	}

	err := EmitOTELSpans(context.Background(), tracer, events, base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	assertAttribute(t, spans[0].Attributes, "ailang.budget.effect", "IO")
	assertIntAttribute(t, spans[0].Attributes, "ailang.budget.used", 3)
	assertIntAttribute(t, spans[0].Attributes, "ailang.budget.limit", 5)
	assertIntAttribute(t, spans[0].Attributes, "ailang.budget.remaining", 2)
}

func TestEmitOTELSpans_ErrorEvent(t *testing.T) {
	exporter, tp := setupTestTracer()
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")

	base := time.Now()
	events := []TraceEvent{
		{Version: "1.0", Event: EventFunctionEnter, TimestampNS: 1000, Depth: 1,
			Function: &FunctionEvent{Name: "divide"}},
		{Version: "1.0", Event: EventError, TimestampNS: 1500, Depth: 1,
			Error: &ErrorEvent{Message: "division by zero", Location: "math.ail:12:5"}},
		{Version: "1.0", Event: EventFunctionExit, TimestampNS: 2000, Depth: 1,
			Function: &FunctionEvent{Name: "divide"}},
	}

	err := EmitOTELSpans(context.Background(), tracer, events, base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	// Error should set span status
	if spans[0].Status.Code != codes.Error {
		t.Errorf("expected error status, got %v", spans[0].Status.Code)
	}
	if spans[0].Status.Description != "division by zero" {
		t.Errorf("expected error description 'division by zero', got %q", spans[0].Status.Description)
	}

	// Should also have an error event
	if len(spans[0].Events) != 1 {
		t.Fatalf("expected 1 span event, got %d", len(spans[0].Events))
	}
	if spans[0].Events[0].Name != "error" {
		t.Errorf("expected event name 'error', got %s", spans[0].Events[0].Name)
	}
}

func TestEmitOTELSpans_FullTrace(t *testing.T) {
	exporter, tp := setupTestTracer()
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")

	base := time.Now()
	// Simulate: module start -> function -> effect -> contract -> budget -> function exit -> module end
	events := []TraceEvent{
		{Version: "1.0", Event: EventModuleStart, TimestampNS: 0,
			Module: &ModuleEvent{Name: "app", Caps: []string{"IO", "FS"}}},
		{Version: "1.0", Event: EventFunctionEnter, TimestampNS: 1000, Depth: 1,
			Function: &FunctionEvent{Name: "processFile", Args: []string{"\"input.txt\""}}},
		{Version: "1.0", Event: EventContractCheck, TimestampNS: 1200, Depth: 1,
			Contract: &ContractEvent{Kind: "requires", Passed: true, Message: "file exists", Function: "processFile"}},
		{Version: "1.0", Event: EventEffect, TimestampNS: 1500, Depth: 1,
			Effect: &EffectEvent{EffectName: "FS", OpName: "readFile", Args: []string{"\"input.txt\""}, Result: "\"data...\""}},
		{Version: "1.0", Event: EventBudgetDelta, TimestampNS: 1600, Depth: 1,
			Budget: &BudgetEvent{Effect: "FS", Used: 1, Limit: 3, Remaining: 2, Physical: 1}},
		{Version: "1.0", Event: EventEffect, TimestampNS: 2000, Depth: 1,
			Effect: &EffectEvent{EffectName: "IO", OpName: "println", Args: []string{"\"result\""}, Result: "()"}},
		{Version: "1.0", Event: EventFunctionExit, TimestampNS: 3000, Depth: 1,
			Function: &FunctionEvent{Name: "processFile", Result: "()", DurationNS: 2000}},
		{Version: "1.0", Event: EventModuleEnd, TimestampNS: 4000,
			Module: &ModuleEvent{Name: "app", DurationNS: 4000}},
	}

	err := EmitOTELSpans(context.Background(), tracer, events, base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	// Expected: 2 effects + 1 function + 1 module = 4 spans
	if len(spans) != 4 {
		t.Fatalf("expected 4 spans, got %d", len(spans))
	}

	// Verify span names (in order of End() calls)
	names := make([]string, len(spans))
	for i, s := range spans {
		names[i] = s.Name
	}

	// Effects end first (instantaneous), then function, then module
	expectedNames := []string{
		"eval.effect.FS.readFile",
		"eval.effect.IO.println",
		"eval.function.processFile",
		"eval.module.app",
	}
	for i, expected := range expectedNames {
		if names[i] != expected {
			t.Errorf("span[%d]: expected %s, got %s", i, expected, names[i])
		}
	}

	// Function span should have contract event
	fnSpan := spans[2]
	if len(fnSpan.Events) != 1 {
		t.Errorf("expected 1 event on function span, got %d", len(fnSpan.Events))
	}

	// Verify hierarchy: effects → function → module
	moduleSpan := spans[3]
	if fnSpan.Parent.SpanID() != moduleSpan.SpanContext.SpanID() {
		t.Error("function span should be child of module span")
	}
	effectSpan := spans[0]
	if effectSpan.Parent.SpanID() != fnSpan.SpanContext.SpanID() {
		t.Error("effect span should be child of function span")
	}
}

func TestEmitOTELSpans_NilPayloads(t *testing.T) {
	exporter, tp := setupTestTracer()
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")

	// Events with nil payloads should be skipped gracefully
	events := []TraceEvent{
		{Version: "1.0", Event: EventFunctionEnter, TimestampNS: 1000},
		{Version: "1.0", Event: EventEffect, TimestampNS: 2000},
		{Version: "1.0", Event: EventContractCheck, TimestampNS: 3000},
		{Version: "1.0", Event: EventBudgetDelta, TimestampNS: 4000},
		{Version: "1.0", Event: EventError, TimestampNS: 5000},
		{Version: "1.0", Event: EventModuleStart, TimestampNS: 6000},
		{Version: "1.0", Event: EventModuleEnd, TimestampNS: 7000},
		{Version: "1.0", Event: EventFunctionExit, TimestampNS: 8000},
	}

	err := EmitOTELSpans(context.Background(), tracer, events, time.Now())
	if err != nil {
		t.Fatalf("unexpected error with nil payloads: %v", err)
	}
	if len(exporter.GetSpans()) != 0 {
		t.Errorf("expected 0 spans from nil payloads, got %d", len(exporter.GetSpans()))
	}
}

func TestEmitOTELSpans_ParentContext(t *testing.T) {
	exporter, tp := setupTestTracer()
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")

	// Create a parent span to verify eval spans are children of it
	ctx, parentSpan := tracer.Start(context.Background(), "ailang run: test.ail")

	events := []TraceEvent{
		{Version: "1.0", Event: EventFunctionEnter, TimestampNS: 1000, Depth: 1,
			Function: &FunctionEvent{Name: "main"}},
		{Version: "1.0", Event: EventFunctionExit, TimestampNS: 2000, Depth: 1,
			Function: &FunctionEvent{Name: "main", Result: "()"}},
	}

	err := EmitOTELSpans(ctx, tracer, events, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	parentSpan.End()

	spans := exporter.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("expected 2 spans (child + parent), got %d", len(spans))
	}

	// The eval function span should be child of the parent "ailang run" span
	evalSpan := spans[0] // Ends first
	runSpan := spans[1]  // Ends second

	if evalSpan.Parent.SpanID() != runSpan.SpanContext.SpanID() {
		t.Error("eval function span should be child of 'ailang run' span")
	}
}

// assertAttribute checks that an attribute with the given key has the given string value.
func assertAttribute(t *testing.T, attrs []attribute.KeyValue, key string, expected string) {
	t.Helper()
	for _, a := range attrs {
		if string(a.Key) == key {
			if a.Value.AsString() != expected {
				t.Errorf("attribute %s: expected %q, got %q", key, expected, a.Value.AsString())
			}
			return
		}
	}
	t.Errorf("attribute %s not found", key)
}

// assertIntAttribute checks that an attribute with the given key has the given int value.
func assertIntAttribute(t *testing.T, attrs []attribute.KeyValue, key string, expected int) {
	t.Helper()
	for _, a := range attrs {
		if string(a.Key) == key {
			if a.Value.AsInt64() != int64(expected) {
				t.Errorf("attribute %s: expected %d, got %d", key, expected, a.Value.AsInt64())
			}
			return
		}
	}
	t.Errorf("attribute %s not found", key)
}
