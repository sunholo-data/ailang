package fileguard

import (
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Write protection inside a sandbox root: the `.git` metadata guard
// (M-EXECUTOR-POLICY-HARDENING M6) and the policy's fs_deny_write patterns
// (M7). This is the ONE matcher the FS effect (internal/effects) and the
// agent tool endpoint (internal/policytool) share — they used to carry two
// copies of the `.git` check, and both compared deny patterns
// case-sensitively (#1559).
//
// Matching is case- and normalization-insensitive on every platform. On a
// case-insensitive volume (macOS APFS by default, Windows NTFS)
// `.CLAUDE/settings.json` IS `.claude/settings.json`, and on APFS a
// decomposed `é` names the same file as a precomposed one; matching the
// spelling instead of the file let a confined program write into a
// deny-listed directory by changing case. Folding fails closed: on a
// case-sensitive volume (most Linux) `.CLAUDE/x` is a different file and is
// refused anyway — over-denying an exotic spelling costs nothing, under-
// denying the real one is the hole.

// Protection is the write-side policy for paths relative to a sandbox root.
type Protection struct {
	// GitDir: any path with a `.git` component is read-only.
	GitDir bool
	// DenyWrite: the policy's fs_deny_write patterns.
	DenyWrite []string
}

// Violation says why a path is protected. The zero value means writable.
type Violation struct {
	// Ancestor: the path can contain a deny-pattern target.
	Ancestor bool
	// GitDir: the path has a `.git` component.
	GitDir bool
	// Pattern: the first fs_deny_write pattern the path matched.
	Pattern string
	// HardLink: the file has more than one hard link, so another name for
	// it may be protected (Root.CheckPath).
	HardLink bool
	// Via: the resolved root-relative name that matched, when it differs
	// from the spelled one (an in-root symlink; Root.CheckPath).
	Via string
}

// Protected reports whether v refuses the write.
func (v Violation) Protected() bool { return v.GitDir || v.Pattern != "" || v.HardLink }

// Check applies the protection to rel, a path relative to the sandbox root
// (slash- or OS-separated; it is cleaned here, so `src/../.git/config` is
// `.git/config`).
func (p Protection) Check(rel string) Violation {
	if p.GitDir && UnderGitDir(rel) {
		return Violation{GitDir: true}
	}
	if pat := MatchDenyWrite(p.DenyWrite, rel); pat != "" {
		return Violation{Pattern: pat}
	}
	return Violation{}
}

// UnderGitDir reports whether rel has a `.git` component, case- and
// normalization-folded. `.gitignore` and `.gitattributes` are ordinary files.
func UnderGitDir(rel string) bool {
	gitKey := foldKey(".git")
	for _, seg := range strings.Split(cleanRel(rel), "/") {
		if foldKey(seg) == gitKey {
			return true
		}
	}
	return false
}

// MatchDenyWrite returns the first pattern rel matches, or "". Patterns are
// globs over slash-separated components: `*`, `?` and `[...]` match within
// one component (path.Match), and a `**` component matches zero or more
// whole components, so `**/*.test.ts` protects `a.test.ts`,
// `src/a.test.ts` and `src/deep/a.test.ts` alike and `docs/**` covers
// `docs` and everything beneath it. A pattern with a slash is anchored at
// the sandbox root (`tests/**` is not `cli/tests/x`; write `**/tests/**`
// for that); a pattern without one is also matched against the base name,
// so `*.yml` protects YAML files at any depth. Pattern and path are both
// folded (see the file comment) before comparison; the returned pattern is
// the operator's original spelling. A malformed glob matches (fails closed).
//
// Before 2026-10-17 (AILANG World report) a `**` that was not a trailing `/**` went to path.Match, where
// it is just `*`: `**/*.test.ts` protected exactly one directory level.
func MatchDenyWrite(patterns []string, rel string) string {
	key := foldKey(cleanRel(rel))
	parts := strings.Split(key, "/")
	base := parts[len(parts)-1]
	for _, orig := range patterns {
		pat := normPattern(orig)
		if matchComponents(strings.Split(pat, "/"), parts) {
			return orig
		}
		if !strings.Contains(pat, "/") {
			if m, err := path.Match(pat, base); m || err != nil {
				return orig
			}
		}
	}
	return ""
}

// normPattern is the folded, slash-separated form of an operator pattern.
func normPattern(orig string) string {
	return foldKey(strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(orig)), "./"))
}

// matchComponents matches pattern components against path components, a
// `**` component standing for zero or more of them.
func matchComponents(pats, parts []string) bool {
	if len(pats) == 0 {
		return len(parts) == 0
	}
	if pats[0] == "**" {
		for i := 0; i <= len(parts); i++ {
			if matchComponents(pats[1:], parts[i:]) {
				return true
			}
		}
		return false
	}
	if len(parts) == 0 {
		return false
	}
	if m, err := path.Match(pats[0], parts[0]); !m && err == nil {
		return false
	}
	return matchComponents(pats[1:], parts[1:])
}

// cleanRel is the slash-separated, cleaned form of a root-relative path with
// no leading "./".
func cleanRel(rel string) string {
	c := path.Clean(filepath.ToSlash(rel))
	return strings.TrimPrefix(c, "./")
}

// foldKey is the comparison key for a path or pattern: NFC-normalized (APFS
// is normalization-insensitive), then every rune replaced by the smallest
// member of its Unicode case-folding orbit. The orbit, not strings.ToLower:
// ToLower leaves `ſ` (U+017F, which folds to `s`) and the Kelvin sign alone
// in some directions, and a volume that folds them would see the same file.
func foldKey(s string) string {
	s = norm.NFC.String(s)
	return strings.Map(foldRune, s)
}

func foldRune(r rune) rune {
	m := r
	for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
		if f < m {
			m = f
		}
	}
	return m
}

// CheckMove protects both direct targets and ancestors for rename/remove.
// Ordinary writes and mkdirs should continue to use Check.
func (p Protection) CheckMove(rel string) Violation {
	if v := p.Check(rel); v.Protected() {
		return v
	}
	if pat := MatchDenyMove(p.DenyWrite, rel); pat != "" {
		return Violation{Pattern: pat, Ancestor: true}
	}
	return Violation{}
}

// MatchDenyMove returns a direct match first, then the first pattern that
// can match strictly beneath rel. Basename matches are deliberately excluded
// from the latter check: those patterns follow files across directory moves.
func MatchDenyMove(patterns []string, rel string) string {
	if pat := MatchDenyWrite(patterns, rel); pat != "" {
		return pat
	}
	key := foldKey(cleanRel(rel))
	for _, orig := range patterns {
		if mayMatchBeneath(normPattern(orig), key) {
			return orig
		}
	}
	return ""
}

// mayMatchBeneath compares folded slash-separated components without touching
// the filesystem. A remaining component represents a protected descendant.
// Encountered malformed globs and ** fail closed.
func mayMatchBeneath(pattern, rel string) bool {
	pats, parts := strings.Split(pattern, "/"), strings.Split(rel, "/")
	for i, part := range parts {
		if i >= len(pats) {
			return false
		}
		if pats[i] == "**" {
			return true
		}
		matched, err := path.Match(pats[i], part)
		if err != nil {
			return true
		}
		if !matched {
			return false
		}
	}
	return len(pats) > len(parts)
}
