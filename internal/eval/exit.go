package eval

// EvalExitCode is a sentinel panic value used by the exit() builtin.
// When panicked with this value, the runtime catches it and calls os.Exit(Code).
// This approach lets the evaluator unwind cleanly, flushing traces and telemetry.
type EvalExitCode struct {
	Code int
}
