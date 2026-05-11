// Package pkg implements the AILANG package system: manifests, lock files,
// dependency resolution, and package loading.
package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// ManifestFile is the canonical manifest filename.
const ManifestFile = "ailang.toml"

// PackageManifest represents the contents of an ailang.toml file.
type PackageManifest struct {
	Package      PackageInfo             `toml:"package"`
	Exports      ExportConfig            `toml:"exports"`
	Dependencies map[string]Dependency   `toml:"dependencies"`
	DevDeps      map[string]Dependency   `toml:"dependencies_dev"`
	Effects      EffectConfig            `toml:"effects"`
	Metadata     map[string]interface{}  `toml:"metadata"`
	Stability    StabilityConfig         `toml:"stability"`
	Cascade      CascadeConfig           `toml:"cascade"`     // M-PKG-AUTONOMOUS-CASCADE-SAFE M3
	AIProviders  []AIProviderSpec        `toml:"ai_provider"` // M-AI-PROVIDER-CONFIG (v0.16.0); see internal/pkg/ai_provider.go
	Extensions   ExtensionRegistryConfig `toml:"extensions"`  // M-AILANG-EXT-REGISTRY-GEN (v0.17.1)
	Assets       AssetConfig             `toml:"assets"`      // M-EXT-PORTABILITY-GATE (v0.19.0)
}

// AssetConfig holds the optional [assets] section in ailang.toml.
// Lists files under the package's assets/ subdirectory that ship with the
// tarball and can be resolved at runtime via std/package.assetPath().
//
// Example:
//
//	[assets]
//	files = ["mcp-call.mjs", "schemas/tool-call.json"]
//
// At publish time, every file listed in assets.files MUST exist under the
// package's assets/ directory or publish is rejected.
type AssetConfig struct {
	// Files declares the expected assets relative to the package's assets/ dir.
	// When non-empty, validates each file exists at publish time.
	// When empty, all files under assets/ are still bundled (declaration is optional).
	Files []string `toml:"files"`
}

// ExtensionRegistryConfig holds the optional [extensions] section in ailang.toml.
// When non-empty, "ailang generate-extension-registry" reads this to emit a static
// dispatch file that wires the listed extension packages into a resolve() function.
//
// Example:
//
//	[extensions]
//	packages      = ["motoko-ext-compaction@0.2.0", "motoko-ext-exa-search@0.4.1"]
//	config_import = "src/core/config.RuntimeConfig"
//	hooks_import  = "src/core/ext/types.ExtensionHooks"
//	output        = "src/core/ext/registry_generated.ail"
type ExtensionRegistryConfig struct {
	// Packages lists extension packages to wire in, each pinned with @version.
	Packages []string `toml:"packages"`
	// ConfigImport is the "module/path.TypeName" of the config type passed to each extension.
	ConfigImport string `toml:"config_import"`
	// HooksImport is the "module/path.TypeName" of the hooks return type.
	HooksImport string `toml:"hooks_import"`
	// RegistryImport is the optional "module/path.TypeName" of the registry container type.
	// When set, the generator also emits parse_core_ext_order() and supporting helpers so
	// the generated file is a complete standalone registry entry point.
	RegistryImport string `toml:"registry_import"`
	// Output is the path for the generated file (default: "registry_generated.ail").
	Output string `toml:"output"`
	// ModuleName overrides the generated module declaration (default: derived from Output path).
	ModuleName string `toml:"module_name"`
}

// CascadeConfig holds the optional [cascade] section in ailang.toml.
// Used by the cascade scheduler to bound the cumulative cost of an
// autonomous cascade across the dependent DAG. Default applied when the
// section is absent: $1.00 USD per cascade root.
//
// Example:
//
//	[cascade]
//	max_cost_usd = 0.50  # Cascades rooted at this package abort after $0.50.
type CascadeConfig struct {
	// MaxCostUSD is the cumulative cost ceiling for a cascade rooted at
	// this package. When the cascade scheduler dispatches dependents and
	// the running per-task cost sum approaches the cap, it aborts further
	// dispatches with a structured event. 0 = use the default (1.0).
	MaxCostUSD float64 `toml:"max_cost_usd"`
}

// DefaultCascadeMaxCostUSD is the cumulative cost ceiling applied when a
// package's ailang.toml does not specify [cascade] max_cost_usd. Chosen so
// a typical small cascade (~3-5 dependents) bumps + tests fit comfortably,
// while a runaway cycle aborts long before cost becomes concerning.
const DefaultCascadeMaxCostUSD = 1.0

// EffectiveMaxCostUSD returns the per-cascade cost cap to enforce, applying
// the default when the manifest is silent.
func (c CascadeConfig) EffectiveMaxCostUSD() float64 {
	if c.MaxCostUSD <= 0 {
		return DefaultCascadeMaxCostUSD
	}
	return c.MaxCostUSD
}

// PackageInfo holds the [package] section.
type PackageInfo struct {
	Name         string `toml:"name"`
	Version      string `toml:"version"`
	Edition      string `toml:"edition"`
	AILANG       string `toml:"ailang"`        // Minimum AILANG version required (e.g., ">=0.9.5")
	ModulePrefix string `toml:"module_prefix"` // Optional: maps existing module paths to package namespace
	Description  string `toml:"description"`
	License      string `toml:"license"`
}

// ExportConfig holds the [exports] section.
type ExportConfig struct {
	Modules []string `toml:"modules"`
}

// Dependency can be a version string, path table, or git table.
type Dependency struct {
	Version string `toml:"version,omitempty"`
	Path    string `toml:"path,omitempty"`
	Git     string `toml:"git,omitempty"`    // git repo URL
	Tag     string `toml:"tag,omitempty"`    // git tag (e.g., "auth-v0.1.0")
	Rev     string `toml:"rev,omitempty"`    // git commit hash (overrides tag)
	Subdir  string `toml:"subdir,omitempty"` // path within git repo (e.g., "packages/auth")
}

// UnmarshalTOML implements custom TOML unmarshalling for Dependency.
// Supports:
//
//	"sunholo/json" = "0.3.1"                                                 (string → version)
//	"shared/utils" = { path = "../utils" }                                   (table → path dep)
//	"sunholo/auth" = { git = "https://...", subdir = "packages/auth", tag = "v0.1.0" }  (table → git dep)
func (d *Dependency) UnmarshalTOML(data interface{}) error {
	switch v := data.(type) {
	case string:
		d.Version = v
		return nil
	case map[string]interface{}:
		if p, ok := v["path"]; ok {
			if ps, ok := p.(string); ok {
				d.Path = ps
			}
		}
		if ver, ok := v["version"]; ok {
			if vs, ok := ver.(string); ok {
				d.Version = vs
			}
		}
		if g, ok := v["git"]; ok {
			if gs, ok := g.(string); ok {
				d.Git = gs
			}
		}
		if t, ok := v["tag"]; ok {
			if ts, ok := t.(string); ok {
				d.Tag = ts
			}
		}
		if r, ok := v["rev"]; ok {
			if rs, ok := r.(string); ok {
				d.Rev = rs
			}
		}
		if s, ok := v["subdir"]; ok {
			if ss, ok := s.(string); ok {
				d.Subdir = ss
			}
		}
		return nil
	default:
		return fmt.Errorf("dependency must be a version string or table, got %T", data)
	}
}

// EffectConfig holds the [effects] section.
type EffectConfig struct {
	Max []string `toml:"max"`
}

// StabilityConfig holds the [stability] section.
type StabilityConfig struct {
	Level string `toml:"level"`
}

// LoadManifest reads and validates an ailang.toml from the given directory.
func LoadManifest(dir string) (*PackageManifest, error) {
	path := filepath.Join(dir, ManifestFile)
	return LoadManifestFile(path)
}

// LoadManifestFile reads and validates an ailang.toml from a specific path.
func LoadManifestFile(path string) (*PackageManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var m PackageManifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("invalid manifest %s: %w", path, err)
	}

	return &m, nil
}

// Validate checks the manifest for required fields and consistency.
func (m *PackageManifest) Validate() error {
	if m.Package.Name == "" {
		return fmt.Errorf("[package].name is required")
	}
	if m.Package.Version == "" {
		return fmt.Errorf("[package].version is required")
	}
	if m.Package.Edition == "" {
		return fmt.Errorf("[package].edition is required")
	}

	// Validate ailang version constraint format if present (optional field)
	if m.Package.AILANG != "" {
		if _, err := ParseVersionConstraint(m.Package.AILANG); err != nil {
			return fmt.Errorf("[package].ailang: %w", err)
		}
	}

	// Validate package name is two-level: vendor/name
	parts := strings.SplitN(m.Package.Name, "/", 3)
	if len(parts) != 2 {
		return fmt.Errorf("[package].name must be vendor/name format, got %q", m.Package.Name)
	}
	if parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("[package].name segments must not be empty: %q", m.Package.Name)
	}

	// Validate module_prefix if set: must not contain "/" and must differ from package name
	if m.Package.ModulePrefix != "" {
		if strings.Contains(m.Package.ModulePrefix, "/") {
			return fmt.Errorf("[package].module_prefix must be a single segment (no slashes), got %q", m.Package.ModulePrefix)
		}
	}

	// Validate exported modules start with package name prefix or module_prefix
	for _, mod := range m.Exports.Modules {
		matchesPkgName := strings.HasPrefix(mod, m.Package.Name+"/") || mod == m.Package.Name
		matchesPrefix := m.Package.ModulePrefix != "" &&
			(strings.HasPrefix(mod, m.Package.ModulePrefix+"/") || mod == m.Package.ModulePrefix)
		if !matchesPkgName && !matchesPrefix {
			if m.Package.ModulePrefix != "" {
				return fmt.Errorf("exported module %q must start with package name %q or module_prefix %q",
					mod, m.Package.Name, m.Package.ModulePrefix)
			}
			return fmt.Errorf("exported module %q must start with package name %q", mod, m.Package.Name)
		}
	}

	// Validate dependencies have version, path, or git
	for name, dep := range m.Dependencies {
		if dep.Version == "" && dep.Path == "" && dep.Git == "" {
			return fmt.Errorf("dependency %q must specify version, path, or git", name)
		}
		if dep.Git != "" && dep.Tag == "" && dep.Rev == "" {
			return fmt.Errorf("git dependency %q must specify tag or rev", name)
		}
		if dep.Path != "" && dep.Git != "" {
			return fmt.Errorf("dependency %q cannot have both path and git", name)
		}
		// Reject non-exact version specifiers — only exact semver allowed in ailang.toml
		if dep.Version != "" && (dep.Version == "latest" || strings.ContainsAny(dep.Version, "^~><=")) {
			return fmt.Errorf("dependency %q has non-exact version %q — ailang.toml requires exact versions (e.g., \"0.1.0\")\n\nUse: ailang install %s@latest\nThis resolves and writes the exact version automatically.", name, dep.Version, name)
		}
	}

	// Validate stability level if set
	if m.Stability.Level != "" {
		switch m.Stability.Level {
		case "experimental", "stable", "frozen":
			// valid
		default:
			return fmt.Errorf("invalid stability level %q, must be experimental|stable|frozen", m.Stability.Level)
		}
	}

	// Validate [[ai_provider]] blocks (M-AI-PROVIDER-CONFIG, v0.16.0)
	if err := validateAIProviders(m.AIProviders); err != nil {
		return err
	}

	// Validate [assets].files entries are clean relative paths (M-EXT-PORTABILITY-GATE, v0.19.0)
	for _, asset := range m.Assets.Files {
		if asset == "" {
			return fmt.Errorf("[assets].files contains empty entry")
		}
		if filepath.IsAbs(asset) {
			return fmt.Errorf("[assets].files entry %q must be relative to assets/", asset)
		}
		clean := filepath.Clean(asset)
		if strings.HasPrefix(clean, "..") || strings.Contains(clean, "../") {
			return fmt.Errorf("[assets].files entry %q must not escape assets/", asset)
		}
	}

	// Validate [extensions] block (M-AILANG-EXT-REGISTRY-GEN, v0.17.1)
	if len(m.Extensions.Packages) > 0 {
		if m.Extensions.ConfigImport == "" {
			return fmt.Errorf("[extensions].config_import is required when packages are listed")
		}
		if m.Extensions.HooksImport == "" {
			return fmt.Errorf("[extensions].hooks_import is required when packages are listed")
		}
	}

	return nil
}

// MapImportToModulePath converts a canonical import path to the local module path.
// If module_prefix is set, remaps "vendor/name/subpath" → "prefix/subpath".
// If not set, returns the import path unchanged.
func (m *PackageManifest) MapImportToModulePath(importPath string) string {
	if m.Package.ModulePrefix == "" {
		return importPath
	}
	// importPath = "vendor/name/subpath" → extract subpath
	prefix := m.Package.Name + "/"
	if strings.HasPrefix(importPath, prefix) {
		subpath := strings.TrimPrefix(importPath, prefix)
		return m.Package.ModulePrefix + "/" + subpath
	}
	// Already uses module_prefix — return as-is
	return importPath
}

// MapModuleToImportPath converts a local module path to the canonical import path.
// If module_prefix is set, remaps "prefix/subpath" → "vendor/name/subpath".
// If not set, returns the module path unchanged.
func (m *PackageManifest) MapModuleToImportPath(modulePath string) string {
	if m.Package.ModulePrefix == "" {
		return modulePath
	}
	prefix := m.Package.ModulePrefix + "/"
	if strings.HasPrefix(modulePath, prefix) {
		subpath := strings.TrimPrefix(modulePath, prefix)
		return m.Package.Name + "/" + subpath
	}
	return modulePath
}

// IsPathDep returns true if the named dependency is a path dependency.
func (m *PackageManifest) IsPathDep(name string) bool {
	dep, ok := m.Dependencies[name]
	return ok && dep.Path != ""
}

// IsGitDep returns true if the named dependency is a git dependency.
func (m *PackageManifest) IsGitDep(name string) bool {
	dep, ok := m.Dependencies[name]
	return ok && dep.Git != ""
}

// InitManifest creates a default ailang.toml in the given directory.
// ailangVersion is the current AILANG compiler version (used to set the ailang constraint).
// Pass empty string to omit the constraint.
func InitManifest(dir, name, ailangVersion string) error {
	path := filepath.Join(dir, ManifestFile)

	// Don't overwrite existing manifest
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists in %s", ManifestFile, dir)
	}

	// Default stability
	stability := "experimental"

	ailangLine := ""
	constraint := FormatVersionConstraint(ailangVersion)
	if constraint != "" {
		ailangLine = fmt.Sprintf("ailang = %q\n", constraint)
	}

	content := fmt.Sprintf(`[package]
name = %q
version = "0.1.0"
edition = "1"
%s
[exports]
modules = [%q]

[effects]
max = []

[stability]
level = %q
`, name, ailangLine, name+"/core", stability)

	return os.WriteFile(path, []byte(content), 0644)
}

// FindManifest walks up from dir looking for ailang.toml.
// Returns the directory containing it, or empty string if not found.
func FindManifest(dir string) string {
	dir, _ = filepath.Abs(dir)
	for {
		if _, err := os.Stat(filepath.Join(dir, ManifestFile)); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
