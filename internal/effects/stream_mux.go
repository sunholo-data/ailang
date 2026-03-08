package effects

import (
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/sunholo/ailang/internal/eval"
)

// selectEventsLoop is the core multi-source event multiplexer.
//
// M-ASYNC-IO: This implements deterministic priority-ordered dispatch over N event
// sources. When multiple sources have events ready simultaneously, the highest-priority
// source always wins (deterministic). When no events are ready, we block on any source
// using reflect.Select (implementation detail — not semantic basis).
//
// Merge rules (v0.9.0):
//  1. Sources sorted by priority descending. Source list order is default priority.
//  2. Each round: non-blocking check of sources in priority order; first ready wins.
//  3. Same-priority band: round-robin (rotating start index) prevents starvation.
//  4. No source ready: block until any source delivers (nondeterministic arrival, traced).
//  5. Lower-priority sources may starve if higher bands are continuously ready (documented).
//  6. Handler returns false: loop stops immediately.
func selectEventsLoop(
	sources []EventSource,
	handler eval.Value,
	fnCaller func(eval.Value, eval.Value) (eval.Value, error),
	idleTimeout time.Duration,
	maxDuration time.Duration,
) error {
	if len(sources) == 0 {
		return fmt.Errorf("selectEvents: no sources provided")
	}

	// Sort sources by priority descending (highest first)
	sorted := make([]EventSource, len(sources))
	copy(sorted, sources)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Priority() > sorted[j].Priority()
	})

	// Group by priority band for round-robin within bands
	bands := groupByPriority(sorted)

	// Timers
	idleTimer := time.NewTimer(idleTimeout)
	defer idleTimer.Stop()

	maxTimer := time.NewTimer(maxDuration)
	defer maxTimer.Stop()

	for {
		// Phase 1: Non-blocking priority-ordered check
		delivered := false
		for bi := range bands {
			b := &bands[bi]
			n := len(b.sources)
			for i := 0; i < n; i++ {
				idx := (b.rotation + i) % n
				src := b.sources[idx]
				select {
				case evt, ok := <-src.Events():
					if !ok {
						continue // Source closed, skip
					}
					// Reset idle timer
					resetTimer(idleTimer, idleTimeout)

					adtVal := eventToADT(evt)
					shouldContinue, err := callHandlerSafe(fnCaller, handler, adtVal)
					if err != nil {
						deliverErrorAndReturn(fnCaller, handler, "handler error: "+err.Error())
						return nil
					}
					if !shouldContinue {
						return nil
					}

					b.rotation = (idx + 1) % n // Advance round-robin
					delivered = true
					goto nextRound
				default:
					continue
				}
			}
		}

		if delivered {
			goto nextRound
		}

		// Phase 2: No events ready — block on any source (+ timers)
		{
			cases := buildSelectCases(sorted, idleTimer, maxTimer)
			chosen, value, ok := reflect.Select(cases)

			if chosen == len(sorted) {
				// Idle timer fired
				deliverErrorAndReturn(fnCaller, handler,
					fmt.Sprintf("idle timeout after %s", idleTimeout))
				return nil
			}
			if chosen == len(sorted)+1 {
				// Max duration timer fired
				deliverErrorAndReturn(fnCaller, handler,
					fmt.Sprintf("max duration %s exceeded", maxDuration))
				return nil
			}

			if !ok {
				// Source channel closed
				// Check if ALL sources are closed
				allClosed := true
				for _, src := range sorted {
					select {
					case _, open := <-src.Events():
						if open {
							allClosed = false
						}
					default:
						allClosed = false
					}
				}
				if allClosed {
					return nil
				}
				continue // Some sources still open
			}

			// Reset idle timer
			resetTimer(idleTimer, idleTimeout)

			evt := value.Interface().(streamEvent)
			adtVal := eventToADT(evt)
			shouldContinue, err := callHandlerSafe(fnCaller, handler, adtVal)
			if err != nil {
				deliverErrorAndReturn(fnCaller, handler, "handler error: "+err.Error())
				return nil
			}
			if !shouldContinue {
				return nil
			}
		}

	nextRound:
	}
}

// groupByPriority groups sorted sources into priority bands.
func groupByPriority(sorted []EventSource) []struct {
	sources  []EventSource
	rotation int
} {
	type band struct {
		sources  []EventSource
		rotation int
	}
	var bands []band
	for _, src := range sorted {
		if len(bands) == 0 || bands[len(bands)-1].sources[0].Priority() != src.Priority() {
			bands = append(bands, band{sources: []EventSource{src}})
		} else {
			bands[len(bands)-1].sources = append(bands[len(bands)-1].sources, src)
		}
	}
	// Convert to the return type
	result := make([]struct {
		sources  []EventSource
		rotation int
	}, len(bands))
	for i, b := range bands {
		result[i].sources = b.sources
		result[i].rotation = b.rotation
	}
	return result
}

// buildSelectCases creates reflect.SelectCase slice for blocking on all sources + timers.
func buildSelectCases(sources []EventSource, idleTimer, maxTimer *time.Timer) []reflect.SelectCase {
	cases := make([]reflect.SelectCase, len(sources)+2)
	for i, src := range sources {
		cases[i] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(src.Events()),
		}
	}
	// Idle timer
	cases[len(sources)] = reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(idleTimer.C),
	}
	// Max duration timer
	cases[len(sources)+1] = reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(maxTimer.C),
	}
	return cases
}

// deliverErrorAndReturn delivers a StreamError(Timeout(...)) event to the handler.
func deliverErrorAndReturn(fnCaller func(eval.Value, eval.Value) (eval.Value, error), handler eval.Value, msg string) {
	errEvt := eventToADT(streamEvent{
		kind:    "error",
		errType: "Timeout",
		text:    msg,
	})
	_, _ = callHandlerSafe(fnCaller, handler, errEvt)
}

// resetTimer safely resets a timer.
func resetTimer(t *time.Timer, d time.Duration) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(d)
}
