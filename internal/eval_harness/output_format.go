package eval_harness

import "strings"

// IsOutputFormatFailure reports whether a stdout mismatch is purely cosmetic:
// every expected line is present and correct, but carried a label prefix or
// differed in surrounding whitespace.
//
// M-EVAL-FAILURE-ATTRIBUTION. This does NOT change the verdict — a run that
// trips this still FAILS, because byte-exact stdout is the determinism contract
// AILANG is built around and the teaching prompt warns about it explicitly. It
// changes only the CATEGORY, so `logic_error` stops meaning both "wrong answer"
// and "right answer, labelled".
//
// # Deliberately conservative
//
// The failure mode to avoid is the mirror of the bug this fixes: over-matching
// here would quietly reclassify genuine wrong answers as cosmetic and inflate
// apparent capability. So a naive `strings.HasSuffix(got, want)` is NOT enough —
// expected "3" against produced "13" satisfies it, and that is a wrong answer.
//
// Two rules, both requiring a structural match:
//
//  1. line counts are equal (no missing or extra output), and
//  2. each produced line either equals its expected line after trimming, or is
//     that expected line preceded by a LABEL — a prefix terminated by ':' or
//     '=' plus optional space. The delimiter requirement is what stops "13"
//     from reading as a labelled "3".
//
// Anything else stays logic_error. When in doubt this returns false: a genuine
// failure mis-labelled as cosmetic is far worse than a cosmetic failure left in
// the logic_error bucket.
func IsOutputFormatFailure(expected, actual string) bool {
	wantLines := splitNonEmptyLines(expected)
	gotLines := splitNonEmptyLines(actual)

	if len(wantLines) == 0 || len(wantLines) != len(gotLines) {
		return false
	}

	sawDifference := false
	for i, want := range wantLines {
		got := gotLines[i]
		switch {
		case got == want:
			// Identical after trimming — no information either way.
		case isLabelledValue(got, want):
			sawDifference = true
		default:
			return false
		}
	}

	// If every line matched exactly, stdout_ok would not have been false for a
	// reason this function can explain (trailing-newline or interior-blank-line
	// differences). Report false and leave it as logic_error rather than
	// claiming a diagnosis we did not make.
	return sawDifference
}

// isLabelledValue reports whether got is want preceded by a label, where a
// label is a non-empty prefix ending in ':' or '=' (with optional trailing
// space). The delimiter is mandatory — that is the whole safety property.
func isLabelledValue(got, want string) bool {
	if !strings.HasSuffix(got, want) || got == want {
		return false
	}
	prefix := strings.TrimSuffix(got, want)
	prefix = strings.TrimRight(prefix, " \t")
	if prefix == "" {
		return false
	}
	last := prefix[len(prefix)-1]
	return last == ':' || last == '='
}

// splitNonEmptyLines splits on newlines, trims each line, and drops blanks so
// that trailing-newline differences do not by themselves defeat the comparison.
func splitNonEmptyLines(s string) []string {
	raw := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		if t := strings.TrimSpace(line); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// CategorizeErrorWithOutput is CategorizeError plus the cosmetic-mismatch
// refinement. Callers that have the expected and actual stdout to hand should
// prefer it; CategorizeError keeps its exact existing behaviour for callers
// that do not, so no existing reader of error_category changes meaning.
func CategorizeErrorWithOutput(compileOk, runtimeOk, stdoutOk bool, expected, actual string) string {
	cat := CategorizeError(compileOk, runtimeOk, stdoutOk)
	if cat == ErrorCategoryLogic && IsOutputFormatFailure(expected, actual) {
		return ErrorCategoryOutputFormat
	}
	return cat
}
