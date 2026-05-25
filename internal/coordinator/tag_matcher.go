package coordinator

import (
	"os"
	"strings"
)

// TagMatches reports whether a worker advertising `advertised` tags can serve
// a task that `required` a set of tags. Semantics:
//
//   - Empty required → matches anything (no constraint). Preserves backwards
//     compatibility with the pre-tag world: a message with no `requires`
//     attribute is claimable by any worker.
//   - Non-empty required → every required tag must be satisfied by at least
//     one advertised tag (set inclusion).
//   - A tag is "satisfied" if the advertised tag equals the required tag
//     exactly, OR if either side uses the prefix-glob form `family:*` that
//     covers the other side's specific tag in the same family.
//
// Case-sensitive throughout. Empty strings in either list are ignored.
func TagMatches(required, advertised []string) bool {
	reqs := dedupeNonEmpty(required)
	if len(reqs) == 0 {
		return true
	}
	ads := dedupeNonEmpty(advertised)
	if len(ads) == 0 {
		return false
	}
	for _, r := range reqs {
		matched := false
		for _, a := range ads {
			if tagPatternMatches(r, a) || tagPatternMatches(a, r) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

// tagPatternMatches reports whether `tag` is covered by `pattern`. Pattern
// supports two forms:
//
//   - Exact match: "ollama:gemma4-26b" matches only "ollama:gemma4-26b".
//   - Prefix glob: a pattern ending in "*" matches any tag whose literal
//     prefix is the part before "*". So "ollama:*" matches "ollama:gemma4-26b"
//     and "ollama:" (empty suffix) but NOT "qwen:30b".
//
// "*" alone matches everything. A star anywhere other than the final position
// is treated literally — we intentionally do NOT support full glob semantics
// because they invite ambiguity in tag routing.
func tagPatternMatches(pattern, tag string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") && !strings.Contains(pattern[:len(pattern)-1], "*") {
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(tag, prefix)
	}
	return pattern == tag
}

// dedupeNonEmpty returns the input list with empty strings removed and
// duplicates collapsed (preserving first occurrence order).
func dedupeNonEmpty(xs []string) []string {
	if len(xs) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(xs))
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if x == "" {
			continue
		}
		if _, dup := seen[x]; dup {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

// ResolveHostID returns the explicit host_id if non-empty, otherwise falls
// back to os.Hostname(). If os.Hostname() fails (rare), returns "unknown-host"
// so the value is never empty — empty values would break routing.
func ResolveHostID(explicit string) string {
	if explicit != "" {
		return explicit
	}
	h, err := os.Hostname()
	if err != nil || h == "" {
		return "unknown-host"
	}
	return h
}
