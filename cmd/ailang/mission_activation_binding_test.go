package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/mission/activation"
	"github.com/sunholo-data/ailang/internal/testutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMissionRetainedBindingMissingReadOnly(t *testing.T) {
	if !activation.HostSupported() {
		t.Skip("local mission activation requires macOS or Linux host locking (activation/lock_other.go)")
	}
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	err := runMissionIteration(context.Background(), "status", []string{"docs", "--work-item", "w", "--activation", "missing"}, &bytes.Buffer{}, missionIterationDeps{})
	if err == nil {
		t.Fatal("missing activation accepted")
	}
	if _, err = os.Stat(filepath.Join(home, ".ailang")); !os.IsNotExist(err) {
		t.Fatalf("read-only lookup created state: %v", err)
	}
	for _, verb := range []string{"resume", "cancel", "iterate"} {
		args := []string{"docs", "--work-item", "w", "--activation", "op"}
		if verb == "iterate" {
			args = args[1:]
		}
		if _, err = parseIterationFlags(verb, args, &bytes.Buffer{}); err == nil {
			t.Fatalf("%s accepted retained write override", verb)
		}
	}
	if err = runMissionConfirmStopped(context.Background(), []string{"docs", "--work-item", "w", "--version", "1", "--attestation", "stopped", "--activation", "op"}, &bytes.Buffer{}, missionIterationDeps{}); missionErrorExitCode(err) != 2 {
		t.Fatalf("confirm accepted retained override: %v", err)
	}
	if _, err = resolveMissionReadBinding("/explicit/binding", "op", "docs", "w"); err == nil {
		t.Fatal("ambiguous placements accepted")
	}
}

func TestMissionRetainedStatusAfterRestore(t *testing.T) {
	if !activation.HostSupported() {
		t.Skip("local mission activation requires macOS or Linux host locking (activation/lock_other.go)")
	}
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	config := filepath.Join(home, ".config", "ailang")
	if err := os.MkdirAll(config, 0700); err != nil {
		t.Fatal(err)
	}
	bindingPath := filepath.Join(config, "mission-runtime.toml")
	prior := []byte("version=1\nstate_db=\"/nonexistent/baseline.db\"\nworkspace_root=\"/nonexistent/baseline\"\n")
	if err := os.WriteFile(bindingPath, prior, 0600); err != nil {
		t.Fatal(err)
	}
	db := filepath.Join(home, "retained.db")
	store, err := coordinator.NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	key := coordinator.MissionWorkItemKey{MissionID: "docs", WorkItemID: "w"}
	b := []byte(`{}`)
	if _, err = store.ClaimMissionWorkItem(context.Background(), coordinator.MissionWorkItemSpec{MissionWorkItemKey: key, SpecJSON: string(b), SpecDigest: fmt.Sprintf("%x", sha256.Sum256(b)), StageIDs: []string{"review"}, TimeoutSeconds: 60}, 30); err != nil {
		t.Fatal(err)
	}
	manager := activation.Manager{Dir: filepath.Join(home, ".ailang", "state", "mission-activations")}
	q := activation.Request{OperationID: "retained", MissionID: "docs", WorkItemID: "w", MarkerPath: filepath.Join(home, "disabled"), BindingPath: bindingPath, Binding: []byte(fmt.Sprintf("version=1\nstate_db=%q\nworkspace_root=%q\n", db, filepath.Join(home, "workspace")))}
	verified := func(context.Context, activation.Record) error { return nil }
	if _, err = manager.Activate(context.Background(), q, verified); err != nil {
		t.Fatal(err)
	}
	if _, err = manager.Recover(context.Background(), q.OperationID, verified); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err = runMissionIteration(context.Background(), "status", []string{"docs", "--work-item", "w", "--activation", "retained", "--json"}, &out, missionIterationDeps{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"work_item_id":"w"`) {
		t.Fatal(out.String())
	}
	if _, err = resolveMissionReadBinding("", "retained", "docs", "other"); err == nil {
		t.Fatal("unrelated work admitted through record")
	}
	after, err := os.ReadFile(bindingPath)
	if err != nil || !bytes.Equal(after, prior) {
		t.Fatalf("restored binding mutated %v", err)
	}
}
