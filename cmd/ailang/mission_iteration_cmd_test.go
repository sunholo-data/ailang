package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/mission/iteration"
)

func iterationTestDeps(t *testing.T) (missionIterationDeps, string) {
	t.Helper()
	dir := t.TempDir()
	db := filepath.Join(dir, "runtime.sqlite")
	binding := filepath.Join(dir, "mission-runtime.toml")
	body := fmt.Sprintf("version=1\nstate_db=%q\nworkspace_root=%q\n", db, filepath.Join(dir, "workspaces"))
	if err := os.WriteFile(binding, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	return missionIterationDeps{BindingPath: binding}, db
}

func TestMissionIterationStatusMissingNeverCreates(t *testing.T) {
	deps, db := iterationTestDeps(t)
	err := runMissionIteration(context.Background(), "status", []string{"docs", "--work-item", "w", "--json"}, &bytes.Buffer{}, deps)
	if missionErrorExitCode(err) != 2 {
		t.Fatalf("expected config failure, got %v", err)
	}
	if _, err := os.Stat(db); !os.IsNotExist(err) {
		t.Fatalf("status created DB: %v", err)
	}
}

func TestMissionIterationFlagsFailClosed(t *testing.T) {
	for _, tc := range []struct {
		verb string
		args []string
	}{
		{"iterate", []string{"--state-db", "elsewhere", "--work-item", "file"}},
		{"iterate", []string{"--work-item", "a", "--work-item", "b"}},
		{"iterate", []string{"--work-item", "a", "extra"}},
		{"status", []string{"docs", "--work-item", "a", "extra"}},
		{"status", []string{"docs", "--work-item", "a", "--dry-run"}},
		{"resume", []string{"docs", "--work-item", "a", "--json"}},
		{"cancel", []string{"docs", "--work-item", "a"}},
		{"cancel", []string{"docs", "--work-item", "a", "--version", "0"}},
	} {
		t.Run(tc.verb+strings.Join(tc.args, "_"), func(t *testing.T) {
			err := runMissionIteration(context.Background(), tc.verb, tc.args, &bytes.Buffer{}, missionIterationDeps{})
			if missionErrorExitCode(err) != 2 {
				t.Fatalf("accepted invalid flags: %v", err)
			}
		})
	}
}

func TestMissionIterationStatusRedactsSnapshotAndCancelVersion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mission iteration execution requires macOS/Linux: Windows descendant cleanup and receipt sync are unsupported")
	}
	deps, db := iterationTestDeps(t)
	store, err := coordinator.NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	snapshot := iteration.Snapshot{Spec: iteration.Spec{Brief: "PRIVATE PROMPT", MissionID: "docs", WorkItemID: "w"}}
	b, _ := json.Marshal(snapshot)
	key := coordinator.MissionWorkItemKey{MissionID: "docs", WorkItemID: "w"}
	item, err := store.ClaimMissionWorkItem(context.Background(), coordinator.MissionWorkItemSpec{MissionWorkItemKey: key, SpecJSON: string(b), SpecDigest: fmt.Sprintf("%x", sha256.Sum256(b)), StageIDs: []string{"write"}, TimeoutSeconds: 60}, 30)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := runMissionIteration(context.Background(), "status", []string{"docs", "--work-item", "w", "--json"}, &out, deps); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "PRIVATE") || strings.Contains(out.String(), item.OwnerToken) || strings.Contains(out.String(), "spec_json") {
		t.Fatalf("status leaked snapshot: %s", out.String())
	}
	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["phase"] != "ready" || got["stage"] != "write" {
		t.Fatalf("bad status %v", got)
	}
	args := []string{"docs", "--work-item", "w", "--version", fmt.Sprint(item.Version + 1)}
	if err := runMissionIteration(context.Background(), "cancel", args, &bytes.Buffer{}, deps); err == nil {
		t.Fatal("stale cancellation accepted")
	}
	args[len(args)-1] = fmt.Sprint(item.Version)
	if err := runMissionIteration(context.Background(), "cancel", args, &bytes.Buffer{}, deps); missionErrorExitCode(err) != 130 {
		t.Fatalf("cancellation: %v", err)
	}
	if err := runMissionIteration(context.Background(), "status", []string{"docs", "--work-item", "w"}, &bytes.Buffer{}, deps); err != nil {
		t.Fatalf("terminal status must be read success: %v", err)
	}
}
