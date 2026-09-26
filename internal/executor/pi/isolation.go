package pi

import "github.com/sunholo-data/ailang/internal/executor"

// Isolation flags for a frozen mission stage, and the token-cap quantity.
//
// ABLATION, 2026-09-14. Asked one model in a gated workspace to run
// `pwd && git rev-parse HEAD` and report whether the sprint-evaluator skill was loaded:
//
//	flags                        bash                     skill
//	(dispatch's current set)     REFUSED by the gate      loaded
//	--no-extensions --approve    ran                      loaded
//	--no-extensions alone        ran                      NOT loaded
//
// The third row is why --approve is not optional. --no-extensions also disables the
// globally installed workspace-trust extension, which had been granting project trust
// headlessly, and without trust pi ignores project-local `.agents/skills`. The
// evaluator's skill IS its terminator (sprint-evaluator: 100-point rubric, threshold 70),
// so dropping extensions without restoring trust trades a visible deadlock for a silently
// unqualified judge — strictly worse, because the result looks like a verdict.
//
// The first row is the defect being repaired: the evaluator's own --tools allowlist omits
// `session_protocol_ack`, so the repo's session-protocol gate arms with no disarm
// reachable and bash is confined to three start-anchored allow-regexes.
func isolationArgs(task *executor.Task) []string {
	if task == nil || !task.IsolateFromProjectExtensions {
		return nil
	}
	return []string{"--no-extensions", "--approve"}
}

// capExceeded reports the canonical processed total and whether it breaches maxTokens.
//
// pi emits per-turn DELTAS in message_end, so its counters are SUMMED by the caller —
// unlike claude and codex, which receive cumulative values and assign. Cache writes are
// tracked separately and are usually 0 on OpenRouter, so this rarely changes pi's own
// number; it exists so every harness tests one expression (executor.TokensProcessed).
// pi reports no reasoning-token count, hence the 0.
func capExceeded(maxTokens, input, cacheWrite, output int) (int, bool) {
	processed := executor.TokensProcessedFrom(input, cacheWrite, output, 0)
	return processed, maxTokens > 0 && processed > maxTokens
}

// chargeCache feeds prompt-cache tokens to the cost budget and reports whether it tripped.
//
// Lives here to keep the stream loop short. Cache is most of a cached run's bill (69% on a
// measured run), and it was invisible to the budget until 2026-09-14. Undeclared rates
// price at zero rather than guessing the input rate, which would overstate ~3.8x and kill
// a healthy run — see executor/cost.go.
func chargeCache(task *executor.Task, cacheRead, cacheWrite int) bool {
	if task == nil || task.Budget == nil || (cacheRead <= 0 && cacheWrite <= 0) {
		return false
	}
	_, exceeded := task.Budget.AddCache(cacheRead, cacheWrite)
	return exceeded
}
