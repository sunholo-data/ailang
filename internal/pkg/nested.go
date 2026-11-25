package pkg

import (
	"os"
	"path/filepath"
)

// IsNestedPackageRoot reports whether dir, met while walking the package at
// root, is the root of ANOTHER package: a directory below root holding its own
// ailang.toml. Such a directory belongs to that package, not to root, so every
// walk that defines root's contents (source discovery, content hash, tarball,
// publish staging) skips it.
//
// M-PKG-PARALLEL-AGENTS C6 (V3): before this, a nested package was compiled into
// the outer one (MOD010 on its module paths) and shipped in its tarball.
func IsNestedPackageRoot(dir, root string) bool {
	if filepath.Clean(dir) == filepath.Clean(root) {
		return false
	}
	st, err := os.Stat(filepath.Join(dir, ManifestFile))
	return err == nil && !st.IsDir()
}
