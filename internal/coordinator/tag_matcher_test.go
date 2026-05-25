package coordinator

import (
	"testing"
)

func TestTagMatches(t *testing.T) {
	tests := []struct {
		name       string
		required   []string
		advertised []string
		want       bool
	}{
		{
			name:       "empty required matches any advertised",
			required:   nil,
			advertised: []string{"ollama:gemma4-26b-ailang"},
			want:       true,
		},
		{
			name:       "empty required, empty advertised — match",
			required:   nil,
			advertised: nil,
			want:       true,
		},
		{
			name:       "non-empty required, empty advertised — no match",
			required:   []string{"ollama:gemma4-26b-ailang"},
			advertised: nil,
			want:       false,
		},
		{
			name:       "exact single tag match",
			required:   []string{"ollama:gemma4-26b-ailang"},
			advertised: []string{"ollama:gemma4-26b-ailang"},
			want:       true,
		},
		{
			name:       "exact single tag mismatch",
			required:   []string{"ollama:gemma4-26b-ailang"},
			advertised: []string{"ollama:qwen3-coder-30b"},
			want:       false,
		},
		{
			name:       "advertised superset of required — match",
			required:   []string{"ollama:gemma4-26b-ailang"},
			advertised: []string{"ollama:gemma4-26b-ailang", "gpu:m4-max", "local-models"},
			want:       true,
		},
		{
			name:       "required superset of advertised — no match",
			required:   []string{"ollama:gemma4-26b-ailang", "gpu:m4-max"},
			advertised: []string{"ollama:gemma4-26b-ailang"},
			want:       false,
		},
		{
			name:       "multiple required all satisfied",
			required:   []string{"ollama:gemma4-26b-ailang", "gpu:m4-max"},
			advertised: []string{"ollama:gemma4-26b-ailang", "gpu:m4-max", "local-models"},
			want:       true,
		},
		{
			name:       "multiple required, one missing — no match",
			required:   []string{"ollama:gemma4-26b-ailang", "gpu:nvidia-a100"},
			advertised: []string{"ollama:gemma4-26b-ailang", "gpu:m4-max"},
			want:       false,
		},
		{
			name:       "glob: advertised family matches required specific",
			required:   []string{"ollama:gemma4-26b-ailang"},
			advertised: []string{"ollama:*"},
			want:       true,
		},
		{
			name:       "glob: advertised family does NOT match different family required",
			required:   []string{"qwen:30b"},
			advertised: []string{"ollama:*"},
			want:       false,
		},
		{
			name:       "glob: required family matches advertised specific",
			required:   []string{"ollama:*"},
			advertised: []string{"ollama:gemma4-26b-ailang"},
			want:       true,
		},
		{
			name:       "mixed exact + glob in advertised",
			required:   []string{"ollama:gemma4-26b-ailang", "gpu:m4-max"},
			advertised: []string{"ollama:*", "gpu:m4-max", "local-models"},
			want:       true,
		},
		{
			name:       "system:heartbeat probe tag",
			required:   []string{"system:heartbeat"},
			advertised: []string{"system:heartbeat", "ollama:gemma4-26b-ailang"},
			want:       true,
		},
		{
			name:       "duplicates in required",
			required:   []string{"ollama:gemma4-26b-ailang", "ollama:gemma4-26b-ailang"},
			advertised: []string{"ollama:gemma4-26b-ailang"},
			want:       true,
		},
		{
			name:       "empty string in required is ignored",
			required:   []string{"", "ollama:gemma4-26b-ailang"},
			advertised: []string{"ollama:gemma4-26b-ailang"},
			want:       true,
		},
		{
			name:       "case-sensitive: different cases do not match",
			required:   []string{"Ollama:Gemma4-26b"},
			advertised: []string{"ollama:gemma4-26b"},
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TagMatches(tt.required, tt.advertised)
			if got != tt.want {
				t.Errorf("TagMatches(%v, %v) = %v, want %v",
					tt.required, tt.advertised, got, tt.want)
			}
		})
	}
}

func TestTagMatchesGlob(t *testing.T) {
	// Focused tests for glob-prefix semantics.
	tests := []struct {
		name    string
		pattern string
		tag     string
		want    bool
	}{
		{name: "exact match (no glob)", pattern: "ollama:gemma4-26b", tag: "ollama:gemma4-26b", want: true},
		{name: "exact mismatch (no glob)", pattern: "ollama:gemma4-26b", tag: "ollama:qwen", want: false},
		{name: "glob matches any suffix", pattern: "ollama:*", tag: "ollama:gemma4-26b-ailang", want: true},
		{name: "glob matches empty suffix", pattern: "ollama:*", tag: "ollama:", want: true},
		{name: "glob does not match other family", pattern: "ollama:*", tag: "qwen:30b", want: false},
		{name: "asterisk only matches everything", pattern: "*", tag: "anything", want: true},
		{name: "trailing star not at prefix is literal", pattern: "ollama:*-ailang", tag: "ollama:gemma4-26b-ailang", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tagPatternMatches(tt.pattern, tt.tag)
			if got != tt.want {
				t.Errorf("tagPatternMatches(%q, %q) = %v, want %v",
					tt.pattern, tt.tag, got, tt.want)
			}
		})
	}
}

func TestResolveHostID(t *testing.T) {
	// Explicit value is returned unchanged.
	if got := ResolveHostID("studio.eval-rig"); got != "studio.eval-rig" {
		t.Errorf("ResolveHostID(explicit) = %q, want studio.eval-rig", got)
	}

	// Empty value falls back to os.Hostname() — just ensure it's non-empty
	// and the fallback path runs without error.
	if got := ResolveHostID(""); got == "" {
		t.Errorf("ResolveHostID(empty) returned empty; want hostname fallback")
	}
}
