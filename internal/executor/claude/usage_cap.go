package claude

import "github.com/sunholo-data/ailang/internal/executor"

// Token-cap accounting for the claude harness, kept out of the streaming loop.
//
// WHY THE COUNTERS ARE ASSIGNED, NOT SUMMED. claude-code emits CUMULATIVE usage in
// message_delta, so the handler tracks the latest value rather than adding each event —
// summing would multiply the same prompt by the turn count. pi and opencode emit per-turn
// or per-step DELTAS and therefore sum. Both shapes are correct for their provider, and
// the difference is why a shared accumulator would be wrong here.
//
// WHY CACHE CREATION ARRIVES LATE, and the limitation that follows. claude-code places
// cache_creation_input_tokens outside the usage block the message_delta handler reads, so
// the in-flight kill cannot see newly cached prompt and under-counts a cached run. The
// result event carries the canonical figure, so the cap is re-tested there. Consequence,
// stated rather than hidden: on this harness an over-cap run is DETECTED when the result
// arrives instead of being KILLED mid-stream. Closing that needs a recorded claude-code
// stream to verify the field's position — guessing a field path inside a kill switch is
// not worth the risk, and there is no fixture in this tree today.
//
// Measured 2026-09-14: a five-file read reported InputTokens=50 with
// CacheCreationInputTokens=44,841, so the omitted bucket was 99.9% of the real input.
// The canonical quantity is defined once in executor.TokensProcessed.

// capExceeded reports the canonical processed total and whether it breaches maxTokens.
//
// maxTokens <= 0 means "no cap", matching every other guard in the tree.
func capExceeded(maxTokens, input, cacheCreation, output int) (int, bool) {
	processed := executor.TokensProcessedFrom(input, cacheCreation, output, 0)
	return processed, maxTokens > 0 && processed > maxTokens
}

// residual returns the non-negative part of final-running, for feeding a cost Budget the
// difference between what the stream reported and what the result event reconciles to.
//
// Non-negative because message_delta values can arrive out of order or duplicated, and a
// negative residual would REFUND tokens into the budget — buying an over-spending run
// more room rather than less.
func residual(final, running int) int {
	if d := final - running; d > 0 {
		return d
	}
	return 0
}
