package effects

// M-BUDGET-SCOPING-BUG: Hierarchical per-invocation budget frames.
//
// This replaces the old cumulative-across-call-chain budget semantics (Merge /
// Reset / ChargeSemanticOnly / PopScopeAndChargeCaller) with a per-execution
// stack of per-invocation frames.
//
// Normative semantics (design_docs/planned/v0_29_0/m-budget-scoping-bug.md):
//
//   - Push a frame on entry to any function whose signature carries an
//     @limit/@min annotation. Frames are per-invocation, not per-function-name:
//     each recursive call pushes its own independent frame.
//   - Pop the frame on EVERY exit (normal, error, unwind). Callers must use a
//     defer so a callee error cannot leave a stale frame on the stack.
//   - Unannotated functions push NO frame; their effects charge whatever
//     annotated ancestor frames are currently active (budgets compose through
//     unannotated intermediates).
//
// Charging rule (bubbling): every matching effect operation charges the current
// (innermost) frame AND all currently-active ancestor frames that constrain that
// effect. A callee's effects therefore count toward an annotated caller's budget,
// and an annotated callee is additionally bounded by its own frame.
//
// @limit deterministic pre-op rule: before an op of cost C, inspect every active
// frame constraining the effect WITHOUT mutating any frame. A frame violates when
// used+C > limit. If one or more frames violate, select the FIRST violating frame
// in innermost-to-outermost stack order, return an error naming that frame,
// increment NOTHING, and do not perform the op. Otherwise atomically increment
// used += C in EVERY matching active frame, then perform the op.

// BudgetFrame records the budget state of a single function invocation.
type BudgetFrame struct {
	FnName string         // Function name for error attribution (may be "")
	limits map[string]int // Effect name → max limit (@limit); absent = no upper bound
	mins   map[string]int // Effect name → min required (@min); absent = no lower bound
	used   map[string]int // Effect name → semantic usage charged to THIS frame
}

// BudgetFrameStack is a per-execution stack of per-invocation budget frames.
//
// It lives on shared enforcement state (referenced from EffContext) so it
// survives the shallow copy performed by WithBudget — the frames are
// per-execution while each frame is per-invocation.
type BudgetFrameStack struct {
	frames []*BudgetFrame
}

// NewBudgetFrameStack creates an empty frame stack.
func NewBudgetFrameStack() *BudgetFrameStack {
	return &BudgetFrameStack{}
}

// Push adds a new per-invocation frame for a function with the given annotations.
//
// limits/mins are copied so the frame is independent of the caller's maps.
func (s *BudgetFrameStack) Push(fnName string, limits, mins map[string]int) {
	f := &BudgetFrame{
		FnName: fnName,
		limits: make(map[string]int, len(limits)),
		mins:   make(map[string]int, len(mins)),
		used:   make(map[string]int),
	}
	for k, v := range limits {
		f.limits[k] = v
	}
	for k, v := range mins {
		f.mins[k] = v
	}
	s.frames = append(s.frames, f)
}

// Pop removes and returns the innermost frame, or nil if the stack is empty.
func (s *BudgetFrameStack) Pop() *BudgetFrame {
	n := len(s.frames)
	if n == 0 {
		return nil
	}
	f := s.frames[n-1]
	s.frames = s.frames[:n-1]
	return f
}

// Depth reports how many frames are currently active.
func (s *BudgetFrameStack) Depth() int {
	return len(s.frames)
}

// Charge applies the deterministic pre-op @limit rule for one effect operation
// of cost 1 against all active frames.
//
// Returns a *BudgetExhaustedError naming the first violating frame
// (innermost-to-outermost) if the op would exceed any active limit; in that case
// NO frame is mutated and the caller must not perform the operation. Otherwise it
// atomically increments used by 1 on every active frame that constrains the
// effect and returns nil.
func (s *BudgetFrameStack) Charge(effect, position string) error {
	const cost = 1

	// Pass 1: inspect every active frame WITHOUT mutating. Select the first
	// violating frame in innermost-to-outermost order.
	for i := len(s.frames) - 1; i >= 0; i-- {
		f := s.frames[i]
		limit, hasLimit := f.limits[effect]
		if !hasLimit {
			continue
		}
		if f.used[effect]+cost > limit {
			// Frame's own semantic count is authoritative for both the
			// semantic and physical fields under per-invocation frames.
			return NewBudgetExhaustedError(effect, limit, f.used[effect], position, f.used[effect])
		}
	}

	// Pass 2: no violation — atomically increment every matching active frame.
	for _, f := range s.frames {
		if _, hasLimit := f.limits[effect]; hasLimit {
			f.used[effect] += cost
		}
	}
	return nil
}

// HasActiveLimit reports whether any active frame constrains the given effect
// with an @limit. Used to decide whether frame-based enforcement applies.
func (s *BudgetFrameStack) HasActiveLimit(effect string) bool {
	for _, f := range s.frames {
		if _, ok := f.limits[effect]; ok {
			return true
		}
	}
	return false
}

// CheckMin verifies the frame met all its @min requirements using its own
// bubbled semantic count. Returns a *BudgetUnderrunError for the first unmet
// minimum, or nil. Called at NORMAL-exit frame pop only (suppressed on error).
func (f *BudgetFrame) CheckMin(position string) error {
	for effect, min := range f.mins {
		if f.used[effect] < min {
			return NewBudgetUnderrunError(effect, min, f.used[effect], position)
		}
	}
	return nil
}

// HasAnnotations reports whether a signature carries any @limit or @min.
func HasAnnotations(limits, mins map[string]int) bool {
	return len(limits) > 0 || len(mins) > 0
}
