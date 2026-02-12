package trace

import "time"

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
}

// NewCollector creates a new trace collector.
func NewCollector() *Collector {
	return &Collector{
		events:         make([]TraceEvent, 0, 256),
		startTime:      time.Now(),
		depth:          0,
		enabled:        true,
		funcEntryTimes: make(map[int]time.Time),
	}
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
	c.events = append(c.events, TraceEvent{
		Version:     traceVersion,
		Event:       EventModuleStart,
		TimestampNS: c.nowNS(),
		Module: &ModuleEvent{
			Name: name,
			Caps: caps,
		},
	})
}

// RecordModuleEnd records module completion.
func (c *Collector) RecordModuleEnd(name string, durationNS int64) {
	if !c.Enabled() {
		return
	}
	c.events = append(c.events, TraceEvent{
		Version:     traceVersion,
		Event:       EventModuleEnd,
		TimestampNS: c.nowNS(),
		Module: &ModuleEvent{
			Name:       name,
			DurationNS: durationNS,
		},
	})
}

// RecordFunctionEnter records function call entry.
func (c *Collector) RecordFunctionEnter(name string, args []string) {
	if !c.Enabled() {
		return
	}
	c.depth++
	c.funcEntryTimes[c.depth] = time.Now()
	c.events = append(c.events, TraceEvent{
		Version:     traceVersion,
		Event:       EventFunctionEnter,
		TimestampNS: c.nowNS(),
		Depth:       c.depth,
		Function: &FunctionEvent{
			Name: name,
			Args: args,
		},
	})
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
	c.events = append(c.events, TraceEvent{
		Version:     traceVersion,
		Event:       EventFunctionExit,
		TimestampNS: c.nowNS(),
		Depth:       c.depth,
		Function: &FunctionEvent{
			Name:       name,
			Result:     result,
			DurationNS: durationNS,
		},
	})
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
	c.events = append(c.events, TraceEvent{
		Version:     traceVersion,
		Event:       EventEffect,
		TimestampNS: c.nowNS(),
		Depth:       c.depth,
		Effect:      &evt,
	})
}

// RecordContractCheck records a contract verification result.
func (c *Collector) RecordContractCheck(kind string, passed bool, msg, location, function string) {
	if !c.Enabled() {
		return
	}
	c.events = append(c.events, TraceEvent{
		Version:     traceVersion,
		Event:       EventContractCheck,
		TimestampNS: c.nowNS(),
		Depth:       c.depth,
		Contract: &ContractEvent{
			Kind:     kind,
			Passed:   passed,
			Message:  msg,
			Location: location,
			Function: function,
		},
	})
}

// RecordBudgetDelta records a budget state change after an effect invocation.
func (c *Collector) RecordBudgetDelta(effect string, used, limit, remaining, physical int) {
	if !c.Enabled() {
		return
	}
	c.events = append(c.events, TraceEvent{
		Version:     traceVersion,
		Event:       EventBudgetDelta,
		TimestampNS: c.nowNS(),
		Depth:       c.depth,
		Budget: &BudgetEvent{
			Effect:    effect,
			Used:      used,
			Limit:     limit,
			Remaining: remaining,
			Physical:  physical,
		},
	})
}

// RecordError records an error event.
func (c *Collector) RecordError(msg, location string) {
	if !c.Enabled() {
		return
	}
	c.events = append(c.events, TraceEvent{
		Version:     traceVersion,
		Event:       EventError,
		TimestampNS: c.nowNS(),
		Depth:       c.depth,
		Error: &ErrorEvent{
			Message:  msg,
			Location: location,
		},
	})
}

// BaseTime returns the collector's creation time.
// Used by EmitOTELSpans to reconstruct absolute timestamps.
func (c *Collector) BaseTime() time.Time {
	if c == nil {
		return time.Time{}
	}
	return c.startTime
}

// nowNS returns nanoseconds since collector creation.
func (c *Collector) nowNS() int64 {
	return time.Since(c.startTime).Nanoseconds()
}
