package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/BurntSushi/toml"
	"github.com/sunholo-data/ailang/internal/strutil"
)

// PathDepRewrite describes a published dependency change. Rendering belongs to
// the CLI: staging is also used while pkg quality writes machine-readable JSON.
type PathDepRewrite struct{ Name, Path, Version string }

// RewritePathDeps replaces targetDir's path dependencies with the exact version
// declared by their local manifest, resolving paths against resolveFromDir.
// It is silent and leaves the target untouched if preparation fails.
func RewritePathDeps(targetDir, resolveFromDir string) ([]PathDepRewrite, error) {
	manifest, err := LoadManifest(targetDir)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(targetDir, ManifestFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)
	var rewrites []PathDepRewrite
	for _, name := range strutil.SortedKeys(manifest.Dependencies) {
		dep := manifest.Dependencies[name]
		if dep.Path == "" {
			continue
		}
		depDir := dep.Path
		if !filepath.IsAbs(depDir) {
			depDir = filepath.Join(resolveFromDir, depDir)
		}
		local, err := LoadManifest(depDir)
		if err != nil {
			return nil, fmt.Errorf("cannot read dependency %s at %s: %w\nPath deps must be resolvable at publish time", name, dep.Path, err)
		}
		pattern := fmt.Sprintf(`"%s"\s*=\s*\{\s*path\s*=\s*"%s"\s*\}`, regexp.QuoteMeta(name), regexp.QuoteMeta(dep.Path))
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("bad rewrite pattern for %s: %w", name, err)
		}
		if !re.MatchString(content) {
			return nil, fmt.Errorf("cannot rewrite path dependency %s in %s: unsupported manifest spelling", name, path)
		}
		content = re.ReplaceAllLiteralString(content, fmt.Sprintf(`"%s" = "%s"`, name, local.Package.Version))
		rewrites = append(rewrites, PathDepRewrite{Name: name, Path: dep.Path, Version: local.Package.Version})
	}
	if len(rewrites) == 0 {
		return nil, nil
	}
	// A regex match in a comment must not hide an unrewritten actual entry.
	var candidate PackageManifest
	if err := toml.Unmarshal([]byte(content), &candidate); err != nil {
		return nil, fmt.Errorf("invalid rewritten manifest: %w", err)
	}
	for name, dep := range candidate.Dependencies {
		if dep.Path != "" {
			return nil, fmt.Errorf("path dependency %s was not rewritten in %s", name, path)
		}
	}
	if err := candidate.Validate(); err != nil {
		return nil, fmt.Errorf("invalid rewritten manifest: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, err
	}
	return rewrites, nil
}
