package eval_harness

import (
	"reflect"
	"testing"
)

func TestAIHandlerArgs(t *testing.T) {
	tests := []struct {
		name string
		caps []string
		want []string
	}{
		// The regression this fixes: ai_effect_json_schema declares
		// caps: ["AI", "IO"] and used to run with --caps but no handler,
		// failing with "no AI model configured" banked as runtime_error.
		{"AI plus IO, the real benchmark shape", []string{"AI", "IO"}, []string{"--ai-stub"}},
		{"AI alone", []string{"AI"}, []string{"--ai-stub"}},
		{"lowercase, author-written YAML", []string{"ai"}, []string{"--ai-stub"}},
		{"mixed case with padding", []string{" Ai "}, []string{"--ai-stub"}},

		// Must NOT fire otherwise — a spurious --ai-stub on a non-AI benchmark
		// would change the command line for every run in the suite.
		{"no caps at all", nil, nil},
		{"empty slice", []string{}, nil},
		{"IO and FS only", []string{"IO", "FS"}, nil},
		{"substring must not match", []string{"AILANG"}, nil},
		{"suffix must not match", []string{"OPENAI"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := aiHandlerArgs(tt.caps)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("aiHandlerArgs(%q) = %q, want %q", tt.caps, got, tt.want)
			}
		})
	}
}
