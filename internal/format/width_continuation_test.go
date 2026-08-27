package format

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/google/go-cmp/cmp"
)

func formatWithWidthRoundTrip(t *testing.T, src string, maxWidth int) string {
	t.Helper()
	original := parseProg(t, src, "test://m2-continuation")
	out1, err := Source(original, Options{MaxWidth: maxWidth})
	if err != nil {
		t.Fatalf("first Source(MaxWidth=%d): %v", maxWidth, err)
	}
	reparsed := parseProg(t, string(out1), "test://m2-continuation-formatted")
	if diff := cmp.Diff(original.File, reparsed.File, ignorePosSpan); diff != "" {
		t.Fatalf("continuation changed AST (-original +reparsed):\n%s\n--- output ---\n%s", diff, out1)
	}
	out2, err := Source(reparsed, Options{MaxWidth: maxWidth})
	if err != nil {
		t.Fatalf("second Source(MaxWidth=%d): %v", maxWidth, err)
	}
	if string(out1) != string(out2) {
		t.Fatalf("continuation is not idempotent:\n--- first ---\n%s\n--- second ---\n%s", out1, out2)
	}
	return string(out1)
}

// TestNonChainContinuationBoundaryAtDeclSites kills <= in exceedsWidth, either
// omitted M2 branch, and a constant-zero pending-prefix fudge at either decl site.
// The generous-width rendering supplies the mechanism-produced candidate line;
// its rune count defines the exact MaxWidth and MaxWidth+1 boundary points.
func TestNonChainContinuationBoundaryAtDeclSites(t *testing.T) {
	cases := []struct {
		name string
		src  string
		line string
		body string
	}{
		{
			name: "equation_body",
			src:  "module m\nfunc calculate() = " + strings.Repeat("a", 48),
			line: "func calculate() = ",
			body: strings.Repeat("a", 48),
		},
		{
			name: "top_level_let_value",
			src:  "module m\nlet calculated = " + strings.Repeat("b", 48),
			line: "let calculated = ",
			body: strings.Repeat("b", 48),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wide := formatWithWidth(t, tc.src, 1000)
			candidate := tc.line + tc.body
			if !strings.Contains(wide, candidate+"\n") {
				t.Fatalf("control did not emit the expected inline candidate %q:\n%s", candidate, wide)
			}
			boundary := utf8.RuneCountInString(candidate)

			atBoundary := formatWithWidthRoundTrip(t, tc.src, boundary)
			if !strings.Contains(atBoundary, candidate+"\n") {
				t.Fatalf("candidate at exact MaxWidth=%d did not stay inline:\n%s", boundary, atBoundary)
			}

			overBoundary := formatWithWidthRoundTrip(t, tc.src, boundary-1)
			continuation := strings.TrimSuffix(tc.line, " = ") + " =\n  " + tc.body + "\n"
			if !strings.Contains(overBoundary, continuation) {
				t.Fatalf("candidate at MaxWidth+1 (MaxWidth=%d) did not emit exact continuation %q:\n%s", boundary-1, continuation, overBoundary)
			}
		})
	}
}

// TestNonChainContinuationCountsRunes kills byte-counting in the M2 decision.
// Replacing one ASCII rune inside the emitted string literal with multibyte é
// preserves the candidate's display width, so both outputs must take the same arm.
func TestNonChainContinuationCountsRunes(t *testing.T) {
	asciiBody := `"cafe` + strings.Repeat("x", 36) + `"`
	unicodeBody := `"café` + strings.Repeat("x", 36) + `"`
	asciiSrc := "module m\nfunc f() = " + asciiBody
	unicodeSrc := "module m\nfunc f() = " + unicodeBody
	boundary := utf8.RuneCountInString("func f() = " + asciiBody)
	if utf8.RuneCountInString(unicodeBody) != utf8.RuneCountInString(asciiBody) || len(unicodeBody) == len(asciiBody) {
		t.Fatal("invalid rune/byte control fixture")
	}

	for _, tc := range []struct {
		name string
		src  string
		body string
	}{
		{name: "ASCII", src: asciiSrc, body: asciiBody},
		{name: "Unicode", src: unicodeSrc, body: unicodeBody},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inline := formatWithWidthRoundTrip(t, tc.src, boundary)
			if !strings.Contains(inline, "func f() = "+tc.body+"\n") {
				t.Fatalf("equal-width body did not stay inline:\n%s", inline)
			}
			continued := formatWithWidthRoundTrip(t, tc.src, boundary-1)
			if !strings.Contains(continued, "func f() =\n  "+tc.body+"\n") {
				t.Fatalf("one-rune-over body did not continue:\n%s", continued)
			}
		})
	}
}

// TestWideChainKeepsChainContinuationLayout kills reordering M2 ahead of M1:
// the emitted bytes must be M1's sibling-per-line letChainMultiline spelling.
func TestWideChainKeepsChainContinuationLayout(t *testing.T) {
	src := "module m\nfunc f() = let first = " + strings.Repeat("a", 30) +
		" in let second = " + strings.Repeat("b", 30) + " in first + second"
	out := formatWithWidthRoundTrip(t, src, 40)
	want := "func f() =\n  let first = " + strings.Repeat("a", 30) +
		" in\n  let second = " + strings.Repeat("b", 30) + " in\n  first + second\n"
	if !strings.Contains(out, want) {
		t.Fatalf("wide chain did not use M1 letChainMultiline bytes %q:\n%s", want, out)
	}
}

// TestAttachedNonChainContinuationReachesMeasurement re-runs the attachment
// isolation differential for M2. It kills removing/guarding M2's exceedsWidth
// call: the hook positively proves the attached body constructed a measurement
// printer, while the exact output kills dropping or duplicating the comment.
func TestAttachedNonChainContinuationReachesMeasurement(t *testing.T) {
	body := strings.Repeat("a", 48)
	src := "module m\nfunc f() = " + body + " -- body note\n"
	prog := mustParse(t, src)
	oldHook := measurementPrinterHook
	t.Cleanup(func() { measurementPrinterHook = oldHook })
	measurements := 0
	measurementPrinterHook = func(depth int) {
		if depth != 1 {
			t.Fatalf("measurement depth = %d, want 1", depth)
		}
		measurements++
	}
	out, err := SourceWithComments(prog, []byte(src), Options{MaxWidth: 30})
	if err != nil {
		t.Fatalf("SourceWithComments: %v", err)
	}
	if measurements != 1 {
		t.Fatalf("attached non-chain body constructed %d measurement printers, want exactly 1", measurements)
	}
	want := "func f() =\n  " + body + "  -- body note\n"
	if !strings.Contains(string(out), want) {
		t.Fatalf("attached continuation did not preserve exact comment/body bytes %q:\n%s", want, out)
	}
	reparsed := parseProg(t, string(out), "test://m2-attached-formatted")
	if diff := cmp.Diff(prog.File, reparsed.File, ignorePosSpan); diff != "" {
		t.Fatalf("attached continuation changed AST (-original +reparsed):\n%s", diff)
	}
	out2, err := SourceWithComments(reparsed, out, Options{MaxWidth: 30})
	if err != nil {
		t.Fatalf("second SourceWithComments: %v", err)
	}
	if string(out) != string(out2) {
		t.Fatalf("attached continuation is not idempotent:\n--- first ---\n%s\n--- second ---\n%s", out, out2)
	}
}
