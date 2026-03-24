package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// VersionConflictError is returned when the dependency graph has incompatible
// version requirements for the same package.
type VersionConflictError struct {
	Package          string // e.g., "sunholo/firestore"
	DirectVersion    string // root manifest's version (empty if not a direct dep)
	ExistingVersion  string // version already resolved
	RequestedVersion string // conflicting version requested
	RequestedBy      string // package that requested the conflicting version
}

func (e *VersionConflictError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "version conflict: %s\n", e.Package)
	if e.DirectVersion != "" {
		fmt.Fprintf(&b, "  root requires: %s\n", e.DirectVersion)
	}
	fmt.Fprintf(&b, "  already resolved: %s\n", e.ExistingVersion)
	fmt.Fprintf(&b, "  transitive requires: %s (via %s)\n", e.RequestedVersion, e.RequestedBy)
	b.WriteString("\nresolution aborted\n\nsuggestion:\n")
	if e.DirectVersion != "" {
		fmt.Fprintf(&b, "  - republish %s against %s@%s\n", e.RequestedBy, e.Package, e.DirectVersion)
		fmt.Fprintf(&b, "  - or change root dependency to %s@%s explicitly\n", e.Package, e.ExistingVersion)
	} else {
		fmt.Fprintf(&b, "  - pin %s in root ailang.toml to resolve ambiguity\n", e.Package)
	}
	return b.String()
}

// ResolvedPackage is a fully resolved dependency with computed hash.
type ResolvedPackage struct {
	Name          string
	Version       string
	AILANG        string // Minimum AILANG version from package manifest
	ContentHash   string
	InterfaceHash string
	Source        string // "path", "git", or "registry"
	Path          string // absolute path for path/git deps
	GitURL        string // git repo URL (git deps only)
	GitRev        string // resolved commit hash (git deps only)
	GitSubdir     string // path within repo (git deps only)
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
	resolvedSet := make(map[string]string) // name → resolved version

	// Collect root direct dependency versions (authoritative)
	directDeps := make(map[string]string) // name → version
	for depName, dep := range manifest.Dependencies {
		if dep.Version != "" {
			directDeps[depName] = dep.Version
		}
	}

	// Shared registry client for all registry lookups in this resolution
	var registryClient *RegistryClient

	var resolve func(m *PackageManifest, dir string, fromRegistry bool, path []string) error
	resolve = func(m *PackageManifest, dir string, fromRegistry bool, path []string) error {
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

		// Resolve each dependency (sorted for deterministic order)
		depNames := make([]string, 0, len(m.Dependencies))
		for n := range m.Dependencies {
			depNames = append(depNames, n)
		}
		sort.Strings(depNames)

		for _, depName := range depNames {
			dep := m.Dependencies[depName]
			// When inside a registry package, path deps can't be resolved locally.
			// Convert them to registry lookups using the registry index.
			if fromRegistry && dep.Path != "" {
				if registryClient == nil {
					registryClient = NewRegistryClient()
				}
				// Look up the dep's latest version from the registry index
				index, err := registryClient.FetchIndex()
				if err != nil {
					return fmt.Errorf("failed to fetch registry index to resolve transitive dep %s: %w", depName, err)
				}
				version := ""
				for _, entry := range index.Packages {
					if entry.Name == depName {
						version = entry.Latest
						break
					}
				}
				if version == "" {
					return fmt.Errorf("transitive dependency %s (path dep in registry package %s) not found in registry", depName, name)
				}
				// Replace the path dep with a registry version dep for resolution
				dep = Dependency{Version: version}
			}

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
				if err := resolve(depManifest, depDir, false, append(path, name)); err != nil {
					return err
				}

				// Add this dep if not already resolved (with version conflict detection)
				if existingVer, already := resolvedSet[depName]; already {
					if existingVer != depManifest.Package.Version {
						return &VersionConflictError{
							Package:          depName,
							DirectVersion:    directDeps[depName],
							ExistingVersion:  existingVer,
							RequestedVersion: depManifest.Package.Version,
							RequestedBy:      name,
						}
					}
				} else {
					hash, err := ContentHash(depDir)
					if err != nil {
						return fmt.Errorf("failed to hash %s: %w", depName, err)
					}

					resolved = append(resolved, ResolvedPackage{
						Name:          depName,
						Version:       depManifest.Package.Version,
						AILANG:        depManifest.Package.AILANG,
						ContentHash:   hash,
						InterfaceHash: InterfaceHash(depManifest),
						Source:        "path",
						Path:          depDir,
						Effects:       depManifest.Effects.Max,
						Exports:       depManifest.Exports.Modules,
					})
					resolvedSet[depName] = depManifest.Package.Version
				}
			} else if dep.Git != "" {
				// Git dependency — clone/fetch to cache, resolve to local path
				cache, err := NewGitCache()
				if err != nil {
					return fmt.Errorf("failed to init git cache: %w", err)
				}
				localPath, resolvedRev, err := cache.Resolve(dep.Git, dep.Tag, dep.Rev, dep.Subdir)
				if err != nil {
					return fmt.Errorf("failed to resolve git dep %s: %w", depName, err)
				}

				depManifest, err := LoadManifest(localPath)
				if err != nil {
					return fmt.Errorf("failed to load git dep %s at %s: %w", depName, localPath, err)
				}

				if depManifest.Package.Name != depName {
					return fmt.Errorf("git dependency name mismatch: declared %q but manifest has name %q",
						depName, depManifest.Package.Name)
				}

				if err := resolve(depManifest, localPath, false, append(path, name)); err != nil {
					return err
				}

				if existingVer, already := resolvedSet[depName]; already {
					if existingVer != depManifest.Package.Version {
						return &VersionConflictError{
							Package:          depName,
							DirectVersion:    directDeps[depName],
							ExistingVersion:  existingVer,
							RequestedVersion: depManifest.Package.Version,
							RequestedBy:      name,
						}
					}
				} else {
					hash, err := ContentHash(localPath)
					if err != nil {
						return fmt.Errorf("failed to hash git dep %s: %w", depName, err)
					}

					resolved = append(resolved, ResolvedPackage{
						Name:          depName,
						Version:       depManifest.Package.Version,
						AILANG:        depManifest.Package.AILANG,
						ContentHash:   hash,
						InterfaceHash: InterfaceHash(depManifest),
						Source:        "git",
						// Path omitted — resolved at runtime from GitURL (portable lock file)
						GitURL:    dep.Git,
						GitRev:    resolvedRev,
						GitSubdir: dep.Subdir,
						Effects:   depManifest.Effects.Max,
						Exports:   depManifest.Exports.Modules,
					})
					resolvedSet[depName] = depManifest.Package.Version
				}
			} else {
				// Registry dependency — download from registry, cache locally
				if existingVer, already := resolvedSet[depName]; already {
					if existingVer != dep.Version {
						return &VersionConflictError{
							Package:          depName,
							DirectVersion:    directDeps[depName],
							ExistingVersion:  existingVer,
							RequestedVersion: dep.Version,
							RequestedBy:      name,
						}
					}
				} else {
					if registryClient == nil {
						registryClient = NewRegistryClient()
					}
					client := registryClient
					cachePath, err := CachedPackagePath(depName, dep.Version)
					if err != nil {
						return fmt.Errorf("failed to compute cache path for %s: %w", depName, err)
					}

					// Check if already cached
					if _, statErr := os.Stat(filepath.Join(cachePath, ManifestFile)); statErr != nil {
						// Not cached — download from registry
						tarballData, err := client.FetchPackage(depName, dep.Version)
						if err != nil {
							return fmt.Errorf("failed to download %s@%s from registry: %w", depName, dep.Version, err)
						}
						if err := os.MkdirAll(cachePath, 0755); err != nil {
							return fmt.Errorf("failed to create cache dir: %w", err)
						}
						if err := ExtractTarball(tarballData, cachePath); err != nil {
							return fmt.Errorf("failed to extract %s@%s: %w", depName, dep.Version, err)
						}
					}

					depManifest, err := LoadManifest(cachePath)
					if err != nil {
						return fmt.Errorf("failed to load cached %s@%s: %w", depName, dep.Version, err)
					}

					if err := resolve(depManifest, cachePath, true, append(path, name)); err != nil {
						return err
					}

					hash, err := ContentHash(cachePath)
					if err != nil {
						return fmt.Errorf("failed to hash %s: %w", depName, err)
					}

					resolved = append(resolved, ResolvedPackage{
						Name:          depName,
						Version:       dep.Version,
						AILANG:        depManifest.Package.AILANG,
						ContentHash:   hash,
						InterfaceHash: InterfaceHash(depManifest),
						Source:        "registry",
						// Path omitted — resolved at runtime from name+version (portable lock file)
						Effects: depManifest.Effects.Max,
						Exports: depManifest.Exports.Modules,
					})
					resolvedSet[depName] = dep.Version
				}
			}
		}

		visited[name] = true
		return nil
	}

	if err := resolve(manifest, rootDir, false, nil); err != nil {
		return nil, err
	}

	// Post-resolution validation: verify direct deps are authoritative
	// If a transitive dep resolved a different version than the root manifest
	// specifies, the resolve() above should have caught it. But as a safety
	// net, verify here too.
	for depName, directVersion := range directDeps {
		if resolvedVersion, ok := resolvedSet[depName]; ok && resolvedVersion != directVersion {
			return nil, &VersionConflictError{
				Package:          depName,
				DirectVersion:    directVersion,
				ExistingVersion:  resolvedVersion,
				RequestedVersion: directVersion,
				RequestedBy:      "(root manifest)",
			}
		}
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
