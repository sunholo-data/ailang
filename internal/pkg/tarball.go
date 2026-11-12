package pkg

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// AssetsDir is the package subdirectory whose contents are bundled verbatim.
const AssetsDir = "assets"

// CreateTarball creates a deterministic tar.gz of a package directory.
// Includes: ailang.toml, *.ail files, AGENT.md, _smoke.ail, anything under assets/.
// Excludes: .git, tests/, ailang.lock.
// Files are sorted and timestamps zeroed for determinism.
func CreateTarball(packageDir string) ([]byte, error) {
	packageDir, err := filepath.Abs(packageDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Collect files to include
	var files []string
	err = filepath.Walk(packageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(packageDir, path)
		if err != nil {
			return err
		}

		// Skip directories and excluded paths
		if info.IsDir() {
			base := filepath.Base(path)
			if base == ".git" || base == "tests" || base == "test" {
				return filepath.SkipDir
			}
			return nil
		}

		// Defense-in-depth: never include a path with ".." segments, even via symlink games.
		clean := filepath.Clean(rel)
		if strings.HasPrefix(clean, "..") || strings.Contains(clean, "../") {
			return fmt.Errorf("invalid relative path in package: %q", rel)
		}

		// Include: ailang.toml, *.ail, AGENT.md, anything under assets/.
		// Use forward slashes so the tarball entry name is stable cross-platform.
		relForward := filepath.ToSlash(rel)
		switch {
		case IsNamedTestBodyFile(filepath.Base(rel)):
			// A leftover `ailang test` body copy (#1502) is never package
			// content; quality refuses it (PUB024) and this keeps the tarball
			// identical to the clean tree's if that gate is ever bypassed.
		case rel == ManifestFile,
			strings.HasSuffix(rel, ".ail"),
			rel == "AGENT.md",
			rel == ChangelogFile, // M-PKG-QUALITY-LADDER M4
			strings.HasPrefix(relForward, AssetsDir+"/"):
			files = append(files, relForward)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk package dir: %w", err)
	}

	// Sort for determinism
	sort.Strings(files)

	// Create tar.gz
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for _, rel := range files {
		fullPath := filepath.Join(packageDir, rel)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", rel, err)
		}

		hdr := &tar.Header{
			Name:    rel,
			Size:    int64(len(data)),
			Mode:    0644,
			ModTime: time.Time{}, // deterministic: zero time
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return nil, fmt.Errorf("failed to write tar header for %s: %w", rel, err)
		}
		if _, err := tw.Write(data); err != nil {
			return nil, fmt.Errorf("failed to write tar data for %s: %w", rel, err)
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// isTarPathTraversal reports whether a tar entry name is unsafe to extract:
// traversal (contains ".."), absolute, or otherwise non-local. Tar entry names
// use forward slashes by convention, so we check those explicitly rather than
// rely solely on filepath.IsAbs (which needs a drive letter on Windows).
func isTarPathTraversal(name string) bool {
	if name == "" {
		return false
	}
	if strings.Contains(name, "..") {
		return true
	}
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, `\`) || filepath.IsAbs(name) {
		return true
	}
	// filepath.IsLocal (Go 1.20+) also rejects volume-rooted and reserved names.
	return !filepath.IsLocal(filepath.FromSlash(name))
}

// ExtractTarball extracts a tar.gz to a destination directory.
//
// It refuses, with a *TarballLimitError, any tarball over the limits in
// tarball_limits.go: compressed size, entry count, per-entry size, total
// extracted size and decompressed stream size. A refusal can leave a partial
// extraction in destDir; callers extract into a fresh directory.
func ExtractTarball(data []byte, destDir string) error {
	if err := CheckTarballSize(int64(len(data))); err != nil {
		return err
	}
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to decompress tarball: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(&cappedReader{r: gr, max: maxTarballStreamBytes})
	var entries int
	var total int64
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tarball entry: %w", err)
		}
		entries++
		if err := checkEntryLimits(hdr, entries, total); err != nil {
			return err
		}
		total += hdr.Size

		// Security: reject path-traversal and absolute entry names outright.
		if isTarPathTraversal(hdr.Name) {
			return fmt.Errorf("invalid path in tarball: %s", hdr.Name)
		}

		destPath := filepath.Join(destDir, filepath.FromSlash(hdr.Name))

		// Defense in depth: verify the joined path stays under destDir. This is
		// the containment idiom the analyzer follows (mirrors internal/builtins/tar.go).
		rel, err := filepath.Rel(destDir, destPath)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("path escapes destination: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destPath, 0o755); err != nil {
				return err
			}
			continue
		case tar.TypeReg:
			// Regular file — written below. (The reader normalizes legacy
			// TypeRegA to TypeReg, so we don't need a separate case.)
		case tar.TypeSymlink, tar.TypeLink:
			// Links are rejected: even a currently-local target could later be
			// resolved to escape destDir. Safer to refuse (matches builtins/tar.go).
			return fmt.Errorf("link entry rejected: %s", hdr.Name)
		default:
			// Skip unsupported entry types (devices, fifos, etc.).
			continue
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}

		if err := writeTarEntry(tr, hdr, destPath); err != nil {
			return err
		}
	}

	return nil
}

// checkEntryLimits refuses an entry that would take the archive past the
// entry-count, per-entry or total-size limit. It runs on every entry, skipped
// types included: archive/tar reads and discards the body of an entry type we
// skip, so an unchecked one would still cost its declared size in decompression.
func checkEntryLimits(hdr *tar.Header, entries int, totalSoFar int64) error {
	if entries > MaxTarballEntries {
		return &TarballLimitError{
			Limit: "entry count", Const: "pkg.MaxTarballEntries", Max: MaxTarballEntries,
			Observed: fmt.Sprintf("entry %d (%s)", entries, hdr.Name),
		}
	}
	if hdr.Size > MaxTarballEntryBytes {
		return &TarballLimitError{
			Limit: "per-entry size", Const: "pkg.MaxTarballEntryBytes", Max: MaxTarballEntryBytes,
			Observed: fmt.Sprintf("entry %s declares %d bytes", hdr.Name, hdr.Size),
		}
	}
	if totalSoFar+hdr.Size > MaxTarballTotalBytes {
		return &TarballLimitError{
			Limit: "total extracted size", Const: "pkg.MaxTarballTotalBytes", Max: MaxTarballTotalBytes,
			Observed: fmt.Sprintf("entry %s brings the total to %d bytes", hdr.Name, totalSoFar+hdr.Size),
		}
	}
	return nil
}

// writeTarEntry streams one regular-file entry to destPath. archive/tar never
// yields more than hdr.Size bytes, which checkEntryLimits has bounded; the
// LimitReader is defence in depth, and a body longer than declared is refused.
func writeTarEntry(tr *tar.Reader, hdr *tar.Header, destPath string) error {
	f, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", destPath, err)
	}
	n, err := io.Copy(f, io.LimitReader(tr, MaxTarballEntryBytes+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(destPath)
		return fmt.Errorf("failed to extract %s: %w", hdr.Name, err)
	}
	if n > MaxTarballEntryBytes {
		_ = os.Remove(destPath) // never leave a truncated file behind
		return &TarballLimitError{
			Limit: "per-entry size", Const: "pkg.MaxTarballEntryBytes", Max: MaxTarballEntryBytes,
			Observed: fmt.Sprintf("entry %s yielded more than %d bytes", hdr.Name, int64(MaxTarballEntryBytes)),
		}
	}
	return nil
}

// TarballHash computes SHA256 of tarball bytes.
func TarballHash(data []byte) string {
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:])
}

// VerifyDeclaredAssets checks that every file listed in [assets].files exists
// under packageDir/assets/. Returns nil if no assets are declared.
//
// This runs at publish time so a typo in ailang.toml fails loud rather than
// shipping a tarball whose runtime assetPath() lookups all return Err.
func VerifyDeclaredAssets(packageDir string, manifest *PackageManifest) error {
	if manifest == nil || len(manifest.Assets.Files) == 0 {
		return nil
	}
	for _, rel := range manifest.Assets.Files {
		full := filepath.Join(packageDir, AssetsDir, rel)
		info, err := os.Stat(full)
		if err != nil {
			return fmt.Errorf("[assets].files: declared asset %q not found at %s", rel, filepath.Join(AssetsDir, rel))
		}
		if info.IsDir() {
			return fmt.Errorf("[assets].files: declared asset %q is a directory; only files allowed", rel)
		}
	}
	return nil
}
