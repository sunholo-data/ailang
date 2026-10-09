package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstalledAuthPathHonorsProfile(t *testing.T) {
	profile := t.TempDir()
	t.Setenv("CODEX_HOME", profile)
	got, err := InstalledAuthPath()
	if err != nil || got != filepath.Join(profile, "auth.json") {
		t.Fatalf("profile path=%q error=%v", got, err)
	}
}

func TestDaemonProtocolMapsExecEvents(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	seen := make(chan []string, 1)
	go func() {
		defer server.Close()
		dec := json.NewDecoder(server)
		enc := json.NewEncoder(server)
		var methods []string
		for {
			var req map[string]any
			if dec.Decode(&req) != nil {
				seen <- methods
				return
			}
			m, _ := req["method"].(string)
			methods = append(methods, m)
			switch m {
			case "initialize":
				enc.Encode(map[string]any{"id": req["id"], "result": map[string]any{}})
			case "initialized":
			case "thread/start":
				enc.Encode(map[string]any{"id": req["id"], "result": map[string]any{"thread": map[string]any{"id": "thread-1"}}})
			case "turn/start":
				enc.Encode(map[string]any{"id": req["id"], "result": map[string]any{"turn": map[string]any{"id": "turn-1"}}})
				enc.Encode(map[string]any{"method": "item/completed", "params": map[string]any{"threadId": "thread-1", "turnId": "turn-1", "item": map[string]any{"type": "commandExecution", "id": "cmd", "command": "echo done", "aggregatedOutput": "done", "exitCode": 0}}})
				enc.Encode(map[string]any{"method": "item/completed", "params": map[string]any{"threadId": "thread-1", "turnId": "turn-1", "item": map[string]any{"type": "agentMessage", "id": "item-1", "text": "done"}}})
				enc.Encode(map[string]any{"method": "turn/completed", "params": map[string]any{"threadId": "thread-1", "turn": map[string]any{"id": "turn-1", "status": "completed"}}})
			case "thread/unsubscribe":
				enc.Encode(map[string]any{"id": req["id"], "result": map[string]any{}})
				seen <- methods
				return
			}
		}
	}()
	var out bytes.Buffer
	err := runDaemonProtocol(context.Background(), client, client, DaemonExecOptions{Model: "model", Directive: "do task", Sandbox: "workspace-write", ApprovalPolicy: "never"}, &out)
	if err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(&out)
	var events []map[string]any
	for {
		var e map[string]any
		if err := dec.Decode(&e); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		events = append(events, e)
	}
	if len(events) != 5 || events[0]["type"] != "thread.started" || events[3]["item"].(map[string]any)["text"] != "done" || events[4]["type"] != "turn.completed" {
		t.Fatalf("events=%v", events)
	}
	if events[2]["item"].(map[string]any)["aggregated_output"] != "done" || events[2]["item"].(map[string]any)["exit_code"] != float64(0) {
		t.Fatalf("command mapping=%v", events[2])
	}
	methods := <-seen
	if methods[len(methods)-1] != "thread/unsubscribe" {
		t.Fatalf("methods=%v", methods)
	}
}

func TestDaemonProtocolCancellationInterruptsOwnedTurn(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	interrupted := make(chan bool, 1)
	go func() {
		defer server.Close()
		dec := json.NewDecoder(server)
		enc := json.NewEncoder(server)
		for {
			var req map[string]any
			if dec.Decode(&req) != nil {
				return
			}
			switch req["method"] {
			case "initialize":
				enc.Encode(map[string]any{"id": req["id"], "result": map[string]any{}})
			case "thread/start":
				enc.Encode(map[string]any{"id": req["id"], "result": map[string]any{"thread": map[string]any{"id": "owned"}}})
			case "turn/start":
				enc.Encode(map[string]any{"id": req["id"], "result": map[string]any{"turn": map[string]any{"id": "turn"}}})
				cancel()
			case "turn/interrupt":
				p := req["params"].(map[string]any)
				interrupted <- p["threadId"] == "owned" && p["turnId"] == "turn"
				enc.Encode(map[string]any{"id": req["id"], "result": map[string]any{}})
				enc.Encode(map[string]any{"method": "turn/completed", "params": map[string]any{"threadId": "owned", "turn": map[string]any{"id": "turn", "status": "interrupted"}}})
			case "thread/unsubscribe":
				enc.Encode(map[string]any{"id": req["id"], "result": map[string]any{}})
				return
			}
		}
	}()
	err := runDaemonProtocol(ctx, client, client, DaemonExecOptions{Model: "model", Directive: "task"}, io.Discard)
	if err != context.Canceled {
		t.Fatalf("cancel=%v", err)
	}
	if !<-interrupted {
		t.Fatal("interrupted wrong owner")
	}
}

func TestDaemonExecMissingAuthRefusesBeforeConnecting(t *testing.T) {
	err := RunDaemonExec(context.Background(), "codex", DaemonExecOptions{Model: "model", Directive: "task", Env: []string{"CODEX_HOME=" + t.TempDir()}}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "login is missing") {
		t.Fatalf("missing auth=%v", err)
	}
}

func TestDaemonConfigIsolatesStartupAndPreservesNestedOwner(t *testing.T) {
	cfg := daemonIsolatedConfig(map[string]any{"shell_environment_policy.inherit": "all", "shell_environment_policy.include_only": []string{"OLD"}, "shell_environment_policy.set.MISSION_ROLE": "executor"}, []string{"CODEX_HOME=/mission", "AILANG_CODEX_RUNTIME=daemon", "PATH=/bin"})
	set := cfg["shell_environment_policy.set"].(map[string]any)
	if cfg["shell_environment_policy.inherit"] != "none" || set["CODEX_HOME"] != "/mission" || set["AILANG_CODEX_RUNTIME"] != "daemon" || set["MISSION_ROLE"] != "executor" {
		t.Fatalf("config=%v", cfg)
	}
	if _, ok := cfg["shell_environment_policy.set.MISSION_ROLE"]; ok {
		t.Fatal("flat override bypasses exact set")
	}
}

func TestDaemonProtocolAmbiguousTurnStartQuarantines(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	go func() {
		defer server.Close()
		dec := json.NewDecoder(server)
		enc := json.NewEncoder(server)
		for {
			var req map[string]any
			if dec.Decode(&req) != nil {
				return
			}
			switch req["method"] {
			case "initialize":
				enc.Encode(map[string]any{"id": req["id"], "result": map[string]any{}})
			case "thread/start":
				enc.Encode(map[string]any{"id": req["id"], "result": map[string]any{"thread": map[string]any{"id": "owned"}}})
			case "turn/start":
				return // accepted turn might exist but response was lost
			}
		}
	}()
	err := runDaemonProtocol(context.Background(), client, client, DaemonExecOptions{Model: "model", Directive: "task"}, io.Discard)
	if !errors.Is(err, ErrProcessTerminationUnconfirmed) {
		t.Fatalf("ambiguous turn=%v", err)
	}
}
