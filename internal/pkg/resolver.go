package pkg

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ResolvedPackage is a fully resolved dependency with computed hash.
type ResolvedPackage struct {
	Name          string
	Version       string
	ContentHash   string
	InterfaceHash string
	Source        string // "path" or "registry"
	Path          string // absolute path for path deps
	Effects       []string
	Exports       []string
}

// ResolveDependencies resolves all dependencies from a manifest.
// rootDir is the directory containing the manifest.
// Returns resolved packages in topological order.
func ResolveDependencies(manifest *PackageManifest, rootDir string) ([]ResolvedPackage, error) {
	rootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve root dir: %w", err)
	}

	// Track visited packages for cycle detection
	visited := make(map[string]bool) // fully resolved
	inStack := make(map[string]bool) // currently being resolved (cycle detection)
	resolved := []ResolvedPackage{}
	resolvedSet := make(map[string]bool)

	var resolve func(m *PackageManifest, dir string, path []string) error
	resolve = func(m *PackageManifest, dir string, path []string) error {
		name := m.Package.Name

		if inStack[name] {
			cycle := append(path, name)
			return fmt.Errorf("circular dependency: %s", strings.Join(cycle, " → "))
		}
		if visited[name] {
			return nil
		}

		inStack[name] = true
		defer func() { delete(inStack, name) }()

		// Resolve each dependency
		for depName, dep := range m.Dependencies {
			if dep.Path != "" {
				// Path dependency — resolve relative to current package dir
				depDir := dep.Path
				if !filepath.IsAbs(depDir) {
					depDir = filepath.Join(dir, depDir)
				}
				depDir, err := filepath.Abs(depDir)
				if err != nil {
					return fmt.Errorf("failed to resolve path for %s: %w", depName, err)
				}

				depManifest, err := LoadManifest(depDir)
				if err != nil {
					return fmt.Errorf("failed to load dependency %s at %s: %w", depName, depDir, err)
				}

				// Verify declared name matches
				if depManifest.Package.Name != depName {
					return fmt.Errorf("dependency name mismatch: declared %q but manifest at %s has name %q",
						depName, depDir, depManifest.Package.Name)
				}

				// Recursively resolve transitive deps
				if err := resolve(depManifest, depDir, append(path, name)); err != nil {
					return err
				}

				// Add this dep if not already resolved
				if !resolvedSet[depName] {
					hash, err := ContentHash(depDir)
					if err != nil {
						return fmt.Errorf("failed to hash %s: %w", depName, err)
					}

					resolved = append(resolved, ResolvedPackage{
						Name:          depName,
						Version:       depManifest.Package.Version,
						ContentHash:   hash,
						InterfaceHash: InterfaceHash(depManifest),
						Source:        "path",
						Path:          depDir,
						Effects:       depManifest.Effects.Max,
						Exports:       depManifest.Exports.Modules,
					})
					resolvedSet[depName] = true
				}
			} else {
				// Registry dependency — for now, just record it unresolved
				// (Phase 2 will add registry resolution)
				if !resolvedSet[depName] {
					resolved = append(resolved, ResolvedPackage{
						Name:    depName,
						Version: dep.Version,
						Source:  "registry",
					})
					resolvedSet[depName] = true
				}
			}
		}

		visited[name] = true
		return nil
	}

	if err := resolve(manifest, rootDir, nil); err != nil {
		return nil, err
	}

	return resolved, nil
}

// BuildDependencyTree returns a printable dependency tree string.
func BuildDependencyTree(manifest *PackageManifest, rootDir string) (string, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s@%s\n", manifest.Package.Name, manifest.Package.Version))

	deps := sortedKeys(manifest.Dependencies)
	for i, name := range deps {
		dep := manifest.Dependencies[name]
		isLast := i == len(deps)-1
		prefix := "├── "
		if isLast {
			prefix = "└── "
		}

		if dep.Path != "" {
			sb.WriteString(fmt.Sprintf("%s%s (path: %s)\n", prefix, name, dep.Path))

			// Load transitive deps if possible
			depDir := dep.Path
			if !filepath.IsAbs(depDir) {
				depDir = filepath.Join(rootDir, depDir)
			}
			depManifest, err := LoadManifest(depDir)
			if err == nil && len(depManifest.Dependencies) > 0 {
				childPrefix := "│   "
				if isLast {
					childPrefix = "    "
				}
				printSubTree(&sb, depManifest, depDir, childPrefix)
			}
		} else {
			sb.WriteString(fmt.Sprintf("%s%s@%s\n", prefix, name, dep.Version))
		}
	}

	return sb.String(), nil
}

func printSubTree(sb *strings.Builder, m *PackageManifest, dir string, indent string) {
	deps := sortedKeys(m.Dependencies)
	for i, name := range deps {
		dep := m.Dependencies[name]
		isLast := i == len(deps)-1
		prefix := indent + "├── "
		if isLast {
			prefix = indent + "└── "
		}

		if dep.Path != "" {
			sb.WriteString(fmt.Sprintf("%s%s (path: %s)\n", prefix, name, dep.Path))
		} else {
			sb.WriteString(fmt.Sprintf("%s%s@%s\n", prefix, name, dep.Version))
		}
	}
}

func sortedKeys(m map[string]Dependency) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Sort for deterministic output
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}
