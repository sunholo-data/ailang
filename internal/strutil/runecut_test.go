package strutil

import (
	"testing"
	"unicode/utf8"
)

func TestRuneSafeCut(t *testing.T) {
	s := "aé中😀b" // 1 + 2 + 3 + 4 + 1 bytes
	cases := map[int]int{
		-1: 0, 0: 0, 1: 1, 2: 1, 3: 3, 4: 3, 5: 3, 6: 6, 7: 6, 8: 6, 9: 6, 10: 10, 11: 11, 12: 11, 100: 11,
	}
	for max, want := range cases {
		got := RuneSafeCut(s, max)
		if got != want {
			t.Errorf("RuneSafeCut(%q, %d) = %d, want %d", s, max, got, want)
		}
		if !utf8.ValidString(s[:got]) {
			t.Errorf("RuneSafeCut(%q, %d) split a rune: %q", s, max, s[:got])
		}
	}
	// Not UTF-8 (a run of continuation bytes): fall back to the byte cut
	// rather than scanning without bound.
	junk := "\x80\x80\x80\x80\x80\x80"
	if got := RuneSafeCut(junk, 5); got != 5 {
		t.Errorf("RuneSafeCut(junk, 5) = %d, want 5", got)
	}
	// A stray continuation byte is a rune of its own: the cut after it stands.
	if got := RuneSafeCut("ab\x80cd", 3); got != 3 {
		t.Errorf("RuneSafeCut(ab\\x80cd, 3) = %d, want 3", got)
	}
}
