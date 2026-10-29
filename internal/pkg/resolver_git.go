package pkg

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/config"
)

// gitFromProvisionedOnly reports whether git dependencies may only be read
// from what the operator already provisioned: under confinement (#1720 D2)
// nothing may be cloned or fetched, and with AILANG_PACKAGE_ROOT set the
// root is exclusive for git exactly as for registry packages (D3).
func gitFromProvisionedOnly() bool {
	return Confined() || config.PackageRoot() != ""
}

// gitPackageDir is the ONE answer to "where does this locked git package
// live", shared by the loader, the content-hash check and the resolver's
// carry-forward, so the three cannot diverge. It creates nothing.
//
//   - AILANG_PACKAGE_ROOT set: <root>/<vendor>/<name>/<version>, the registry
//     layout, and nothing else (D3).
//   - otherwise: the URL-keyed clone under ~/.ailang/cache/git, plus subdir.
func gitPackageDir(locked *LockedPackage) (string, error) {
	if config.PackageRoot() != "" {
		return PackageDir(locked.Name, locked.Version)
	}
	if locked.GitURL == "" {
		return "", fmt.Errorf("git package %s has no git_url in %s; run 'ailang lock' to resolve", locked.Name, LockFileName)
	}
	cache, err := NewGitCache()
	if err != nil {
		return "", err
	}
	dir := cache.CacheDir(locked.GitURL)
	if locked.GitSubdir != "" {
		dir = filepath.Join(dir, locked.GitSubdir)
	}
	return dir, nil
}

// lockedGitDir is the directory a locked git package loads from, read-only:
// the D3 package root exclusively when AILANG_PACKAGE_ROOT is set, else the
// URL-keyed clone, else (old lock files) the recorded Path. The loader and
// ValidateContentHashesFrom both call it.
func lockedGitDir(locked *LockedPackage) (string, error) {
	if config.PackageRoot() != "" {
		dir, err := gitPackageDir(locked)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(dir); err != nil {
			return "", fmt.Errorf("git package %s@%s is not provisioned in AILANG_PACKAGE_ROOT at %s; ask the operator to provision it", locked.Name, locked.Version, dir)
		}
		return dir, nil
	}
	var dir string
	if locked.GitURL != "" {
		d, err := gitPackageDir(locked)
		if err != nil {
			return "", fmt.Errorf("failed to locate git cache for %s: %w", locked.Name, err)
		}
		dir = d
	}
	// Fallback to stored Path for old lock files or if computed dir doesn't exist
	if dir == "" || func() bool { _, e := os.Stat(dir); return e != nil }() {
		if locked.Path != "" {
			dir = locked.Path
		}
	}
	if dir == "" {
		return "", fmt.Errorf("git package %s has no cached path; run 'ailang lock' to resolve", locked.Name)
	}
	if _, err := os.Stat(dir); err != nil {
		return "", fmt.Errorf("git package %s cache not found at %s; run 'ailang lock' to re-fetch", locked.Name, dir)
	}
	return dir, nil
}

// gitNotProvisioned is the refusal when a git dependency cannot be carried
// forward. Confined, it is the canonical AuthNetwork refusal (resolving it
// would need a clone); unconfined with a package root, it names the root.
func gitNotProvisioned(depName, reason string) error {
	what := fmt.Sprintf("resolving git dependency %s (%s)", depName, reason)
	if err := Require(AuthNetwork, what, GitRemedy); err != nil {
		return err
	}
	return fmt.Errorf("git dependency %s is not provisioned in AILANG_PACKAGE_ROOT (%s); ask the operator to provision it", depName, reason)
}

// provisionedGitDep is D2's carry-forward: a git dependency is resolved with
// NO git invocation when the previous ailang.lock already pins it (same
// git_url, tag or rev, and subdir) and the provisioned directory still hashes
// to the locked content and interface. Anything else is a refusal — a
// changed pin needs a fetch, which is operator authority.
// It returns the directory and the locked commit.
func provisionedGitDep(depName string, dep Dependency, prev *LockFile) (string, string, error) {
	if prev == nil {
		return "", "", gitNotProvisioned(depName, "no previous "+LockFileName+" pins it")
	}
	p, ok := prev.FindPackage(depName)
	if !ok || p.Source != "git" {
		return "", "", gitNotProvisioned(depName, "not pinned in "+LockFileName)
	}
	switch {
	case p.GitURL != dep.Git:
		return "", "", gitNotProvisioned(depName, fmt.Sprintf("git url changed from %q", p.GitURL))
	case p.GitSubdir != dep.Subdir:
		return "", "", gitNotProvisioned(depName, fmt.Sprintf("subdir changed from %q", p.GitSubdir))
	case dep.Rev != "" && p.GitRev != dep.Rev:
		return "", "", gitNotProvisioned(depName, fmt.Sprintf("rev changed from %q", p.GitRev))
	case dep.Rev == "" && p.GitTag != dep.Tag:
		return "", "", gitNotProvisioned(depName, fmt.Sprintf("tag %q is not the one %s pins (%q)", dep.Tag, LockFileName, p.GitTag))
	}
	dir, err := gitPackageDir(p)
	if err != nil {
		return "", "", err
	}
	if _, err := os.Stat(dir); err != nil {
		return "", "", gitNotProvisioned(depName, "not present at "+dir)
	}
	hash, err := ContentHash(dir)
	if err != nil {
		return "", "", fmt.Errorf("failed to hash git dep %s at %s: %w", depName, dir, err)
	}
	if hash != p.ContentHash {
		return "", "", gitNotProvisioned(depName, fmt.Sprintf("provisioned content %s does not match locked %s", hash, p.ContentHash))
	}
	m, err := LoadManifest(dir)
	if err != nil {
		return "", "", fmt.Errorf("failed to load git dep %s at %s: %w", depName, dir, err)
	}
	if m.Package.Version != p.Version || (p.InterfaceHash != "" && InterfaceHash(m) != p.InterfaceHash) {
		return "", "", gitNotProvisioned(depName, "provisioned manifest does not match "+LockFileName)
	}
	return dir, p.GitRev, nil
}

// resolveGitSource returns the directory and commit for a git dependency:
// carried forward from the provisioned copy when gitFromProvisionedOnly,
// cloned/fetched into the git cache otherwise.
func resolveGitSource(depName string, dep Dependency, prev *LockFile) (string, string, error) {
	if gitFromProvisionedOnly() {
		return provisionedGitDep(depName, dep, prev)
	}
	cache, err := NewGitCache()
	if err != nil {
		return "", "", fmt.Errorf("failed to init git cache: %w", err)
	}
	localPath, rev, err := cache.Resolve(dep.Git, dep.Tag, dep.Rev, dep.Subdir)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve git dep %s: %w", depName, err)
	}
	return localPath, rev, nil
}

// gitTag is the tag a lock records: none when an exact rev overrides it.
func gitTag(dep Dependency) string {
	if dep.Rev != "" {
		return ""
	}
	return dep.Tag
}
