package pkg

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// stagePublishedPackage regenerates the smoke execution lock. Previous metadata
// is only an input to the existing confined-git carry-forward, never a copied
// path lock used by the child. Resolver I/O retains its authority chokepoints.
func stagePublishedPackage(sourceDir, stageDir string) error {
	if err := copyPackageContents(sourceDir, stageDir); err != nil {
		return fmt.Errorf("copy package: %w", err)
	}
	previous, err := LoadLockFile(stageDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read previous smoke lock: %w", err)
	}
	if _, err := RewritePathDeps(stageDir, sourceDir); err != nil {
		return fmt.Errorf("rewrite smoke manifest: %w", err)
	}
	if err := os.Remove(filepath.Join(stageDir, LockFileName)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove copied smoke lock: %w", err)
	}
	manifest, err := LoadManifest(stageDir)
	if err != nil {
		return err
	}
	lock, err := ResolveLockFrom(manifest, stageDir, "ailang publish smoke stage", previous)
	if err != nil {
		return fmt.Errorf("resolve published smoke dependencies: %w", err)
	}
	if err := lock.Save(stageDir); err != nil {
		return fmt.Errorf("save published smoke lock: %w", err)
	}
	return nil
}
