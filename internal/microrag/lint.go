package microrag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// BuiltinResolver looks up a builtin spec by name and returns the parsed JSON
// envelope from `ailang builtins show <name> --json`. Tests can stub this.
type BuiltinResolver interface {
	Resolve(name string) (*BuiltinSpecJSON, error)
}

// BuiltinSpecJSON mirrors the envelope emitted by `ailang builtins show --json`.
// Only the fields the lint nudge needs are unmarshalled.
type BuiltinSpecJSON struct {
	Name      string `json:"name"`
	Module    string `json:"module"`
	Signature string `json:"signature"`
	IsPure    bool   `json:"is_pure"`
	Effect    string `json:"effect,omitempty"`
	Metadata  *struct {
		Description string `json:"description,omitempty"`
		Examples    []struct {
			Code        string `json:"code"`
			Description string `json:"description,omitempty"`
		} `json:"examples,omitempty"`
		Since string `json:"since,omitempty"`
	} `json:"metadata,omitempty"`
}

// CLIBuiltinResolver shells out to `ailang builtins show <name> --json`.
type CLIBuiltinResolver struct {
	Binary string
}

func (r *CLIBuiltinResolver) Resolve(name string) (*BuiltinSpecJSON, error) {
	bin := r.Binary
	if bin == "" {
		bin = "ailang"
	}
	cmd := exec.Command(bin, "builtins", "show", name, "--json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// Exit code 1 means not found — return nil, nil so the caller can skip silently.
		return nil, nil
	}
	var spec BuiltinSpecJSON
	if err := json.Unmarshal(stdout.Bytes(), &spec); err != nil {
		return nil, fmt.Errorf("parse builtins show JSON: %w", err)
	}
	if spec.Name == "" {
		return nil, nil
	}
	return &spec, nil
}

// LintRequest is the input to a lint pass. Code is the file content
// being inspected (post-edit).
type LintRequest struct {
	FilePath string
	Code     string
}

// LintNudge is one builtin first-use nudge.
type LintNudge struct {
	Name          string `json:"name"`
	Module        string `json:"module"`
	Signature     string `json:"signature"`
	Description   string `json:"description,omitempty"`
	Example       string `json:"example,omitempty"`
	InjectionText string `json:"injection_text"`
	Tokens        int    `json:"tokens"`
}

// LintResult is the structured output of one lint pass.
type LintResult struct {
	Nudges []LintNudge `json:"nudges,omitempty"`
	State  string      `json:"microrag_state"`
	Reason string      `json:"reason,omitempty"`
}

// Linter holds runtime configuration for the first-use builtin nudge pass.
type Linter struct {
	Resolver   BuiltinResolver
	SessionDir string
	MaxNudges  int // hard cap per call; 0 → 2
	MaxTokens  int // per-nudge budget; 0 → 80
}

// identifierRe matches identifiers in function-call position: foo(...).
// First char lowercase + word boundary catches builtins (which are all lowercase
// or _-prefixed). Underscores match the leading-underscore wrapper convention.
var identifierRe = regexp.MustCompile(`(?m)\b(_?[a-z][A-Za-z0-9_]*)\s*\(`)

// Lint scans req.Code for first-use builtin invocations and emits short
// signature nudges for any builtin not yet seen this session.
func (l *Linter) Lint(req LintRequest) (*LintResult, error) {
	_, finish := startLintSpan(context.Background(), len(req.Code))
	res, err := l.lintInner(req)
	finish(res)
	return res, err
}

func (l *Linter) lintInner(req LintRequest) (*LintResult, error) {
	if !EnabledFromEnv() {
		return &LintResult{State: "disabled", Reason: "env_disabled"}, nil
	}
	candidates := extractCandidates(req.Code)
	if len(candidates) == 0 {
		return &LintResult{State: "on", Reason: "no_candidates"}, nil
	}
	seen, err := l.loadSeen()
	if err != nil {
		// Not fatal — on read error treat as empty-set and continue.
		seen = map[string]bool{}
	}
	maxNudges := l.MaxNudges
	if maxNudges <= 0 {
		maxNudges = 2
	}
	maxTokens := l.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 80
	}

	out := &LintResult{State: "on"}
	dryrun := DryrunFromEnv()
	if dryrun {
		out.State = "dryrun"
	}

	var fresh []string
	for _, name := range candidates {
		if len(out.Nudges) >= maxNudges {
			break
		}
		if seen[name] {
			continue
		}
		spec, err := l.Resolver.Resolve(name)
		if err != nil {
			continue
		}
		if spec == nil {
			// Not a builtin — record as seen so we don't re-resolve every keystroke.
			seen[name] = true
			fresh = append(fresh, name)
			continue
		}
		nudge := buildNudge(spec, maxTokens)
		seen[name] = true
		fresh = append(fresh, name)
		if !dryrun {
			out.Nudges = append(out.Nudges, nudge)
		}
	}
	if len(fresh) > 0 {
		_ = l.appendSeen(fresh)
	}
	if len(out.Nudges) == 0 && out.Reason == "" {
		out.Reason = "no_first_use"
	}
	return out, nil
}

// extractCandidates pulls deduplicated identifier-in-call-position names
// from the given code. Order is preserved (first appearance wins) so the
// nudge order matches the source order — easier for AI consumption.
func extractCandidates(code string) []string {
	matches := identifierRe.FindAllStringSubmatch(code, -1)
	seen := map[string]bool{}
	var out []string
	for _, m := range matches {
		name := m[1]
		if seen[name] {
			continue
		}
		// Skip language keywords that look like calls in source.
		if isKeyword(name) {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

func isKeyword(s string) bool {
	switch s {
	case "if", "then", "else", "let", "in", "match", "case", "func", "type",
		"module", "import", "export", "true", "false", "do", "yield",
		"return", "for", "while", "with":
		return true
	}
	return false
}

// buildNudge formats a single first-use snippet under the per-nudge token cap.
func buildNudge(spec *BuiltinSpecJSON, maxTokens int) LintNudge {
	// Header + signature is the load-bearing part — never truncate that.
	var b strings.Builder
	fmt.Fprintf(&b, "═ μRAG/builtin: %s ═\n", spec.Name)
	fmt.Fprintf(&b, "  %s\n", spec.Signature)
	if spec.Module != "" && spec.Module != "$builtin" {
		public := strings.TrimPrefix(spec.Name, "_")
		fmt.Fprintf(&b, "  import %s (%s)\n", spec.Module, public)
	}
	desc := ""
	example := ""
	if spec.Metadata != nil {
		desc = spec.Metadata.Description
		if len(spec.Metadata.Examples) > 0 {
			example = spec.Metadata.Examples[0].Code
		}
	}
	if desc != "" {
		fmt.Fprintf(&b, "  %s\n", desc)
	}
	if example != "" {
		fmt.Fprintf(&b, "  ex: %s\n", example)
	}
	text := b.String()
	tokens := approxTokens(text)
	if tokens > maxTokens {
		text = truncateToTokens(text, maxTokens)
		tokens = maxTokens
	}
	return LintNudge{
		Name:          spec.Name,
		Module:        spec.Module,
		Signature:     spec.Signature,
		Description:   desc,
		Example:       example,
		InjectionText: text,
		Tokens:        tokens,
	}
}

// --- Builtins-seen ledger -------------------------------------------------

func (l *Linter) seenPath() string {
	return filepath.Join(l.SessionDir, "builtins_seen.txt")
}

func (l *Linter) loadSeen() (map[string]bool, error) {
	data, err := os.ReadFile(l.seenPath())
	if err != nil {
		return map[string]bool{}, err
	}
	out := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out[line] = true
		}
	}
	return out, nil
}

func (l *Linter) appendSeen(names []string) error {
	if err := os.MkdirAll(l.SessionDir, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(l.seenPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, n := range names {
		if _, err := f.WriteString(n + "\n"); err != nil {
			return err
		}
	}
	return nil
}
