package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestServeAPI_MCPToolSurface(t *testing.T) {
	binary := buildAilang(t)

	moduleRoot := t.TempDir()
	modulePath := filepath.Join(moduleRoot, "api", "surface.ail")
	if err := os.MkdirAll(filepath.Dir(modulePath), 0755); err != nil {
		t.Fatal(err)
	}
	content := `module api/surface

@route("GET", "/status")
export pure func status() -> string = "ok"

export pure func helper() -> string = "helper"
`
	if err := os.WriteFile(modulePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name       string
		suppress   bool
		wantSubmit bool
	}{
		{"default", false, true},
		{"suppressed", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			names := probeServeAPIMCPTools(t, binary, modulePath, tc.suppress)
			if len(names) == 0 {
				t.Fatal("instrument failure - positive control missing: empty tools/list")
			}
			if !names["status"] {
				t.Fatalf("instrument failure - positive control status missing; tools: %v", names)
			}
			if names["submit_feedback"] != tc.wantSubmit {
				t.Errorf("submit_feedback present = %v, want %v; tools: %v", names["submit_feedback"], tc.wantSubmit, names)
			}
		})
	}
}

func TestCompileCacheClear_Artifacts(t *testing.T) {
	binary := buildAilang(t)

	t.Run("success", func(t *testing.T) {
		root := t.TempDir()
		cacheRoot := filepath.Join(root, "isolated-cache")
		t.Setenv("AILANG_CACHE_DIR", cacheRoot)
		compileRoot := filepath.Join(cacheRoot, "compile")
		modulesRoot := filepath.Join(compileRoot, "modules")
		if err := os.MkdirAll(filepath.Join(modulesRoot, "orphan", "nested"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(modulesRoot, "orphan", "nested", ".artifacts-dead.tmp"), []byte("partial"), 0o644); err != nil {
			t.Fatal(err)
		}
		manifest := `{"version":"v4","entries":{"seed":{"cache_key":"key","iface_digest":"digest","compile_time_ms":1,"timestamp":"2026-01-01T00:00:00Z"}}}`
		if err := os.WriteFile(filepath.Join(compileRoot, "manifest.json"), []byte(manifest), 0o644); err != nil {
			t.Fatal(err)
		}
		compileSentinel := filepath.Join(compileRoot, "keep.txt")
		cacheSentinel := filepath.Join(cacheRoot, "package-cache", "keep.txt")
		if err := os.WriteFile(compileSentinel, []byte("keep"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(cacheSentinel), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(cacheSentinel, []byte("keep"), 0o644); err != nil {
			t.Fatal(err)
		}

		stdout, stderr, exitCode := runAilangBin(t, binary, "cache", "compile-clear")
		if exitCode != 0 {
			t.Fatalf("compile-clear exit=%d, want 0; stdout=%q stderr=%q", exitCode, stdout, stderr)
		}
		if !strings.Contains(stdout, "Cleared 1 cached compilation entries") {
			t.Fatalf("success line missing from stdout %q; stderr=%q", stdout, stderr)
		}
		if _, err := os.Stat(modulesRoot); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("modules subtree still exists or stat failed: %v", err)
		}
		for _, sentinel := range []string{compileSentinel, cacheSentinel} {
			if data, err := os.ReadFile(sentinel); err != nil || string(data) != "keep" {
				t.Errorf("sentinel %s changed or removed: data=%q err=%v", sentinel, data, err)
			}
		}
	})

	t.Run("failure has no success line", func(t *testing.T) {
		root := t.TempDir()
		cacheRoot := filepath.Join(root, "isolated-cache")
		t.Setenv("AILANG_CACHE_DIR", cacheRoot)
		compileRoot := filepath.Join(cacheRoot, "compile")
		modulesRoot := filepath.Join(compileRoot, "modules", "partial")
		if err := os.MkdirAll(modulesRoot, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(modulesRoot, "core.gob"), []byte("partial"), 0o644); err != nil {
			t.Fatal(err)
		}
		// A directory at the manifest path deterministically makes Save fail on
		// every supported platform while keeping all writes under t.TempDir.
		if err := os.Mkdir(filepath.Join(compileRoot, "manifest.json"), 0o755); err != nil {
			t.Fatal(err)
		}

		stdout, stderr, exitCode := runAilangBin(t, binary, "cache", "compile-clear")
		if exitCode == 0 {
			t.Fatalf("compile-clear exit=0, want nonzero; stdout=%q stderr=%q", stdout, stderr)
		}
		if strings.Contains(stdout, "Cleared ") {
			t.Fatalf("success line printed on failure: stdout=%q stderr=%q", stdout, stderr)
		}
		if !strings.Contains(stderr, "Error") {
			t.Fatalf("failure diagnostic missing: stdout=%q stderr=%q", stdout, stderr)
		}
	})
}

func TestServeAPI_DivergentCacheTools(t *testing.T) {
	binary := buildAilang(t)
	root := t.TempDir()
	cacheRoot := filepath.Join(root, "isolated-cache")
	modulePath := filepath.Join(root, "entry.ail")

	writeRoutes := func(count int) {
		t.Helper()
		var source strings.Builder
		source.WriteString("module entry\n")
		for i := 1; i <= count; i++ {
			fmt.Fprintf(&source, "\n@route(\"POST\", \"/f%d\")\nexport pure func f%d(x: float) -> float = x + %d.0\n", i, i, i)
		}
		if err := os.WriteFile(modulePath, []byte(source.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	wantNames := func(count int) []string {
		names := make([]string, count)
		for i := range names {
			names[i] = fmt.Sprintf("f%d", i+1)
		}
		return names
	}
	assertSurface := func(want []string, got mcpProbeResult) {
		t.Helper()
		if strings.Join(got.tools, ",") != strings.Join(want, ",") {
			t.Fatalf("tools/list = %v, want exactly %v; stderr:\n%s", got.tools, want, got.stderr)
		}
	}

	writeRoutes(6)
	first := probeServeAPICacheMCP(t, binary, modulePath, cacheRoot, false)
	assertSurface(wantNames(6), first)
	moduleID, oldKey, _ := readCompileManifestEntry(t, cacheRoot)
	moduleDir := compileArtifactDir(cacheRoot, moduleID)
	oldFiles := snapshotRegularFiles(t, moduleDir)
	if _, ok := oldFiles["artifacts.json"]; !ok {
		t.Fatal("six-route artifact snapshot omitted old artifacts.json stamp")
	}

	writeRoutes(7)
	fresh := probeServeAPICacheMCP(t, binary, modulePath, cacheRoot, true)
	assertSurface(wantNames(7), fresh)
	if fresh.callText != "42" {
		t.Fatalf("fresh f7 call text = %q, want 42; stderr:\n%s", fresh.callText, fresh.stderr)
	}
	gotModuleID, freshKey, _ := readCompileManifestEntry(t, cacheRoot)
	if gotModuleID != moduleID || freshKey == oldKey {
		t.Fatalf("source edit manifest = module %q key %q, want module %q and key different from %q", gotModuleID, freshKey, moduleID, oldKey)
	}

	// Restore the complete six-route artifact set, including its old stamp,
	// while deliberately retaining the seven-route manifest entry.
	restoreRegularFiles(t, moduleDir, oldFiles)
	_, retainedKey, _ := readCompileManifestEntry(t, cacheRoot)
	if retainedKey != freshKey {
		t.Fatalf("artifact restoration changed manifest key: got %q want %q", retainedKey, freshKey)
	}
	repaired := probeServeAPICacheMCP(t, binary, modulePath, cacheRoot, true)
	assertSurface(wantNames(7), repaired)
	if repaired.callText != "42" {
		t.Fatalf("repaired f7 call text = %q, want 42; stderr:\n%s", repaired.callText, repaired.stderr)
	}
	if !strings.Contains(repaired.stderr, "CACHE_INVALID module="+moduleID) ||
		!strings.Contains(repaired.stderr, "reason=ARTIFACT_INVALID") ||
		!strings.Contains(repaired.stderr, "artifacts.json") {
		t.Fatalf("old-stamp authorization diagnostic missing; stderr:\n%s", repaired.stderr)
	}

	// A second restart must consume the repaired, verified artifacts rather
	// than compiling again. The cache entry timestamp is written only when a
	// compilation is published, so byte equality pins the verified-hit path.
	_, _, repairedTimestamp := readCompileManifestEntry(t, cacheRoot)
	warm := probeServeAPICacheMCP(t, binary, modulePath, cacheRoot, true)
	assertSurface(wantNames(7), warm)
	if warm.callText != "42" {
		t.Fatalf("warm f7 call text = %q, want 42; stderr:\n%s", warm.callText, warm.stderr)
	}
	_, _, warmTimestamp := readCompileManifestEntry(t, cacheRoot)
	if warmTimestamp != repairedTimestamp {
		t.Fatalf("second restart republished cache entry: timestamp %q -> %q", repairedTimestamp, warmTimestamp)
	}
	if strings.Contains(warm.stderr, "CACHE_INVALID") {
		t.Fatalf("verified restart reported invalid cache; stderr:\n%s", warm.stderr)
	}

	// Poison exactly one payload with its old six-route bytes while retaining
	// the fresh seven-route stamp. This isolates per-blob hash verification.
	freshFiles := snapshotRegularFiles(t, moduleDir)
	if bytes.Equal(oldFiles["core.gob"], freshFiles["core.gob"]) {
		t.Fatal("instrument failure: six-route and seven-route core.gob are identical")
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "core.gob"), oldFiles["core.gob"], 0o644); err != nil {
		t.Fatal(err)
	}
	poisoned := probeServeAPICacheMCP(t, binary, modulePath, cacheRoot, true)
	assertSurface(wantNames(7), poisoned)
	if poisoned.callText != "42" {
		t.Fatalf("hash-repaired f7 call text = %q, want 42; stderr:\n%s", poisoned.callText, poisoned.stderr)
	}
	if !strings.Contains(poisoned.stderr, "CACHE_INVALID module="+moduleID) ||
		!strings.Contains(poisoned.stderr, "core.gob") ||
		!strings.Contains(poisoned.stderr, "reason=ARTIFACT_INVALID") {
		t.Fatalf("single-blob hash-failure diagnostic missing; stderr:\n%s", poisoned.stderr)
	}
}

// TestServeAPI_RouteIfaceMismatchFromCache is the M4 arm that
// TestServeAPI_DivergentCacheTools cannot be: every divergence that test
// constructs is caught by M1's per-blob hash verification before
// registerModule ever runs, so it passes unchanged with M4's production
// diagnostic reverted. This one hand-writes an artifact set that is
// hash-VALID and logically incomplete — iface.json loses the export for a
// function the source still declares with @route, and artifacts.json is
// re-stamped to match the new bytes — so M1 accepts it and the route/iface
// invariant is the only thing standing between the operator and a serve-api
// that silently publishes a short tool surface.
func TestServeAPI_RouteIfaceMismatchFromCache(t *testing.T) {
	binary := buildAilang(t)
	root := t.TempDir()
	cacheRoot := filepath.Join(root, "isolated-cache")
	modulePath := filepath.Join(root, "entry.ail")

	var source strings.Builder
	source.WriteString("module entry\n")
	for i := 1; i <= 7; i++ {
		fmt.Fprintf(&source, "\n@route(\"POST\", \"/f%d\")\nexport pure func f%d(x: float) -> float = x + %d.0\n", i, i, i)
	}
	if err := os.WriteFile(modulePath, []byte(source.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	// Warm the cache and pin the healthy control: seven tools, f7 callable.
	warm := probeServeAPICacheMCP(t, binary, modulePath, cacheRoot, true)
	want := []string{"f1", "f2", "f3", "f4", "f5", "f6", "f7"}
	if strings.Join(warm.tools, ",") != strings.Join(want, ",") {
		t.Fatalf("warm tools/list = %v, want exactly %v; stderr:\n%s", warm.tools, want, warm.stderr)
	}
	if warm.callText != "42" {
		t.Fatalf("warm f7 call text = %q, want 42; stderr:\n%s", warm.callText, warm.stderr)
	}

	moduleID, _, _ := readCompileManifestEntry(t, cacheRoot)
	moduleDir := compileArtifactDir(cacheRoot, moduleID)

	// Drop f7 from the cached interface while leaving the source untouched.
	ifacePath := filepath.Join(moduleDir, "iface.json")
	ifaceRaw, err := os.ReadFile(ifacePath)
	if err != nil {
		t.Fatal(err)
	}
	var ifaceDoc map[string]json.RawMessage
	if err := json.Unmarshal(ifaceRaw, &ifaceDoc); err != nil {
		t.Fatal(err)
	}
	var exports map[string]json.RawMessage
	if err := json.Unmarshal(ifaceDoc["exports"], &exports); err != nil {
		t.Fatal(err)
	}
	if _, ok := exports["f7"]; !ok {
		t.Fatalf("instrument failure: cached iface has no f7 export to remove: %s", ifaceRaw)
	}
	delete(exports, "f7")
	if exports["f6"] == nil {
		t.Fatal("instrument failure: removing f7 also removed the f6 control")
	}
	patchedExports, err := json.Marshal(exports)
	if err != nil {
		t.Fatal(err)
	}
	ifaceDoc["exports"] = patchedExports
	patchedIface, err := json.Marshal(ifaceDoc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ifacePath, patchedIface, 0o644); err != nil {
		t.Fatal(err)
	}

	// Re-stamp so the tampering is invisible to M1's hash verification. Without
	// this the run would fail as ARTIFACT_INVALID and prove nothing about M4.
	stampPath := filepath.Join(moduleDir, "artifacts.json")
	stampRaw, err := os.ReadFile(stampPath)
	if err != nil {
		t.Fatal(err)
	}
	var stamp struct {
		Version  string            `json:"version"`
		ModuleID string            `json:"module_id"`
		CacheKey string            `json:"cache_key"`
		SHA256   map[string]string `json:"sha256"`
	}
	if err := json.Unmarshal(stampRaw, &stamp); err != nil {
		t.Fatal(err)
	}
	previous, ok := stamp.SHA256["iface.json"]
	if !ok {
		t.Fatalf("instrument failure: stamp records no iface.json hash: %s", stampRaw)
	}
	sum := sha256.Sum256(patchedIface)
	updated := hex.EncodeToString(sum[:])
	if updated == previous {
		t.Fatal("instrument failure: patched iface.json hashes to the original value")
	}
	stamp.SHA256["iface.json"] = updated
	patchedStamp, err := json.MarshalIndent(stamp, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stampPath, patchedStamp, 0o644); err != nil {
		t.Fatal(err)
	}

	stderr, exitErr := runServeAPIMCPExpectingExit(t, binary, modulePath, cacheRoot)
	if exitErr == nil {
		t.Fatalf("serve-api started on a route/interface mismatch; stderr:\n%s", stderr)
	}
	for _, needle := range []string{"CACHE_ROUTE_IFACE_MISMATCH", "f7", "compile-clear"} {
		if !strings.Contains(stderr, needle) {
			t.Fatalf("mismatch diagnostic missing %q; stderr:\n%s", needle, stderr)
		}
	}
	// The failure must be the route invariant, not M1's hash gate: the
	// re-stamped artifacts are internally consistent by construction.
	if strings.Contains(stderr, "reason=ARTIFACT_INVALID") {
		t.Fatalf("re-stamped artifacts still failed hash verification; stderr:\n%s", stderr)
	}
}

// runServeAPIMCPExpectingExit starts serve-api with stdin closed and returns
// its stderr plus the process error. A healthy server blocks on stdin, so the
// bounded context is what stops it; a refusing server exits on its own.
func runServeAPIMCPExpectingExit(t *testing.T, binary, modulePath, cacheRoot string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "serve-api", "--mcp", "--routes-only", "--no-feedback-tool", modulePath)
	cmd.Env = appendEnvOverride(os.Environ(), "AILANG_CACHE_DIR", cacheRoot)
	cmd.Stdin = strings.NewReader("")
	var stderr, stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err := cmd.Run()
	if ctx.Err() != nil {
		t.Fatalf("serve-api did not exit within the bound; stderr:\n%s", stderr.String())
	}
	return stderr.String(), err
}

type mcpProbeResult struct {
	tools    []string
	callText string
	stderr   string
}

func probeServeAPICacheMCP(t *testing.T, binary, modulePath, cacheRoot string, callF7 bool) mcpProbeResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "serve-api", "--mcp", "--routes-only", "--no-feedback-tool", modulePath)
	cmd.Env = appendEnvOverride(os.Environ(), "AILANG_CACHE_DIR", cacheRoot)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	cleanup := func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
	cleaned := false
	defer func() {
		if !cleaned {
			cleanup()
		}
	}()

	scanner := bufio.NewScanner(stdout)
	send := func(value any) {
		t.Helper()
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fmt.Fprintf(stdin, "%s\n", data); err != nil {
			t.Fatalf("write MCP request: %v; stderr:\n%s", err, stderr.String())
		}
	}
	rpc := func(id int, method string, params any) json.RawMessage {
		t.Helper()
		send(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
		for scanner.Scan() {
			var response struct {
				ID     json.RawMessage `json:"id"`
				Result json.RawMessage `json:"result"`
				Error  json.RawMessage `json:"error"`
			}
			if err := json.Unmarshal(scanner.Bytes(), &response); err != nil {
				continue
			}
			var gotID int
			if json.Unmarshal(response.ID, &gotID) != nil || gotID != id {
				continue
			}
			if len(response.Error) != 0 && string(response.Error) != "null" {
				t.Fatalf("MCP %s error: %s; stderr:\n%s", method, response.Error, stderr.String())
			}
			return response.Result
		}
		if ctx.Err() != nil {
			t.Fatalf("MCP %s timeout: %v; stderr:\n%s", method, ctx.Err(), stderr.String())
		}
		t.Fatalf("MCP stdout closed during %s: %v; stderr:\n%s", method, scanner.Err(), stderr.String())
		return nil
	}

	rpc(1, "initialize", map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "cache-divergence-test", "version": "1"},
	})
	send(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
	var listed struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(rpc(2, "tools/list", map[string]any{}), &listed); err != nil {
		t.Fatalf("decode tools/list: %v", err)
	}
	result := mcpProbeResult{tools: make([]string, 0, len(listed.Tools))}
	for _, tool := range listed.Tools {
		result.tools = append(result.tools, tool.Name)
	}
	sort.Strings(result.tools)
	if callF7 {
		var called struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		}
		if err := json.Unmarshal(rpc(3, "tools/call", map[string]any{
			"name": "f7", "arguments": map[string]any{"x": 35},
		}), &called); err != nil {
			t.Fatalf("decode tools/call: %v", err)
		}
		if called.IsError || len(called.Content) != 1 || called.Content[0].Type != "text" {
			t.Fatalf("f7 call result = %#v", called)
		}
		result.callText = called.Content[0].Text
	}
	cleanup()
	cleaned = true
	result.stderr = stderr.String()
	return result
}

func appendEnvOverride(env []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(env)+1)
	for _, item := range env {
		if !strings.HasPrefix(item, prefix) {
			result = append(result, item)
		}
	}
	return append(result, prefix+value)
}

func readCompileManifestEntry(t *testing.T, cacheRoot string) (moduleID, cacheKey, timestamp string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(cacheRoot, "compile", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Entries map[string]struct {
			CacheKey  string `json:"cache_key"`
			Timestamp string `json:"timestamp"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Entries) != 1 {
		t.Fatalf("manifest entries = %d, want 1: %s", len(manifest.Entries), data)
	}
	for id, entry := range manifest.Entries {
		return id, entry.CacheKey, entry.Timestamp
	}
	panic("unreachable")
}

func compileArtifactDir(cacheRoot, moduleID string) string {
	name := strings.NewReplacer("/", "__", "\\", "__").Replace(moduleID)
	return filepath.Join(cacheRoot, "compile", "modules", name)
}

func snapshotRegularFiles(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	files := make(map[string][]byte)
	for _, entry := range entries {
		if entry.Type().IsRegular() {
			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			files[entry.Name()] = data
		}
	}
	return files
}

func restoreRegularFiles(t *testing.T, dir string, files map[string][]byte) {
	t.Helper()
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func probeServeAPIMCPTools(t *testing.T, binary, modulePath string, suppress bool) map[string]bool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	args := []string{"serve-api", "--mcp"}
	if suppress {
		args = append(args, "--no-feedback-tool")
	}
	args = append(args, modulePath)
	cmd := exec.CommandContext(ctx, binary, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	send := func(value any) {
		t.Helper()
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fmt.Fprintf(stdin, "%s\n", data); err != nil {
			t.Fatalf("write request: %v; stderr:\n%s", err, stderr.String())
		}
	}
	send(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "surface-test", "version": "1"},
		},
	})

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		var response struct {
			ID     json.RawMessage `json:"id"`
			Result struct {
				Tools []struct {
					Name string `json:"name"`
				} `json:"tools"`
			} `json:"result"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &response); err != nil {
			continue
		}
		var id int
		if err := json.Unmarshal(response.ID, &id); err != nil {
			continue
		}
		switch id {
		case 1:
			send(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
			send(map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": map[string]any{}})
		case 2:
			names := make(map[string]bool, len(response.Result.Tools))
			for _, tool := range response.Result.Tools {
				names[tool.Name] = true
			}
			return names
		}
	}
	if ctx.Err() != nil {
		t.Fatalf("MCP probe timeout: %v; stderr:\n%s", ctx.Err(), stderr.String())
	}
	t.Fatalf("MCP stdout closed before tools/list response: %v; stderr:\n%s", scanner.Err(), stderr.String())
	return nil
}
