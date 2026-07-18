package format

import "testing"

// comments_test.go proves HasComments is a lossless lexical preflight, not a
// substring search: `--` and `//` introducers inside strings, char literals,
// regex literals, and triple-quoted quasiquote templates are NOT comments, while
// real leading/trailing/floating comments ARE detected.

func TestHasComments_DetectsRealComments(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"leading", "-- a leading comment\nmodule m\nfunc f() = 1\n"},
		{"trailing_same_line", "module m\nfunc f() = 1 -- trailing comment\n"},
		{"floating_between_decls", "module m\nfunc a() = 1\n\n-- floating comment\n\nfunc b() = 2\n"},
		{"slash_slash", "module m\nfunc f() = 1 // c-style comment\n"},
		{"double_dash_only", "-- just a comment\n"},
		{"comment_before_close", "module m\nfunc f() {\n  let a = 1\n  -- before close\n  a\n}\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			has, err := HasComments([]byte(tc.src))
			if err != nil {
				t.Fatal(err)
			}
			if !has {
				t.Errorf("expected a comment to be detected in:\n%s", tc.src)
			}
		})
	}
}

func TestHasComments_IgnoresCommentLikeTextInLiterals(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"string_double_dash", `module m` + "\n" + `func f() = "a -- b"` + "\n"},
		{"string_slash_slash", `module m` + "\n" + `func f() = "http://example.com"` + "\n"},
		{"char_dash", `module m` + "\n" + `func f() = '-'` + "\n"},
		{"quasiquote_double_dash", `module m` + "\n" + `func f() = sql"""SELECT -- not a comment"""` + "\n"},
		{"quasiquote_slash", `module m` + "\n" + `func f() = url"""http://x.y"""` + "\n"},
		{"string_with_escaped_quote_then_dashes", `module m` + "\n" + `func f() = "quote \" then -- dashes"` + "\n"},
		{"minus_operator_not_comment", "module m\nfunc f(a, b) = a - b\n"},
		{"single_minus_then_space", "module m\nfunc f(x) = x - 1\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			has, err := HasComments([]byte(tc.src))
			if err != nil {
				t.Fatal(err)
			}
			if has {
				t.Errorf("comment-like text in a literal must NOT be detected as a comment:\n%s", tc.src)
			}
		})
	}
}

// TestHasComments_MixedLiteralThenRealComment ensures a real comment AFTER a
// literal containing comment-like text is still found (the scanner resumes
// correctly past the literal).
func TestHasComments_MixedLiteralThenRealComment(t *testing.T) {
	src := `module m` + "\n" + `func f() = "url http://x" -- real comment here` + "\n"
	has, err := HasComments([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Errorf("expected the trailing real comment to be detected past the string literal:\n%s", src)
	}
}

// TestHasComments_CleanFile confirms a comment-free file reports false.
func TestHasComments_CleanFile(t *testing.T) {
	src := "module m\n\nfunc f() = let a = 1; a\n"
	has, err := HasComments([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Errorf("clean file must report no comments:\n%s", src)
	}
}
