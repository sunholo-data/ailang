package main

import "testing"

// TestCohortSourceRefPrefix locks the delimiter behavior so a baseline id like
// "v1.0" cannot accidentally also match "v1.05" cohorts.
func TestCohortSourceRefPrefix(t *testing.T) {
	cases := []struct{ in, want string }{
		{"v1.0", "v1.0/"},
		{"v1.0/", "v1.0/"},
		{"v1.0/agent/baseline", "v1.0/agent/baseline/"},
		{"", ""},
	}
	for _, c := range cases {
		if got := cohortSourceRefPrefix(c.in); got != c.want {
			t.Errorf("cohortSourceRefPrefix(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
