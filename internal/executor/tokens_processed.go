package executor

// The canonical token quantity, and why the guards needed one.
//
// Every executor already normalises its provider's raw counters to one contract:
// InputTokens is FRESH input, DISJOINT from CacheReadInputTokens and
// CacheCreationInputTokens. The providers disagree wildly about the raw shape —
// codex's `cached_input_tokens` is a SUBSET of `input_tokens` (OpenAI Responses
// semantics) and is split out, while claude's and opencode's cache counters are
// EXCLUSIVE of input and are added on — and each harness has a test pinning its own
// normalisation. That layer is sound.
//
// What had no single definition was the quantity the THRASH GUARD tests. All five
// guards hand-rolled `inputTokens + outputTokens`, which omits
// CacheCreationInputTokens entirely. On a harness that caches aggressively that is
// nearly the whole prompt.
//
// Measured 2026-09-14 with `ailang mission role-run`: two requests identical except the
// model — same executor role, same workspace, same instructions (read five named files,
// one tool call each, then reply DONE):
//
//	harness  model               Input  CacheCreation  CacheRead  Output
//	pi       pi-or-minimax-m3   35,992              0      1,024     311
//	claude   claude-haiku-4-5        50         44,841    172,565     648
//
// Identical work. `Input+Output` reads 36,303 for pi and 698 for claude — a 52x
// divergence in the number a cap is compared against, which is why an executor
// appeared to run comfortably inside 70,000 while an evaluator died four times at
// 100,000. TokensProcessed reads 36,303 and 45,539: the same work, the same order of
// magnitude, the residual being claude's 6 turns against pi's 2 plus its own system
// prompt.
//
// CacheReadInputTokens is deliberately EXCLUDED. It is the reused prefix, re-sent by
// the harness on every turn, so including it makes the total grow quadratically with
// turn count and measures conversation length rather than work done — pi reported
// 679,040 cache reads on a 23-turn stage that processed 101,542. A guard on that
// number would fire on any long-but-productive stage. The eval BANKING layer does
// include it (`Input + CacheCreation + CacheRead`, see
// codex/cached_tokens_test.go) because there the question is "what did the provider
// process", not "is this agent looping".

// TokensProcessed is the canonical cumulative work a stage has done: fresh input,
// newly cached input, visible output and hidden reasoning.
//
// This is the quantity a token cap should be compared against, and the one every
// thrash guard should use, so that a cap means the same thing on every harness.
// Negative components are clamped rather than trusted: a malformed provider stream
// must not be able to shrink the total and buy an agent more budget.
func (r *Result) TokensProcessed() int {
	if r == nil {
		return 0
	}
	return nonNeg(r.InputTokens) + nonNeg(r.CacheCreationInputTokens) +
		nonNeg(r.OutputTokens) + nonNeg(r.ReasonTokens)
}

// TokensProcessedFrom is the same definition for callers accumulating counters
// in-flight, before a Result exists — which is every harness's streaming loop.
func TokensProcessedFrom(input, cacheCreation, output, reason int) int {
	return nonNeg(input) + nonNeg(cacheCreation) + nonNeg(output) + nonNeg(reason)
}

func nonNeg(n int) int {
	if n < 0 {
		return 0
	}
	return n
}
