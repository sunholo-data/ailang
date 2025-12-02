package effects

import (
	"fmt"

	"github.com/sunholo/ailang/internal/eval"
)

// DebugContext collects debug data during execution
//
// The debug context is the host-side handler for the Debug effect.
// It accumulates logs and assertions during AILANG execution, with
// collection only available from the host (not from AILANG code).
//
// This implements the "write-only from AILANG" design:
//   - AILANG code can call Debug.log and Debug.check
//   - Only the host can call Collect() and Reset()
//   - Prevents AILANG code from branching on its own debug state
//
// Note: We use "check" instead of "assert" because "assert" is a reserved keyword
//
// Thread-safety: DebugContext is designed for single-threaded use
// within one evaluation. Create a new context for each step/tick.
type DebugContext struct {
	logs       []LogEntry
	assertions []AssertionResult
	timestamp  int64 // Logical time (host-defined meaning)
}

// LogEntry represents a single log message
type LogEntry struct {
	Message   string
	Location  string // "file.ail:42" (auto-injected by compiler)
	Timestamp int64  // Logical time (host-defined: tick, test index, etc.)
}

// AssertionResult represents the result of a debug assertion
type AssertionResult struct {
	Passed   bool
	Message  string
	Location string // "file.ail:42" (auto-injected by compiler)
}

// DebugOutput is the structured output from debug collection
type DebugOutput struct {
	Logs       []LogEntry
	Assertions []AssertionResult
}

// NewDebugContext creates a new debug context
//
// The context starts empty with timestamp 0. Call SetTimestamp()
// before running AILANG code if you want meaningful timestamps.
func NewDebugContext() *DebugContext {
	return &DebugContext{
		logs:       make([]LogEntry, 0),
		assertions: make([]AssertionResult, 0),
		timestamp:  0,
	}
}

// SetTimestamp sets the logical timestamp for subsequent log entries
//
// The meaning of timestamp is host-defined:
//   - Game engine: frame/tick index
//   - Test runner: test case index
//   - CLI: 0 or monotonic counter
//
// AILANG code MUST NOT rely on timestamp meaning.
func (d *DebugContext) SetTimestamp(t int64) {
	d.timestamp = t
}

// Log adds a log entry to the debug context
//
// This is called by the Debug.log effect operation.
// Location is auto-injected by the compiler.
func (d *DebugContext) Log(msg, location string) {
	d.logs = append(d.logs, LogEntry{
		Message:   msg,
		Location:  location,
		Timestamp: d.timestamp,
	})
}

// Check records an assertion result (formerly Assert)
//
// This is called by the Debug.check effect operation.
// Location is auto-injected by the compiler.
// Assertions are collected, not thrown - the program continues.
// Note: Named "Check" because "assert" is a reserved keyword in AILANG.
func (d *DebugContext) Check(cond bool, msg, location string) {
	d.assertions = append(d.assertions, AssertionResult{
		Passed:   cond,
		Message:  msg,
		Location: location,
	})
}

// Collect returns all accumulated debug data (HOST-ONLY)
//
// This method is NOT exposed to AILANG code. Only the host
// can collect debug output after AILANG execution completes.
func (d *DebugContext) Collect() DebugOutput {
	return DebugOutput{
		Logs:       d.logs,
		Assertions: d.assertions,
	}
}

// Reset clears the debug context for the next step/tick
//
// Call this between ticks/steps to start fresh.
func (d *DebugContext) Reset() {
	d.logs = d.logs[:0]
	d.assertions = d.assertions[:0]
}

// HasFailedAssertions returns true if any assertions failed
func (d *DebugContext) HasFailedAssertions() bool {
	for _, a := range d.assertions {
		if !a.Passed {
			return true
		}
	}
	return false
}

// FailedAssertions returns only the failed assertions
func (d *DebugContext) FailedAssertions() []AssertionResult {
	var failed []AssertionResult
	for _, a := range d.assertions {
		if !a.Passed {
			failed = append(failed, a)
		}
	}
	return failed
}

// init registers Debug effect operations
// Note: We use "check" instead of "assert" because "assert" is a reserved keyword
func init() {
	RegisterOp("Debug", "log", debugLog)
	RegisterOp("Debug", "check", debugCheck)
}

// debugLog implements Debug.log(msg: string) -> ()
//
// Adds a log entry with the given message. Location is extracted
// from a hidden argument (position 1) injected by the compiler.
//
// Parameters:
//   - ctx: Effect context
//   - args: [StringValue (msg), StringValue (location)]
//
// Returns:
//   - UnitValue on success
func debugLog(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("E_DEBUG_TYPE_ERROR: log: expected at least 1 argument, got %d", len(args))
	}

	msg, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_DEBUG_TYPE_ERROR: log: expected string message, got %T", args[0])
	}

	// Location is optional (for backwards compatibility during development)
	location := "unknown"
	if len(args) >= 2 {
		if loc, ok := args[1].(*eval.StringValue); ok {
			location = loc.Value
		}
	}

	// Get or create debug context
	if ctx.Debug == nil {
		ctx.Debug = NewDebugContext()
	}
	ctx.Debug.Log(msg.Value, location)

	return &eval.UnitValue{}, nil
}

// debugCheck implements Debug.check(cond: bool, msg: string) -> ()
//
// Records an assertion result. The assertion is collected, not thrown.
// Location is extracted from a hidden argument (position 2) injected
// by the compiler.
//
// Note: Named "check" because "assert" is a reserved keyword in AILANG.
//
// Parameters:
//   - ctx: Effect context
//   - args: [BoolValue (cond), StringValue (msg), StringValue (location)]
//
// Returns:
//   - UnitValue on success (even if assertion fails)
func debugCheck(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("E_DEBUG_TYPE_ERROR: check: expected at least 2 arguments, got %d", len(args))
	}

	cond, ok := args[0].(*eval.BoolValue)
	if !ok {
		return nil, fmt.Errorf("E_DEBUG_TYPE_ERROR: check: expected bool condition, got %T", args[0])
	}

	msg, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_DEBUG_TYPE_ERROR: check: expected string message, got %T", args[1])
	}

	// Location is optional (for backwards compatibility during development)
	location := "unknown"
	if len(args) >= 3 {
		if loc, ok := args[2].(*eval.StringValue); ok {
			location = loc.Value
		}
	}

	// Get or create debug context
	if ctx.Debug == nil {
		ctx.Debug = NewDebugContext()
	}
	ctx.Debug.Check(cond.Value, msg.Value, location)

	return &eval.UnitValue{}, nil
}
