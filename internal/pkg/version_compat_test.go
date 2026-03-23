package pkg

import "testing"

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input               string
		major, minor, patch int
		err                 bool
	}{
		{"0.9.5", 0, 9, 5, false},
		{"1.0.0", 1, 0, 0, false},
		{"v0.10.3", 0, 10, 3, false},
		{"0.9", 0, 9, 0, false},
		{"10.20.30", 10, 20, 30, false},
		{"0.9.5-rc1", 0, 9, 5, false}, // pre-release stripped
		{"dev", 999, 999, 999, false},
		{"unknown", 999, 999, 999, false},
		{"", 999, 999, 999, false},
		{"abc", 0, 0, 0, true},
		{"1", 0, 0, 0, true},
		{"a.b.c", 0, 0, 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			sv, err := ParseSemver(tc.input)
			if tc.err {
				if err == nil {
					t.Fatalf("expected error for %q", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.input, err)
			}
			if sv.Major != tc.major || sv.Minor != tc.minor || sv.Patch != tc.patch {
				t.Errorf("got %d.%d.%d, want %d.%d.%d", sv.Major, sv.Minor, sv.Patch, tc.major, tc.minor, tc.patch)
			}
		})
	}
}

func TestSemverGte(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"1.0.0", "0.9.5", true},
		{"0.9.5", "0.9.5", true}, // equal
		{"0.9.5", "0.9.6", false},
		{"0.10.0", "0.9.5", true}, // 0.10 > 0.9
		{"0.9.0", "0.10.0", false},
		{"1.0.0", "0.99.99", true},
		{"0.9.5", "1.0.0", false},
	}
	for _, tc := range tests {
		t.Run(tc.a+">="+tc.b, func(t *testing.T) {
			a, _ := ParseSemver(tc.a)
			b, _ := ParseSemver(tc.b)
			if got := a.gte(b); got != tc.want {
				t.Errorf("(%s).gte(%s) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestParseVersionConstraint(t *testing.T) {
	tests := []struct {
		input string
		want  string
		err   bool
	}{
		{">=0.9.5", "0.9.5", false},
		{">= 0.10.0", "0.10.0", false},
		{">=1.0.0", "1.0.0", false},
		{"0.9.5", "0.9.5", false}, // bare version = >=
		{"", "", true},
		{"~0.9.5", "", true},     // unsupported
		{"^1.0", "", true},       // unsupported
		{">=0.9,<1.0", "", true}, // ranges unsupported
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := ParseVersionConstraint(tc.input)
			if tc.err {
				if err == nil {
					t.Fatalf("expected error for %q", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSatisfiesAILANGVersion(t *testing.T) {
	tests := []struct {
		requirement    string
		currentVersion string
		want           bool
	}{
		// No requirement = always satisfied
		{"", "0.9.5", true},
		{"", "0.1.0", true},

		// dev/unknown always satisfy
		{">=1.0.0", "dev", true},
		{">=99.0.0", "unknown", true},

		// Normal checks
		{">=0.9.5", "0.9.5", true},
		{">=0.9.5", "0.9.6", true},
		{">=0.9.5", "0.10.0", true},
		{">=0.9.5", "1.0.0", true},
		{">=0.9.5", "0.9.4", false},
		{">=0.9.5", "0.8.0", false},
		{">=0.10.0", "0.9.5", false}, // 0.10 > 0.9 numerically
		{">=1.0.0", "0.99.99", false},

		// Bare version (treated as >=)
		{"0.9.5", "0.9.5", true},
		{"0.9.5", "0.9.4", false},

		// With v prefix
		{">=0.9.5", "v0.9.5", true},
		{">=0.9.5", "v0.9.4", false},
	}
	for _, tc := range tests {
		name := tc.requirement + "_vs_" + tc.currentVersion
		t.Run(name, func(t *testing.T) {
			got, err := SatisfiesAILANGVersion(tc.requirement, tc.currentVersion)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("SatisfiesAILANGVersion(%q, %q) = %v, want %v",
					tc.requirement, tc.currentVersion, got, tc.want)
			}
		})
	}
}

func TestFormatVersionConstraint(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"0.9.5", ">=0.9.5"},
		{"v0.9.5", ">=0.9.5"},
		{"1.0.0", ">=1.0.0"},
		{"dev", ""},
		{"unknown", ""},
		{"", ""},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			if got := FormatVersionConstraint(tc.input); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
