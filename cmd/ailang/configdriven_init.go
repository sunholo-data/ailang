package main

import (
	"fmt"
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/configdriven"
	"github.com/sunholo-data/ailang/internal/pkg"
)

// ManifestSource pairs a parsed PackageManifest with the absolute path of
// the ailang.toml it came from. The path is used in error messages so users
// can locate the file declaring a conflicting [[ai_provider]] block.
type ManifestSource struct {
	Manifest *pkg.PackageManifest
	Path     string
}

// RegisterConfigDrivenProviders walks a list of manifests, harvests every
// [[ai_provider]] block, and registers each as a config-driven AI provider
// against the supplied registry. Built-in provider names are reserved —
// registering a config-driven provider with a built-in name is allowed but
// flagged via Diagnostics(); the built-in wins at dispatch time.
//
// Per-manifest validation already happens at LoadManifest time
// (validateAIProviders in internal/pkg/ai_provider.go) — this function
// performs *cross-manifest* duplicate-name detection. On conflict, returns
// a structured error naming both source manifests so the user can resolve
// (rename one, remove one, or aliasing — out of scope for v1).
//
// Idempotent: re-registering the exact same provider+source is a no-op.
// Safe to call multiple times in the same session (e.g. in tests).
func RegisterConfigDrivenProviders(registry *ai.ProviderRegistry, sources []ManifestSource) error {
	if registry == nil {
		registry = ai.GlobalProviderRegistry
	}

	for _, src := range sources {
		if src.Manifest == nil {
			continue
		}
		absPath := src.Path
		if abs, err := filepath.Abs(absPath); err == nil {
			absPath = abs
		}

		for i := range src.Manifest.AIProviders {
			spec := &src.Manifest.AIProviders[i]
			provider := configdriven.New(spec)
			if err := registry.Register(spec.Name, provider, absPath); err != nil {
				return fmt.Errorf("failed to register [[ai_provider]] from %s: %w", absPath, err)
			}
		}
	}
	return nil
}

// LookupConfigDrivenProvider is a convenience for dispatch sites: returns
// the registered provider for the given name, or nil if not registered.
// Built-ins are NOT consulted here — dispatch must check built-ins first.
func LookupConfigDrivenProvider(name string) ai.Provider {
	p, ok := ai.GlobalProviderRegistry.Lookup(name)
	if !ok {
		return nil
	}
	return p
}

// HarvestAndRegisterFromDir scans the ailang.toml in dir (and its dependency
// manifests via the lock file), harvests every [[ai_provider]] block, and
// registers each in the global registry. Called at CLI startup so that
// dispatch sites in setupAIHandler[FromConfig|Direct] can resolve
// config-driven providers.
//
// Bare projects (no ailang.toml in any ancestor) silently produce no
// registrations — that's fine, dispatch falls through to the built-in error
// path. Errors loading specific dependency manifests are NOT fatal: we
// register what we can and skip what we can't, since a missing
// [[ai_provider]] block in one dependency shouldn't break the whole project.
//
// Cross-package duplicate-name conflicts ARE fatal: returning the error
// causes the CLI to surface it to the user with both source paths.
func HarvestAndRegisterFromDir(dir string) error {
	manifestDir := pkg.FindManifest(dir)
	if manifestDir == "" {
		return nil // bare project — no manifests to harvest
	}

	rootManifest, err := pkg.LoadManifest(manifestDir)
	if err != nil {
		return nil // malformed root manifest — pipeline will report it
	}

	sources := []ManifestSource{{
		Manifest: rootManifest,
		Path:     filepath.Join(manifestDir, pkg.ManifestFile),
	}}

	// Walk dependencies via the lock file. If the lock file is missing or
	// unreadable, we can still register the root's providers — the lock is
	// only needed to look up dep manifests.
	if lf, err := pkg.LoadLockFile(manifestDir); err == nil {
		pkgLoader := pkg.NewPackageLoader(lf, manifestDir)
		for depName := range rootManifest.Dependencies {
			depManifest, err := pkgLoader.LoadManifestByName(depName)
			if err != nil || depManifest == nil {
				continue // skip — not fatal, just missing AI provider data from this dep
			}
			// Source identifier for error messages. The on-disk path of a
			// transitive dep manifest isn't readily exposed by PackageLoader,
			// so we use a logical "package@version" identifier — sufficient
			// for the cross-package conflict diagnostic which exists only to
			// help the user identify which package is the offender.
			sources = append(sources, ManifestSource{
				Manifest: depManifest,
				Path:     fmt.Sprintf("%s@%s (transitive dep)", depManifest.Package.Name, depManifest.Package.Version),
			})
		}
	}

	return RegisterConfigDrivenProviders(nil, sources)
}
