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

func contentHashDrift(name, locked, current string) error {
	return fmt.Errorf("dependency %s content changed (locked: %s, current: %s)\nRun 'ailang lock' to update", name, locked, current)
}
