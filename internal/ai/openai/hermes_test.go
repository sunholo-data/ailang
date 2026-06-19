package openai

import "testing"

func TestExtractHermesToolCalls(t *testing.T) {
	cases := []struct {
		name      string
		in        string
		wantNames []string
		wantArgs  []string
	}{
		{name: "none", in: "just some prose, no tool call", wantNames: nil},
		{
			name:      "single in reasoning, nested braces in content",
			in:        `I'll write it. <tool_call>{"name":"WriteFile","arguments":{"path":"solution.ail","content":"module m\n{ x }"}}</tool_call>`,
			wantNames: []string{"WriteFile"},
			wantArgs:  []string{`{"path":"solution.ail","content":"module m\n{ x }"}`},
		},
		{
			name:      "two calls",
			in:        "<tool_call>{\"name\":\"ReadFile\",\"arguments\":{\"path\":\"a\"}}</tool_call>\nthen\n<tool_call>{\"name\":\"BashExec\",\"arguments\":{\"cmd\":\"ls\"}}</tool_call>",
			wantNames: []string{"ReadFile", "BashExec"},
		},
		{
			name:      "malformed json skipped",
			in:        `<tool_call>{not json}</tool_call>`,
			wantNames: nil,
		},
		{
			name:      "missing arguments defaults to empty object",
			in:        `<tool_call>{"name":"RunTests"}</tool_call>`,
			wantNames: []string{"RunTests"},
			wantArgs:  []string{"{}"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractHermesToolCalls(c.in)
			if len(got) != len(c.wantNames) {
				t.Fatalf("got %d calls, want %d (%v)", len(got), len(c.wantNames), got)
			}
			for i, want := range c.wantNames {
				if got[i].Name != want {
					t.Errorf("call[%d].Name = %q, want %q", i, got[i].Name, want)
				}
			}
			for i, want := range c.wantArgs {
				if got[i].Arguments != want {
					t.Errorf("call[%d].Arguments = %q, want %q", i, got[i].Arguments, want)
				}
			}
		})
	}
}
