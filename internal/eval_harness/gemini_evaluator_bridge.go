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
