package main

import "testing"

// TestDistillAgentLine covers the compact agent-format renderer (M-AILANG-SEMANTIC-CONTEXT R1):
// structured errors use their fields; not-yet-structured errors have their embedded
// "at file:line:col:" location distilled to the front and the noisy framing stripped.
func TestDistillAgentLine(t *testing.T) {
	cases := []struct {
		name string
		in   checkJSONError
		file string
		want string
	}{
		{
			name: "structured error uses fields + suggestion",
			in:   checkJSONError{Code: "MOD010", Message: "module path mismatch", File: "a.ail", Line: 1, Column: 8, Suggestion: "rename module to a"},
			file: "a.ail",
			want: "a.ail:1:8 MOD010: module path mismatch → rename module to a",
		},
		{
			name: "fallback type error distills embedded location and code",
			in:   checkJSONError{Code: "ERROR", Message: "type error in scratch_r1 (decl 0): at scratch_r1.ail:3:20: No instance for Num[string] in scope. Import std/prelude or define instance", File: "scratch_r1.ail"},
			file: "scratch_r1.ail",
			want: "scratch_r1.ail:3:20 TYPE_ERROR: No instance for Num[string] in scope. Import std/prelude or define instance",
		},
		{
			name: "fallback without parseable location keeps message, uses fallback file",
			in:   checkJSONError{Code: "ERROR", Message: "something went wrong"},
			file: "b.ail",
			want: "b.ail ERROR: something went wrong",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := distillAgentLine(tc.in, tc.file); got != tc.want {
				t.Errorf("distillAgentLine:\n got:  %q\n want: %q", got, tc.want)
			}
		})
	}
}
