package claude

import "github.com/sunholo-data/ailang/internal/executor"

// Token-cap accounting for the claude harness, kept out of the streaming loop.
//
// HOW THE COUNTERS ACCUMULATE. claude-code's usage is cumulative WITHIN a turn and RESETS
// at each message_start, so a run total is the SUM of the per-turn finals. This file said
// the opposite until 2026-09-14 — "CUMULATIVE, so assign the latest value" — which kept
// only the last turn. Recorded proof is testdata/claude_stream_partial.ndjson, six turns
// whose per-turn deltas reproduce the result event exactly (50 input, 49,214 cache
// creation, 315,648 cache read, 706 output); assigning read 8 and 54.
//
// CACHE CREATION IS PRESENT IN FLIGHT. This file previously recorded that claude-code
// placed cache_creation_input_tokens outside the usage block the message_delta handler
// reads, and the in-flight kill was abandoned on that basis. The recorded stream shows it
// inside both message_start.message.usage and message_delta.usage. So the cap is now
// enforced in flight — a run is KILLED when it breaches, not merely noticed afterwards —
// and the result-event test remains as a backstop.
//
// That earlier note was inference, not measurement, and it cost the cap its teeth: with
// cache creation pinned at 0 and only the last turn's input and output assigned, the guard
// compared 62 against a cap of 20,000 on a run that processed 49,970. Asserted now by
// TestClaudeTokenCapKillsInFlight.
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

// maxInt keeps a within-turn counter monotonic. Duplicate or out-of-order stream events
// must not be able to lower a running total, which would refund work already done and buy
// an over-budget agent more room.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// killFinishReason names why a run ended when one of the guards stopped it.
//
// Without this a thrash kill inherited the provider's own subtype and was banked as
// FinishStop — indistinguishable from a clean finish, so the failure-attribution layer
// could not see cap kills at all.
func killFinishReason(thrashKilled, costKilled bool, fallback string) string {
	switch {
	case thrashKilled:
		return executor.FinishThrashAborted
	case costKilled:
		return executor.FinishCostExhausted
	default:
		return fallback
	}
}

// usageGuard owns claude's token accounting for one run: it folds each turn's usage into
// the run totals, charges the cost budget the increment, and enforces the token cap by
// KILLING the process rather than noticing afterwards.
//
// It is a type rather than a handful of locals in the streaming loop because the loop's
// job is to say WHEN something arrived, not what it means — and because accounting that
// lives in a 700-line goroutine is accounting nobody can test on its own.
type usageGuard struct {
	task *executor.Task
	kill func()

	// committed* are the turns that have ended; cur* is the turn in progress. Claude's
	// counters restart at every message_start, so the run total is their sum.
	committedInput, committedCacheCreation, committedCacheRead, committedOutput int
	curInput, curCacheCreation, curCacheRead, curOutput                         int

	// final* are the result event's canonical totals, once it has arrived.
	finalized                                                   bool
	finalInput, finalCacheCreation, finalCacheRead, finalOutput int

	// What has already been charged to the cost Budget, so each event adds only its own
	// increment instead of re-charging the running total.
	chargedInput, chargedOutput int

	costKilled     bool
	thrashKilled   bool
	thrashKilledAt int
}

func newUsageGuard(task *executor.Task, kill func()) *usageGuard {
	return &usageGuard{task: task, kill: kill}
}

// endTurn banks the turn that just finished, because the next turn's counters start again
// from zero.
func (g *usageGuard) endTurn() {
	g.committedInput += g.curInput
	g.committedCacheCreation += g.curCacheCreation
	g.committedCacheRead += g.curCacheRead
	g.committedOutput += g.curOutput
	g.curInput, g.curCacheCreation, g.curCacheRead, g.curOutput = 0, 0, 0, 0
}

// fold takes one turn-scoped usage block — from message_start or message_delta — and
// applies the budget and cap consequences.
func (g *usageGuard) fold(usage map[string]interface{}) {
	if usage == nil {
		return
	}
	// Within one turn these only grow; max() absorbs duplicate or out-of-order events
	// without letting a stale one shrink the total and buy the agent more budget.
	g.curInput = maxInt(g.curInput, intFromAny(usage["input_tokens"]))
	g.curCacheCreation = maxInt(g.curCacheCreation, intFromAny(usage["cache_creation_input_tokens"]))
	g.curCacheRead = maxInt(g.curCacheRead, intFromAny(usage["cache_read_input_tokens"]))
	g.curOutput = maxInt(g.curOutput, intFromAny(usage["output_tokens"]))
	g.charge()
	g.enforceCap()
}

// finalize reconciles against the result event, which is canonical, and re-tests the cap
// as a backstop behind the in-flight kill. A nil result still banks the final turn, so a
// stream cut short reports what it actually did.
func (g *usageGuard) finalize(final *claudeHeadlessResult) {
	g.endTurn()
	if final == nil {
		return
	}
	g.finalized = true
	g.finalInput = final.Usage.InputTokens
	g.finalCacheCreation = final.Usage.CacheCreationInputTokens
	g.finalCacheRead = final.Usage.CacheReadInputTokens
	g.finalOutput = final.Usage.OutputTokens
	// With per-turn summing the stream and the result agree, so this residual is normally
	// zero. It stays because a stream cut short mid-turn can still leave one.
	g.charge()
	g.enforceCapNoKill()
}

// charge feeds the cost budget only what it has not already been told about.
func (g *usageGuard) charge() {
	if g.task == nil || g.task.Budget == nil {
		return
	}
	deltaIn := residual(g.inputTokens(), g.chargedInput)
	deltaOut := residual(g.outputTokens(), g.chargedOutput)
	if deltaIn == 0 && deltaOut == 0 {
		return
	}
	g.chargedInput += deltaIn
	g.chargedOutput += deltaOut
	if _, exceeded := g.task.Budget.Add(deltaIn, deltaOut); exceeded {
		g.costKilled = true
		if g.kill != nil {
			g.kill()
		}
	}
}

func (g *usageGuard) enforceCap() {
	if g.enforceCapNoKill() && g.kill != nil {
		g.kill()
	}
}

// enforceCapNoKill records a breach and reports whether this call is the one that found
// it, so the result-event backstop can flag a run whose process has already exited.
func (g *usageGuard) enforceCapNoKill() bool {
	if g.task == nil || g.thrashKilled {
		return false
	}
	tp, over := capExceeded(g.task.MaxTokensPerBench, g.inputTokens(),
		g.cacheCreationTokens(), g.outputTokens())
	if !over {
		return false
	}
	g.thrashKilled = true
	g.thrashKilledAt = tp
	return true
}

// The run totals. Before the result event these are the summed turns; after it they are
// the provider's own canonical figures.
func (g *usageGuard) inputTokens() int {
	if g.finalized {
		return g.finalInput
	}
	return g.committedInput + g.curInput
}

func (g *usageGuard) cacheCreationTokens() int {
	if g.finalized {
		return g.finalCacheCreation
	}
	return g.committedCacheCreation + g.curCacheCreation
}

func (g *usageGuard) cacheReadTokens() int {
	if g.finalized {
		return g.finalCacheRead
	}
	return g.committedCacheRead + g.curCacheRead
}

func (g *usageGuard) outputTokens() int {
	if g.finalized {
		return g.finalOutput
	}
	return g.committedOutput + g.curOutput
}
