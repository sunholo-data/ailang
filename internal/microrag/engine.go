package microrag

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// EnvEnabled is the master eval-toggle env var (0/1). Default: enabled.
const (
	EnvEnabled = "AILANG_MICRORAG_ENABLED"
	EnvRoutes  = "AILANG_MICRORAG_ROUTES"
	EnvDryrun  = "AILANG_MICRORAG_DRYRUN"
	EnvSession = "AILANG_MICRORAG_SESSION"

	searchCacheTTLSecs = 240
	embedCacheTTLDays  = 1
)

// SearchHit is the engine's view of a single brain search result.
// Decoupled from internal/effects.BrainSearchResult so the engine can
// be tested without spinning up SQLite.
type SearchHit struct {
	Tier      string  `json:"tier"`
	Namespace string  `json:"namespace"`
	Key       string  `json:"key"`
	Score     float64 `json:"score"`
	Content   string  `json:"content"`
	UpdatedAt int64   `json:"updated_at_ms"`
	Source    string  `json:"source"`
}

// SearchEnvelope mirrors the JSON written by `ailang cache search --json`.
type SearchEnvelope struct {
	Scope   string      `json:"scope"`
	Count   int         `json:"count"`
	QueryMs int64       `json:"query_ms"`
	Results []SearchHit `json:"results"`
}

// Searcher abstracts the brain — production wires to `ailang cache search --json`,
// tests wire to a stub.
type Searcher interface {
	Search(query, namespace string, limit int) ([]SearchHit, error)
}

// Injection is the structured output of one engine call.
type Injection struct {
	InjectionText string  `json:"injection_text"`
	SnippetID     string  `json:"snippet_id"`
	Tokens        int     `json:"tokens"`
	Namespace     string  `json:"ns"`
	Score         float64 `json:"score"`
	State         string  `json:"microrag_state"` // "on" | "dryrun" | "disabled"
}

// LedgerEntry is one line in injections.jsonl.
type LedgerEntry struct {
	TS        int64   `json:"ts"`
	SnippetID string  `json:"snippet_id"`
	Tokens    int     `json:"tokens"`
	FilePath  string  `json:"file_path"`
	Namespace string  `json:"ns"`
	State     string  `json:"microrag_state"`
	Score     float64 `json:"score"`
}

// Engine holds runtime configuration + side-effect roots (filesystem,
// searcher). All disk paths are derived from sessionDir.
type Engine struct {
	Cfg        *Config
	Searcher   Searcher
	SessionDir string
	Now        func() time.Time // injectable for tests
}

// Request is the input to one engine call.
type Request struct {
	ToolName string // Edit | Write | Read
	FilePath string
	Content  string // for Edit/Write — first ~2KB used for query enrichment
}

// ContextResult is what the CLI shell-out emits.
type ContextResult struct {
	Injection *Injection `json:"injection,omitempty"`
	State     string     `json:"microrag_state"`
	Reason    string     `json:"reason,omitempty"` // why no injection (disabled, no_route, dedup, ...)
}

// EnabledFromEnv returns the engine on/off state from env, defaulting to true.
// Invalid values default to true ("don't fail closed" — see design Risks table).
func EnabledFromEnv() bool {
	v := strings.TrimSpace(os.Getenv(EnvEnabled))
	if v == "" {
		return true
	}
	return v != "0" && strings.ToLower(v) != "false"
}

// DryrunFromEnv returns whether dryrun mode is active.
func DryrunFromEnv() bool {
	v := strings.TrimSpace(os.Getenv(EnvDryrun))
	return v == "1" || strings.ToLower(v) == "true"
}

// RoutesAllowlist returns the AILANG_MICRORAG_ROUTES allowlist as a set,
// or nil if no allowlist is set (means: allow all).
func RoutesAllowlist() map[string]bool {
	v := strings.TrimSpace(os.Getenv(EnvRoutes))
	if v == "" {
		return nil
	}
	out := map[string]bool{}
	for _, s := range strings.Split(v, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			out[s] = true
		}
	}
	return out
}

// Context is the main entry point. Returns a ContextResult describing whether
// an injection happened and why. Side effect: appends to injections.jsonl on
// success (or on dryrun).
func (e *Engine) Context(req Request) (*ContextResult, error) {
	_, finish := startContextSpan(context.Background(), req)
	res, err := e.contextInner(req)
	finish(res)
	return res, err
}

func (e *Engine) contextInner(req Request) (*ContextResult, error) {
	if !e.Cfg.Enabled || !EnabledFromEnv() {
		return &ContextResult{State: "disabled", Reason: "env or config disabled"}, nil
	}
	route := e.Cfg.MatchRoute(req.FilePath)
	if route == nil {
		return &ContextResult{State: "on", Reason: "no_route"}, nil
	}
	if route.KB == "skip" {
		return &ContextResult{State: "on", Reason: "kb_skip"}, nil
	}
	if al := RoutesAllowlist(); al != nil && !al[route.KB] {
		return &ContextResult{State: "on", Reason: "kb_not_in_allowlist"}, nil
	}

	query := buildQuery(req)
	queryHash := hash(query + "|" + route.KB)

	// Search cache lookup.
	if hit, ok := e.readSearchCache(queryHash); ok {
		return e.maybeInject(req, route, hit, queryHash, true)
	}

	hits, err := e.Searcher.Search(query, route.KB, 3)
	if err != nil {
		return &ContextResult{State: "on", Reason: fmt.Sprintf("search_error: %v", err)}, nil
	}
	e.writeSearchCache(queryHash, hits)
	return e.maybeInject(req, route, hits, queryHash, false)
}

func (e *Engine) maybeInject(req Request, route *Route, hits []SearchHit, queryHash string, fromCache bool) (*ContextResult, error) {
	// Filter to relevance floor.
	filtered := make([]SearchHit, 0, len(hits))
	for _, h := range hits {
		if h.Score >= route.RelevanceFloor {
			filtered = append(filtered, h)
		}
	}
	if len(filtered) == 0 {
		return &ContextResult{State: "on", Reason: "below_floor"}, nil
	}
	// Top-1 only (per design: pointer not body).
	sort.SliceStable(filtered, func(i, j int) bool { return filtered[i].Score > filtered[j].Score })
	top := filtered[0]

	snippetID := hashSnippet(top)
	window := e.Cfg.WindowFor(route.KB)
	bypass := e.Cfg.BypassFor(route.KB)

	bypassFired := top.Score >= bypass
	if !bypassFired && e.dedupSuppressed(snippetID, window) {
		return &ContextResult{State: "on", Reason: "dedup_suppressed"}, nil
	}
	if e.budgetExhausted() {
		return &ContextResult{State: "on", Reason: "session_budget_exhausted"}, nil
	}

	state := "on"
	if DryrunFromEnv() {
		state = "dryrun"
	}
	text := e.formatInjection(route, top)
	tokens := approxTokens(text)
	if tokens > route.MaxTokensPerInjection {
		text = truncateToTokens(text, route.MaxTokensPerInjection)
		tokens = route.MaxTokensPerInjection
	}
	inj := &Injection{
		InjectionText: text,
		SnippetID:     snippetID,
		Tokens:        tokens,
		Namespace:     route.KB,
		Score:         top.Score,
		State:         state,
	}
	e.appendLedger(LedgerEntry{
		TS:        e.now().UnixMilli(),
		SnippetID: snippetID,
		Tokens:    tokens,
		FilePath:  req.FilePath,
		Namespace: route.KB,
		State:     state,
		Score:     top.Score,
	})
	reason := "injected"
	if fromCache {
		reason = "injected_from_cache"
	}
	if bypassFired {
		reason = "injected_bypass"
	}
	if state == "dryrun" {
		reason = "dryrun_logged"
		// In dryrun mode emit no injection text to the harness.
		return &ContextResult{Injection: nil, State: state, Reason: reason}, nil
	}
	return &ContextResult{Injection: inj, State: state, Reason: reason}, nil
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

// --- Query building -------------------------------------------------------

func buildQuery(req Request) string {
	const maxContent = 2048
	c := req.Content
	if len(c) > maxContent {
		c = c[:maxContent]
	}
	return req.FilePath + "\n" + c
}

// --- Search-result cache --------------------------------------------------

func (e *Engine) searchCachePath(queryHash string) string {
	return filepath.Join(e.SessionDir, "search_cache", queryHash+".json")
}

func (e *Engine) readSearchCache(queryHash string) ([]SearchHit, bool) {
	p := e.searchCachePath(queryHash)
	st, err := os.Stat(p)
	if err != nil {
		return nil, false
	}
	if e.now().Sub(st.ModTime()) > time.Duration(searchCacheTTLSecs)*time.Second {
		return nil, false
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, false
	}
	var hits []SearchHit
	if err := json.Unmarshal(data, &hits); err != nil {
		return nil, false
	}
	return hits, true
}

func (e *Engine) writeSearchCache(queryHash string, hits []SearchHit) {
	p := e.searchCachePath(queryHash)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	data, err := json.Marshal(hits)
	if err != nil {
		return
	}
	_ = os.WriteFile(p, data, 0o644)
}

// --- Dedup ledger ---------------------------------------------------------

func (e *Engine) ledgerPath() string {
	return filepath.Join(e.SessionDir, "injections.jsonl")
}

// dedupSuppressed returns true if the snippet appeared in the ledger more
// recently than `window` cumulative tokens ago.
func (e *Engine) dedupSuppressed(snippetID string, window int) bool {
	p := e.ledgerPath()
	data, err := os.ReadFile(p)
	if err != nil {
		return false
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	tokensSince := 0
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		var entry LedgerEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.SnippetID == snippetID {
			return tokensSince < window
		}
		tokensSince += entry.Tokens
		if tokensSince >= window {
			return false
		}
	}
	return false
}

// budgetExhausted returns true if cumulative session tokens exceed the budget.
func (e *Engine) budgetExhausted() bool {
	p := e.ledgerPath()
	data, err := os.ReadFile(p)
	if err != nil {
		return false
	}
	total := 0
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry LedgerEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		total += entry.Tokens
	}
	return total >= e.Cfg.SessionBudget
}

func (e *Engine) appendLedger(entry LedgerEntry) {
	p := e.ledgerPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(data, '\n'))
}

// --- Formatting -----------------------------------------------------------

func (e *Engine) formatInjection(route *Route, hit SearchHit) string {
	marker := "🧠 μRAG"
	border := "━━━"
	if e.Cfg.MarkerStyle == "ascii" {
		marker = "[uRAG]"
		border = "==="
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s [%s] %s\n", border, marker, route.KB, border)
	body := hit.Content
	body = strings.TrimSpace(body)
	if hit.Source != "" {
		fmt.Fprintf(&b, "→ %s\n", hit.Source)
	}
	for _, line := range strings.Split(body, "\n") {
		fmt.Fprintf(&b, "  %s\n", line)
	}
	fmt.Fprintf(&b, "%s%s\n", border, strings.Repeat(border, 4))
	return b.String()
}

// approxTokens is a deliberate over-approximation: ~4 chars/token, never zero.
// Keeps injection budget enforcement simple without a real tokenizer.
func approxTokens(s string) int {
	if s == "" {
		return 0
	}
	t := len(s) / 4
	if t == 0 {
		t = 1
	}
	return t
}

// truncateToTokens cuts the injection at approximately maxTokens tokens.
// Preserves leading marker line (best-effort) by truncating from the body.
func truncateToTokens(s string, maxTokens int) string {
	maxBytes := maxTokens * 4
	if len(s) <= maxBytes {
		return s
	}
	return s[:maxBytes]
}

// --- Hashing --------------------------------------------------------------

func hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:16]) // 128-bit prefix is plenty
}

func hashSnippet(h SearchHit) string {
	return hash(h.Namespace + "|" + h.Key + "|" + h.Content)
}

// DefaultSessionDir returns the session-scoped state directory.
// Honors AILANG_MICRORAG_SESSION; otherwise uses a per-pid fallback so that
// concurrent sessions don't share a ledger.
func DefaultSessionDir() string {
	sid := strings.TrimSpace(os.Getenv(EnvSession))
	if sid == "" {
		sid = fmt.Sprintf("pid-%d", os.Getpid())
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ailang", "state", "microrag", sid)
}
