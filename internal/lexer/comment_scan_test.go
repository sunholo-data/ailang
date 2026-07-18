package lexer

import "testing"

// comment_scan_test.go verifies ScanForComment (the opt-in comment preflight)
// detects real comments correctly AND leaves NextToken() output byte-for-byte
// unchanged — the parser-visible token stream must be unaffected.

func TestScanForComment_Detection(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want bool
	}{
		{"leading_dash", "-- comment\nx", true},
		{"trailing_dash", "x -- comment", true},
		{"slash_slash", "x // comment", true},
		{"no_comment", "let x = 1", false},
		{"minus_operator", "a - b", false},
		{"string_with_dashes", `"a -- b"`, false},
		{"string_with_slashes", `"http://x"`, false},
		{"char_dash", `'-'`, false},
		{"quasiquote_dashes", `sql"""SELECT -- x"""`, false},
		{"string_then_real_comment", `"http://x" -- real`, true},
		{"escaped_quote_then_dash", `"a \" -- b"`, false},
		{"empty", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ScanForComment([]byte(tc.src))
			if got != tc.want {
				t.Errorf("ScanForComment(%q) = %v, want %v", tc.src, got, tc.want)
			}
		})
	}
}

// TestNextTokenUnaffectedByScan proves that running ScanForComment does not alter
// the token stream a subsequent fresh lexer produces (they are independent), and
// that comments are still skipped exactly as before on the parser path.
func TestNextTokenUnaffectedByScan(t *testing.T) {
	sources := []string{
		"let x = 1 -- trailing\nlet y = 2",
		"module m\nfunc f() = 1 // c\n",
		`func g() = "http://x -- y"`,
		"a - b - c",
		`sql"""SELECT -- not a comment"""`,
	}
	for _, src := range sources {
		// Baseline token stream.
		before := collectTokens(src)
		// Run the comment scan (must not touch any shared lexer state).
		_ = ScanForComment([]byte(src))
		// Token stream after the scan must be identical.
		after := collectTokens(src)

		if len(before) != len(after) {
			t.Fatalf("token count changed for %q: %d -> %d", src, len(before), len(after))
		}
		for i := range before {
			if before[i].Type != after[i].Type || before[i].Literal != after[i].Literal {
				t.Errorf("token %d changed for %q:\n before {%v %q}\n after  {%v %q}",
					i, src, before[i].Type, before[i].Literal, after[i].Type, after[i].Literal)
			}
		}
	}
}

// TestCommentsStillSkippedByLexer is a regression guard: the lexer's own comment
// skipping (the parser path) is unchanged — a `--` comment produces no token.
func TestCommentsStillSkippedByLexer(t *testing.T) {
	toks := collectTokens("x -- comment\ny")
	// Expect: IDENT(x), IDENT(y), EOF — the comment yields no token.
	if len(toks) != 3 {
		t.Fatalf("expected 3 tokens (x, y, EOF), got %d: %v", len(toks), toks)
	}
	if toks[0].Literal != "x" || toks[1].Literal != "y" || toks[2].Type != EOF {
		t.Errorf("unexpected token stream: %v", toks)
	}
	for _, tok := range toks {
		if tok.Type == COMMENT {
			t.Error("lexer must not emit COMMENT tokens on the parser path")
		}
	}
}
