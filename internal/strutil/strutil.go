// Package strutil holds the small helpers that every package used to carry
// its own copy of — truncation, a file-exists probe, sorted map keys. Stdlib
// leaf; language-core and platform packages both import it.
//
// Truncation had three semantics across eleven copies (cut at max-3 bytes
// plus "...", cut at max bytes plus "...", and a rune-safe byte cap with a
// mark). The two that survive are named by what the cap measures:
// Truncate caps display WIDTH in runes, TruncateBytes caps payload SIZE in
// bytes. Both are rune-safe — the byte-index copies split multi-byte
// characters and emitted invalid UTF-8. M-V1-SIMPLIFY-S3 M5.
package strutil

import (
	"os"
	"sort"
	"unicode/utf8"
)

// Ellipsis is what Truncate appends. Three ASCII dots, not U+2026, so the
// result is the same width in a terminal and in a fixed-width table.
const Ellipsis = "..."

// Truncate caps s at max runes of display width. Policy (the one policy —
// display.Truncate and telemetry.Truncate delegate here):
//
//   - max <= 0 means no cap: s comes back unchanged.
//   - s within the cap comes back unchanged.
//   - otherwise the first max-3 runes plus Ellipsis, so the result is never
//     wider than max. For ASCII input this is byte-identical to the old
//     `s[:max-3] + "..."`; for multi-byte input it no longer splits a rune.
//   - when the cap leaves no room for the mark (max <= 3) the first max
//     runes are returned bare — a mark that consumes the whole budget says
//     nothing.
func Truncate(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	keep := max - len(Ellipsis)
	mark := Ellipsis
	if keep <= 0 {
		keep, mark = max, ""
	}
	i := 0
	for n := 0; n < keep; n++ {
		_, size := utf8.DecodeRuneInString(s[i:])
		i += size
	}
	return s[:i] + mark
}

// TruncateBytes caps s at max BYTES — a payload limit, not a display one —
// appending mark when it cuts, and never splitting a rune. The result is at
// most max bytes as long as mark itself fits.
func TruncateBytes(s string, max int, mark string) string {
	if len(s) <= max {
		return s
	}
	keep := max - len(mark)
	if keep < 0 {
		keep = 0
	}
	for keep > 0 && !utf8.RuneStart(s[keep]) {
		keep--
	}
	return s[:keep] + mark
}

// FileExists reports whether path names something that can be stat'ed.
// A permission error reads as absent, the way every prior copy read it.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// SortedKeys returns the keys of m in ascending order, for deterministic
// iteration and output.
func SortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
