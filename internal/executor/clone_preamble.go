package executor

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// M-GEMINI-REPO-MOUNT Phase 2 — clone-over-egress shared helpers.
//
// The clone-review preamble lives here, in ONE place, because BOTH the CLI
// (`ailang exec --clone-repo`) and the eval-harness bridge (RunGeminiEvaluator)
// build it. Keeping it in a single builder is a hard invariant: if the two
// callers each hand-rolled the directive they would drift, and a drifted
// directive silently changes what the sandbox clones/checks. The executor
// package is the lowest common dependency of both callers (cmd/ailang/exec.go
// and internal/eval_harness both already import it), with no import cycle.

// SHA40 matches a syntactically-valid, full-length (40-hex) git object id.
// Abbreviated SHAs are rejected on purpose: the arbitrary-SHA fetch-by-SHA
// path pins an exact commit, and the HEAD-review evidence check needs an
// unambiguous full id to record as the reviewed revision.
var SHA40 = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

// IsValidSHA40 reports whether s is a full 40-hex git SHA.
func IsValidSHA40(s string) bool { return SHA40.MatchString(s) }

// cloneURLRE is the only shape of --clone-repo the preamble will interpolate
// into shell text the agent is told to run EXACTLY (M-EXECUTOR-ENV-HARDENING
// D5): https, a plain host[:port], and a path of URL-safe characters. Every
// shell metacharacter (; | & $ ` ' " \ < > ( ) { } * ? ! # ~ whitespace) and
// userinfo (user:token@host) are outside it by construction, so validation is
// an allowlist, not a hunt for bad bytes.
var cloneURLRE = regexp.MustCompile(`^https://[A-Za-z0-9](?:[A-Za-z0-9.-]*[A-Za-z0-9])?(?::[0-9]{1,5})?/[A-Za-z0-9._/%+-]+$`)

// ValidateCloneURL reports why repoURL may not be cloned by the clone-review
// preamble, or nil. It is a named error, never a silent rewrite.
func ValidateCloneURL(repoURL string) error {
	if !cloneURLRE.MatchString(repoURL) {
		return fmt.Errorf("--clone-repo must be a plain https git URL (https://host/owner/repo[.git], no credentials, query, fragment, whitespace or shell metacharacters), got %q", repoURL)
	}
	u, err := url.Parse(repoURL)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("--clone-repo must be a plain https git URL with a host, got %q", repoURL)
	}
	if strings.Contains(u.Path, "/../") || strings.HasSuffix(u.Path, "/..") || strings.Contains(u.Path, "//") {
		return fmt.Errorf("--clone-repo path must not contain empty or '..' segments, got %q", repoURL)
	}
	return nil
}

// BuildClonePreamble renders the canonical clone-review directive preamble for
// an egress-enabled sandbox. It is bounded-by-construction: BOTH modes are
// shallow (`--depth 1`), so neither walks full history.
//
//   - HEAD review (sha == ""):        `git clone --depth 1 <repoURL>`
//   - Arbitrary SHA (sha != ""):      shallow fetch-by-SHA of exactly that commit
//
// In both modes the agent must echo `git rev-parse HEAD` on its own line so the
// caller's evidence check can confirm which revision was actually reviewed.
//
// repoURL must be a plain https URL (ValidateCloneURL — checked here too, so
// the eval-harness caller is covered without its own copy). When sha
// is non-empty it must be a valid 40-hex SHA; an invalid sha returns an error
// rather than emitting an unbounded/ambiguous directive (no silent fallback).
func BuildClonePreamble(repoURL, sha string) (string, error) {
	repoURL = strings.TrimSpace(repoURL)
	sha = strings.TrimSpace(sha)
	if repoURL == "" {
		return "", fmt.Errorf("clone preamble: repo URL is required")
	}
	if err := ValidateCloneURL(repoURL); err != nil {
		return "", fmt.Errorf("clone preamble: %w", err)
	}

	var b strings.Builder
	b.WriteString("You are a read-only reviewer running in a sandbox with network egress.\n")
	b.WriteString("Obtain the source by running these commands EXACTLY, in order:\n\n")

	if sha == "" {
		// HEAD review — probe-R-proven recipe.
		fmt.Fprintf(&b, "  git clone --depth 1 %s repo\n", repoURL)
		b.WriteString("  cd repo\n")
	} else {
		if !IsValidSHA40(sha) {
			return "", fmt.Errorf("clone preamble: --clone-sha must be a 40-hex git SHA, got %q", sha)
		}
		// Arbitrary SHA — shallow fetch-by-SHA (bounded, no full history walk).
		b.WriteString("  mkdir repo && cd repo\n")
		b.WriteString("  git init -q\n")
		fmt.Fprintf(&b, "  git remote add origin %s\n", repoURL)
		fmt.Fprintf(&b, "  git fetch --depth 1 origin %s\n", sha)
		b.WriteString("  git checkout --detach FETCH_HEAD\n")
	}

	b.WriteString("\nThen, on a line by itself, echo the exact revision you checked out:\n")
	b.WriteString("  git rev-parse HEAD\n\n")
	b.WriteString("Then perform the review (run `ailang check` where relevant; you may download a ")
	b.WriteString("pinned Linux `ailang` release binary over the same egress) and emit the structured ")
	b.WriteString("verdict JSON as instructed below.\n\n")

	return b.String(), nil
}

// ValidateCloneFlags enforces the CLI clone-flag rules independently of any
// dispatch, so it is unit-testable without a live executor:
//
//   - --clone-sha without --clone-repo         → error
//   - --clone-repo with --api-only             → error (API path has no sandbox)
//   - --clone-repo on a non-egress resolution  → error (resolvedExecName must be
//     an executor advertising CapNetworkEgress; today only managed_agents)
//   - --clone-repo not a plain https URL       → error (ValidateCloneURL)
//
// It returns whether egress is requested (cloneRepo non-empty) so the caller can
// set Task.RequiresEgress. When no clone flags are set it is a no-op (false, nil).
func ValidateCloneFlags(cloneRepo, cloneSHA string, apiOnly bool, resolvedExecName string, egressCapable bool) (requiresEgress bool, err error) {
	cloneRepo = strings.TrimSpace(cloneRepo)
	cloneSHA = strings.TrimSpace(cloneSHA)

	if cloneSHA != "" && cloneRepo == "" {
		return false, fmt.Errorf("--clone-sha requires --clone-repo")
	}
	if cloneRepo == "" {
		return false, nil
	}
	if apiOnly {
		return false, fmt.Errorf("--clone-repo is not supported with --api-only (the API path has no sandbox to clone into); use the agentic gemini executor")
	}
	if !egressCapable {
		return false, fmt.Errorf("--clone-repo requires an egress-capable agentic executor (only 'gemini'/managed_agents qualifies), but the resolved executor %q does not support network egress", resolvedExecName)
	}
	if err := ValidateCloneURL(cloneRepo); err != nil {
		return false, err
	}
	if cloneSHA != "" && !IsValidSHA40(cloneSHA) {
		return false, fmt.Errorf("--clone-sha must be a 40-hex git SHA, got %q", cloneSHA)
	}
	return true, nil
}
