package main

import "testing"

func TestCellEscapesMDXOutsideCodeSpans(t *testing.T) {
	cases := []struct{ in, want string }{
		// The 2026-10-15 docs-build break: a bare JSON object read as a JSX expression.
		{`e.g. '{"sort":"throughput"}'`, `e.g. '&#123;"sort":"throughput"&#125;'`},
		{"root <dir>/compile", "root &lt;dir&gt;/compile"},
		// Code spans are literal in MDX: entities there would render verbatim.
		{"`GET /api/<id>` and `{}`", "`GET /api/<id>` and `{}`"},
		{"a|b\nc", `a\|b c`},
	}
	for _, c := range cases {
		if got := cell(c.in); got != c.want {
			t.Errorf("cell(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
