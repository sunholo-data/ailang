package effects

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// Invalid cache names must not displace a valid release or prerelease.
func TestAssetVersionOrderingPreservesReleases(t *testing.T) {
	for _, pair := range [][2]string{
		{"dev", "0.1.0"},
		{"unknown", "0.1.0"},
		{"v1.2", "0.1.0"},
		{"1.0.0-", "0.1.0"},
		{"1.0.0-rc.", "0.1.0"},
		{"1.0.0+", "0.1.0"},
		{"0.9.0", "0.10.0"},
		{"1.0.0-rc1", "1.0.0"},
		{"1.0.0-rc.2", "1.0.0-rc.10"},
		{"1.0.0-rc.10", "1.0.0"},
	} {
		if !semverLess(pair[0], pair[1]) || semverLess(pair[1], pair[0]) {
			t.Errorf("want %s < %s", pair[0], pair[1])
		}
	}
}

// TestSemverPrereleasePrecedence verifies SemVer 2.0.0 §11.4 ordering:
//   - dot-separated numeric identifiers compare as integers (rc.2 < rc.10)
//   - numeric identifiers sort before non-numeric identifiers (1 < alpha, alpha.1 < alpha.beta)
//   - non-numeric identifiers compare lexically in ASCII (alpha < beta < rc)
//   - shorter prefix sorts before longer when equal (alpha < alpha.1, rc.1 < rc.1.1)
//   - pre-release sorts before normal release (1.0.0-rc.10 < 1.0.0)
func TestSemverPrereleasePrecedence(t *testing.T) {
	chains := [][]string{
		// Official SemVer 2.0.0 §11.4 chain + multi-digit RCs
		{
			"1.0.0-0",
			"1.0.0-1",
			"1.0.0-2",
			"1.0.0-10",
			"1.0.0-alpha",
			"1.0.0-alpha.1",
			"1.0.0-alpha.2",
			"1.0.0-alpha.10",
			"1.0.0-alpha.beta",
			"1.0.0-beta",
			"1.0.0-beta.2",
			"1.0.0-beta.11",
			"1.0.0-rc.1",
			"1.0.0-rc.1.1",
			"1.0.0-rc.2",
			"1.0.0-rc.10",
			"1.0.0",
			"1.0.1-rc.1",
			"1.0.1",
		},
		// Optional leading 'v' prefix and huge numeric pre-release identifiers (no int overflow)
		{
			"v0.64.0",
			"v1.0.0-rc.1",
			"v1.0.0-rc.2",
			"v1.0.0-rc.10",
			"v1.0.0-rc.999999999999999999999",
			"v1.0.0-rc.1000000000000000000000",
			"v1.0.0-rc.alpha",
			"v1.0.0",
		},
	}

	for ci, chain := range chains {
		for i := 0; i < len(chain); i++ {
			if semverLess(chain[i], chain[i]) {
				t.Errorf("chain[%d]: irreflexivity violated for %q", ci, chain[i])
			}
			for j := i + 1; j < len(chain); j++ {
				if !semverLess(chain[i], chain[j]) {
					t.Errorf("chain[%d]: want %q < %q, got false", ci, chain[i], chain[j])
				}
				if semverLess(chain[j], chain[i]) {
					t.Errorf("chain[%d]: asymmetry violated: %q < %q was true", ci, chain[j], chain[i])
				}
			}
		}
	}
}

// TestSemverBuildMetadataIgnored verifies SemVer 2.0.0 §10: build metadata (+...)
// does not affect version precedence on either releases or pre-releases.
func TestSemverBuildMetadataIgnored(t *testing.T) {
	equalPairs := [][2]string{
		{"1.0.0+20130313", "1.0.0+20261126"},
		{"1.0.0", "1.0.0+build.1"},
		{"1.0.0-rc.2+build.99", "1.0.0-rc.2+build.1"},
		{"1.0.0-rc.2", "1.0.0-rc.2+sha.deadbeef"},
	}
	for _, p := range equalPairs {
		if semverLess(p[0], p[1]) || semverLess(p[1], p[0]) {
			t.Errorf("expected equal precedence for %q and %q", p[0], p[1])
		}
	}
	if !semverLess("1.0.0-rc.2+zzz", "1.0.0-rc.10+aaa") {
		t.Error("expected 1.0.0-rc.2+zzz < 1.0.0-rc.10+aaa")
	}
}

// TestPkgAssetPath_PicksHighestRCBeforeFinal verifies end-to-end cache selection
// picks 1.0.0-rc.10 over 1.0.0-rc.2, and then 1.0.0 once installed.
func TestPkgAssetPath_PicksHighestRCBeforeFinal(t *testing.T) {
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	base := filepath.Join(home, ".ailang", "cache", "registry", "test", "rcpkg")

	for _, v := range []string{"0.64.0", "1.0.0-rc.1", "1.0.0-rc.2", "1.0.0-rc.10"} {
		dir := filepath.Join(base, v, AssetsDir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "ver.txt"), []byte(v), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	got, err := pkgAssetPath(nil, []eval.Value{
		&eval.StringValue{Value: "test/rcpkg"},
		&eval.StringValue{Value: "ver.txt"},
	})
	if err != nil {
		t.Fatalf("pkgAssetPath: %v", err)
	}
	gotPath := got.(*eval.TaggedValue).Fields[0].(*eval.StringValue).Value
	if want := filepath.Join(base, "1.0.0-rc.10", AssetsDir, "ver.txt"); gotPath != want {
		t.Errorf("during RC soak: expected %q, got %q", want, gotPath)
	}

	// Installing final 1.0.0 supersedes 1.0.0-rc.10.
	finalDir := filepath.Join(base, "1.0.0", AssetsDir)
	if err := os.MkdirAll(finalDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(finalDir, "ver.txt"), []byte("1.0.0"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err = pkgAssetPath(nil, []eval.Value{
		&eval.StringValue{Value: "test/rcpkg"},
		&eval.StringValue{Value: "ver.txt"},
	})
	if err != nil {
		t.Fatalf("pkgAssetPath: %v", err)
	}
	gotPath = got.(*eval.TaggedValue).Fields[0].(*eval.StringValue).Value
	if want := filepath.Join(base, "1.0.0", AssetsDir, "ver.txt"); gotPath != want {
		t.Errorf("after final 1.0.0: expected %q, got %q", want, gotPath)
	}
}
