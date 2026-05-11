package eval_analysis

import "testing"

func TestShouldExcludeFromCapability(t *testing.T) {
	tests := []struct {
		category string
		want     bool
	}{
		// Excluded — provider noise, not capability signal.
		{"quota_exhausted", true},
		{"rate_limit", true},

		// Included — capability signal.
		{"timeout", false},
		{"cost_killed", false},
		{"step_exhausted", false},
		{"compile_error", false},
		{"runtime_error", false},
		{"logic_error", false},
		{"verify_error", false},
		{"none", false},

		// Catch-all — kept excluded during the legacy-JSON transition period
		// (see capability.go docs for rationale).
		{"api_error", true},

		// Unknown future categories default to included (defensive).
		{"future_category_xyz", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			got := ShouldExcludeFromCapability(tt.category)
			if got != tt.want {
				t.Errorf("ShouldExcludeFromCapability(%q) = %v, want %v",
					tt.category, got, tt.want)
			}
		})
	}
}
