package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/sunholo/ailang/internal/schema"
)

// LockFileName is the canonical lock file name.
const LockFileName = "ailang.lock"

// LockFileSchema is the schema identifier for lock files.
const LockFileSchema = "ailang.lock/v1"

// LockFile represents the resolved dependency graph.
type LockFile struct {
	Schema      string          `json:"schema"`
	Version     string          `json:"schema_version"`
	GeneratedAt time.Time       `json:"generated_at"`
	Generator   string          `json:"generator"`
	Packages    []LockedPackage `json:"packages"`
}

// LockedPackage is a resolved dependency entry in the lock file.
type LockedPackage struct {
	Name          string   `json:"name"`
	Version       string   `json:"version"`
	ContentHash   string   `json:"content_hash"`
	InterfaceHash string   `json:"interface_hash,omitempty"`
	Source        string   `json:"source"` // "path" or "registry"
	Path          string   `json:"path,omitempty"`
	Effects       []string `json:"effects"`
	Exports       []string `json:"exports"`
}

// NewLockFile creates a lock file from resolved packages.
func NewLockFile(packages []LockedPackage, generator string) *LockFile {
	// Sort packages by name for determinism
	sorted := make([]LockedPackage, len(packages))
	copy(sorted, packages)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	// Sort effects and exports within each package
	for i := range sorted {
		sort.Strings(sorted[i].Effects)
		sort.Strings(sorted[i].Exports)
	}

	return &LockFile{
		Schema:      LockFileSchema,
		Version:     "1.0.0",
		GeneratedAt: time.Now().UTC(),
		Generator:   generator,
		Packages:    sorted,
	}
}

// Save writes the lock file as deterministic JSON.
func (lf *LockFile) Save(dir string) error {
	path := filepath.Join(dir, LockFileName)

	data, err := schema.MarshalDeterministic(lf)
	if err != nil {
		return fmt.Errorf("failed to marshal lock file: %w", err)
	}

	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		return fmt.Errorf("failed to format lock file: %w", err)
	}
	buf.WriteByte('\n')

	return os.WriteFile(path, buf.Bytes(), 0644)
}

// LoadLockFile reads and validates a lock file from a directory.
func LoadLockFile(dir string) (*LockFile, error) {
	path := filepath.Join(dir, LockFileName)
	return LoadLockFileFromPath(path)
}

// LoadLockFileFromPath reads and validates a lock file from a specific path.
func LoadLockFileFromPath(path string) (*LockFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var lf LockFile
	if err := json.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	if err := lf.Validate(); err != nil {
		return nil, fmt.Errorf("invalid lock file %s: %w", path, err)
	}

	return &lf, nil
}

// Validate checks the lock file for consistency.
func (lf *LockFile) Validate() error {
	if !schema.Accepts(lf.Schema, LockFileSchema) {
		return fmt.Errorf("unsupported lock file schema: %s (expected %s)", lf.Schema, LockFileSchema)
	}

	// Check for duplicate package names
	seen := make(map[string]bool)
	for _, p := range lf.Packages {
		if seen[p.Name] {
			return fmt.Errorf("duplicate package in lock file: %s", p.Name)
		}
		seen[p.Name] = true

		if p.ContentHash == "" {
			return fmt.Errorf("package %s missing content_hash", p.Name)
		}
		if p.Source == "" {
			return fmt.Errorf("package %s missing source", p.Name)
		}
	}

	return nil
}

// FindPackage looks up a package by name in the lock file.
func (lf *LockFile) FindPackage(name string) (*LockedPackage, bool) {
	for i := range lf.Packages {
		if lf.Packages[i].Name == name {
			return &lf.Packages[i], true
		}
	}
	return nil, false
}

// ValidateContentHashes re-computes content hashes for all path dependencies
// and verifies they match the lock file. Returns an error if any dependency
// has changed since the lock file was generated.
func (lf *LockFile) ValidateContentHashes() error {
	for _, p := range lf.Packages {
		if p.Source != "path" || p.Path == "" {
			continue
		}
		currentHash, err := ContentHash(p.Path)
		if err != nil {
			return fmt.Errorf("failed to hash dependency %s at %s: %w", p.Name, p.Path, err)
		}
		if currentHash != p.ContentHash {
			return fmt.Errorf("dependency %s content changed (locked: %s, current: %s)\nRun 'ailang lock' to update", p.Name, p.ContentHash[:24]+"...", currentHash[:24]+"...")
		}
	}
	return nil
}

// ValidateAgainstManifest checks that the lock file is consistent with a manifest.
// Returns an error if the manifest declares dependencies not in the lock file.
func (lf *LockFile) ValidateAgainstManifest(m *PackageManifest) error {
	for name := range m.Dependencies {
		if _, found := lf.FindPackage(name); !found {
			return fmt.Errorf("dependency %q in manifest but not in lock file; run 'ailang lock' to update", name)
		}
	}
	return nil
}
