package trace

// Label scopes (M-TRACE-LABEL-AWARE, design_docs/planned/v0_36_0/m-trace-label-aware.md).
//
// IFC labels are erased before runtime, and the collector only ever receives
// strings, so it cannot tell a <secret> value from any other. The static IFC
// pass can: it reports every call site at which a labelled value is passed,
// captured or produced, and the elaborated Core App carries that label to the
// evaluator. The evaluator pushes a label scope for the DYNAMIC EXTENT of such
// a call, and inside it the collector records no value at all — function
// arguments and results, effect arguments and results — only the structure
// (names, arity, span tree), with each value replaced by <redacted:LABEL>.
//
// Redacting the whole extent rather than individual arguments is what closes
// the higher-order and polymorphic cases (the design's open questions 2 and
// 3): a label-transparent function such as `forward(x: string) = inner(x)`
// does not know statically that its caller handed it a secret, but it runs
// inside the caller's scope. The cost is over-redaction of values computed
// from unlabelled inputs inside a labelled call — the safe direction.
//
// The scope is a stack on the collector, guarded by its mutex. Concurrent
// evaluation sharing one collector can over-redact the other goroutine's
// events while a scope is open; it cannot under-redact its own.

// LabelRedactedMarker is what a value recorded inside a label scope becomes.
func LabelRedactedMarker(label string) string {
	return "<redacted:" + label + ">"
}

// PushLabelScope opens a label scope and returns the function that closes it.
// Scopes nest LIFO. A nil collector returns a no-op so callers need no guard.
func (c *Collector) PushLabelScope(label string) func() {
	if c == nil {
		return func() {}
	}
	c.mu.Lock()
	c.labelScopes = append(c.labelScopes, label)
	depth := len(c.labelScopes)
	c.mu.Unlock()
	return func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		// Truncate to below this scope's own slot: closing an outer scope
		// first (never done by the evaluator) still leaves no scope open
		// that belonged to it.
		if len(c.labelScopes) >= depth {
			c.labelScopes = c.labelScopes[:depth-1]
		}
	}
}

// LabelScope returns the innermost open label, or "" when none is open.
func (c *Collector) LabelScope() string {
	if c == nil {
		return ""
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.labelScopeLocked()
}

func (c *Collector) labelScopeLocked() string {
	if len(c.labelScopes) == 0 {
		return ""
	}
	return c.labelScopes[len(c.labelScopes)-1]
}

// redactLabelScope replaces every value on evt while a label scope is open.
// It runs inside record(), the one path every Record* method takes, so a site
// that rendered a value itself (the AI effects, a raw RecordEffect) is covered
// as well as the ones that go through EffContext.RenderTraceValue. Reports
// whether it applied. Caller holds c.mu.
func (c *Collector) redactLabelScope(evt *TraceEvent) bool {
	label := c.labelScopeLocked()
	if label == "" {
		return false
	}
	marker := LabelRedactedMarker(label)
	redact := func(s string) string {
		if s == "" {
			return s
		}
		return marker
	}
	if evt.Function != nil {
		for i, a := range evt.Function.Args {
			evt.Function.Args[i] = redact(a)
		}
		evt.Function.Result = redact(evt.Function.Result)
	}
	if evt.Effect != nil {
		for i, a := range evt.Effect.Args {
			evt.Effect.Args[i] = redact(a)
		}
		evt.Effect.Result = redact(evt.Effect.Result)
	}
	return true
}
