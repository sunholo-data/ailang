// Package runner is the `ailang run` pipeline as a library: resolve the
// capabilities a program is granted, wire the effect handlers (Net, Process,
// Stream, SharedMem, SharedIndex, Env, Debug, Trace), compile the file through
// internal/pipeline, and execute the entry point — on the tree-walking
// evaluator or the bytecode VM with evaluator fallback, once or per batch
// input.
//
// It is not the evaluator (internal/eval) and it is not flag parsing: the
// CLI builds an Options and calls Run, and process-level concerns (telemetry
// init, GOGC, os.Exit) stay in cmd/ailang. Its sibling internal/pipeline
// compiles; runner runs.
package runner
