package microrag

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// UserPromptRequest is the input to a UserPromptSubmit-driven engine call.
// The user's prompt itself becomes the embedding query — no filepath, no
// content blob to dilute the signal. This is the embedding-query shape
// embeddings excel at (per ADR-002).
type UserPromptRequest struct {
	Prompt     string
	Namespaces []string // empty → defaultUserPromptNamespaces
}

// UserPromptResult is the CLI shell-out envelope for a UserPromptSubmit call.
// Mirrors ContextResult but with no file-routing concept — namespaces are
// queried directly.
type UserPromptResult struct {
	Injection *Injection `json:"injection,omitempty"`
	State     string     `json:"microrag_state"`
	Reason    string     `json:"reason,omitempty"`
}

// defaultUserPromptNamespaces is what we query when the caller doesn't pin
// the namespace list. These two corpora are what `ailang micro-rag bootstrap`
// populates and are the natural answer-bearing surfaces for "how do I X in
// AILANG?" questions.
var defaultUserPromptNamespaces = []string{"ailang-syntax", "ailang-builtins"}

const (
	// userPromptRelevanceFloor is intentionally tighter than the file-content
	// default (0.30). It gates a hook that fires on EVERY user turn, so a false
	// positive costs ~430 tokens of context repeatedly across a session — the
	// error is asymmetric and the floor should lean toward staying quiet.
	//
	// Calibrated 2026-07-31 against a labelled panel on the
	// ollama:embeddinggemma bootstrap corpus (613 frames). The panel spans BOTH
	// prompt lengths, because a short-prompt-only panel is not representative:
	//
	//	              short              long
	//	on-topic      0.61 - 0.78        0.81, 0.85
	//	off-topic     0.27 - 0.59        0.65, 0.60
	//
	// Length is not the confounder — long ON-topic prompts score HIGHER than
	// long off-topic ones, so relevance still dominates. The genuine overlap is
	// the weakest terse syntax questions (0.61-0.65) against real long repo-ops
	// prompts (max 0.65), which no single absolute threshold can separate.
	//
	// 0.70 resolves it in favour of staying quiet. Nothing in the panel scores
	// between 0.66 and 0.72, so 0.70 retains exactly what 0.66 would while
	// buying a 0.07 margin over the highest measured off-topic score instead of
	// 0.01. The cost is explicit: three terse short-form questions (0.61 "FS
	// effect + Result", 0.63 "Option/Some/None", 0.65 "record row polymorphism")
	// no longer inject. Their long forms — how a user actually asks when stuck —
	// score 0.81+, and `ailang prompt` plus the use-ailang skill still cover the
	// terse case.
	//
	// History: 0.50 came from an ASSUMED ~0.45 off-topic ceiling that was never
	// measured (real off-topic reaches 0.65). A first correction to 0.60 was
	// calibrated on short prompts only and still admitted the 0.65 long case.
	// Re-run TestUserPrompt_FloorSeparatesCalibrationPanel after any reindex and
	// use EnvUserPromptFloor to re-tune without a rebuild.
	userPromptRelevanceFloor = 0.70

	// userPromptMinLen — below this we skip the search entirely. Embedding
	// signal on <20 chars of prompt is dominated by noise.
	userPromptMinLen = 20

	// userPromptMaxTokens caps the per-injection size. Higher than file-route
	// default (150) because the chosen chunk *is* the answer to the user's
	// question — over-truncation hurts more than verbosity here.
	userPromptMaxTokens = 200
)

// UserPromptFloorFromEnv returns the user-prompt relevance floor, honouring
// EnvUserPromptFloor when it parses to a value in (0,1]. Anything else falls
// back to the calibrated default — an unparseable or out-of-range override is
// a config typo, and silently widening the gate is the failure mode this
// hook's token cost cannot afford.
func UserPromptFloorFromEnv() float64 {
	v := strings.TrimSpace(os.Getenv(EnvUserPromptFloor))
	if v == "" {
		return userPromptRelevanceFloor
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f <= 0 || f > 1 {
		return userPromptRelevanceFloor
	}
	return f
}

// userPromptBypassFor returns the dedup-bypass threshold for the user-prompt
// path, floored at relevanceFloor+bypassBand.
//
// The configured default bypass (0.70) was chosen when this path's relevance
// floor was 0.50, leaving a 0.20 band in which the dedup ledger could actually
// suppress a repeat. Raising the floor to 0.70 collapsed that band to zero:
// every hit that cleared the floor also cleared the bypass, so dedup silently
// stopped working and the same chunk could re-inject on every turn — the exact
// per-turn cost this hook's floor exists to control.
//
// Deriving the bypass from the live floor keeps the band intact even when the
// floor is overridden via EnvUserPromptFloor.
func userPromptBypassFor(cfg *Config, ns string) float64 {
	const bypassBand = 0.20
	minBypass := UserPromptFloorFromEnv() + bypassBand
	if b := cfg.BypassFor(ns); b > minBypass {
		return b
	}
	return minBypass
}

// UserPrompt searches the configured namespaces using `req.Prompt` as the
// query and returns the highest-scoring injection that clears the user-
// prompt relevance floor. Reuses the engine's dedup ledger and session
// budget so a popular chunk doesn't fire twice in one session.
func (e *Engine) UserPrompt(req UserPromptRequest) (*UserPromptResult, error) {
	_, finish := startUserPromptSpan(context.Background(), req)
	res, err := e.userPromptInner(req)
	finish(res)
	return res, err
}

func (e *Engine) userPromptInner(req UserPromptRequest) (*UserPromptResult, error) {
	if !e.Cfg.Enabled || !EnabledFromEnv() {
		return &UserPromptResult{State: "disabled", Reason: "env or config disabled"}, nil
	}
	prompt := strings.TrimSpace(req.Prompt)
	if len(prompt) < userPromptMinLen {
		return &UserPromptResult{State: "on", Reason: "prompt_too_short"}, nil
	}
	namespaces := req.Namespaces
	if len(namespaces) == 0 {
		namespaces = defaultUserPromptNamespaces
	}
	if al := RoutesAllowlist(); al != nil {
		kept := namespaces[:0]
		for _, ns := range namespaces {
			if al[ns] {
				kept = append(kept, ns)
			}
		}
		namespaces = kept
		if len(namespaces) == 0 {
			return &UserPromptResult{State: "on", Reason: "kb_not_in_allowlist"}, nil
		}
	}

	// Top-1 from each namespace, then pick overall winner.
	var allHits []SearchHit
	for _, ns := range namespaces {
		hits, err := e.searchUserPromptNamespace(prompt, ns)
		if err != nil {
			continue
		}
		if len(hits) > 0 {
			allHits = append(allHits, hits[0])
		}
	}
	if len(allHits) == 0 {
		return &UserPromptResult{State: "on", Reason: "no_hits"}, nil
	}
	sort.SliceStable(allHits, func(i, j int) bool { return allHits[i].Score > allHits[j].Score })
	top := allHits[0]
	if top.Score < UserPromptFloorFromEnv() {
		return &UserPromptResult{State: "on", Reason: "below_floor"}, nil
	}

	snippetID := hashSnippet(top)
	window := e.Cfg.WindowFor(top.Namespace)
	bypass := userPromptBypassFor(e.Cfg, top.Namespace)

	bypassFired := top.Score >= bypass
	if !bypassFired && e.dedupSuppressed(snippetID, window) {
		return &UserPromptResult{State: "on", Reason: "dedup_suppressed"}, nil
	}
	if e.budgetExhausted() {
		return &UserPromptResult{State: "on", Reason: "session_budget_exhausted"}, nil
	}

	state := "on"
	if DryrunFromEnv() {
		state = "dryrun"
	}
	text := e.formatUserPromptInjection(top)
	tokens := approxTokens(text)
	if tokens > userPromptMaxTokens {
		text = truncateToTokens(text, userPromptMaxTokens)
		tokens = userPromptMaxTokens
	}
	inj := &Injection{
		InjectionText: text,
		SnippetID:     snippetID,
		Tokens:        tokens,
		Namespace:     top.Namespace,
		Score:         top.Score,
		State:         state,
	}
	e.appendLedger(LedgerEntry{
		TS:        e.now().UnixMilli(),
		SnippetID: snippetID,
		Tokens:    tokens,
		FilePath:  "user-prompt://" + ledgerSummary(prompt, 80),
		Namespace: top.Namespace,
		State:     state,
		Score:     top.Score,
	})
	reason := "injected"
	if bypassFired {
		reason = "injected_bypass"
	}
	if state == "dryrun" {
		return &UserPromptResult{Injection: nil, State: state, Reason: "dryrun_logged"}, nil
	}
	return &UserPromptResult{Injection: inj, State: state, Reason: reason}, nil
}

// searchUserPromptNamespace runs a single-namespace top-1 search with cache
// reuse. The cache key is namespaced separately from file-content queries so
// the two retrieval paths don't collide.
func (e *Engine) searchUserPromptNamespace(query, ns string) ([]SearchHit, error) {
	queryHash := hash(query + "|" + ns + "|user-prompt")
	if hits, ok := e.readSearchCache(queryHash); ok {
		return hits, nil
	}
	hits, err := e.Searcher.Search(query, ns, 1)
	if err != nil {
		return nil, err
	}
	e.writeSearchCache(queryHash, hits)
	return hits, nil
}

func (e *Engine) formatUserPromptInjection(hit SearchHit) string {
	marker := "🧠 μRAG"
	border := "━━━"
	if e.Cfg.MarkerStyle == "ascii" {
		marker = "[uRAG]"
		border = "==="
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s [%s] %s\n", border, marker, hit.Namespace, border)
	if hit.Source != "" {
		fmt.Fprintf(&b, "→ %s\n", hit.Source)
	}
	body := strings.TrimSpace(hit.Content)
	for _, line := range strings.Split(body, "\n") {
		fmt.Fprintf(&b, "  %s\n", line)
	}
	fmt.Fprintf(&b, "%s%s\n", border, strings.Repeat(border, 4))
	return b.String()
}

// ledgerSummary normalises a prompt to a single-line truncated string for
// the ledger's FilePath field. Only used for human inspection of the ledger.
func ledgerSummary(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n]
}
