package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Each call launches a fresh CLI dispatcher process against the same isolated
// local authority. No live state, vault or environment token is consulted.
func TestAccessApprovalsProcess(t *testing.T) {
	for i, arg := range os.Args {
		if arg == "--access-approval-child" {
			dispatchCommand("access-approvals", os.Args[i+1:])
			os.Exit(0)
		}
	}
	dir := filepath.Join(t.TempDir(), "authority")
	call := func(command string, in any, wantOK bool) map[string]any {
		t.Helper()
		data, _ := json.Marshal(in)
		cmd := exec.Command(os.Args[0], "-test.run=^TestAccessApprovalsProcess$", "--", "--access-approval-child", "--state-dir", dir, "--source", "daneel-local", command)
		cmd.Stdin = bytes.NewReader(data)
		cmd.Env = append(os.Environ(), "AILANG_STORAGE_COORDINATOR=local")
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		var result map[string]any
		if wantOK {
			if err != nil || stderr.Len() != 0 {
				t.Fatalf("%s failed: %v %s", command, err, stderr.String())
			}
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatalf("%s output: %s", command, stdout.String())
			}
		} else {
			if err == nil || stdout.Len() != 0 {
				t.Fatalf("%s unexpectedly succeeded", command)
			}
			if err := json.Unmarshal(stderr.Bytes(), &result); err != nil {
				t.Fatalf("error not JSON: %s", stderr.String())
			}
		}
		return result
	}
	r := call("request", map[string]any{"owner": "mark", "worker": "machine:daneel", "task": "process-fixture", "attempt": "1", "operation": "dummy", "resource_alias": "canary", "purpose": "process fixture", "policy_version": "1", "version": 1, "arguments": map[string]any{"message": "process canary"}}, true)
	id := r["id"]
	call("decide", map[string]any{"id": id, "actor": "mark", "version": 1, "decision": "approved", "idempotency_key": "d"}, true)
	bind := map[string]any{"id": id, "worker": r["worker"], "task": r["task"], "attempt": r["attempt"], "operation": r["operation"], "arguments_digest": r["arguments_digest"], "policy_version": r["policy_version"], "version": 1, "idempotency_key": "c"}
	r = call("claim", bind, true)
	bind["claim_id"] = r["execution"].(map[string]any)["claim_id"]
	r = call("execute", bind, true)
	revision := r["revision"]
	r = call("execute", bind, true)
	if r["revision"] != revision {
		t.Fatal("process replay changed receipt")
	}
	receipt := call("receipt", map[string]any{"id": id}, true)
	if receipt["execution"].(map[string]any)["state"] != "completed" {
		t.Fatal(receipt)
	}
	call("decide", map[string]any{"id": id, "actor": "imposter", "version": 1, "decision": "approved", "idempotency_key": "d"}, false)
}
