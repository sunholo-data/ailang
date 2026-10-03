package pkg

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
)

// LockedFromResolved converts resolver output into lock file entries.
func LockedFromResolved(resolved []ResolvedPackage) []LockedPackage {
	locked := make([]LockedPackage, len(resolved))
	for i, r := range resolved {
		locked[i] = LockedPackage(r)
	}
	return locked
}

// ResolveLock resolves manifest's dependencies from dir and returns the lock
// file `ailang lock` would write there. Path dependencies are recorded
// relative to dir with forward slashes (ailang#1498), so the result does not
// depend on where the checkout lives.
func ResolveLock(manifest *PackageManifest, dir, generator string) (*LockFile, error) {
	var packages []LockedPackage
	if len(manifest.Dependencies) > 0 {
		resolved, err := ResolveDependencies(manifest, dir)
		if err != nil {
			return nil, err
		}
		packages = LockedFromResolved(resolved)
	}
	return NewLockFile(packages, generator), nil
}

// CheckLock re-resolves dir's ailang.toml and compares the result with the
// committed ailang.lock, ignoring the generation metadata (timestamp,
// generator, ailang_version). It returns one line per drifted entry; an empty
// slice means the lock is current. Because path dependencies are relative,
// the answer is the same in every checkout of the same tree.
//
// A missing or unreadable lock is an error, not drift.
func CheckLock(dir string) ([]string, error) {
	current, err := LoadLockFile(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("no %s in %s; run 'ailang lock' to create it", LockFileName, dir)
		}
		return nil, err
	}
	manifest, err := LoadManifest(dir)
	if err != nil {
		return nil, err
	}
	expected, err := ResolveLock(manifest, dir, "")
	if err != nil {
		return nil, fmt.Errorf("dependency resolution failed: %w", err)
	}
	return LockDrift(current, expected), nil
}

// LockDrift lists the differences between a committed lock and a freshly
// resolved one, package by package.
func LockDrift(current, expected *LockFile) []string {
	var drift []string
	for _, exp := range expected.Packages {
		cur, ok := current.FindPackage(exp.Name)
		if !ok {
			drift = append(drift, fmt.Sprintf("%s: missing from %s", exp.Name, LockFileName))
			continue
		}
		drift = append(drift, packageDrift(*cur, exp)...)
	}
	for _, cur := range current.Packages {
		if _, ok := expected.FindPackage(cur.Name); !ok {
			drift = append(drift, fmt.Sprintf("%s: in %s but no longer a dependency", cur.Name, LockFileName))
		}
	}
	return drift
}

func packageDrift(cur, exp LockedPackage) []string {
	var out []string
	if cur.Source == "path" && IsAbsoluteCrossPlatform(cur.Path) {
		out = append(out, fmt.Sprintf("%s: path is absolute (%s), written by an older ailang; should be %q relative to %s",
			cur.Name, cur.Path, exp.Path, ManifestFile))
	} else if cur.Path != exp.Path {
		out = append(out, fmt.Sprintf("%s: path %q, expected %q", cur.Name, cur.Path, exp.Path))
	}
	fields := []struct{ name, cur, exp string }{
		{"source", cur.Source, exp.Source},
		{"version", cur.Version, exp.Version},
		{"content_hash", cur.ContentHash, exp.ContentHash},
		{"interface_hash", cur.InterfaceHash, exp.InterfaceHash},
		{"ailang", cur.AILANG, exp.AILANG},
		{"git_url", cur.GitURL, exp.GitURL},
		{"git_rev", cur.GitRev, exp.GitRev},
		{"git_subdir", cur.GitSubdir, exp.GitSubdir},
	}
	for _, f := range fields {
		if f.cur != f.exp {
			out = append(out, fmt.Sprintf("%s: %s %q, expected %q", cur.Name, f.name, f.cur, f.exp))
		}
	}
	if !sameStrings(cur.Effects, exp.Effects) {
		out = append(out, fmt.Sprintf("%s: effects [%s], expected [%s]", cur.Name, strings.Join(cur.Effects, ", "), strings.Join(exp.Effects, ", ")))
	}
	if !sameStrings(cur.Exports, exp.Exports) {
		out = append(out, fmt.Sprintf("%s: exports [%s], expected [%s]", cur.Name, strings.Join(cur.Exports, ", "), strings.Join(exp.Exports, ", ")))
	}
	return out
}

// sameStrings treats nil and empty as equal (JSON round-trips both as []/null).
func sameStrings(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}

// staleAbsolutePathHint explains a path dependency the lock names by an
// absolute directory that is not there — the signature of a lock written by
// ailang <= v0.32 on another machine or checkout.
func staleAbsolutePathHint(p *LockedPackage) string {
	if p.Source != "path" || !IsAbsoluteCrossPlatform(p.Path) {
		return ""
	}
	return fmt.Sprintf("\n%s records %s by the absolute path %s (written by an older ailang in another checkout); run 'ailang lock' to rewrite it relative to %s",
		LockFileName, p.Name, p.Path, ManifestFile)
}
