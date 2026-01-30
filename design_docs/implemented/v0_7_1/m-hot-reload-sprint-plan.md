# Sprint Plan: M-HOT-RELOAD - Hot Reload for serve-api

**Sprint ID**: M-HOT-RELOAD
**Design Doc**: [m-hot-reload-serve-api.md](m-hot-reload-serve-api.md)
**Duration**: 1 day (~4 hours implementation + testing)
**Risk Level**: Low
**Total LOC Estimate**: ~270

## Summary

Add `--watch` flag to `ailang serve-api` that monitors `.ail` files for changes and automatically recompiles/reloads modules without restarting the server. Requires cache invalidation APIs across 3 layers (loader, runtime, engine) and a new file watcher using fsnotify.

## Milestones

### M1: Cache Invalidation APIs (~25 LOC)

Add public methods to clear cached modules across all layers.

**Tasks:**
1. Add `DeleteInstance(modulePath)` to `internal/runtime/runtime.go`
2. Add `GetLoader()` to `internal/runtime/runtime.go` to expose loader
3. Add `DeleteCached(modulePath)` to `internal/loader/loader.go`
4. Add `InvalidateModule(modulePath)` to `internal/embed/embed.go` that clears all 3 caches
5. Write unit tests verifying cache invalidation

**Files:**
- `internal/runtime/runtime.go` (+10 LOC)
- `internal/loader/loader.go` (+5 LOC)
- `internal/embed/embed.go` (+10 LOC)
- `internal/embed/embed_test.go` (+30 LOC tests)

**Acceptance Criteria:**
- `Engine.InvalidateModule(path)` clears loader cache, runtime instance, and compiled flag
- `runtime.HasInstance(path)` returns false after `DeleteInstance(path)`
- Existing tests pass (no regression)

### M2: File Watcher (~150 LOC)

Create fsnotify-based file watcher that triggers module reload on .ail file changes.

**Tasks:**
1. Add `github.com/fsnotify/fsnotify` dependency
2. Create `internal/apiserver/watcher.go` with `startWatcher()`, `watchLoop()`, `reloadFile()`
3. Add `getWatchDirs()` helper to collect directories from loaded modules
4. Implement 200ms debounce to batch rapid saves
5. Implement graceful degradation (log compile errors, keep serving old version)
6. Write unit tests for watcher

**Files:**
- `go.mod` / `go.sum` (+1 dependency)
- `internal/apiserver/watcher.go` (new, ~120 LOC)
- `internal/apiserver/watcher_test.go` (new, ~80 LOC)

**Acceptance Criteria:**
- File watcher detects .ail file changes (Write and Create events)
- Non-.ail files are ignored
- Compile errors don't crash the server
- Debounce prevents multiple rapid reloads
- Watcher cleans up on server close

### M3: CLI Integration & Documentation (~30 LOC)

Wire `--watch` flag through CLI to server config and update docs.

**Tasks:**
1. Add `--watch` flag to `cmd/ailang/serve_api.go`
2. Add `Watch bool` to `apiserver.Config` struct
3. Add `watcher *fsnotify.Watcher` field to `apiserver.Server` struct
4. Start watcher in `Server.Start()` when `Watch` is true
5. Clean up watcher in `Server.Close()`
6. Update `cmd/ailang/help.go` to mention `--watch`
7. Update `docs/docs/guides/serve-api.md` with hot reload docs
8. Update CHANGELOG.md

**Files:**
- `cmd/ailang/serve_api.go` (+5 LOC)
- `internal/apiserver/server.go` (+15 LOC)
- `cmd/ailang/help.go` (+1 LOC)
- `docs/docs/guides/serve-api.md` (+20 LOC)
- `CHANGELOG.md` (+10 LOC)

**Acceptance Criteria:**
- `ailang serve-api --watch ./api/` starts with file watching enabled
- `ailang serve-api ./api/` (without --watch) does NOT watch files
- Help text shows `--watch` flag
- Documentation updated with hot reload section
- CHANGELOG updated

### M4: End-to-End Verification

Manual and automated testing of the full hot reload flow.

**Tasks:**
1. Run `ailang serve-api --watch examples/web_api_demo/api/`
2. Curl endpoint, verify response
3. Edit `.ail` file, save
4. Curl same endpoint, verify updated response
5. Introduce syntax error, verify server logs error but keeps serving
6. Fix syntax error, verify reload succeeds
7. Run full test suite: `make test`

**Acceptance Criteria:**
- Hot reload works end-to-end (edit file -> new response in <500ms)
- Graceful degradation on compile error
- All existing tests pass
- `examples/web_api_demo/test.sh` still passes

## Dependencies

```
M1 (Cache APIs) -> M2 (Watcher) -> M3 (CLI + Docs) -> M4 (E2E Verification)
```

All milestones are sequential.

## Success Metrics

- [ ] `--watch` flag available on `ailang serve-api`
- [ ] File changes trigger automatic module reload
- [ ] Compile errors don't crash the server
- [ ] Reload latency < 500ms
- [ ] All existing tests pass
- [ ] Documentation updated
- [ ] CHANGELOG updated
