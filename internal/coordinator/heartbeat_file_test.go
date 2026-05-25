package coordinator

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

func TestFileHeartbeatStore_MissingFile_EmptyList(t *testing.T) {
	tmp := t.TempDir()
	s := NewFileHeartbeatStore(filepath.Join(tmp, "hb.json"))
	got, err := s.List(context.Background(), time.Hour)
	if err != nil {
		t.Fatalf("List on missing file returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("List on missing file returned %d entries, want 0", len(got))
	}
}

func TestFileHeartbeatStore_PutCreatesFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "hb.json")
	s := NewFileHeartbeatStore(path)

	hb := WorkerHeartbeat{HostID: "studio", LastSeen: time.Now(), Tags: []string{"ollama:gemma4"}, Type: "bare-metal"}
	if err := s.Put(context.Background(), hb); err != nil {
		t.Fatalf("Put error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file at %s after Put: %v", path, err)
	}
	got, _ := s.List(context.Background(), time.Hour)
	if len(got) != 1 || got[0].HostID != "studio" {
		t.Errorf("List = %v, want [studio]", got)
	}
}

func TestFileHeartbeatStore_PutOverwritesByHostID(t *testing.T) {
	tmp := t.TempDir()
	s := NewFileHeartbeatStore(filepath.Join(tmp, "hb.json"))

	older := WorkerHeartbeat{HostID: "studio", ActiveTasks: 1, LastSeen: time.Now(), Type: "bare-metal"}
	newer := WorkerHeartbeat{HostID: "studio", ActiveTasks: 5, LastSeen: time.Now(), Type: "bare-metal"}
	_ = s.Put(context.Background(), older)
	_ = s.Put(context.Background(), newer)

	got, _ := s.List(context.Background(), time.Hour)
	if len(got) != 1 {
		t.Fatalf("List len = %d, want 1 (same host_id should overwrite)", len(got))
	}
	if got[0].ActiveTasks != 5 {
		t.Errorf("ActiveTasks = %d, want 5 (latest write wins)", got[0].ActiveTasks)
	}
}

func TestFileHeartbeatStore_StaleEntriesExcluded(t *testing.T) {
	tmp := t.TempDir()
	s := NewFileHeartbeatStore(filepath.Join(tmp, "hb.json"))

	fresh := WorkerHeartbeat{HostID: "fresh", LastSeen: time.Now(), Type: "bare-metal"}
	stale := WorkerHeartbeat{HostID: "stale", LastSeen: time.Now().Add(-10 * time.Minute), Type: "bare-metal"}
	_ = s.Put(context.Background(), fresh)
	_ = s.Put(context.Background(), stale)

	got, _ := s.List(context.Background(), 5*time.Minute)
	if len(got) != 1 {
		t.Fatalf("List len = %d, want 1 (stale excluded)", len(got))
	}
	if got[0].HostID != "fresh" {
		t.Errorf("HostID = %q, want fresh", got[0].HostID)
	}
}

func TestFileHeartbeatStore_MultipleHostsSortedByID(t *testing.T) {
	tmp := t.TempDir()
	s := NewFileHeartbeatStore(filepath.Join(tmp, "hb.json"))

	now := time.Now()
	for _, h := range []string{"laptop", "studio", "cloud-run"} {
		_ = s.Put(context.Background(), WorkerHeartbeat{HostID: h, LastSeen: now, Type: "bare-metal"})
	}

	got, _ := s.List(context.Background(), time.Hour)
	if len(got) != 3 {
		t.Fatalf("List len = %d, want 3", len(got))
	}
	ids := make([]string, len(got))
	for i, hb := range got {
		ids[i] = hb.HostID
	}
	if !sort.StringsAreSorted(ids) {
		t.Errorf("List should sort by HostID; got %v", ids)
	}
}

func TestFileHeartbeatStore_CrossProcessVisibility(t *testing.T) {
	// Two stores pointing at the same file simulate the daemon-writes / CLI-reads
	// pattern. Whatever the writer Puts must be visible to a separate reader.
	tmp := t.TempDir()
	path := filepath.Join(tmp, "hb.json")
	writer := NewFileHeartbeatStore(path)
	reader := NewFileHeartbeatStore(path) // separate instance, same path

	hb := WorkerHeartbeat{HostID: "studio", LastSeen: time.Now(), Tags: []string{"ollama:*"}, Type: "bare-metal"}
	if err := writer.Put(context.Background(), hb); err != nil {
		t.Fatalf("writer.Put: %v", err)
	}

	got, err := reader.List(context.Background(), time.Hour)
	if err != nil {
		t.Fatalf("reader.List: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("reader saw %d entries, want 1 (writer's heartbeat should be visible)", len(got))
	}
	if got[0].HostID != "studio" || len(got[0].Tags) != 1 {
		t.Errorf("reader saw %+v, want studio with [ollama:*] tags", got[0])
	}
}

func TestDefaultHeartbeatPath(t *testing.T) {
	if got := DefaultHeartbeatPath("/tmp/state"); got != "/tmp/state/worker_heartbeats.json" {
		t.Errorf("DefaultHeartbeatPath = %q, want /tmp/state/worker_heartbeats.json", got)
	}
	// Empty stateDir falls back to ~/.ailang/state — just verify suffix.
	got := DefaultHeartbeatPath("")
	if filepath.Base(got) != "worker_heartbeats.json" {
		t.Errorf("DefaultHeartbeatPath('') basename = %q, want worker_heartbeats.json", filepath.Base(got))
	}
}
