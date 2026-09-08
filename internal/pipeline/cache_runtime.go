package pipeline

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type cacheDependencies struct {
	newStore func(string) (*CacheStore, error)
	stderr   io.Writer
}

func productionCacheDependencies() cacheDependencies {
	return cacheDependencies{newStore: NewCacheStore, stderr: os.Stderr}
}

type cacheRuntime struct {
	store         *CacheStore
	stderr        io.Writer
	invalidWarned map[string]bool
	writeWarned   map[string]bool
}

func newCacheRuntime(projectDir string, deps cacheDependencies) *cacheRuntime {
	runtime := &cacheRuntime{
		stderr:        deps.stderr,
		invalidWarned: make(map[string]bool),
		writeWarned:   make(map[string]bool),
	}
	if runtime.stderr == nil {
		runtime.stderr = io.Discard
	}
	store, err := deps.newStore(projectDir)
	if err != nil {
		runtime.warnWrite("initialization", "", cacheRootPath(projectDir), err)
		return runtime
	}
	runtime.store = store
	return runtime
}

func cacheRootPath(projectDir string) string {
	if override := os.Getenv("AILANG_CACHE_DIR"); override != "" {
		return filepath.Join(override, "compile")
	}
	return filepath.Join(projectDir, ".ailang", "cache", "compile")
}

func (runtime *cacheRuntime) load(moduleID, expectedKey string) (*CachedModule, *CacheEntry, bool) {
	if runtime == nil || runtime.store == nil {
		return nil, nil, false
	}
	entry, ok := runtime.store.Lookup(moduleID, expectedKey)
	if !ok {
		return nil, nil, false
	}
	cached, err := runtime.store.LoadArtifacts(moduleID, expectedKey)
	if err != nil {
		runtime.warnInvalid(moduleID, err)
		return nil, entry, false
	}
	return cached, entry, true
}

func (runtime *cacheRuntime) publish(moduleID, cacheKey string, entry *CacheEntry, cached *CachedModule) bool {
	if runtime == nil || runtime.store == nil {
		return false
	}
	if err := runtime.store.StoreArtifacts(moduleID, cacheKey, cached); err != nil {
		stage := "publication"
		var artifactErr *cacheArtifactError
		if errors.As(err, &artifactErr) && artifactErr.Stage != "" {
			stage = artifactErr.Stage
		}
		runtime.warnWrite(stage, moduleID, artifactErrorPath(err, runtime.store.moduleArtifactDir(moduleID)), err)
		return false
	}
	runtime.store.Store(moduleID, entry)
	return true
}

func (runtime *cacheRuntime) save() {
	if runtime == nil || runtime.store == nil {
		return
	}
	if err := runtime.store.Save(); err != nil {
		runtime.warnWrite("manifest_save", "", filepath.Join(runtime.store.dir, "manifest.json"), err)
	}
}

func (runtime *cacheRuntime) warnInvalid(moduleID string, err error) {
	if runtime.invalidWarned[moduleID] {
		return
	}
	runtime.invalidWarned[moduleID] = true
	path := ""
	reason := artifactInvalidReason
	scope := ""
	var limit int64
	var artifactErr *cacheArtifactError
	if errors.As(err, &artifactErr) {
		path = artifactErr.Path
		reason = artifactErr.Reason
		scope = artifactErr.Scope
		limit = artifactErr.LimitBytes
	}
	fmt.Fprintf(runtime.stderr, "CACHE_INVALID module=%s path=%s reason=%s", moduleID, path, reason)
	if scope != "" {
		fmt.Fprintf(runtime.stderr, " scope=%s limit_bytes=%d", scope, limit)
	}
	fmt.Fprintln(runtime.stderr, "; recompiling")
}

func (runtime *cacheRuntime) warnSourceUnavailable(moduleID string) {
	fmt.Fprintf(runtime.stderr, "CACHE_SOURCE_UNAVAILABLE module=%s; bypassing compilation cache\n", moduleID)
}

func (runtime *cacheRuntime) warnWrite(stage, moduleID, path string, err error) {
	key := stage + "\x00" + moduleID
	if runtime.writeWarned[key] {
		return
	}
	runtime.writeWarned[key] = true
	fmt.Fprint(runtime.stderr, "CACHE_WRITE_FAILED")
	if moduleID != "" {
		fmt.Fprintf(runtime.stderr, " module=%s", moduleID)
	}
	fmt.Fprintf(runtime.stderr, " stage=%s path=%s: %v; using fresh compilation\n", stage, path, err)
}

func artifactErrorPath(err error, fallback string) string {
	var artifactErr *cacheArtifactError
	if errors.As(err, &artifactErr) && artifactErr.Path != "" {
		return artifactErr.Path
	}
	return fallback
}
