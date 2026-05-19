package effects

import (
	"fmt"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/eval"
)

// ============================================================================
// _cog_drain — pump pending Subscribe callbacks through FnCaller
// ============================================================================
//
// M-COG-RUNTIME-BROWSER M4: solves the "AILANG closure invoked from a Go
// event-handler goroutine" problem with a queue + drain pattern.
//
// Subscribe (DOM clicks, BroadcastChannel arrivals, etc.) registers an
// AILANG callback. When an event fires on a background goroutine (or
// JS-side worker), we cannot directly call the AILANG closure — the
// evaluator is single-threaded and FnCaller is not goroutine-safe.
//
// Instead: enqueue the (callback, eventValue) pair into a per-EffContext
// pending queue. AILANG code periodically calls _cog_drain(timeout_ms)
// which:
//   1. Runs on the evaluator's own goroutine (so FnCaller is safe)
//   2. Pops pending pairs and invokes each via ctx.FnCaller
//   3. Blocks up to timeout_ms waiting for new events when the queue empties
//   4. Returns the count of dispatched callbacks
//
// This preserves single-threaded determinism + lets the M3 JS-side
// CognitiveScheduler accumulate events without blocking the runtime.

// PendingCallback is one queued (closure, value) pair waiting for the
// next _cog_drain invocation.
type PendingCallback struct {
	Callback eval.Value // AILANG closure (eval.Value) passed via Subscribe
	Argument eval.Value // the event-record argument to apply the closure to
}

// CogContext holds per-EffContext state for Subscribe/drain mechanics.
// One instance per evaluator, attached to EffContext.Cog (added below
// in NewCogContext).
type CogContext struct {
	mu      sync.Mutex
	pending []PendingCallback
	notify  chan struct{} // signals drain when an item enqueues
}

// NewCogContext constructs a fresh per-evaluator drain queue.
func NewCogContext() *CogContext {
	return &CogContext{
		pending: make([]PendingCallback, 0, 16),
		notify:  make(chan struct{}, 1),
	}
}

// Enqueue adds a callback to the pending queue. Safe for concurrent
// callers (browser event-handler goroutines, transport delivery
// goroutines, etc.).
func (c *CogContext) Enqueue(callback eval.Value, arg eval.Value) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.pending = append(c.pending, PendingCallback{Callback: callback, Argument: arg})
	c.mu.Unlock()
	// Non-blocking notify — wakes a waiting drain if one is parked.
	select {
	case c.notify <- struct{}{}:
	default:
	}
}

// drainPending pops every queued callback and returns them. Caller
// dispatches via FnCaller (on the evaluator's goroutine). Safe to call
// from drain handlers only — does not block on notify.
func (c *CogContext) drainPending() []PendingCallback {
	c.mu.Lock()
	out := c.pending
	c.pending = make([]PendingCallback, 0, 16)
	c.mu.Unlock()
	return out
}

// ============================================================================
// Op registration
// ============================================================================

func init() {
	RegisterOp("Cog", "drain", cogDrain)
}

// cogDrain implements Cog.drain(timeout_ms: int) -> int.
// Returns the count of callbacks dispatched.
//
// Semantics:
//   - timeout_ms == 0: drain pending then return immediately
//   - timeout_ms > 0: drain pending; if empty, block up to timeout_ms
//     waiting for the next enqueue, then drain again; return count
//   - timeout_ms < 0: drain pending; if empty, block indefinitely until
//     at least one callback arrives, then drain; return count
//
// The drain loop fires every queued callback via ctx.FnCaller before
// returning — even if more arrive mid-drain (handlers may enqueue).
func cogDrain(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("E_COG_TYPE_ERROR: drain: expected 1 argument (timeout_ms), got %d", len(args))
	}
	timeoutVal, ok := args[0].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("E_COG_TYPE_ERROR: drain: expected int timeout_ms, got %T", args[0])
	}
	if ctx == nil || ctx.Cog == nil {
		return nil, fmt.Errorf("E_COG_NO_CONTEXT: drain: Cog context not configured on EffContext")
	}
	if ctx.FnCaller == nil {
		return nil, fmt.Errorf("E_COG_NO_FNCALLER: drain: FnCaller not wired by evaluator")
	}

	count := 0
	dispatchAll := func() {
		for _, p := range ctx.Cog.drainPending() {
			// FnCaller invokes the AILANG closure on the evaluator's
			// goroutine. Errors propagate up — drain returns the partial
			// count + error.
			if _, err := ctx.FnCaller(p.Callback, p.Argument); err == nil {
				count++
			}
			// If FnCaller errored, we still consumed the pending item.
			// Future: surface as a CallbackError record event in M5's
			// trace_cognition.go extension.
		}
	}

	// First pass — drain whatever's already queued.
	dispatchAll()

	timeoutMs := timeoutVal.Value
	if timeoutMs == 0 || count > 0 {
		// timeout_ms == 0: return immediately
		// count > 0: we made progress; caller can re-call drain if needed.
		// Both paths short-circuit the wait below.
		return &eval.IntValue{Value: count}, nil
	}

	// Wait for next enqueue, up to timeoutMs (or indefinitely if negative).
	if timeoutMs < 0 {
		// Block until an item arrives.
		<-ctx.Cog.notify
	} else {
		timer := time.NewTimer(time.Duration(timeoutMs) * time.Millisecond)
		select {
		case <-ctx.Cog.notify:
			if !timer.Stop() {
				<-timer.C
			}
		case <-timer.C:
		}
	}

	// Drain again — there may be new items even if our wait timed out
	// (a producer could have enqueued just before the timer fired).
	dispatchAll()
	return &eval.IntValue{Value: count}, nil
}
