package eval

// TraceLabelScoper is implemented by effect contexts that can open an IFC
// label scope on their trace collector (M-TRACE-LABEL-AWARE). Inside a scope
// the collector records structure but no value.
//
// PushTraceLabelScope returns the function that closes the scope, or nil when
// no enabled collector is attached (then there is nothing to redact).
type TraceLabelScoper interface {
	PushTraceLabelScope(label string) func()
}

// enterTraceLabelScope opens a label scope for a call the static IFC pass
// marked (core.App.TraceRedact), returning its closer or nil.
//
// Called only for marked calls, so an unmarked program pays nothing; a marked
// call with tracing off pays one interface assertion.
func (e *CoreEvaluator) enterTraceLabelScope(label string) func() {
	if label == "" {
		return nil
	}
	s, ok := e.effContext.(TraceLabelScoper)
	if !ok {
		return nil
	}
	return s.PushTraceLabelScope(label)
}
