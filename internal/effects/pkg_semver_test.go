package effects

import "testing"

// Mirrored compatibility cases from pkg.ParseSemver, including permissive
// signed components and prerelease stripping rather than strict SemVer rules.
func TestAssetVersionCompatibility(t *testing.T) {
	for _, tc := range []struct {
		version string
		want    [3]int
		valid   bool
	}{
		{"0.10.0", [3]int{0, 10, 0}, true}, {"v1.2", [3]int{1, 2, 0}, true},
		{"1.2.3-rc1", [3]int{1, 2, 3}, true}, {"dev", [3]int{999, 999, 999}, true},
		{"unknown", [3]int{999, 999, 999}, true}, {"", [3]int{999, 999, 999}, true},
		{"-1.2.3", [3]int{-1, 2, 3}, true}, {"1.2.3+build", [3]int{}, false},
		{"junk", [3]int{}, false}, {"1.x", [3]int{}, false}, {"1.2.x", [3]int{}, false},
	} {
		got, err := assetVersion(tc.version)
		if (err == nil) != tc.valid || tc.valid && got != tc.want {
			t.Errorf("%q: %v %v", tc.version, got, err)
		}
	}
	if compareAssetVersions("0.10.0", "0.9.0") <= 0 {
		t.Fatal("numeric minor ordering")
	}
	if compareAssetVersions("1.0.0", "v1.0.0") >= 0 {
		t.Fatal("deterministic tie-break")
	}
}
