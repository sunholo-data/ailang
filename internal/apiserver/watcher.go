package apiserver

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// startWatcher initializes the file watcher and begins watching directories
// containing loaded .ail files. Changes trigger cache invalidation and recompilation.
func (s *Server) startWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	s.watcher = watcher

	// Watch directories containing loaded modules
	dirs := s.getWatchDirs()
	for _, dir := range dirs {
		if err := watcher.Add(dir); err != nil {
			log.Printf("  Warning: cannot watch %s: %v", dir, err)
		} else {
			log.Printf("  Watching: %s", dir)
		}
	}

	go s.watchLoop()
	return nil
}

// watchLoop processes fsnotify events with debouncing to batch rapid saves.
func (s *Server) watchLoop() {
	// Debounce: collect events for 200ms before reloading
	var debounce *time.Timer
	pendingFiles := map[string]bool{}

	for {
		select {
		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}
			// Only care about .ail files (ignore temp files from editors/sed)
			base := filepath.Base(event.Name)
			if !strings.HasSuffix(event.Name, ".ail") || strings.Contains(base, "!") || strings.HasPrefix(base, ".") {
				continue
			}
			// Only care about Write and Create events
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			pendingFiles[event.Name] = true
			if debounce != nil {
				debounce.Stop()
			}
			debounce = time.AfterFunc(200*time.Millisecond, func() {
				for file := range pendingFiles {
					s.reloadFile(file)
				}
				pendingFiles = map[string]bool{}
			})

		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

// reloadFile recompiles a single .ail file and swaps it in atomically.
// On compile error, the previous version continues serving (graceful degradation).
func (s *Server) reloadFile(absPath string) {
	// Find module path for this file
	relPath, err := filepath.Rel(s.basePath, absPath)
	if err != nil {
		log.Printf("  Hot reload: cannot resolve %s relative to %s: %v", absPath, s.basePath, err)
		return
	}
	modulePath := strings.TrimSuffix(filepath.ToSlash(relPath), ".ail")

	// First, try to compile the new source via loadFile (which updates s.modules metadata).
	// loadFile only does pipeline.ModeCheck, so it doesn't affect the engine's runtime caches.
	// If compilation fails, we keep the old engine caches intact for graceful degradation.
	if err := s.loadFile(absPath); err != nil {
		log.Printf("  Hot reload FAILED for %s: %v", modulePath, err)
		log.Printf("  Previous version still serving (graceful degradation)")
		return
	}

	// Compilation succeeded - now invalidate engine caches so the next Call() picks up
	// the fresh source. The engine will lazily recompile on the next request.
	s.engine.InvalidateModule(modulePath)

	log.Printf("  Hot reloaded: %s", modulePath)
}

// getWatchDirs returns the unique set of directories containing loaded .ail files.
func (s *Server) getWatchDirs() []string {
	seen := map[string]bool{}
	var dirs []string

	for _, absPath := range s.watchPaths {
		dir := filepath.Dir(absPath)
		if !seen[dir] {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}

	return dirs
}
