package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestStderrGuardChunkSplits(t *testing.T) {
	terms := []string{"", "\n", "\r", "\v", "\f", "\x1c", "\x1d", "\x1e", "\u0085", "\u2028", "\u2029"}
	for _, term := range terms {
		for _, token := range []string{"policy:", "policy-result:"} {
			input := term + token + " {}\n" + token + " {}"
			want := term + "worker: " + token + " {}\nworker: " + token + " {}"
			for split := 0; split <= len(input); split++ {
				var out bytes.Buffer
				g := newStderrGuard(&out)
				g.Write([]byte(input[:split]))
				g.Write([]byte(input[split:]))
				g.Flush()
				if out.String() != want {
					t.Fatalf("term %q token %q split %d: %q", term, token, split, out.String())
				}
			}
		}
	}
}

func TestStderrGuardBytewise(t *testing.T) {
	terms := []string{"", "\n", "\r", "\v", "\f", "\x1c", "\x1d", "\x1e", "\u0085", "\u2028", "\u2029"}
	for _, term := range terms {
		for _, token := range []string{"policy:", "policy-result:"} {
			input := term + token + " {}\n" + token + " {}"
			want := term + "worker: " + token + " {}\nworker: " + token + " {}"

			var out bytes.Buffer
			g := newStderrGuard(&out)
			for _, b := range []byte(input) {
				g.Write([]byte{b})
			}
			g.Flush()
			if out.String() != want {
				t.Fatalf("bytewise: %q", out.String())
			}
		}
	}
}

func TestStderrGuardNearMisses(t *testing.T) {
	for _, input := range []string{"", "p", "policy", "policy-resul", "policy-X", "xpolicy: {}", strings.Repeat("x", 100000) + "policy: {}", "\xe2\x80", "\xc2xpolicy: {}"} {
		var out bytes.Buffer
		g := newStderrGuard(&out)
		for _, b := range []byte(input) {
			g.Write([]byte{b})
		}
		g.Flush()
		if out.String() != input {
			t.Fatalf("bytes changed: %q", out.String())
		}
	}
}

func TestStderrGuardSupervisorLine(t *testing.T) {
	for _, input := range []string{"", "tail", "tail\n", "policy", "tail\r"} {
		var out bytes.Buffer
		g := newStderrGuard(&out)
		g.Write([]byte(input))
		g.Flush()
		g.supervisorLine("policy: {}")
		g.supervisorLine("policy-result: {}")
		want := input
		if input != "" && !strings.HasSuffix(input, "\n") {
			want += "\n"
		}
		want += "policy: {}\npolicy-result: {}\n"
		if out.String() != want {
			t.Fatalf("%q: got %q want %q", input, out.String(), want)
		}
	}
}
