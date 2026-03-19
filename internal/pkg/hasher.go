package pkg

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ContentHash computes a SHA256 hash over all .ail source files in dir,
// sorted by relative path. Same files in same order = same hash.
func ContentHash(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	// Collect all .ail files relative to dir
	var files []string
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".ail") {
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			files = append(files, rel)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to walk directory: %w", err)
	}

	// Sort for determinism
	sort.Strings(files)

	h := sha256.New()
	for _, rel := range files {
		// Write the relative path as a separator (so renaming changes the hash)
		fmt.Fprintf(h, "file:%s\n", rel)

		f, err := os.Open(filepath.Join(dir, rel))
		if err != nil {
			return "", fmt.Errorf("failed to open %s: %w", rel, err)
		}
		if _, err := io.Copy(h, f); err != nil {
			f.Close()
			return "", fmt.Errorf("failed to hash %s: %w", rel, err)
		}
		f.Close()
	}

	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// InterfaceHash computes a SHA256 hash over a package's public interface:
// package name, edition, sorted exported module list, and sorted max effects.
// This hash stays the same when only internal code changes — it only changes
// when the package's public surface (exports, effects) changes.
//
// Explicitly excluded: source formatting, comments, internal modules,
// declaration order, source file contents.
func InterfaceHash(m *PackageManifest) string {
	h := sha256.New()

	// Package identity
	fmt.Fprintf(h, "name:%s\n", m.Package.Name)
	fmt.Fprintf(h, "edition:%s\n", m.Package.Edition)

	// Sorted exported modules
	exports := make([]string, len(m.Exports.Modules))
	copy(exports, m.Exports.Modules)
	sort.Strings(exports)
	for _, mod := range exports {
		fmt.Fprintf(h, "export:%s\n", mod)
	}

	// Sorted max effects
	effects := make([]string, len(m.Effects.Max))
	copy(effects, m.Effects.Max)
	sort.Strings(effects)
	for _, eff := range effects {
		fmt.Fprintf(h, "effect:%s\n", eff)
	}

	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}
