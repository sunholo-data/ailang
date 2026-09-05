package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
