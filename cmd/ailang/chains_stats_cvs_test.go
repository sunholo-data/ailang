package main

import "testing"

// TestCohortSourceRefPrefix locks the delimiter behavior so a baseline id like
// "v1.0" cannot accidentally also match "v1.05" cohorts.
//
// NOTE (M4a, BF-2): the already-slash-suffixed inputs below are no longer
// REACHABLE from either CLI surface — validateBaselineID rejects '/' in a
// baseline id on the freeze side AND the query side. They stay covered here
// because this function is the idempotent normalizer both sides call, and its
// contract (never double-append a delimiter) is what makes the writer/reader
// round-trip in eval_suite_cohort_test.go sound.
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
