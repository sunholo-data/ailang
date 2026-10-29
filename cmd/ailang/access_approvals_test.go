package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestAccessApprovalsJSONContract(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "runtime")
	args := []string{"--state-dir", dir, "--source", "daneel-local", "request"}
	stdin := `{"owner":"mark","worker":"machine:daneel","task":"task-1","attempt":"attempt-1","operation":"dummy","resource_alias":"canary","purpose":"review this","arguments":{"message":"canary"},"policy_version":"1","version":1}`
	var out bytes.Buffer
	if err := runAccessApprovals(args, bytes.NewBufferString(stdin), &out); err != nil {
		t.Fatal(err)
	}
	var record map[string]any
	if err := json.Unmarshal(out.Bytes(), &record); err != nil {
		t.Fatalf("not a JSON record: %s", out.String())
	}
	if record["source"] != "daneel-local" || record["decision"] != "pending" {
		t.Fatal(record)
	}
	for _, bad := range []string{stdin + ` {}`, `{"owner":"mark","shell":"true"}`, `{broken}`} {
		out.Reset()
		if err := runAccessApprovals(args, bytes.NewBufferString(bad), &out); err == nil {
			t.Fatalf("invalid protocol accepted: %s", bad)
		}
		if out.Len() != 0 {
			t.Fatal("error emitted success JSON")
		}
	}
	out.Reset()
	if err := runAccessApprovals([]string{"list"}, bytes.NewBufferString(`{"owner":"mark"}`), &out); err == nil {
		t.Fatal("implicit source accepted")
	}
	if _, ok := lookupCommand("access-approvals"); !ok {
		t.Fatal("command is not registered")
	}
}
