package strutil_test

import (
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"

	"github.com/sunholo-data/ailang/internal/strutil"
)

func TestTruncateMatchesTheOldByteSemanticsForASCII(t *testing.T) {
	old := func(s string, max int) string { // the eight `s[:max-3] + "..."` copies
		if len(s) <= max {
			return s
		}
		return s[:max-3] + "..."
	}
	for _, s := range []string{"", "a", "hello", "the quick brown fox", "exactly-twenty-chars"} {
		for _, max := range []int{4, 5, 10, 20, 40} {
			if got, want := strutil.Truncate(s, max), old(s, max); got != want {
				t.Errorf("Truncate(%q, %d) = %q, old semantics gave %q", s, max, got, want)
			}
		}
	}
}

func TestTruncateIsRuneAware(t *testing.T) {
	s := "Zürich café — naïve résumé"
	got := strutil.Truncate(s, 10)
	if !utf8.ValidString(got) {
		t.Fatalf("Truncate produced invalid UTF-8: %q", got)
	}
	if n := utf8.RuneCountInString(got); n != 10 {
		t.Errorf("width = %d runes, want 10 (%q)", n, got)
	}
	if got != "Zürich "+strutil.Ellipsis {
		t.Errorf("got %q", got)
	}
	if strutil.Truncate(s, 100) != s {
		t.Error("a string within the cap must come back unchanged")
	}
	if got := strutil.Truncate("abcdef", 3); got != "abc" {
		t.Errorf("no room for the mark: got %q, want \"abc\"", got)
	}
	if got := strutil.Truncate("abcdef", 0); got != "abcdef" {
		t.Errorf("max 0 means no cap: got %q", got)
	}
}

func TestTruncateBytesCapsPayloadWithoutSplittingRunes(t *testing.T) {
	const mark = "…[truncated]"
	s := "report — with em dashes — and arrows → everywhere"
	for _, max := range []int{len(mark), len(mark) + 1, 20, 25, 30, len(s) - 1} {
		got := strutil.TruncateBytes(s, max, mark)
		if len(got) > max {
			t.Errorf("max=%d: %d bytes, cap exceeded (%q)", max, len(got), got)
		}
		if !utf8.ValidString(got) {
			t.Errorf("max=%d: invalid UTF-8 %q", max, got)
		}
		if got[len(got)-len(mark):] != mark {
			t.Errorf("max=%d: missing mark: %q", max, got)
		}
	}
	if strutil.TruncateBytes(s, len(s), mark) != s {
		t.Error("a string at the cap must come back unchanged")
	}
	if got := strutil.TruncateBytes("abc", 1, "…"); got != "…" {
		t.Errorf("a cap smaller than the mark still marks: %q", got)
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x")
	if strutil.FileExists(p) {
		t.Fatal("absent file reported present")
	}
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !strutil.FileExists(p) || !strutil.FileExists(dir) {
		t.Fatal("present file or dir reported absent")
	}
}

func TestSortedKeys(t *testing.T) {
	got := strutil.SortedKeys(map[string]int{"b": 1, "a": 2, "c": 3})
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("got %v", got)
	}
	if got := strutil.SortedKeys(map[string]struct{}{}); len(got) != 0 {
		t.Errorf("empty map: got %v", got)
	}
}
