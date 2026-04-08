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
// See design_docs/planned/v0_11_0/m-mcp-cascade-postmortem-and-fixes.md.
//
// This test asserts that LoadModules registers each .ail file exactly once,
// regardless of how many other local files import it.
func TestLoadModules_NoDuplicateRegistration(t *testing.T) {
	srv := newColdStartTestServer(t)
	defer srv.Close()

	// Walk the loaded module table and assert no module appears under two keys.
	srv.mu.RLock()
	defer srv.mu.RUnlock()

	if len(srv.modules) == 0 {
		t.Fatal("no modules registered — fixture broken")
	}

	// Build a reverse index: file path → keys it's registered under.
	// We don't have the absolute file path on ModuleInfo directly, but
	// we can use the Path field (which the loader sets to the registration
	// key) and assert it matches the map key.
	for key, info := range srv.modules {
		if info == nil {
			t.Errorf("nil ModuleInfo for key %q", key)
			continue
		}
		if info.Path != key {
			t.Errorf("ModuleInfo.Path %q does not match map key %q (indicates duplicate registration under different keys)", info.Path, key)
		}
	}

	// Strict count: the fixture has exactly 5 local files. Anything more
	// means a module was registered twice.
	const expectedModules = 5
	if len(srv.modules) != expectedModules {
		var keys []string
		for k := range srv.modules {
			keys = append(keys, k)
		}
		t.Errorf("expected exactly %d modules registered, got %d: %v", expectedModules, len(srv.modules), keys)
	}
}

// TestLoadModules_KeyConsistency asserts that every registration key in
// s.modules is a forward-slash relative path under basePath, NOT a
// canonical-ID-derived form (which is what the dep-discovery loop used to
// produce when it disagreed with the main path).
func TestLoadModules_KeyConsistency(t *testing.T) {
	srv := newColdStartTestServer(t)
	defer srv.Close()

	srv.mu.RLock()
	defer srv.mu.RUnlock()

	for key := range srv.modules {
		// Keys must be forward-slash relative paths, never absolute,
		// never containing ".." segments, never with .ail suffix.
		if filepath.IsAbs(key) {
			t.Errorf("module key %q is absolute — should be relative to basePath", key)
		}
		if strings.HasSuffix(key, ".ail") {
			t.Errorf("module key %q has .ail suffix — should be stripped", key)
		}
		if strings.Contains(key, "..") {
			t.Errorf("module key %q contains '..' segments", key)
		}
		if strings.Contains(key, "\\") {
			t.Errorf("module key %q uses backslash separator — should be forward slash", key)
		}
	}
}

// TestLoadModules_DepDiscoveryNoRedundantWork captures log output during
// LoadModules and asserts that the "Loaded package module: ... routes
// discovered" line (emitted ONLY by the dep-discovery loop) appears at
// most once per module across the entire LoadModules pass.
//
// Before v0.10.11, the dep-discovery loop in loadFile ran for every
// .ail file in the directory, processing every transitive dependency
// regardless of whether it had already been seen. With 5 fixture files
// importing each other, dep-discovery would emit ~10 redundant "Loaded
// package module" lines (the 4 deps × multiple visits). After v0.10.11,
// the existence check at the top of the loop short-circuits repeat
// visits, so each module is processed at most once.
//
// This is the most direct measure of the cold-start regression: the
// extract* work in dep-discovery is the dominant cost, and the log
// line appears exactly once per `extract*` invocation.
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

	// Count "Loaded package module:" occurrences per module name.
	counts := make(map[string]int)
	for _, line := range strings.Split(buf.String(), "\n") {
		const marker = "Loaded package module:"
		idx := strings.Index(line, marker)
		if idx < 0 {
			continue
		}
		rest := strings.TrimSpace(line[idx+len(marker):])
		// rest looks like: "lib/d (1 exports, routes discovered)"
		if sp := strings.Index(rest, " "); sp > 0 {
			rest = rest[:sp]
		}
		counts[rest]++
	}

	// Each route-bearing dep should be discovered AT MOST once. Before
	// v0.10.11 the same module was logged 2-4 times due to the missing
	// existence check in front of extract*.
	for mod, n := range counts {
		if n > 1 {
			t.Errorf("dep-discovery processed module %q %d times — expected at most 1 (regression: extract* runs redundantly)", mod, n)
		}
	}
	// Sanity: at least SOME deps should have been discovered, otherwise
	// the fixture is wrong and the test is vacuous.
	if len(counts) == 0 {
		t.Logf("captured log output:\n%s", buf.String())
		t.Fatal("no dep-discovery activity logged — fixture broken or dep-discovery loop dead")
	}
}

// TestLoadModules_DepDiscoverySkipsEntryPoint verifies that the
// dep-discovery loop in loadFile does NOT re-register the entry-point
// file under a different key. The main path of loadFile already handles
// the entry-point; dep-discovery should only register OTHER files.
func TestLoadModules_DepDiscoverySkipsEntryPoint(t *testing.T) {
	srv := newColdStartTestServer(t)
	defer srv.Close()

	srv.mu.RLock()
	defer srv.mu.RUnlock()

	// Each of the 5 fixture files should appear under exactly one key
	// matching its filepath.Rel form. The fixture filenames map to keys:
	// main.ail → "main", lib/a.ail → "lib/a", etc.
	expectedKeys := []string{"main", "lib/a", "lib/b", "lib/c", "lib/d"}
	for _, key := range expectedKeys {
		if _, ok := srv.modules[key]; !ok {
			var actual []string
			for k := range srv.modules {
				actual = append(actual, k)
			}
			t.Errorf("expected key %q in s.modules; got %v", key, actual)
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
		for dir := cwd; dir != "/"; dir = filepath.Dir(dir) {
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
