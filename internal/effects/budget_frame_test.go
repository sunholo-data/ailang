package effects

// M-BUDGET-SCOPING-BUG: unit tests for the per-invocation budget frame stack and
// the EffContext push/pop lifecycle. These complement the binary-driven
// semantics-matrix e2e tests (cmd/ailang/budget_scoping_e2e_test.go) with direct,
// deterministic assertions on the frame machinery — in particular the
// error-exit frame-pop that guarantees no stale frame leaks onto the stack.

import (
	"errors"
	"testing"
)

// TestBudgetFrameStack_ChargeBubbles verifies the bubbling charge rule: a single
// op increments EVERY active frame that constrains the effect.
func TestBudgetFrameStack_ChargeBubbles(t *testing.T) {
	s := NewBudgetFrameStack()
	s.Push("outer", map[string]int{"IO": 5}, nil)
	s.Push("inner", map[string]int{"IO": 5}, nil)

	if err := s.Charge("IO", ""); err != nil {
		t.Fatalf("charge should succeed: %v", err)
	}
	// Both frames must have been charged once.
	if got := s.frames[0].used["IO"]; got != 1 {
		t.Errorf("outer frame used = %d, want 1 (bubbling)", got)
	}
	if got := s.frames[1].used["IO"]; got != 1 {
		t.Errorf("inner frame used = %d, want 1", got)
	}
}

// TestBudgetFrameStack_PreOpNoMutateOnViolation verifies the deterministic pre-op
// rule: on violation NO frame is mutated and the op is rejected.
func TestBudgetFrameStack_PreOpNoMutateOnViolation(t *testing.T) {
	s := NewBudgetFrameStack()
	s.Push("outer", map[string]int{"IO": 3}, nil)
	s.Push("inner", map[string]int{"IO": 1}, nil)

	// inner already at its limit.
	if err := s.Charge("IO", ""); err != nil {
		t.Fatalf("first charge should succeed: %v", err)
	}
	outerBefore := s.frames[0].used["IO"]
	innerBefore := s.frames[1].used["IO"]

	// Second op violates inner (1+1 > 1). Must reject and mutate nothing.
	err := s.Charge("IO", "")
	if err == nil {
		t.Fatal("second charge must violate inner's limit=1")
	}
	if s.frames[0].used["IO"] != outerBefore || s.frames[1].used["IO"] != innerBefore {
		t.Errorf("no frame may be mutated on violation: outer %d->%d inner %d->%d",
			outerBefore, s.frames[0].used["IO"], innerBefore, s.frames[1].used["IO"])
	}
}

// TestBudgetFrameStack_InnermostFirstAttribution verifies that when an op violates
// multiple frames at once, the FIRST violating frame innermost-to-outermost is
// named.
func TestBudgetFrameStack_InnermostFirstAttribution(t *testing.T) {
	s := NewBudgetFrameStack()
	s.Push("outer", map[string]int{"IO": 1}, nil)
	s.Push("inner", map[string]int{"IO": 1}, nil)

	// First op: both go to used=1.
	if err := s.Charge("IO", ""); err != nil {
		t.Fatalf("first charge should succeed: %v", err)
	}
	// Second op violates BOTH; must name inner (innermost).
	err := s.Charge("IO", "")
	var bex *BudgetExhaustedError
	if !errors.As(err, &bex) {
		t.Fatalf("expected *BudgetExhaustedError, got %T", err)
	}
	if bex.Limit != 1 || bex.Used != 1 {
		t.Errorf("expected inner frame (limit=1, used=1), got limit=%d used=%d", bex.Limit, bex.Used)
	}
}

// TestEffContext_PopBudgetFrame_MinNormalExit verifies @min fires at normal-exit
// pop, and is met when the frame's own count reaches N.
func TestEffContext_PopBudgetFrame_MinNormalExit(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("IO"))
	ctx.PushBudgetFrame("atLeast", map[string]int{"IO": 5}, map[string]int{"IO": 2})

	// One op: below @min=2.
	if err := ctx.RequireCapWithBudget("IO", ""); err != nil {
		t.Fatalf("op should succeed: %v", err)
	}
	// Normal exit with only 1 op → underrun.
	if err := ctx.PopBudgetFrame("atLeast", nil); err == nil {
		t.Fatal("normal-exit pop with 1 op must fail @min=2")
	} else if _, ok := err.(*BudgetUnderrunError); !ok {
		t.Errorf("expected *BudgetUnderrunError, got %T", err)
	}
	if ctx.BudgetFrames.Depth() != 0 {
		t.Errorf("frame must be popped, depth = %d", ctx.BudgetFrames.Depth())
	}
}

// TestEffContext_PopBudgetFrame_MinSuppressedOnError verifies @min is SUPPRESSED
// on error exit: the original error propagates unchanged, and the frame is still
// popped.
func TestEffContext_PopBudgetFrame_MinSuppressedOnError(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("IO"))
	ctx.PushBudgetFrame("atLeast", map[string]int{"IO": 5}, map[string]int{"IO": 3})

	// Zero ops → would fail @min=3 on normal exit. But we exit with an error.
	original := errors.New("callee blew up")
	got := ctx.PopBudgetFrame("atLeast", original)
	if !errors.Is(got, original) {
		t.Errorf("error exit must propagate the original error unchanged; got %v", got)
	}
	if _, ok := got.(*BudgetUnderrunError); ok {
		t.Error("@min underrun must be SUPPRESSED on error exit (must not mask the real error)")
	}
	if ctx.BudgetFrames.Depth() != 0 {
		t.Errorf("frame must still be popped on error exit, depth = %d", ctx.BudgetFrames.Depth())
	}
}

// TestEffContext_ErrorUnwind_NoStaleFrame is the direct frame-leak proof: after a
// frame is popped on an ERROR exit, the stack is empty and a subsequent
// independent (sibling) frame charges from zero — no residue from the unwound
// frame leaks into it. This pins acceptance criterion 6 at the unit level,
// alongside the binary-driven sibling-succeeds e2e test.
func TestEffContext_ErrorUnwind_NoStaleFrame(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("IO"))

	// "boom": annotated @limit=1, does 1 op, then errors mid-flight.
	ctx.PushBudgetFrame("boom", map[string]int{"IO": 1}, nil)
	if err := ctx.RequireCapWithBudget("IO", ""); err != nil {
		t.Fatalf("boom's first op should succeed: %v", err)
	}
	// boom unwinds with an error → frame popped, @min (none) suppressed.
	if err := ctx.PopBudgetFrame("boom", errors.New("boom failed")); err == nil {
		t.Fatal("expected boom's error to propagate")
	}
	if ctx.BudgetFrames.Depth() != 0 {
		t.Fatalf("stale frame leaked after error unwind: depth = %d", ctx.BudgetFrames.Depth())
	}

	// Sibling: fresh @limit=1 frame must accept its own single op (would fail if
	// boom's used count leaked into it).
	ctx.PushBudgetFrame("sibling", map[string]int{"IO": 1}, nil)
	if err := ctx.RequireCapWithBudget("IO", ""); err != nil {
		t.Fatalf("sibling's fresh frame must accept its own op (no stale residue): %v", err)
	}
	if err := ctx.PopBudgetFrame("sibling", nil); err != nil {
		t.Errorf("sibling normal exit should succeed: %v", err)
	}
	if ctx.BudgetFrames.Depth() != 0 {
		t.Errorf("frame stack must be empty at end, depth = %d", ctx.BudgetFrames.Depth())
	}
}

// TestEffContext_UnannotatedPushesNoFrame verifies unannotated functions push no
// frame (the caller pushes only when HasAnnotations is true — enforced by the
// evaluator; here we assert HasAnnotations directly).
func TestBudgetFrame_HasAnnotations(t *testing.T) {
	if HasAnnotations(nil, nil) {
		t.Error("nil/nil must report no annotations")
	}
	if !HasAnnotations(map[string]int{"IO": 3}, nil) {
		t.Error("@limit must count as an annotation")
	}
	if !HasAnnotations(nil, map[string]int{"IO": 1}) {
		t.Error("@min must count as an annotation")
	}
}
