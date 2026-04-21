package apiserver

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/pipeline"
)

// LoadProject is the M-SERVEAPI-UNIFY replacement for the per-file
// loadFile + dep-discovery loop. It walks basePath for .ail files and
// compiles each one through the pipeline, registering every module in
// result.Modules via the single-write-site registerModule.
//
// Duplicate work is avoided by tracking compiled physical paths: once a
// file has been seen in some result.Modules pass, a direct pipeline run
// for that file is skipped (the loader cache may also short-circuit but
// this avoids even the cache lookup).
//
// Unlike the old path, LoadProject has ONE registration site and no
// per-file dedup glue. Projection drift between main path and
// dep-discovery is structurally impossible because there is no
// dep-discovery.
func (s *Server) LoadProject(ctx context.Context, basePath string) error {
	// Discover all .ail files in basePath.
	var files []string
	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip package cache directories — those are loaded via
			// import resolution, not file walking.
			if info.Name() == "pkg" && path != basePath {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".ail") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk %s: %w", basePath, err)
	}

	// Compile each file, register every module we see. Skip files whose
	// physical path has already been seen via a prior compile's
	// result.Modules (loader cache + idempotent registerModule make this
	// safe but dynamic dedup saves cache lookups).
	seenPhysical := make(map[string]bool)

	for _, file := range files {
		absPath, err := filepath.Abs(file)
		if err != nil {
			return fmt.Errorf("abs %s: %w", file, err)
		}
		if resolved, err := filepath.EvalSymlinks(absPath); err == nil {
			absPath = resolved
		}
		if seenPhysical[absPath] {
			continue
		}

		cfg := pipeline.Config{
			Mode:         pipeline.ModeCheck,
			RelaxModules: true,
		}
		src := pipeline.Source{Filename: absPath}

		result, err := pipeline.RunWithContext(ctx, cfg, src)
		if err != nil {
			return fmt.Errorf("compile %s: %w", file, err)
		}
		if len(result.Errors) > 0 {
			return fmt.Errorf("compile errors for %s: %v", file, result.Errors)
		}

		// Preload every transitive module into the runtime, then register
		// every local one via registerModule. Package files are filtered
		// inside registerModule by the under-basePath check.
		if result.Modules != nil {
			for modID, loaded := range result.Modules {
				s.engine.PreloadModule(modID, loaded)
				// Aliasing support: also preload under declared path.
				if loaded.Path != "" && loaded.Path != modID {
					s.engine.PreloadModule(loaded.Path, loaded)
				}
				physical, _, regErr := s.registerModule(loaded)
				if regErr != nil {
					return regErr
				}
				if physical != "" {
					seenPhysical[physical] = true
				}
			}
		}

		// Track this file for watch-mode hot reload.
		s.watchPaths = append(s.watchPaths, absPath)
	}

	// Eager-load every registered module exactly once. Unlike the old
	// LoadModules, this iterates s.modules by CanonicalID (not the
	// physical-path key) because engine.Load expects canonical IDs.
	// Because each entry in s.modules is unique per physical file,
	// engine.Load runs at most once per module (the v0.10.10 regression
	// cannot recur).
	s.mu.RLock()
	ids := make([]string, 0, len(s.modules))
	for _, entry := range s.modules {
		if entry == nil || entry.CanonicalID == "" {
			continue
		}
		// Skip pkg/ modules — they're already preloaded via
		// PreloadModule above; eager-loading them can corrupt canonical
		// paths (same reason as old LoadModules).
		if strings.HasPrefix(entry.CanonicalID, "pkg/") {
			continue
		}
		ids = append(ids, entry.CanonicalID)
	}
	s.mu.RUnlock()

	for _, id := range ids {
		if err := s.engine.Load(id); err != nil {
			log.Printf("  Warning: eager load failed for %s: %v", id, err)
		}
	}

	return nil
}
