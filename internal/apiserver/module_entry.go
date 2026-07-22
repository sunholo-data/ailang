package apiserver

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/loader"
)

// registerModule is the SOLE write site for s.modules.
//
// M-SERVEAPI-UNIFY: Before v0.10.12, two paths wrote to s.modules — the
// main loadFile path (keyed by filepath.Rel) and the dep-discovery loop
// (keyed by a string-trimmed canonical ID). They drifted (the v0.10.7 →
// v0.10.11 cascade). This function replaces both: every module goes
// through here, computes its identity (PhysicalPath) and all projections
// (CanonicalID, DeclaredPath, RelPath) ONCE, and writes to the map keyed
// by PhysicalPath. Consumers (HTTP routes, OpenAPI, MCP, A2A) read the
// projection they need from the ModuleInfo fields — they NEVER re-derive
// from source data.
//
// Returns the normalized map key (PhysicalPath) and a bool indicating
// whether a new entry was created. If the file is not a local module
// (lives outside basePath), returns ("", false, nil) without error — the
// caller should skip it silently. If the file path is unreadable, also
// returns (nil) without error: not-my-concern errors are not fatal.
//
// Idempotent: a repeat call with the same PhysicalPath is a no-op
// (returns the existing key, false, nil).
func (s *Server) registerModule(loaded *loader.LoadedModule) (string, bool, error) {
	if loaded == nil || loaded.File == nil || loaded.Iface == nil {
		return "", false, nil
	}
	if loaded.File.Path == "" {
		return "", false, nil // no source path → cannot be a local file
	}

	// Compute identity: symlink-resolved absolute path.
	absFile, err := filepath.Abs(loaded.File.Path)
	if err != nil {
		return "", false, nil
	}
	if resolved, err := filepath.EvalSymlinks(absFile); err == nil {
		absFile = resolved
	}
	absFile = filepath.Clean(absFile)

	// Under-basePath filter — the ONLY filter. If the physical file
	// doesn't live under s.normalizedBasePath on disk, it's a package
	// file (or stdlib) and serve-api doesn't register its routes.
	if !strings.HasPrefix(absFile+string(filepath.Separator), s.normalizedBasePath) &&
		absFile != strings.TrimSuffix(s.normalizedBasePath, string(filepath.Separator)) {
		declaredPath := ""
		if loaded.File.Module != nil {
			declaredPath = loaded.File.Module.Path
		}
		// Diagnostic: log non-stdlib, non-pkg-cache rejections so operators
		// can see when a deeply-namespaced declared path causes a project
		// module to be loaded from a cache location instead of basePath.
		// Skip the noise-floor cases (stdlib + dependency cache resolutions).
		if declaredPath != "" &&
			!strings.HasPrefix(declaredPath, "std/") &&
			!strings.Contains(absFile, "/.ailang/") &&
			!strings.Contains(absFile, "/std/") {
			log.Printf("  Skipped: %s (declared %q resolves outside basePath %s — file at %s)",
				filepath.Base(absFile), declaredPath, s.normalizedBasePath, absFile)
		}
		// M-SERVEAPI-SURFACE-DROPS: track the drop so ValidateRegistration
		// can fail-fast on @route-bearing rejections (and /health can
		// surface partial registration). Stdlib + cache-noise drops are
		// recorded too — ValidateRegistration partitions by annotation,
		// not by log filtering.
		s.recordDrop(loaded, absFile, declaredPath)
		return "", false, nil
	}

	physicalPath := absFile

	// Idempotency check.
	s.mu.RLock()
	if _, exists := s.modules[physicalPath]; exists {
		s.mu.RUnlock()
		return physicalPath, false, nil
	}
	s.mu.RUnlock()

	// Compute all projections ONCE.
	relPath, relErr := filepath.Rel(s.normalizedBasePath, absFile)
	if relErr != nil || strings.HasPrefix(relPath, "..") {
		// Shouldn't happen given the under-basePath check above, but
		// fall back to the filename if it does.
		relPath = filepath.Base(absFile)
	}
	rel := filepath.ToSlash(strings.TrimSuffix(relPath, ".ail"))

	declaredPath := ""
	if loaded.File.Module != nil {
		declaredPath = loaded.File.Module.Path
	}

	// Build the ModuleInfo. Exports are populated via the existing
	// extract* helpers, which take *ModuleInfo and *ast.File.
	info := extractModuleInfo(loaded.Iface)
	info.PhysicalPath = physicalPath
	info.CanonicalID = loaded.Path
	info.DeclaredPath = declaredPath
	info.Path = rel // backwards-compat: Path is the RelPath projection
	info.File = loaded.File
	info.Iface = loaded.Iface

	extractParamInfo(info, loaded.File)
	extractRouteAnnotations(info, loaded.File)
	extractNoExposeAnnotations(info, loaded.File)
	extractMCPNameAnnotations(info, loaded.File)
	extractNoMCPAnnotations(info, loaded.File)
	extractDocComments(info, loaded.File, absFile)

	// Write.
	s.mu.Lock()
	// Re-check under the write lock (another goroutine may have
	// registered while we were computing projections).
	if existing, exists := s.modules[physicalPath]; exists {
		s.mu.Unlock()
		_ = existing
		return physicalPath, false, nil
	}
	s.modules[physicalPath] = info
	s.mu.Unlock()

	log.Printf("  Registered: %s (%d exports)", rel, len(info.Exports))
	return physicalPath, true, nil
}

// recordDrop appends a DroppedModule entry to s.droppedModules. Called
// from registerModule's under-basePath rejection branch. Annotations are
// scanned from the loaded.File's function declarations: currently only
// "@route" is surfaced (it's the trigger for ValidateRegistration's
// fail-fast). Other annotations may be added later without changing the
// API — DroppedModule.Annotations is a free-form string slice.
func (s *Server) recordDrop(loaded *loader.LoadedModule, absFile, declaredPath string) {
	drop := DroppedModule{
		PhysicalPath: absFile,
		DeclaredPath: declaredPath,
		FileBaseName: filepath.Base(absFile),
		Reason:       "outside-basePath",
	}
	if loaded.File != nil {
		for _, fn := range loaded.File.Funcs {
			if fn.GetAnnotation("route") != nil {
				drop.Annotations = append(drop.Annotations, "@route")
				break // presence-check only; one entry is enough
			}
		}
	}

	s.mu.Lock()
	s.droppedModules = append(s.droppedModules, drop)
	s.mu.Unlock()
}

// findModuleByRelPath looks up a registered module by its RelPath
// projection (the forward-slash relative path that consumers see in URL
// routes and module dispatch). Returns (nil, false) if no module with
// that RelPath is registered.
//
// M-SERVEAPI-UNIFY: s.modules is keyed by PhysicalPath, but consumers
// (HTTP handlers, A2A dispatch) work in RelPath terms because that's
// what URLs encode. This helper bridges the two without re-introducing
// a parallel map.
//
// The caller MUST hold s.mu (read or write lock).
func (s *Server) findModuleByRelPath(rel string) (*ModuleInfo, bool) {
	for _, info := range s.modules {
		if info != nil && info.Path == rel {
			return info, true
		}
	}
	return nil, false
}
