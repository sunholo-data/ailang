package microrag

import (
	"context"
	"fmt"
	"sort"
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
	// default (0.30). Empirical: on the bootstrap corpus, clearly off-topic
	// prompts ("what's the weather?") still score ~0.45 due to baseline
	// English-vocabulary similarity. Real syntax questions land 0.7+ so 0.50
	// gives a comfortable noise margin without filtering legitimate hits.
	userPromptRelevanceFloor = 0.50

	// userPromptMinLen — below this we skip the search entirely. Embedding
	// signal on <20 chars of prompt is dominated by noise.
	userPromptMinLen = 20

	// userPromptMaxTokens caps the per-injection size. Higher than file-route
	// default (150) because the chosen chunk *is* the answer to the user's
	// question — over-truncation hurts more than verbosity here.
	userPromptMaxTokens = 200
)

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
	if top.Score < userPromptRelevanceFloor {
		return &UserPromptResult{State: "on", Reason: "below_floor"}, nil
	}

	snippetID := hashSnippet(top)
	window := e.Cfg.WindowFor(top.Namespace)
	bypass := e.Cfg.BypassFor(top.Namespace)

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
