// Package types provides type inference debugging infrastructure.
//
// TypeDebugSink is an orthogonal debug architecture that avoids scattered
// "if debug { ... }" statements throughout the type checker. Instead, events
// are emitted to a sink interface, with NoOpDebugSink providing zero overhead
// in production and VerboseDebugSink collecting events for formatted output.
//
// Usage:
//
//	// Production (zero overhead)
//	tc := NewCoreTypeChecker()
//	tc.DebugSink = NoOpDebugSink{}
//
//	// Debug mode (collects events)
//	sink := NewVerboseDebugSink()
//	tc.DebugSink = sink
//	// ... run type checking ...
//	for _, event := range sink.Events() {
//	    fmt.Println(event)
//	}
package types

// TypeDebugSink receives events during type inference for debugging.
// Implementations should be designed for zero overhead when not actively
// collecting events (NoOpDebugSink) or collecting structured events
// for later formatting (VerboseDebugSink).
type TypeDebugSink interface {
	// OnFreshTypeVar is called when a fresh type variable is created.
	// The tv parameter is the type variable (typically *TVar2).
	OnFreshTypeVar(tv Type, nodeID uint64, origin OriginKind)

	// OnUnify is called when two types are unified.
	OnUnify(left, right Type, result Type, nodeID uint64)

	// OnSubstitute is called when a type variable is substituted.
	// The tv parameter is the type variable being substituted.
	OnSubstitute(tv Type, resolved Type)

	// OnDefault is called when a type variable is defaulted.
	// The tv parameter is the type variable being defaulted.
	OnDefault(tv Type, defaulted Type, reason string)

	// OnConstraintAdd is called when a constraint is added.
	OnConstraintAdd(className string, ty Type, nodeID uint64)

	// OnConstraintResolve is called when a constraint is resolved.
	OnConstraintResolve(className string, ty Type, method string, nodeID uint64)
}

// DebugEventKind identifies the type of debug event.
type DebugEventKind int

const (
	EventFreshTypeVar DebugEventKind = iota
	EventUnify
	EventSubstitute
	EventDefault
	EventConstraintAdd
	EventConstraintResolve
)

// String returns a human-readable name for the event kind.
func (k DebugEventKind) String() string {
	switch k {
	case EventFreshTypeVar:
		return "fresh_type_var"
	case EventUnify:
		return "unify"
	case EventSubstitute:
		return "substitute"
	case EventDefault:
		return "default"
	case EventConstraintAdd:
		return "constraint_add"
	case EventConstraintResolve:
		return "constraint_resolve"
	default:
		return "unknown"
	}
}

// DebugEvent captures a single type inference event for later formatting.
type DebugEvent struct {
	Kind      DebugEventKind
	NodeID    uint64
	TypeVar   Type   // For fresh/substitute/default events (typically *TVar2)
	Left      Type   // For unify events
	Right     Type   // For unify events
	Result    Type   // For unify events, or resolved type for substitute
	Defaulted Type   // For default events
	Reason    string // For default events
	ClassName string // For constraint events
	Method    string // For constraint resolve events
	Origin    OriginKind
}

// NoOpDebugSink is a debug sink that does nothing.
// It provides zero overhead in production - all methods are empty and inlined.
type NoOpDebugSink struct{}

// Compile-time check that NoOpDebugSink implements TypeDebugSink
var _ TypeDebugSink = NoOpDebugSink{}

func (NoOpDebugSink) OnFreshTypeVar(Type, uint64, OriginKind)          {}
func (NoOpDebugSink) OnUnify(Type, Type, Type, uint64)                 {}
func (NoOpDebugSink) OnSubstitute(Type, Type)                          {}
func (NoOpDebugSink) OnDefault(Type, Type, string)                     {}
func (NoOpDebugSink) OnConstraintAdd(string, Type, uint64)             {}
func (NoOpDebugSink) OnConstraintResolve(string, Type, string, uint64) {}

// VerboseDebugSink collects debug events for later formatting.
// Events are stored in order and can be retrieved via Events().
type VerboseDebugSink struct {
	events []DebugEvent
}

// Compile-time check that VerboseDebugSink implements TypeDebugSink
var _ TypeDebugSink = (*VerboseDebugSink)(nil)

// NewVerboseDebugSink creates a new verbose debug sink.
func NewVerboseDebugSink() *VerboseDebugSink {
	return &VerboseDebugSink{
		events: make([]DebugEvent, 0, 64), // Pre-allocate for typical programs
	}
}

// Events returns all collected debug events.
func (s *VerboseDebugSink) Events() []DebugEvent {
	return s.events
}

// Clear removes all collected events.
func (s *VerboseDebugSink) Clear() {
	s.events = s.events[:0]
}

func (s *VerboseDebugSink) OnFreshTypeVar(tv Type, nodeID uint64, origin OriginKind) {
	s.events = append(s.events, DebugEvent{
		Kind:    EventFreshTypeVar,
		TypeVar: tv,
		NodeID:  nodeID,
		Origin:  origin,
	})
}

func (s *VerboseDebugSink) OnUnify(left, right Type, result Type, nodeID uint64) {
	s.events = append(s.events, DebugEvent{
		Kind:   EventUnify,
		Left:   left,
		Right:  right,
		Result: result,
		NodeID: nodeID,
	})
}

func (s *VerboseDebugSink) OnSubstitute(tv Type, resolved Type) {
	s.events = append(s.events, DebugEvent{
		Kind:    EventSubstitute,
		TypeVar: tv,
		Result:  resolved,
	})
}

func (s *VerboseDebugSink) OnDefault(tv Type, defaulted Type, reason string) {
	s.events = append(s.events, DebugEvent{
		Kind:      EventDefault,
		TypeVar:   tv,
		Defaulted: defaulted,
		Reason:    reason,
	})
}

func (s *VerboseDebugSink) OnConstraintAdd(className string, ty Type, nodeID uint64) {
	s.events = append(s.events, DebugEvent{
		Kind:      EventConstraintAdd,
		ClassName: className,
		Result:    ty,
		NodeID:    nodeID,
	})
}

func (s *VerboseDebugSink) OnConstraintResolve(className string, ty Type, method string, nodeID uint64) {
	s.events = append(s.events, DebugEvent{
		Kind:      EventConstraintResolve,
		ClassName: className,
		Result:    ty,
		Method:    method,
		NodeID:    nodeID,
	})
}
