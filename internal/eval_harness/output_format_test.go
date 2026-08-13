package eval_harness

import "testing"

func TestIsOutputFormatFailure(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		actual   string
		want     bool
	}{
		// The motivating case: tree_transformation_pipeline, or-deepseek-v4-pro-0813,
		// 2026-08-13 core run. Every value correct, every line labelled.
		{
			name:     "labelled values, the real regression",
			expected: "[1, 2, 3, 4, 5, 6, 7]\n[2, 4, 6, 8, 10, 12, 14]\n28\n3\n",
			actual: "treeToList(tree): [1, 2, 3, 4, 5, 6, 7]\n" +
				"treeToList(mapTree(double, tree)): [2, 4, 6, 8, 10, 12, 14]\n" +
				"foldTree(add, 0, tree): 28\n" +
				"treeDepth(tree): 3\n",
			want: true,
		},
		{name: "equals delimiter", expected: "40", actual: "result = 40", want: true},
		{name: "colon no space", expected: "40", actual: "Result:40", want: true},
		{name: "one labelled line among exact ones", expected: "1\n2", actual: "1\nsecond: 2", want: true},

		// THE safety property. A naive HasSuffix would call each of these
		// cosmetic and silently convert a wrong answer into a formatting nit —
		// the mirror of the bug this whole change exists to fix.
		{name: "wrong number sharing a suffix", expected: "3", actual: "13", want: false},
		{name: "wrong number, longer", expected: "40", actual: "1240", want: false},
		{name: "prefix without a delimiter", expected: "40", actual: "answer 40", want: false},
		{name: "negated value", expected: "5", actual: "-5", want: false},

		// Structural mismatches are never cosmetic.
		{name: "missing a line", expected: "1\n2\n3", actual: "a: 1\nb: 2", want: false},
		{name: "extra line", expected: "1", actual: "a: 1\nb: 2", want: false},
		{name: "genuinely wrong values", expected: "1\n2", actual: "a: 9\nb: 8", want: false},
		{name: "empty expected", expected: "", actual: "anything", want: false},
		{name: "empty actual", expected: "1", actual: "", want: false},

		// Exact match reports false: stdout_ok being false was not explained by
		// this function, so it must not claim the diagnosis.
		{name: "identical modulo trailing newline", expected: "40\n", actual: "40", want: false},
		{name: "identical", expected: "40", actual: "40", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsOutputFormatFailure(tt.expected, tt.actual); got != tt.want {
				t.Errorf("IsOutputFormatFailure(%q, %q) = %v, want %v",
					tt.expected, tt.actual, got, tt.want)
			}
		})
	}
}

func TestCategorizeErrorWithOutput(t *testing.T) {
	tests := []struct {
		name                           string
		compileOk, runtimeOk, stdoutOk bool
		expected, actual               string
		want                           string
	}{
		{
			name:      "cosmetic mismatch becomes output_format",
			compileOk: true, runtimeOk: true, stdoutOk: false,
			expected: "40", actual: "result: 40",
			want: ErrorCategoryOutputFormat,
		},
		{
			name:      "genuine wrong answer stays logic_error",
			compileOk: true, runtimeOk: true, stdoutOk: false,
			expected: "40", actual: "41",
			want: ErrorCategoryLogic,
		},
		// The refinement must only ever touch logic_error. A compile or runtime
		// failure has no meaningful stdout to compare.
		{
			name:      "compile failure unaffected",
			compileOk: false, runtimeOk: false, stdoutOk: false,
			expected: "40", actual: "result: 40",
			want: ErrorCategoryCompile,
		},
		{
			name:      "runtime failure unaffected",
			compileOk: true, runtimeOk: false, stdoutOk: false,
			expected: "40", actual: "result: 40",
			want: ErrorCategoryRuntime,
		},
		{
			name:      "success unaffected",
			compileOk: true, runtimeOk: true, stdoutOk: true,
			expected: "40", actual: "40",
			want: ErrorCategoryNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CategorizeErrorWithOutput(tt.compileOk, tt.runtimeOk, tt.stdoutOk, tt.expected, tt.actual)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// CategorizeError must keep its exact existing behaviour so that no current
// reader of error_category changes meaning as a side effect of this work.
func TestCategorizeErrorUnchanged(t *testing.T) {
	cases := []struct {
		compileOk, runtimeOk, stdoutOk bool
		want                           string
	}{
		{false, false, false, ErrorCategoryCompile},
		{true, false, false, ErrorCategoryRuntime},
		{true, true, false, ErrorCategoryLogic},
		{true, true, true, ErrorCategoryNone},
	}
	for _, c := range cases {
		if got := CategorizeError(c.compileOk, c.runtimeOk, c.stdoutOk); got != c.want {
			t.Errorf("CategorizeError(%v,%v,%v) = %q, want %q",
				c.compileOk, c.runtimeOk, c.stdoutOk, got, c.want)
		}
	}
}
