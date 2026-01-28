# M-HOT-RELOAD: Hot Reload for serve-api

**Status**: Planned
**Target**: v0.7.1
**Priority**: P1 (Medium)
**Estimated**: 1 day
**Dependencies**: serve-api (implemented)
**Created**: 2026-01-28

## Problem Statement

When developing with `ailang serve-api`, any change to `.ail` source files requires manually stopping and restarting the server. This breaks the development flow, especially when paired with a React frontend (Vite already hot-reloads JS, but the AILANG backend doesn't).

**Current State:**
- Edit `.ail` file -> must restart `ailang serve-api` manually
- Server startup takes ~3s (module compilation)
- No file watching infrastructure in the codebase (`fsnotify` not in go.mod)
- Existing `ailang watch` command is a stub (runs file once, no actual watching)

**Impact:**
- Developers lose flow during API iteration
- Mismatch with frontend DX (Vite hot-reloads instantly, backend requires restart)
- Simple fix: watch files, clear caches, recompile on change

## Goals

**Primary Goal:** Automatically recompile and reload AILANG modules when `.ail` files change, without restarting the server.

**Success Metrics:**
- File save -> module available via API in <500ms
- No server downtime during reload
- Compile errors logged but don't crash server (graceful degradation)
- Works with `--watch` flag (opt-in, not default)

## Axiom Compliance

| Axiom | Score | Rationale |
|-------|-------|-----------|
| A1: Determinism | +1 | Reload produces identical result to restart |
| A7: Machines First | +1 | Improves tooling for AI-driven development |
| A9: Cost Visibility | 0 | Neutral |
| A10: Composability | 0 | Neutral |
| **Net** | **+2** | Accept |

No axiom violations. Hot reload is a pure DX improvement that doesn't affect language semantics.

## Solution Design

### Architecture Overview

Three layers of caching must be invalidated on file change:

```
Layer 1: loader.ModuleLoader.cache     map[string]*LoadedModule
Layer 2: runtime.ModuleRuntime.instances  map[string]*ModuleInstance  (has sync.Once)
Layer 3: embed.Engine.compiled          map[string]bool
Layer 4: apiserver.Server.modules       map[string]*ModuleInfo      (metadata only)
```

On file change, all four layers are cleared for that module path, and `loadFile()` is called again. The next API request will use the fresh module.

**Key constraint:** `ModuleInstance.initOnce` is a `sync.Once` that cannot be reset. The instance must be fully deleted from `runtime.instances` and recreated.

### Implementation Plan

#### Phase 1: Cache Invalidation APIs (~50 LOC)

Add public methods to clear cached modules:

**`internal/runtime/runtime.go`** - Add `DeleteInstance`:
```go
// DeleteInstance removes a cached module instance, forcing re-evaluation on next load.
func (rt *ModuleRuntime) DeleteInstance(modulePath string) {
    delete(rt.instances, modulePath)
}
```

**`internal/loader/loader.go`** - Add `DeleteCached`:
```go
// DeleteCached removes a module from the loader cache, forcing re-load on next access.
func (ml *ModuleLoader) DeleteCached(modulePath string) {
    delete(ml.cache, modulePath)
}
```

**`internal/runtime/runtime.go`** - Expose loader for cache clearing:
```go
// GetLoader returns the module loader (for cache invalidation).
func (rt *ModuleRuntime) GetLoader() *loader.ModuleLoader {
    return rt.loader
}
```

**`internal/embed/embed.go`** - Add `InvalidateModule`:
```go
// InvalidateModule clears all caches for a module, forcing recompilation on next call.
func (e *Engine) InvalidateModule(modulePath string) {
    e.mu.Lock()
    defer e.mu.Unlock()
    delete(e.compiled, modulePath)
    e.runtime.DeleteInstance(modulePath)
    e.runtime.GetLoader().DeleteCached(modulePath)
}
```

#### Phase 2: File Watcher in apiserver (~120 LOC)

**Add `fsnotify` dependency:**
```bash
go get github.com/fsnotify/fsnotify
```

**`internal/apiserver/watcher.go`** (new file):
```go
package apiserver

import (
    "log"
    "path/filepath"
    "strings"
    "time"

    "github.com/fsnotify/fsnotify"
)

// startWatcher watches loaded .ail files for changes and triggers reload.
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
            log.Printf("Warning: cannot watch %s: %v", dir, err)
        }
    }

    go s.watchLoop()
    return nil
}

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
            if !strings.HasSuffix(event.Name, ".ail") {
                continue
            }
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

func (s *Server) reloadFile(absPath string) {
    // Find module path for this file
    relPath, err := filepath.Rel(s.basePath, absPath)
    if err != nil {
        log.Printf("Cannot resolve %s relative to %s: %v", absPath, s.basePath, err)
        return
    }
    modulePath := strings.TrimSuffix(filepath.ToSlash(relPath), ".ail")

    // Invalidate engine caches
    s.engine.InvalidateModule(modulePath)

    // Re-compile and update Server.modules
    if err := s.loadFile(absPath); err != nil {
        log.Printf("Hot reload FAILED for %s: %v", modulePath, err)
        log.Printf("  Previous version still serving (graceful degradation)")
        return
    }

    log.Printf("Hot reloaded: %s", modulePath)
}
```

#### Phase 3: CLI Flag & Server Integration (~30 LOC)

**`cmd/ailang/serve_api.go`** - Add `--watch` flag:
```go
watchFlag := fs.Bool("watch", false, "Watch .ail files for changes and hot-reload")
```

**`internal/apiserver/server.go`** - Add watcher field and config:
```go
type Config struct {
    // ... existing fields ...
    Watch bool // Enable file watching for hot reload
}

type Server struct {
    // ... existing fields ...
    watcher *fsnotify.Watcher
}
```

Start watcher in `Server.Start()` if `Config.Watch` is true:
```go
if s.config.Watch {
    if err := s.startWatcher(); err != nil {
        return err
    }
    log.Println("Hot reload enabled (watching for .ail file changes)")
}
```

Clean up in `Server.Close()`:
```go
if s.watcher != nil {
    s.watcher.Close()
}
```

### Phase 4 (Future): Dependency-Aware Reload

**Not in scope for v0.7.1.** When module A imports module B and B changes, A is NOT automatically reloaded. The user must save A (or any file) to trigger a full reload.

Future enhancement: build reverse dependency graph from module imports and cascade invalidation. This is ~100 LOC but requires parsing all module declarations to build the dep graph, which adds complexity.

## Files to Create/Modify

| File | Action | LOC |
|------|--------|-----|
| `internal/embed/embed.go` | Modify | +10 (InvalidateModule) |
| `internal/runtime/runtime.go` | Modify | +10 (DeleteInstance, GetLoader) |
| `internal/loader/loader.go` | Modify | +5 (DeleteCached) |
| `internal/apiserver/watcher.go` | Create | ~120 |
| `internal/apiserver/server.go` | Modify | +15 (watcher field, start/stop) |
| `cmd/ailang/serve_api.go` | Modify | +5 (--watch flag) |
| `cmd/ailang/help.go` | Modify | +1 (update help text) |
| `go.mod` / `go.sum` | Modify | +1 dependency (fsnotify) |
| `internal/apiserver/watcher_test.go` | Create | ~80 |
| `docs/docs/guides/serve-api.md` | Modify | +20 (document --watch) |

**Total: ~270 LOC**

## Testing Strategy

1. **Unit tests** for cache invalidation: call `InvalidateModule`, verify `HasInstance` returns false
2. **Integration test**: load module, call function, invalidate, modify source, call again, verify new result
3. **Manual test**: run `ailang serve-api --watch ./api/`, edit `.ail` file, curl endpoint, verify updated response
4. **Graceful degradation test**: introduce syntax error in `.ail` file, verify server logs error but continues serving previous version

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| `sync.Once` prevents re-evaluation | High | Delete entire instance, let next call recreate |
| Loader cache stale after invalidation | Medium | Also clear `loader.cache` entry |
| Race between reload and request | Low | Server.mu already protects `modules` map; Engine.mu protects `compiled` |
| Dependent modules see stale imports | Medium | Document limitation; defer cascade reload to Phase 4 |
| fsnotify platform differences | Low | Well-maintained library; works on macOS, Linux, Windows |

## Related Documents

- [serve-api plan](../../../.claude/plans/ticklish-knitting-waterfall.md) - Original serve-api implementation plan
- [Go Interop embed API](../../implemented/v0_7_0/m-go-interop-embed-api.md) - embed.Engine architecture
