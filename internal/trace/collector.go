package trace

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

const traceVersion = "1.0"

// Collector accumulates trace events during AILANG program execution.
// Designed for single-threaded use within one evaluation.
// Set on EffContext.Trace before execution begins.
type Collector struct {
	events    []TraceEvent
	startTime time.Time
	depth     int
	enabled   bool

	// Track function entry times for duration calculation
	funcEntryTimes map[int]time.Time // depth -> entry time

	// OnEvent is called for each trace event as it is recorded.
	// Used by WASM to stream events to JavaScript in real-time.
	// Nil means no streaming (events are only accumulated in events[]).
	OnEvent func(TraceEvent)

	// OTEL-compatible span IDs (M-WASM-TRACE)
	traceID   string   // consistent across all events in one execution
	spanStack []string // stack of active span IDs (push on enter, pop on exit)
}

// NewCollector creates a new trace collector.
func NewCollector() *Collector {
	return &Collector{
		events:         make([]TraceEvent, 0, 256),
		startTime:      time.Now(),
		depth:          0,
		enabled:        true,
		funcEntryTimes: make(map[int]time.Time),
		traceID:        generateID(16), // 32-hex-char trace ID (W3C trace-context)
		spanStack:      make([]string, 0, 16),
	}
}

// generateID returns a random hex string of the given byte length (2*n hex chars).
func generateID(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// pushSpan generates a new span ID, pushes it onto the stack, and returns it.
func (c *Collector) pushSpan() string {
	id := generateID(8) // 16-hex-char span ID
	c.spanStack = append(c.spanStack, id)
	return id
}

// popSpan removes and returns the top span ID from the stack.
// Returns empty string if stack is empty.
func (c *Collector) popSpan() string {
	if len(c.spanStack) == 0 {
		return ""
	}
	id := c.spanStack[len(c.spanStack)-1]
	c.spanStack = c.spanStack[:len(c.spanStack)-1]
	return id
}

// currentSpanID returns the span ID at the top of the stack (current parent).
func (c *Collector) currentSpanID() string {
	if len(c.spanStack) == 0 {
		return ""
	}
	return c.spanStack[len(c.spanStack)-1]
}

// Enabled returns whether trace collection is active.
func (c *Collector) Enabled() bool {
	return c != nil && c.enabled
}

// Events returns all collected events.
func (c *Collector) Events() []TraceEvent {
	if c == nil {
		return nil
	}
	return c.events
}

// RecordModuleStart records module entry.
func (c *Collector) RecordModuleStart(name string, caps []string) {
	if !c.Enabled() {
		return
	}
	parentSpan := c.currentSpanID()
	spanID := c.pushSpan()
	evt := TraceEvent{
		Version:      traceVersion,
		Event:        EventModuleStart,
		TimestampNS:  c.nowNS(),
		TraceID:      c.traceID,
		SpanID:       spanID,
		ParentSpanID: parentSpan,
		Module: &ModuleEvent{
			Name: name,
			Caps: caps,
		},
	}
	c.events = append(c.events, evt)
	c.notify(evt)
}

// RecordModuleEnd records module completion.
func (c *Collector) RecordModuleEnd(name string, durationNS int64) {
	if !c.Enabled() {
		return
	}
	spanID := c.popSpan()
	evt := TraceEvent{
		Version:     traceVersion,
		Event:       EventModuleEnd,
		TimestampNS: c.nowNS(),
		TraceID:     c.traceID,
		SpanID:      spanID,
		Module: &ModuleEvent{
			Name:       name,
			DurationNS: durationNS,
		},
	}
	c.events = append(c.events, evt)
	c.notify(evt)
}

// RecordFunctionEnter records function call entry.
func (c *Collector) RecordFunctionEnter(name string, args []string) {
	if !c.Enabled() {
		return
	}
	c.depth++
	c.funcEntryTimes[c.depth] = time.Now()
	parentSpan := c.currentSpanID()
	spanID := c.pushSpan()
	evt := TraceEvent{
		Version:      traceVersion,
		Event:        EventFunctionEnter,
		TimestampNS:  c.nowNS(),
		Depth:        c.depth,
		TraceID:      c.traceID,
		SpanID:       spanID,
		ParentSpanID: parentSpan,
		Function: &FunctionEvent{
			Name: name,
			Args: args,
		},
	}
	c.events = append(c.events, evt)
	c.notify(evt)
}

// RecordFunctionExit records function return.
func (c *Collector) RecordFunctionExit(name string, result string) {
	if !c.Enabled() {
		return
	}
	var durationNS int64
	if entry, ok := c.funcEntryTimes[c.depth]; ok {
		durationNS = time.Since(entry).Nanoseconds()
		delete(c.funcEntryTimes, c.depth)
	}
	spanID := c.popSpan()
	evt := TraceEvent{
		Version:     traceVersion,
		Event:       EventFunctionExit,
		TimestampNS: c.nowNS(),
		Depth:       c.depth,
		TraceID:     c.traceID,
		SpanID:      spanID,
		Function: &FunctionEvent{
			Name:       name,
			Result:     result,
			DurationNS: durationNS,
		},
	}
	c.events = append(c.events, evt)
	c.notify(evt)
	if c.depth > 0 {
		c.depth--
	}
}

// RecordEffect records an effect invocation.
func (c *Collector) RecordEffect(effectName, opName string, args []string, result string) {
	if !c.Enabled() {
		return
	}
	evt := EffectEvent{
		EffectName: effectName,
		OpName:     opName,
		Args:       args,
		Result:     result,
	}
	// Flag known non-deterministic operations for replay tolerance
	if IsNonDeterministic(effectName, opName) {
		f := false
		evt.Deterministic = &f
	}
	traceEvt := TraceEvent{
		Version:     traceVersion,
		Event:       EventEffect,
		TimestampNS: c.nowNS(),
		Depth:       c.depth,
		TraceID:     c.traceID,
		SpanID:      c.currentSpanID(),
		Effect:      &evt,
	}
	c.events = append(c.events, traceEvt)
	c.notify(traceEvt)
}

// RecordAIEffect records an AI effect invocation with optional routing metadata.
//
// Behaves like RecordEffect with EffectName="AI" but additionally attaches
// the supplied ResolvedRoute (which may be nil for non-routed AI calls).
// Used by the AI effect ops to surface OpenRouter routing decisions in the
// trace stream.
func (c *Collector) RecordAIEffect(opName string, args []string, result string, route *ResolvedRoute) {
	if !c.Enabled() {
		return
	}
	evt := EffectEvent{
		EffectName: "AI",
		OpName:     opName,
		Args:       args,
		Result:     result,
		Route:      route,
	}
	if IsNonDeterministic("AI", opName) {
		f := false
		evt.Deterministic = &f
	}
	traceEvt := TraceEvent{
		Version:     traceVersion,
		Event:       EventEffect,
		TimestampNS: c.nowNS(),
		Depth:       c.depth,
		TraceID:     c.traceID,
		SpanID:      c.currentSpanID(),
		Effect:      &evt,
	}
	c.events = append(c.events, traceEvt)
	c.notify(traceEvt)
}

// RecordContractCheck records a contract verification result.
func (c *Collector) RecordContractCheck(kind string, passed bool, msg, location, function string) {
	if !c.Enabled() {
		return
	}
	evt := TraceEvent{
		Version:     traceVersion,
		Event:       EventContractCheck,
		TimestampNS: c.nowNS(),
		Depth:       c.depth,
		TraceID:     c.traceID,
		SpanID:      c.currentSpanID(),
		Contract: &ContractEvent{
			Kind:     kind,
			Passed:   passed,
			Message:  msg,
			Location: location,
			Function: function,
		},
	}
	c.events = append(c.events, evt)
	c.notify(evt)
}

// RecordBudgetDelta records a budget state change after an effect invocation.
func (c *Collector) RecordBudgetDelta(effect string, used, limit, remaining, physical int) {
	if !c.Enabled() {
		return
	}
	evt := TraceEvent{
		Version:     traceVersion,
		Event:       EventBudgetDelta,
		TimestampNS: c.nowNS(),
		Depth:       c.depth,
		TraceID:     c.traceID,
		SpanID:      c.currentSpanID(),
		Budget: &BudgetEvent{
			Effect:    effect,
			Used:      used,
			Limit:     limit,
			Remaining: remaining,
			Physical:  physical,
		},
	}
	c.events = append(c.events, evt)
	c.notify(evt)
}

// RecordError records an error event.
func (c *Collector) RecordError(msg, location string) {
	if !c.Enabled() {
		return
	}
	evt := TraceEvent{
		Version:     traceVersion,
		Event:       EventError,
		TimestampNS: c.nowNS(),
		Depth:       c.depth,
		TraceID:     c.traceID,
		SpanID:      c.currentSpanID(),
		Error: &ErrorEvent{
			Message:  msg,
			Location: location,
		},
	}
	c.events = append(c.events, evt)
	c.notify(evt)
}

// BaseTime returns the collector's creation time.
// Used by EmitOTELSpans to reconstruct absolute timestamps.
func (c *Collector) BaseTime() time.Time {
	if c == nil {
		return time.Time{}
	}
	return c.startTime
}

// notify dispatches an event to the OnEvent callback if set.
func (c *Collector) notify(evt TraceEvent) {
	if c.OnEvent != nil {
		c.OnEvent(evt)
	}
}

// nowNS returns nanoseconds since collector creation.
func (c *Collector) nowNS() int64 {
	return time.Since(c.startTime).Nanoseconds()
}
