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

// CreateTarball creates a deterministic tar.gz of a package directory.
// Includes: ailang.toml, *.ail files, AGENT.md. Excludes: .git, tests/, ailang.lock.
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

		// Include: ailang.toml, *.ail, AGENT.md
		if rel == ManifestFile || strings.HasSuffix(rel, ".ail") || rel == "AGENT.md" {
			files = append(files, rel)
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

// ExtractTarball extracts a tar.gz to a destination directory.
func ExtractTarball(data []byte, destDir string) error {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to decompress tarball: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tarball entry: %w", err)
		}

		// Security: prevent path traversal
		cleanName := filepath.Clean(hdr.Name)
		if strings.HasPrefix(cleanName, "..") {
			return fmt.Errorf("invalid path in tarball: %s", hdr.Name)
		}

		destPath := filepath.Join(destDir, cleanName)

		if hdr.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return err
			}
			continue
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		data, err := io.ReadAll(tr)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", hdr.Name, err)
		}

		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", destPath, err)
		}
	}

	return nil
}

// TarballHash computes SHA256 of tarball bytes.
func TarballHash(data []byte) string {
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:])
}
