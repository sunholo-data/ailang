package eval_harness

import (
	"strings"
	"testing"
)

func TestNormalizeSource(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"crlf to lf", "a\r\nb", "a\nb"},
		{"bare cr to lf", "a\rb", "a\nb"},
		{"strips all trailing newlines", "abc\n\n\n", "abc"},
		{"preserves interior newlines", "a\n\nb\n", "a\n\nb"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeSource(tt.in); got != tt.want {
				t.Errorf("NormalizeSource(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSourceConstraintsCheck(t *testing.T) {
	tests := []struct {
		name       string
		sc         SourceConstraints
		code       string
		violations int
		contains   string // substring expected in first violation ("" = no check)
	}{
		{
			name:       "exact bytes pass",
			sc:         SourceConstraints{ExactBytes: 5},
			code:       "abcde\n", // trailing newline stripped -> 5
			violations: 0,
		},
		{
			name:       "exact bytes fail reports delta",
			sc:         SourceConstraints{ExactBytes: 5},
			code:       "abcdefg",
			violations: 1,
			contains:   "(+2)",
		},
		{
			name:       "max bytes pass at limit",
			sc:         SourceConstraints{MaxBytes: 3},
			code:       "abc",
			violations: 0,
		},
		{
			name:       "max bytes fail",
			sc:         SourceConstraints{MaxBytes: 3},
			code:       "abcd",
			violations: 1,
			contains:   "1 over",
		},
		{
			name:       "banned chars found with line number",
			sc:         SourceConstraints{BannedChars: "0123456789"},
			code:       "abc\ndef4\n",
			violations: 1,
			contains:   "line 2",
		},
		{
			name:       "banned chars absent",
			sc:         SourceConstraints{BannedChars: "0123456789"},
			code:       "print(len('xx'))",
			violations: 0,
		},
		{
			name:       "banned substring found",
			sc:         SourceConstraints{BannedSubstrings: []string{"import os"}},
			code:       "import sys\nimport os\n",
			violations: 1,
			contains:   "line 2",
		},
		{
			name:       "multiple violations all reported",
			sc:         SourceConstraints{ExactBytes: 1, BannedChars: "9"},
			code:       "x = 9",
			violations: 2,
		},
		{
			name:       "crlf does not inflate byte count",
			sc:         SourceConstraints{ExactBytes: 3},
			code:       "a\r\nb", // normalized "a\nb" = 3 bytes
			violations: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.sc.Check(tt.code)
			if len(got) != tt.violations {
				t.Fatalf("Check() = %d violations %v, want %d", len(got), got, tt.violations)
			}
			if tt.contains != "" && len(got) > 0 && !strings.Contains(got[0], tt.contains) {
				t.Errorf("violation %q does not contain %q", got[0], tt.contains)
			}
		})
	}
}

func TestSourceConstraintsValidate(t *testing.T) {
	if err := (&SourceConstraints{}).Validate(); err == nil {
		t.Error("empty constraints must be rejected")
	}
	if err := (&SourceConstraints{ExactBytes: 10, MaxBytes: 20}).Validate(); err == nil {
		t.Error("exact_bytes + max_bytes must be mutually exclusive")
	}
	if err := (&SourceConstraints{ExactBytes: 400}).Validate(); err != nil {
		t.Errorf("valid exact_bytes rejected: %v", err)
	}
	if err := (&SourceConstraints{BannedChars: "0123456789"}).Validate(); err != nil {
		t.Errorf("valid banned_chars rejected: %v", err)
	}
}

func TestSourceConstraintsMinBytesAndCharRun(t *testing.T) {
	sc := SourceConstraints{MinBytes: 10}
	if v := sc.Check("short"); len(v) != 1 || !strings.Contains(v[0], "5 short") {
		t.Errorf("min_bytes violation not reported: %v", v)
	}
	if v := sc.Check("exactly10!"); len(v) != 0 {
		t.Errorf("min_bytes false positive: %v", v)
	}

	run := SourceConstraints{MaxCharRun: 2}
	if v := run.Check("ab--cd"); len(v) != 0 {
		t.Errorf("run of 2 must pass: %v", v)
	}
	if v := run.Check("ab---cd"); len(v) != 1 || !strings.Contains(v[0], "3+ times") {
		t.Errorf("run of 3 must fail: %v", v)
	}
	if v := run.Check("aa\naa"); len(v) != 0 {
		t.Errorf("runs broken by newline must pass: %v", v)
	}

	if err := (&SourceConstraints{ExactBytes: 10, MinBytes: 5}).Validate(); err == nil {
		t.Error("exact_bytes + min_bytes must be rejected")
	}
	if err := (&SourceConstraints{MinBytes: 20, MaxBytes: 10}).Validate(); err == nil {
		t.Error("min > max must be rejected")
	}
	if err := (&SourceConstraints{ExactBytes: 512, MaxCharRun: 2}).Validate(); err != nil {
		t.Errorf("exact_bytes + max_char_run must be valid: %v", err)
	}
}
