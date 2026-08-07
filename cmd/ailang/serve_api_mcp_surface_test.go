package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
