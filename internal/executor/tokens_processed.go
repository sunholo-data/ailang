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

// PER-HARNESS ACCUMULATION SEMANTICS — the table, in one place, because they differ and
// nothing asserted which was which until 2026-09-14.
//
//	harness    stream shape                      code does   cache buckets
//	pi         per-turn DELTAS (message_end)     SUM (+=)    write/read reported separately
//	opencode   per-step DELTAS (step_finish)     SUM (+=)    EXCLUSIVE of input/output
//	claude     per-TURN, cumulative within turn  SUM of      creation present in flight, in
//	                                             turns       message_start AND message_delta
//	codex      CUMULATIVE (usage events)         ASSIGN (=)  cached is a SUBSET of input
//	motoko     sums, then final usage wins       BOTH        reported separately
//
// Two consequences that have each already caused a wrong conclusion:
//
//  1. codex's guard must NOT gain a cache term. Its inputTokens is already the WHOLE
//     input and splitCodexInputTokens separates the cached part at the end
//     (total-preserving, per its test), so adding one would double-count.
//  2. Comparing a token cap between two roles is meaningless unless both run on the same
//     harness. Measured: the identical five-file read charged 36,303 on pi and 698 on
//     claude under the old Input+Output expression.
//
// The claude row was WRONG until 2026-09-14, and it was wrong in the direction that
// disarms a guard. It read "CUMULATIVE (message_delta), ASSIGN (=), creation arrives only
// at result" — inferred, never measured. A recorded stream
// (executor/claude/testdata/claude_stream_partial.ndjson) shows the counters reset every
// turn and cache creation present in flight, so assigning kept one turn's tail: 62 tokens
// weighed against a cap of 20,000 on a run that processed 49,970. Pinned now by
// TestClaudeStreamUsageIsPerTurnAndSums and TestClaudeTokenCapKillsInFlight.
//
// EVIDENCE BEHIND EACH ROW — audited 2026-09-21, and the previous version of this note
// ("Still NOT pinned by a recorded stream: opencode, codex and motoko") was wrong about two
// of the three. Two different things can back a row, and conflating them is how the claude
// row stayed wrong:
//
//	                        pins the PROVIDER's wire shape   pins OUR code's handling
//	a RECORDED capture      yes                              yes
//	a SYNTHETIC fixture     no                               yes
//
//	harness    fixture                                  provenance   what is pinned
//	pi         executor/pi/testdata/{v0_73_1,v0_85_1}   RECORDED     both
//	claude     claude/testdata/claude_stream_partial    RECORDED     both
//	opencode   opencode/testdata/opencode_response      RECORDED     both
//	motoko     motoko/testdata/session_*.jsonl          SYNTHETIC    our parser only
//	codex      codex/testdata/codex_response.jsonl      —            NOTHING: no usage at all
//
// opencode IS pinned at the executor level: TestExecuteStreaming_CacheTokens runs the real
// executor against the recorded stream and asserts the SUMMED totals across its three
// step_finish events (input 5, output 184, cache write 34,583, cache read 17,223), which is
// exactly this table's "per-step DELTAS, SUM, cache EXCLUSIVE of input/output".
//
// motoko's BOTH is really EITHER/OR, and both branches are asserted: run_summary wins when
// present (TestParseSessionJSONL), and the per-turn sums are the FALLBACK when the JSONL
// truncated before it (parser.go:547). All four summed counters now have an arm —
// TestParseSessionJSONL_NoSummarySumsCacheBuckets closed the cache half, which was live code
// no fixture reached. But every motoko fixture is hand-built, so none of it is evidence about
// what motoko actually emits.
//
// codex is the one genuine blank: its fixture contains ZERO usage lines, so the
// "CUMULATIVE, ASSIGN, cached is a SUBSET of input" row rests entirely on
// TestSplitCodexInputTokens, a unit test of the split function against values a human typed.
// To settle it, a capture needs >= 2 usage events in one run with DIFFERENT totals — that is
// the minimum that distinguishes CUMULATIVE-assign from DELTA-sum, and it is the exact
// property the claude fixture had to have. Until then the row is a belief, and the claude row
// is what a belief costs: 62 tokens weighed against a cap of 20,000 on a run that processed
// 49,970.
