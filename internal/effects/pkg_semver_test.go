package effects

import "testing"

// Invalid cache names must not displace a valid release or prerelease.
func TestAssetVersionOrderingPreservesReleases(t *testing.T) {
	for _, pair := range [][2]string{
		{"dev", "0.1.0"}, {"unknown", "0.1.0"}, {"v1.2", "0.1.0"},
		{"0.9.0", "0.10.0"}, {"1.0.0-rc1", "1.0.0"},
	} {
		if !semverLess(pair[0], pair[1]) || semverLess(pair[1], pair[0]) {
			t.Errorf("want %s < %s", pair[0], pair[1])
		}
	}
}
