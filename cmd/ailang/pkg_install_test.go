package main

import "testing"

// TestLooksLikeToolchainVersion guards the disambiguation that turns
// `ailang install v0.23.0` (a common mistake — install is for packages, not
// the toolchain) into an actionable error instead of "invalid package name".
func TestLooksLikeToolchainVersion(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		// version-like → true
		{"v0.22.0", true},
		{"0.23.0", true},
		{"v1", true},
		{"1.2.3", true},
		{"v0.0.1", true},
		// package specs / other → false
		{"sunholo/auth", false},
		{"sunholo/auth@0.1.0", false}, // '@' splitting happens before this; name still has '/'
		{"latest", false},
		{"", false},
		{"v", false}, // bare 'v' with no digits
		{"version2", false},
		{"v1.x", false}, // non-numeric component
		{".5", false},   // must start with a digit (after optional 'v')
	}
	for _, c := range cases {
		if got := looksLikeToolchainVersion(c.in); got != c.want {
			t.Errorf("looksLikeToolchainVersion(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
