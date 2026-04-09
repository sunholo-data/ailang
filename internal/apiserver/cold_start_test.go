package apiserver

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoadModules_NoDuplicateRegistration is a regression test for the
// v0.10.10 cold-start cascade. Before v0.10.11, populating ast.File.Path
// reanimated the dep-discovery loop in loadFile, which used a different
// key derivation than the main path — causing each local module to be
// registered under TWO keys in s.modules, doubling eager-load work.
//
// After M-SERVEAPI-UNIFY (v0.10.12), s.modules is keyed by PhysicalPath
// (the symlink-resolved absolute file path) and there is exactly one
// registration site (registerModule). The "no duplicates" invariant now
// reduces to: each PhysicalPath appears at most once.
//
// See design_docs/planned/v0_11_0/m-serveapi-unify.md.
func TestLoadModules_NoDuplicateRegistration(t *testing.T) {
	srv := newColdStartTestServer(t)
	defer srv.Close()

	srv.mu.RLock()
	defer srv.mu.RUnlock()

	if len(srv.modules) == 0 {
		t.Fatal("no modules registered — fixture broken")
	}

	// Every entry's PhysicalPath must equal its map key. The map is
	// keyed by PhysicalPath, so any divergence indicates a stale write
	// that bypassed registerModule.
	seenPhysical := make(map[string]bool)
	for key, info := range srv.modules {
		if info == nil {
			t.Errorf("nil ModuleInfo for key %q", key)
			continue
		}
		if info.PhysicalPath != "" && info.PhysicalPath != key {
			t.Errorf("ModuleInfo.PhysicalPath %q does not match map key %q (indicates stale write site)", info.PhysicalPath, key)
		}
		if seenPhysical[key] {
			t.Errorf("duplicate PhysicalPath in s.modules: %q", key)
		}
		seenPhysical[key] = true
	}

	// Strict count: the fixture has exactly 5 local files. Anything more
	// means a module was registered twice.
	const expectedModules = 5
	if len(srv.modules) != expectedModules {
		var rels []string
		for _, info := range srv.modules {
			if info != nil {
				rels = append(rels, info.Path)
			}
		}
		t.Errorf("expected exactly %d modules registered, got %d: %v", expectedModules, len(srv.modules), rels)
	}
}

// TestLoadModules_KeyConsistency asserts that every registration key in
// s.modules is now an absolute, symlink-resolved file path (the
// PhysicalPath identity established by M-SERVEAPI-UNIFY). The RelPath
// projection lives in info.Path and must be a forward-slash relative
// form under basePath.
func TestLoadModules_KeyConsistency(t *testing.T) {
	srv := newColdStartTestServer(t)
	defer srv.Close()

	srv.mu.RLock()
	defer srv.mu.RUnlock()

	for key, info := range srv.modules {
		// Keys must be absolute physical paths.
		if !filepath.IsAbs(key) {
			t.Errorf("module key %q is not absolute — should be PhysicalPath", key)
		}
		if !strings.HasSuffix(key, ".ail") {
			t.Errorf("module key %q missing .ail suffix — PhysicalPath should be the on-disk file path", key)
		}
		if info == nil {
			continue
		}
		// info.Path is the RelPath projection: forward-slash, no .ail,
		// relative under basePath.
		rel := info.Path
		if filepath.IsAbs(rel) {
			t.Errorf("info.Path %q is absolute — should be relative", rel)
		}
		if strings.HasSuffix(rel, ".ail") {
			t.Errorf("info.Path %q has .ail suffix — should be stripped", rel)
		}
		if strings.Contains(rel, "..") {
			t.Errorf("info.Path %q contains '..' segments", rel)
		}
		if strings.Contains(rel, "\\") {
			t.Errorf("info.Path %q uses backslash separator — should be forward slash", rel)
		}
	}
}

// TestLoadModules_DepDiscoveryNoRedundantWork captures log output during
// LoadModules and asserts that no module is registered more than once.
//
// Before v0.10.11, the dep-discovery loop in loadFile ran for every
// .ail file in the directory and could register the same module under
// two different keys. After M-SERVEAPI-UNIFY (v0.10.12), there is no
// dep-discovery loop at all — LoadProject does a single pipeline pass
// per file, and registerModule's idempotency check guarantees each
// PhysicalPath is logged ("Registered:") at most once.
func TestLoadModules_DepDiscoveryNoRedundantWork(t *testing.T) {
	// Capture the standard logger's output during LoadModules.
	var buf bytes.Buffer
	origOut := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(origOut) })

	srv := newColdStartTestServer(t)
	defer srv.Close()

	// Restore the writer before scanning so we don't capture our own
	// test output.
	log.SetOutput(origOut)

	// Count "Registered:" occurrences per module name (the rel-path
	// printed by registerModule).
	counts := make(map[string]int)
	for _, line := range strings.Split(buf.String(), "\n") {
		const marker = "Registered:"
		idx := strings.Index(line, marker)
		if idx < 0 {
			continue
		}
		rest := strings.TrimSpace(line[idx+len(marker):])
		// rest looks like: "lib/d (1 exports)"
		if sp := strings.Index(rest, " "); sp > 0 {
			rest = rest[:sp]
		}
		counts[rest]++
	}

	for mod, n := range counts {
		if n > 1 {
			t.Errorf("registerModule processed module %q %d times — expected exactly 1 (idempotency check broken)", mod, n)
		}
	}
	if len(counts) == 0 {
		t.Logf("captured log output:\n%s", buf.String())
		t.Fatal("no registration activity logged — fixture broken or registerModule not invoked")
	}
}

// TestLoadModules_DepDiscoverySkipsEntryPoint verifies that every
// fixture file ends up in s.modules under its expected RelPath
// projection. After M-SERVEAPI-UNIFY, the keys are PhysicalPaths, so
// we look up modules by their info.Path (RelPath) instead of by key.
func TestLoadModules_DepDiscoverySkipsEntryPoint(t *testing.T) {
	srv := newColdStartTestServer(t)
	defer srv.Close()

	srv.mu.RLock()
	defer srv.mu.RUnlock()

	// Build a set of RelPaths from info.Path values.
	rels := make(map[string]bool)
	for _, info := range srv.modules {
		if info != nil {
			rels[info.Path] = true
		}
	}

	expectedRels := []string{"main", "lib/a", "lib/b", "lib/c", "lib/d"}
	for _, want := range expectedRels {
		if !rels[want] {
			var actual []string
			for r := range rels {
				actual = append(actual, r)
			}
			t.Errorf("expected RelPath %q in s.modules; got %v", want, actual)
		}
	}
}

// newColdStartTestServer builds a fixture matching the shape that
// triggered the v0.10.10 regression on docparse: multiple local files,
// each with `@route` annotations, where one route file imports another.
//
// Real-world repro: docparse has services/api_keys.ail (@route), and
// services/mcp_tools.ail (@route, imports api_keys). serve-api walks
// the directory and calls loadFile for each. The dep-discovery loop
// inside loadFile(api_keys) sees mcp_tools imported transitively and
// (pre-v0.10.11) registered it under a key different from the one
// loadFile(mcp_tools) used on the main path → two entries for the
// same file in s.modules → eager-load runs twice → 2× cold start.
//
// We model this with three local files, each declaring an @route, with
// imports forming a chain so the dep-discovery loop runs over multiple
// route-bearing modules.
func newColdStartTestServer(t *testing.T) *Server {
	t.Helper()
	tmpDir := t.TempDir()

	// The pipeline's module loader uses CWD as basePath for resolving
	// bare imports like `lib/a`. Chdir into the fixture so transitive
	// imports resolve, and restore on cleanup.
	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origCwd) })

	// lib/d.ail — leaf, has @route.
	mustWrite(t, tmpDir, "lib/d.ail", `module lib/d

@route("GET", "/d")
export pure func dRoute() -> string = "d"
`)

	// lib/c.ail — imports d, has @route.
	mustWrite(t, tmpDir, "lib/c.ail", `module lib/c

import lib/d (dRoute)

@route("GET", "/c")
export pure func cRoute() -> string = dRoute()
`)

	// lib/b.ail — imports c, has @route.
	mustWrite(t, tmpDir, "lib/b.ail", `module lib/b

import lib/c (cRoute)

@route("GET", "/b")
export pure func bRoute() -> string = cRoute()
`)

	// lib/a.ail — imports b, has @route.
	mustWrite(t, tmpDir, "lib/a.ail", `module lib/a

import lib/b (bRoute)

@route("GET", "/a")
export pure func aRoute() -> string = bRoute()
`)

	// main.ail — imports a (transitively all of b, c, d), has @route.
	mainPath := filepath.Join(tmpDir, "main.ail")
	mustWrite(t, tmpDir, "main.ail", `module main

import lib/a (aRoute)

@route("GET", "/main")
export pure func mainRoute() -> string = aRoute()
`)
	_ = mainPath

	// Find stdlib (same dance as testServer).
	stdlibPath := os.Getenv("AILANG_STDLIB_PATH")
	if stdlibPath == "" {
		cwd, _ := os.Getwd()
		for dir := cwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			if _, err := os.Stat(filepath.Join(dir, "stdlib")); err == nil {
				stdlibPath = dir
				break
			}
		}
	}
	if stdlibPath != "" {
		os.Setenv("AILANG_STDLIB_PATH", stdlibPath)
	}

	srv := New(tmpDir, Config{Port: "0", CORS: true})
	// Load via the directory path so EACH .ail file goes through
	// loadFile separately — exactly the path docparse uses with
	// `ailang serve-api docparse/`. This is what triggers dep-discovery
	// across all sibling route files.
	if err := srv.LoadModules([]string{tmpDir}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}
	return srv
}

// TestLoadProject_DocparseShape exercises the realistic shape that
// triggered the v0.10.7 → v0.10.11 cascade on docparse: multiple local
// route-bearing files alongside a `pkg/` directory containing more
// route-bearing AILANG sources. The pkg/ directory holds vendored
// package files which the loader can resolve via imports but which
// MUST NOT be registered as local serve-api routes.
//
// Acceptance criteria from M-SERVEAPI-UNIFY M4:
//   - Local route files are registered exactly once each.
//   - pkg/ files are NOT registered as local routes (under-basePath
//     filter applied via PhysicalPath, not derived module IDs).
//   - Idempotency: a second LoadModules call doesn't change the set.
//   - Total registration count is exact (no duplicates, no leaks).
func TestLoadProject_DocparseShape(t *testing.T) {
	tmpDir := t.TempDir()

	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origCwd) })

	// --- Local services (the project's own files) ---
	mustWrite(t, tmpDir, "services/api_keys.ail", `module services/api_keys

@route("GET", "/api/v1/keys")
export pure func listKeys() -> string = "[]"
`)

	mustWrite(t, tmpDir, "services/mcp_tools.ail", `module services/mcp_tools

import services/api_keys (listKeys)

@route("POST", "/api/v1/mcp/parse")
export pure func mcpParse() -> string = listKeys()

@route("GET", "/api/v1/mcp/list")
export pure func mcpList() -> string = "tools"
`)

	mustWrite(t, tmpDir, "main.ail", `module main

import services/mcp_tools (mcpParse)

@route("GET", "/")
export pure func root() -> string = mcpParse()
`)

	// --- Vendored pkg/ files (must NOT be registered as local routes) ---
	// These live under tmpDir/pkg/ which LoadProject's walk should skip.
	mustWrite(t, tmpDir, "pkg/vendor/leaf.ail", `module pkg/vendor/leaf

@route("GET", "/should-not-appear")
export pure func leafRoute() -> string = "leaked"
`)

	// Find stdlib (same dance as testServer/newColdStartTestServer).
	stdlibPath := os.Getenv("AILANG_STDLIB_PATH")
	if stdlibPath == "" {
		cwd, _ := os.Getwd()
		for dir := cwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			if _, err := os.Stat(filepath.Join(dir, "stdlib")); err == nil {
				stdlibPath = dir
				break
			}
		}
	}
	if stdlibPath != "" {
		os.Setenv("AILANG_STDLIB_PATH", stdlibPath)
	}

	srv := New(tmpDir, Config{Port: "0", CORS: true})
	defer srv.Close()
	if err := srv.LoadModules([]string{tmpDir}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}

	// Acceptance: exactly 3 local modules, no pkg/ files registered.
	srv.mu.RLock()
	rels := make(map[string]bool)
	physicalPaths := make(map[string]bool)
	for key, info := range srv.modules {
		if info == nil {
			t.Errorf("nil ModuleInfo for key %q", key)
			continue
		}
		rels[info.Path] = true
		if physicalPaths[info.PhysicalPath] {
			t.Errorf("duplicate PhysicalPath registered: %q", info.PhysicalPath)
		}
		physicalPaths[info.PhysicalPath] = true
		// pkg/ files must never reach s.modules.
		if strings.Contains(info.Path, "pkg/") || strings.HasPrefix(info.Path, "pkg/") {
			t.Errorf("pkg/ file leaked into s.modules: info.Path=%q", info.Path)
		}
	}
	count := len(srv.modules)
	srv.mu.RUnlock()

	expectedRels := []string{"main", "services/api_keys", "services/mcp_tools"}
	const expectedCount = 3
	if count != expectedCount {
		var got []string
		for r := range rels {
			got = append(got, r)
		}
		t.Errorf("expected %d local modules, got %d: %v", expectedCount, count, got)
	}
	for _, want := range expectedRels {
		if !rels[want] {
			t.Errorf("missing expected RelPath %q in s.modules", want)
		}
	}

	// Idempotency: a second LoadModules pass must be a no-op on counts.
	if err := srv.LoadModules([]string{tmpDir}); err != nil {
		t.Fatalf("second LoadModules: %v", err)
	}
	srv.mu.RLock()
	count2 := len(srv.modules)
	srv.mu.RUnlock()
	if count2 != expectedCount {
		t.Errorf("LoadModules not idempotent: first=%d second=%d", expectedCount, count2)
	}
}

func mustWrite(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
