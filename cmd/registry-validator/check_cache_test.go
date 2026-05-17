package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestCachedPackages_PassNamingGate is M-EXT-AUTHOR-DX M3's pre-launch
// verification: run the production validateToolNames against every cached
// version of every published sunholo/motoko_ext_* package and confirm zero
// false-positive rejections. Cache is populated by `ailang lock` during
// normal development.
//
// Run with: go test -v -run TestCachedPackages_PassNamingGate ./cmd/registry-validator/
//
// Skip if cache unavailable (so CI without ~/.ailang doesn't fail).
func TestCachedPackages_PassNamingGate(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	cacheRoot := filepath.Join(home, ".ailang", "cache", "registry", "sunholo")
	if _, err := os.Stat(cacheRoot); err != nil {
		t.Skip("no ~/.ailang/cache/registry/sunholo — run `ailang lock` in a project that pins sunholo packages first")
	}

	entries, err := os.ReadDir(cacheRoot)
	if err != nil {
		t.Skipf("can't read %s: %v", cacheRoot, err)
	}

	type flagged struct{ pkg, version, name, reason string }
	var failures []flagged
	scanned := 0

	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "motoko_ext_") {
			continue
		}
		pkgDir := filepath.Join(cacheRoot, e.Name())
		versions, err := os.ReadDir(pkgDir)
		if err != nil {
			continue
		}
		versionNames := make([]string, 0, len(versions))
		for _, v := range versions {
			if v.IsDir() {
				versionNames = append(versionNames, v.Name())
			}
		}
		sort.Strings(versionNames)
		for _, v := range versionNames {
			scanned++
			_, badName, reason := validateToolNames(filepath.Join(pkgDir, v))
			if badName != "" {
				failures = append(failures, flagged{
					pkg: e.Name(), version: v, name: badName, reason: reason,
				})
			}
		}
	}

	t.Logf("Scanned %d cached package versions across %d packages", scanned, len(entries))
	if len(failures) == 0 {
		t.Logf("✓ All cached published packages pass the M3 naming gate (zero false positives)")
		return
	}
	t.Errorf("FAILURES: %d packages would be rejected on re-publish:", len(failures))
	for _, f := range failures {
		t.Errorf("  - sunholo/%s@%s: %q (%s)", f.pkg, f.version, f.name, f.reason)
	}
}
