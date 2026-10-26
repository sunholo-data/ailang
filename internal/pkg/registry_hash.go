package pkg

import (
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/config"
)

// RegistryContentError marks registry trust failures that must stop compilation.
// Path dependency drift keeps its existing authoring warning behavior.
type RegistryContentError struct{ Err error }

func (e *RegistryContentError) Error() string { return e.Err.Error() }
func (e *RegistryContentError) Unwrap() error { return e.Err }

// validateRegistryContentHashes runs before path validation so a path warning
// cannot mask a registry trust failure in the same lock file.
func (lf *LockFile) validateRegistryContentHashes() error {
	for _, p := range lf.Packages {
		if p.Source != "registry" {
			continue
		}
		dir, err := PackageDir(p.Name, p.Version)
		if err != nil {
			return &RegistryContentError{fmt.Errorf("cannot resolve registry dependency %s: %w", p.Name, err)}
		}
		if config.PackageRoot() != "" {
			if _, statErr := os.Stat(dir); os.IsNotExist(statErr) {
				return &RegistryContentError{fmt.Errorf("registry package %s@%s is not provisioned in AILANG_PACKAGE_ROOT at %s; ask the operator to provision it", p.Name, p.Version, dir)}
			}
		}
		hash, err := ContentHash(dir)
		if err != nil {
			return &RegistryContentError{fmt.Errorf("failed to hash registry dependency %s at %s: %w", p.Name, dir, err)}
		}
		if hash != p.ContentHash {
			return &RegistryContentError{contentHashDrift(p.Name, p.ContentHash, hash)}
		}
	}
	return nil
}

// validateGitContentHashes is #1720 T1/T2 (D4): a git dependency's content is
// verified against the lock exactly like a registry one, and a mismatch is
// fatal. The URL-keyed clone is shared by every project that names the URL,
// so a different revision checked out by another project, or an edit in the
// cache, used to compile silently; it is now drift.
func (lf *LockFile) validateGitContentHashes() error {
	for i := range lf.Packages {
		p := &lf.Packages[i]
		if p.Source != "git" {
			continue
		}
		dir, err := lockedGitDir(p)
		if err != nil {
			return &RegistryContentError{err}
		}
		hash, err := ContentHash(dir)
		if err != nil {
			return &RegistryContentError{fmt.Errorf("failed to hash git dependency %s at %s: %w", p.Name, dir, err)}
		}
		if hash != p.ContentHash {
			return &RegistryContentError{contentHashDrift(p.Name, p.ContentHash, hash)}
		}
	}
	return nil
}

func contentHashDrift(name, locked, current string) error {
	return fmt.Errorf("dependency %s content changed (locked: %s, current: %s)\nRun 'ailang lock' to update", name, locked, current)
}
