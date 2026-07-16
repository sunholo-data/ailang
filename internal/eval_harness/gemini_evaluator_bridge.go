// INJECT-IN diff bundle bridge for a sandboxed gemini sprint EVALUATOR.
//
// This is the INVERSE of managed_agents_bridge.go (the extract-out bridge).
// The managed_agents executor runs the agent in a Google-hosted sandbox with
// NO repo/file upload — the request body carries only Directive + SystemPrompt
// (managed_agents.go:164-176). So a gemini evaluator sees no local repo and
// cannot inspect a sprint's UNCOMMITTED worktree changes.
//
// This file packages a sprint worktree's uncommitted diff into a size-bounded
// "diff bundle" and injects it INTO the evaluator directive, so the sandboxed
// agent can read and REASON about the changes (reasoning-only: no local test
// re-runs). It then parses a structured verdict back out of the agent's fenced
// response (mirroring how the extract-out bridge parses a fenced block).
//
// Lives in eval_harness/ (not the executor) because this is caller POLICY, the
// same rationale as the sibling managed_agents_bridge.go (file header lines
// 8-11): the executor stays policy-free. This file touches ZERO executor code
// and does NOT import internal/mission/quorum (GeminiVerdict is a SEPARATE type
// in a different domain — see its doc-comment).
package eval_harness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// defaultBundleMaxBytes is the default bundle-text ceiling (256 KiB). Well
// under managed_agents token limits; the directive also competes with the
// design doc + plan, so we stay conservative and configurable.
const defaultBundleMaxBytes = 256 * 1024

// binaryScanBytes is how many leading bytes we scan for a NUL to classify a
// file as binary.
const binaryScanBytes = 8 * 1024

// BundleOptions configures BuildDiffBundle.
type BundleOptions struct {
	// MaxBytes is the ceiling for total bundle text. Full-file-content
	// appendices (and an untracked file's body) are dropped whole to stay
	// under it; the diff + NEW-FILE headers are never dropped. Zero means the
	// default (256 KiB).
	MaxBytes int
}

// maxBytes returns the effective ceiling, applying the default.
func (o BundleOptions) maxBytes() int {
	if o.MaxBytes <= 0 {
		return defaultBundleMaxBytes
	}
	return o.MaxBytes
}

// Bundle is the result of packaging a worktree's uncommitted changes.
type Bundle struct {
	// Text is the deterministic bundle: the unified diff of tracked changes,
	// `+++ NEW FILE` views of untracked files, and full changed-file contents,
	// with LOUD truncation markers for anything dropped.
	Text string
	// Truncated is true if any file content was dropped (binary/generated/
	// over-ceiling). Propagates to DegradationInfo.
	Truncated bool
	// DroppedFiles lists every dropped path with its reason (also echoed as a
	// marker line in Text). No silent drops.
	DroppedFiles []string
	// Bytes is len(Text).
	Bytes int
}

// DegradationInfo carries the degradation signal from bundle building (and,
// via the caller, a backend error) into the verdict. Any set field means the
// verdict is VerificationDegraded — a partial/lenient verdict can never
// masquerade as a full pass (CLAUDE.md "no silent fallbacks").
type DegradationInfo struct {
	// Truncated is true if the bundle dropped any file content.
	Truncated bool
	// DroppedFiles mirrors Bundle.DroppedFiles.
	DroppedFiles []string
	// BackendError is non-empty if the executor runner errored / returned
	// Success==false. Set by the caller (RunGeminiEvaluator), not the builder.
	BackendError string
}

// degraded reports whether any degradation signal is present.
func (d DegradationInfo) degraded() bool {
	return d.Truncated || len(d.DroppedFiles) > 0 || strings.TrimSpace(d.BackendError) != ""
}

// reason composes a human-readable degradation reason (non-empty iff degraded).
func (d DegradationInfo) reason() string {
	var parts []string
	if strings.TrimSpace(d.BackendError) != "" {
		parts = append(parts, "backend error: "+strings.TrimSpace(d.BackendError))
	}
	if d.Truncated || len(d.DroppedFiles) > 0 {
		parts = append(parts, fmt.Sprintf("bundle truncated: %d file(s) dropped and not shown to the evaluator (%s)",
			len(d.DroppedFiles), strings.Join(d.DroppedFiles, "; ")))
	}
	return strings.Join(parts, " | ")
}

// generatedGlobs are path substrings/suffixes marking generated or vendored
// files that are listed but never inlined (they carry little review signal and
// large blast radius).
var generatedSuffixes = []string{".pb.go", ".min.js", "_generated.go"}
var generatedPathParts = []string{"/vendor/", "/dist/"}
var generatedExact = []string{"go.sum"}

// changedFile is one entry in the changed-file set enumerated from git status.
type changedFile struct {
	path      string // repo-relative path
	untracked bool   // true if `??` (new/untracked) — git diff would miss it
}

// BuildDiffBundle packages a worktree's uncommitted changes into a bounded,
// deterministic bundle for a reasoning-only evaluator.
//
// It enumerates the changed-file set via `git status --porcelain -z` (modified
// + staged + UNTRACKED — untracked files are load-bearing: a sprint frequently
// CREATES new files and `git diff` alone would miss them), emits the tracked
// unified diff (`git diff` + `git diff --cached`, ALWAYS included, never
// dropped), synthesizes `+++ NEW FILE:` headers for untracked files, then
// appends full file contents subject to a drop-order + byte ceiling with LOUD
// truncation markers. It never silently drops.
func BuildDiffBundle(worktree string, opts BundleOptions) (Bundle, error) {
	if strings.TrimSpace(worktree) == "" {
		return Bundle{}, fmt.Errorf("BuildDiffBundle: empty worktree path")
	}
	files, err := enumerateChangedFiles(worktree)
	if err != nil {
		return Bundle{}, err
	}

	tracked := gitDiff(worktree, false) + gitDiff(worktree, true)

	var b strings.Builder
	b.WriteString("=== SPRINT DIFF BUNDLE (uncommitted worktree changes) ===\n\n")

	// The tracked unified diff is ALWAYS first and never dropped (the change
	// signal is load-bearing).
	b.WriteString("=== UNIFIED DIFF (tracked changes) ===\n")
	if strings.TrimSpace(tracked) == "" {
		b.WriteString("(no tracked modifications)\n")
	} else {
		b.WriteString(tracked)
		if !strings.HasSuffix(tracked, "\n") {
			b.WriteByte('\n')
		}
	}
	b.WriteByte('\n')

	// Partition into inline candidates vs must-drop (binary/generated). Files
	// are processed in a stable sorted order for determinism.
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })

	type candidate struct {
		cf      changedFile
		content string
		size    int
	}
	var inlineCandidates []candidate
	var dropped []string
	droppedMarkers := &strings.Builder{}

	dropMarker := func(path, human, reason string) {
		line := fmt.Sprintf("=== BUNDLE TRUNCATED: dropped %s (%s, %s) — NOT shown to evaluator ===",
			path, human, reason)
		droppedMarkers.WriteString(line + "\n")
		dropped = append(dropped, fmt.Sprintf("%s (%s)", path, reason))
	}

	for _, cf := range files {
		full := filepath.Join(worktree, cf.path)
		data, readErr := os.ReadFile(full)
		if readErr != nil {
			// A staged deletion (or a race) — the file is gone. It shows in the
			// unified diff already; nothing to inline. Not a drop.
			continue
		}
		if isBinary(data) {
			dropMarker(cf.path, humanSize(len(data)), "binary")
			continue
		}
		if isGeneratedPath(cf.path) {
			dropMarker(cf.path, humanSize(len(data)), "generated")
			continue
		}
		inlineCandidates = append(inlineCandidates, candidate{cf: cf, content: string(data), size: len(data)})
	}

	// For untracked (new) files, emit a `+++ NEW FILE:` HEADER for every one
	// (never dropped, so a new file is never silently invisible) even if its
	// body is later dropped for the ceiling.
	var newFileHeaders strings.Builder
	for _, c := range inlineCandidates {
		if c.cf.untracked {
			newFileHeaders.WriteString(fmt.Sprintf("+++ NEW FILE: %s\n", c.cf.path))
		}
	}
	// Also emit headers for untracked files that were dropped as binary/
	// generated so they are still visible.
	for _, cf := range files {
		if !cf.untracked {
			continue
		}
		alreadyInline := false
		for _, c := range inlineCandidates {
			if c.cf.path == cf.path {
				alreadyInline = true
				break
			}
		}
		if !alreadyInline {
			newFileHeaders.WriteString(fmt.Sprintf("+++ NEW FILE: %s\n", cf.path))
		}
	}
	if newFileHeaders.Len() > 0 {
		b.WriteString("=== NEW (untracked) FILES ===\n")
		b.WriteString(newFileHeaders.String())
		b.WriteByte('\n')
	}

	// Byte-ceiling enforcement over the full-file appendices. Drop order: the
	// binary/generated files are already dropped above; here we drop the
	// LARGEST remaining whole files until the projected total is under the
	// ceiling. Drop candidates are ordered (size desc, path asc) for a stable
	// total order.
	ceiling := opts.maxBytes()

	// Build the appendix body deterministically (sorted by path).
	renderAppendix := func(c candidate) string {
		hdr := "----- FULL FILE: "
		if c.cf.untracked {
			hdr = "----- FULL FILE (new): "
		}
		return fmt.Sprintf("%s%s -----\n%s\n----- END FILE: %s -----\n\n",
			hdr, c.cf.path, ensureTrailingNL(c.content), c.cf.path)
	}

	// Projected fixed prefix = everything already written + the eventual
	// dropped-markers block. We size the appendices against the remaining
	// budget.
	fixedPrefix := b.Len() + len("=== FULL FILE CONTENTS ===\n\n") + droppedMarkers.Len()

	// Decide which appendices to keep. Greedy: drop largest-first until the
	// kept appendices fit the remaining budget.
	keep := make(map[string]bool, len(inlineCandidates))
	for _, c := range inlineCandidates {
		keep[c.cf.path] = true
	}
	appendixBytes := func() int {
		total := 0
		for _, c := range inlineCandidates {
			if keep[c.cf.path] {
				total += len(renderAppendix(c))
			}
		}
		return total
	}
	// Order for dropping: size desc, then path asc (stable total order).
	dropOrder := make([]candidate, len(inlineCandidates))
	copy(dropOrder, inlineCandidates)
	sort.Slice(dropOrder, func(i, j int) bool {
		if dropOrder[i].size != dropOrder[j].size {
			return dropOrder[i].size > dropOrder[j].size
		}
		return dropOrder[i].cf.path < dropOrder[j].cf.path
	})
	for _, c := range dropOrder {
		if fixedPrefix+appendixBytes() <= ceiling {
			break
		}
		if keep[c.cf.path] {
			keep[c.cf.path] = false
			dropMarker(c.cf.path, humanSize(c.size), "over-ceiling")
			fixedPrefix = b.Len() + len("=== FULL FILE CONTENTS ===\n\n") + droppedMarkers.Len()
		}
	}

	// Emit dropped markers (LOUD) first, then the kept full-file appendices.
	if droppedMarkers.Len() > 0 {
		b.WriteString("=== DROPPED / TRUNCATED FILES ===\n")
		b.WriteString(droppedMarkers.String())
		b.WriteByte('\n')
	}

	// Kept appendices in stable path order.
	keptSorted := make([]candidate, 0, len(inlineCandidates))
	for _, c := range inlineCandidates {
		if keep[c.cf.path] {
			keptSorted = append(keptSorted, c)
		}
	}
	sort.Slice(keptSorted, func(i, j int) bool { return keptSorted[i].cf.path < keptSorted[j].cf.path })
	if len(keptSorted) > 0 {
		b.WriteString("=== FULL FILE CONTENTS ===\n\n")
		for _, c := range keptSorted {
			b.WriteString(renderAppendix(c))
		}
	}

	text := b.String()
	sort.Strings(dropped)
	return Bundle{
		Text:         text,
		Truncated:    len(dropped) > 0,
		DroppedFiles: dropped,
		Bytes:        len(text),
	}, nil
}

// enumerateChangedFiles parses `git status --porcelain -z` into the changed
// file set. The -z form is NUL-separated (no quoting), so paths with spaces or
// special chars are handled safely. Renames (R) carry two paths.
func enumerateChangedFiles(worktree string) ([]changedFile, error) {
	out, err := runGit(worktree, "status", "--porcelain", "-z")
	if err != nil {
		return nil, fmt.Errorf("BuildDiffBundle: git status failed: %w", err)
	}
	var files []changedFile
	seen := map[string]bool{}
	entries := strings.Split(out, "\x00")
	for i := 0; i < len(entries); i++ {
		e := entries[i]
		if len(e) < 3 {
			continue
		}
		xy := e[:2]
		path := e[3:]
		untracked := xy == "??"
		// Renames: `R  old\x00new` — the NEW path is the next NUL field.
		if (xy[0] == 'R' || xy[0] == 'C') && i+1 < len(entries) {
			path = entries[i+1]
			i++
		}
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		files = append(files, changedFile{path: path, untracked: untracked})
	}
	return files, nil
}

// gitDiff returns `git diff` (staged=false) or `git diff --cached`
// (staged=true) output. Errors are swallowed to empty (a repo with no HEAD yet
// still yields a usable bundle from untracked files).
func gitDiff(worktree string, staged bool) string {
	args := []string{"diff"}
	if staged {
		args = []string{"diff", "--cached"}
	}
	out, err := runGit(worktree, args...)
	if err != nil {
		return ""
	}
	return out
}

// runGit runs git in worktree with deterministic, locale-independent output.
func runGit(worktree string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = worktree
	// Force stable, locale-independent, color-free output for determinism.
	cmd.Env = append(os.Environ(),
		"LC_ALL=C",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_PAGER=cat",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

// isBinary reports whether data looks binary (a NUL byte in the first 8 KiB).
func isBinary(data []byte) bool {
	n := len(data)
	if n > binaryScanBytes {
		n = binaryScanBytes
	}
	return bytes.IndexByte(data[:n], 0) >= 0
}

// isGeneratedPath reports whether path is generated/vendored (never inlined).
func isGeneratedPath(path string) bool {
	slashed := "/" + filepath.ToSlash(path)
	base := filepath.Base(path)
	for _, exact := range generatedExact {
		if base == exact {
			return true
		}
	}
	for _, suf := range generatedSuffixes {
		if strings.HasSuffix(path, suf) {
			return true
		}
	}
	for _, part := range generatedPathParts {
		if strings.Contains(slashed, part) {
			return true
		}
	}
	return false
}

// humanSize renders a byte count like "412 KiB" / "37 B".
func humanSize(n int) string {
	switch {
	case n >= 1024*1024:
		return fmt.Sprintf("%d MiB", n/(1024*1024))
	case n >= 1024:
		return fmt.Sprintf("%d KiB", n/1024)
	default:
		return fmt.Sprintf("%d B", n)
	}
}

// ensureTrailingNL guarantees s ends with exactly one newline.
func ensureTrailingNL(s string) string {
	if s == "" {
		return ""
	}
	return strings.TrimRight(s, "\n") + "\n"
}

// ============================================================================
// M2 — Reasoning-only evaluator directive + structured verdict parse/validate
// ============================================================================

// geminiPassThreshold is the sprint-evaluator convention: pass requires
// score >= 70 (and no blockers). Matches the sprint-evaluator skill's
// pass_threshold.
const geminiPassThreshold = 70

// evaluatorReasoningInstruction is the compile-time reasoning-only directive
// prelude, mirroring managedAgentsBridgeInstruction's role for the extract-out
// path. It tells the sandboxed evaluator it has NO repo/test access and must
// judge only from the supplied bundle, then emit a fenced json verdict.
const evaluatorReasoningInstruction = "IMPORTANT — Reasoning-only sprint evaluation. " +
	"You are running in a sandbox with NO access to the repository or local tests. " +
	"Everything you can inspect is in the DIFF BUNDLE below. " +
	"Do NOT claim you ran any test. Judge ONLY from the supplied diff + file contents " +
	"against the design doc's acceptance criteria. " +
	"At the very END of your response, output your verdict as a single fenced ```json block " +
	"with fields: score (0-100), pass (bool), blockers (string[])."

// evaluatorTruncationNote is appended when the bundle was truncated so the
// evaluator qualifies its confidence rather than silently over-trusting a
// partial view.
const evaluatorTruncationNote = "\nThe DIFF BUNDLE is marked TRUNCATED: some files were NOT shown to you. " +
	"Note in blockers which unseen files limit your confidence."

// GeminiVerdict is the reasoning-only sprint-evaluation verdict returned by a
// sandboxed gemini evaluator.
//
// SCHEMA NOTE (adaptation, not a byte-mirror): this is a COMPACT ADAPTATION of
// the sprint-evaluator skill's evaluation_report_schema.md, whose fields are
// total_score/result/hard_fails/feedback. The mapping is:
//
//	total_score        -> Score   (0..100)
//	result == "pass"   -> Pass     (at threshold 70)
//	hard_fails + feedback -> Blockers (compact stand-in)
//
// It is intentionally compact for a single-shot reasoning-only verdict; the
// full evaluation_report_schema.md remains the shape for the local sonnet
// sprint-evaluator, not this sandboxed gemini one.
//
// It is DISTINCT from quorum.ReviewResult (pass/reject/strongest_objection),
// which is a design-DOC-REVIEW verdict in a different domain and MUST NOT be
// conflated or overloaded — GeminiVerdict does not import or extend it.
type GeminiVerdict struct {
	// Score is 0..100; pass threshold is 70 (sprint-evaluator convention).
	Score int `json:"score"`
	// Pass is score >= 70 AND no blockers.
	Pass bool `json:"pass"`
	// Blockers are hard-fail reasons; a non-empty Blockers ⇒ Pass==false
	// regardless of Score.
	Blockers []string `json:"blockers"`

	// VerificationDegraded is set (with a non-empty DegradedReason) whenever
	// the bundle was truncated / files were dropped / the backend errored, so
	// a partial or lenient verdict can NEVER masquerade as a full pass
	// (CLAUDE.md "no silent fallbacks"). It is stamped by the CALLER, not
	// trusted from the model.
	VerificationDegraded bool   `json:"verification_degraded"`
	DegradedReason       string `json:"degraded_reason,omitempty"`
}

// BuildEvaluatorDirective composes the reasoning-only evaluator directive: the
// reasoning-only instruction, the design doc + sprint plan + acceptance
// criteria, and the diff bundle. A truncated bundle appends the "note unseen
// files in blockers" line. Mirrors managedAgentsBridgeInstruction's role.
func BuildEvaluatorDirective(designDoc, sprintPlan, acceptanceCriteria string, bundle Bundle) string {
	var b strings.Builder
	b.WriteString(evaluatorReasoningInstruction)
	if bundle.Truncated {
		b.WriteString(evaluatorTruncationNote)
	}
	b.WriteString("\n\n=== DESIGN DOC ===\n")
	b.WriteString(strings.TrimSpace(designDoc))
	b.WriteString("\n\n=== SPRINT PLAN ===\n")
	b.WriteString(strings.TrimSpace(sprintPlan))
	if strings.TrimSpace(acceptanceCriteria) != "" {
		b.WriteString("\n\n=== ACCEPTANCE CRITERIA ===\n")
		b.WriteString(strings.TrimSpace(acceptanceCriteria))
	}
	b.WriteString("\n\n")
	b.WriteString(bundle.Text)
	return b.String()
}

// ParseGeminiVerdict extracts the LAST fenced json block from the agent's
// response, unmarshals it into a GeminiVerdict, validates it, and STAMPS the
// caller-supplied degradation (truncation / dropped files / backend error).
// A missing fence, non-JSON body, or gate violation is a HARD ERROR — never a
// coerced/lenient pass (CLAUDE.md "no silent fallbacks").
func ParseGeminiVerdict(response string, degraded DegradationInfo) (GeminiVerdict, error) {
	block := LastFencedBlock(response)
	if strings.TrimSpace(block) == "" {
		return GeminiVerdict{}, fmt.Errorf("gemini verdict: no fenced ```json block found in response (%.200q)", response)
	}
	var v GeminiVerdict
	if err := json.Unmarshal([]byte(strings.TrimSpace(block)), &v); err != nil {
		return GeminiVerdict{}, fmt.Errorf("gemini verdict: fenced block is not valid JSON: %w (block: %.200q)", err, block)
	}
	// Stamp degradation from the caller BEFORE validating, so a degraded
	// verdict with an empty reason cannot slip through.
	if degraded.degraded() {
		v.VerificationDegraded = true
		v.DegradedReason = degraded.reason()
	}
	if err := ValidateGeminiVerdict(&v); err != nil {
		return GeminiVerdict{}, err
	}
	return v, nil
}

// ValidateGeminiVerdict enforces the sprint-eval contract as HARD errors:
//   - Score ∈ [0,100];
//   - a non-empty Blockers ⇒ Pass==false (a blocker cannot coexist with pass);
//   - VerificationDegraded ⇒ DegradedReason non-empty.
//
// Any violation is an error, never a coerced pass (CLAUDE.md "no silent
// fallbacks", mirroring quorum.ValidateReviewResult discipline).
func ValidateGeminiVerdict(v *GeminiVerdict) error {
	if v.Score < 0 || v.Score > 100 {
		return fmt.Errorf("gemini verdict: score %d out of range [0,100]", v.Score)
	}
	if len(v.Blockers) > 0 && v.Pass {
		return fmt.Errorf("gemini verdict: pass==true with %d blocker(s) is contradictory: %v", len(v.Blockers), v.Blockers)
	}
	if v.VerificationDegraded && strings.TrimSpace(v.DegradedReason) == "" {
		return fmt.Errorf("gemini verdict: verification_degraded==true requires a non-empty degraded_reason")
	}
	return nil
}

// ============================================================================
// M3 — Caller seam + caller-enforced degradation stamping
// ============================================================================

// EvalRunnerOutput is the minimal executor result the caller needs: it mirrors
// the fields `ailang exec gemini --json` already emits (exec.go:281-294) that
// matter for verdict extraction and degradation. The default production runner
// populates it from the executor; unit tests return a canned value.
type EvalRunnerOutput struct {
	// Success is false when the backend errored — the caller stamps
	// VerificationDegraded and NEVER treats the result as a real pass.
	Success bool
	// Output is the agent's raw text response (contains the fenced verdict).
	Output string
	// Error is the executor/backend error text (surfaced in DegradedReason).
	Error string
}

// EvalRunner invokes the sandboxed evaluator with a composed directive +
// system prompt and returns its output. It is INJECTABLE so tests pass a stub
// and NEVER make a live Vertex/managed_agents call. The default production
// runner (DefaultGeminiRunner) shells `ailang exec gemini --json`.
type EvalRunner func(directive, systemPrompt string) (EvalRunnerOutput, error)

// EvalOptions configures RunGeminiEvaluator.
type EvalOptions struct {
	// Bundle configures BuildDiffBundle (byte ceiling etc.).
	Bundle BundleOptions
	// AcceptanceCriteria is the criteria text folded into the directive.
	AcceptanceCriteria string
	// SystemPrompt is passed through to the runner (executor SystemInstruction).
	SystemPrompt string
	// Runner is the injectable executor seam. If nil, DefaultGeminiRunner is
	// used (production path — shells `ailang exec gemini --json`; NEVER
	// exercised by the test suite).
	Runner EvalRunner
}

// RunGeminiEvaluator composes the reasoning-only evaluation end-to-end:
// BuildDiffBundle → BuildEvaluatorDirective → injected runner →
// ParseGeminiVerdict, and CALLER-ENFORCES degradation. On bundle truncation OR
// a backend error, the returned verdict has VerificationDegraded==true with a
// non-empty reason — a model/stub pass:true under degradation can NEVER stand
// as a real pass (CLAUDE.md "no silent fallbacks", Principle 2). Mirrors
// quorum.RunAgenticReviewer's policy-layer boundary.
func RunGeminiEvaluator(worktree, designDoc, sprintPlan string, opts EvalOptions) (*GeminiVerdict, error) {
	bundle, err := BuildDiffBundle(worktree, opts.Bundle)
	if err != nil {
		return nil, fmt.Errorf("RunGeminiEvaluator: build bundle: %w", err)
	}

	// Seed degradation from the bundle (truncation / dropped files).
	degraded := DegradationInfo{
		Truncated:    bundle.Truncated,
		DroppedFiles: bundle.DroppedFiles,
	}

	directive := BuildEvaluatorDirective(designDoc, sprintPlan, opts.AcceptanceCriteria, bundle)

	runner := opts.Runner
	if runner == nil {
		runner = DefaultGeminiRunner
	}
	out, runErr := runner(directive, opts.SystemPrompt)
	if runErr != nil {
		degraded.BackendError = runErr.Error()
	} else if !out.Success {
		msg := strings.TrimSpace(out.Error)
		if msg == "" {
			msg = "executor reported failure with no error text"
		}
		degraded.BackendError = msg
	}

	// On a backend error we have no trustworthy model verdict at all. Return a
	// degraded, NON-PASS verdict carrying the error — never a fabricated pass.
	if strings.TrimSpace(degraded.BackendError) != "" {
		v := &GeminiVerdict{
			Score:                0,
			Pass:                 false,
			Blockers:             []string{"backend error: no verdict obtained"},
			VerificationDegraded: true,
			DegradedReason:       degraded.reason(),
		}
		return v, nil
	}

	// Backend OK: parse the model verdict, then the caller-supplied degradation
	// (bundle truncation) is stamped inside ParseGeminiVerdict — so a stub/model
	// pass:true UNDER truncation still surfaces VerificationDegraded==true.
	v, err := ParseGeminiVerdict(out.Output, degraded)
	if err != nil {
		return nil, fmt.Errorf("RunGeminiEvaluator: parse verdict: %w", err)
	}
	return &v, nil
}

// DefaultGeminiRunner is the production runner: it shells
// `ailang exec gemini --json` (which routes to the managed_agents executor,
// exec.go:317-322) and parses the --json envelope. It is the default when
// EvalOptions.Runner is nil and is NEVER invoked by the test suite (all tests
// inject a stub) — so no test makes a live Vertex/managed_agents call.
func DefaultGeminiRunner(directive, systemPrompt string) (EvalRunnerOutput, error) {
	args := []string{"exec", "gemini", "--json", "--prompt", directive}
	if strings.TrimSpace(systemPrompt) != "" {
		args = append(args, "--system-prompt", systemPrompt)
	}
	cmd := exec.Command("ailang", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	// The --json envelope from exec.go:281-294.
	var env struct {
		Success bool   `json:"success"`
		Output  string `json:"output"`
		Error   string `json:"error"`
	}
	if jerr := json.Unmarshal(stdout.Bytes(), &env); jerr != nil {
		// No parseable envelope — surface the raw failure loudly.
		return EvalRunnerOutput{}, fmt.Errorf("ailang exec gemini: unparseable --json output: %v (stderr: %s)",
			jerr, strings.TrimSpace(stderr.String()))
	}
	if runErr != nil && env.Error == "" {
		env.Error = strings.TrimSpace(stderr.String())
	}
	return EvalRunnerOutput{Success: env.Success, Output: env.Output, Error: env.Error}, nil
}
