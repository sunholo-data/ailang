package coordinator

import (
	"context"
	"io"
	"log"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestWorkerHeartbeat_Fields(t *testing.T) {
	// Sanity: the heartbeat struct exposes the fields the design doc names.
	// The compile-time existence of these fields is the test (build will fail if missing).
	hb := WorkerHeartbeat{
		HostID:      "studio.eval-rig",
		Tags:        []string{"ollama:gemma4-26b-ailang", "gpu:m4-max"},
		ActiveTasks: 1,
		LastSeen:    time.Now(),
		Version:     "v0.24.0",
		UptimeSecs:  3600,
		Type:        "bare-metal",
	}
	if hb.HostID == "" {
		t.Error("HostID should be set")
	}
	if len(hb.Tags) == 0 {
		t.Error("Tags should be set")
	}
}

func TestMemoryHeartbeatStore_PutAndList(t *testing.T) {
	store := NewMemoryHeartbeatStore()

	now := time.Now()
	hb := WorkerHeartbeat{
		HostID:      "studio.eval-rig",
		Tags:        []string{"ollama:gemma4-26b-ailang"},
		ActiveTasks: 0,
		LastSeen:    now,
		Version:     "v0.24.0",
		UptimeSecs:  100,
		Type:        "bare-metal",
	}
	if err := store.Put(context.Background(), hb); err != nil {
		t.Fatalf("Put error: %v", err)
	}

	listed, err := store.List(context.Background(), 5*time.Minute)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("List returned %d, want 1", len(listed))
	}
	if listed[0].HostID != "studio.eval-rig" {
		t.Errorf("HostID = %q, want studio.eval-rig", listed[0].HostID)
	}
	if len(listed[0].Tags) != 1 || listed[0].Tags[0] != "ollama:gemma4-26b-ailang" {
		t.Errorf("Tags = %v, want [ollama:gemma4-26b-ailang]", listed[0].Tags)
	}
}

func TestMemoryHeartbeatStore_StaleEntriesExcluded(t *testing.T) {
	store := NewMemoryHeartbeatStore()

	fresh := WorkerHeartbeat{HostID: "fresh", LastSeen: time.Now(), Type: "bare-metal"}
	stale := WorkerHeartbeat{HostID: "stale", LastSeen: time.Now().Add(-10 * time.Minute), Type: "bare-metal"}
	if err := store.Put(context.Background(), fresh); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(context.Background(), stale); err != nil {
		t.Fatal(err)
	}

	listed, err := store.List(context.Background(), 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("List returned %d, want only fresh (1)", len(listed))
	}
	if listed[0].HostID != "fresh" {
		t.Errorf("HostID = %q, want fresh", listed[0].HostID)
	}

	// With a longer maxAge, both should appear.
	all, err := store.List(context.Background(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Errorf("List returned %d with hour maxAge, want 2", len(all))
	}
}

func TestMemoryHeartbeatStore_PutOverwritesByHostID(t *testing.T) {
	store := NewMemoryHeartbeatStore()

	first := WorkerHeartbeat{HostID: "studio", LastSeen: time.Now(), ActiveTasks: 1, Type: "bare-metal"}
	second := WorkerHeartbeat{HostID: "studio", LastSeen: time.Now(), ActiveTasks: 5, Type: "bare-metal"}

	_ = store.Put(context.Background(), first)
	_ = store.Put(context.Background(), second)

	listed, _ := store.List(context.Background(), time.Hour)
	if len(listed) != 1 {
		t.Fatalf("Put on same HostID should overwrite, not append; got %d entries", len(listed))
	}
	if listed[0].ActiveTasks != 5 {
		t.Errorf("ActiveTasks = %d, want 5 (latest Put)", listed[0].ActiveTasks)
	}
}

func TestMemoryHeartbeatStore_MultipleHostsListedSorted(t *testing.T) {
	store := NewMemoryHeartbeatStore()

	hosts := []string{"laptop", "studio", "cloud-run"}
	now := time.Now()
	for _, h := range hosts {
		_ = store.Put(context.Background(), WorkerHeartbeat{HostID: h, LastSeen: now, Type: "bare-metal"})
	}

	listed, _ := store.List(context.Background(), time.Hour)
	if len(listed) != 3 {
		t.Fatalf("expected 3 hosts; got %d", len(listed))
	}
	// List should be sorted by HostID for stable display.
	got := make([]string, len(listed))
	for i, hb := range listed {
		got[i] = hb.HostID
	}
	want := []string{"cloud-run", "laptop", "studio"}
	if !sort.StringsAreSorted(got) {
		t.Errorf("List should return entries sorted by HostID; got %v", got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d] HostID = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestHeartbeatWriter_StartTickStop(t *testing.T) {
	// HeartbeatWriter should write at the configured interval and stop cleanly
	// when context is cancelled.
	store := NewMemoryHeartbeatStore()

	source := &fakeHeartbeatSource{
		hostID: "studio.eval-rig",
		tags:   []string{"ollama:gemma4-26b-ailang"},
	}
	w := NewHeartbeatWriter(store, source, 30*time.Millisecond, newSilentHBLogger())
	ctx, cancel := context.WithCancel(context.Background())
	w.Start(ctx)

	// Wait long enough for at least 2 ticks.
	time.Sleep(120 * time.Millisecond)

	// Stop the writer.
	cancel()
	w.Wait()

	listed, _ := store.List(context.Background(), time.Hour)
	if len(listed) != 1 {
		t.Fatalf("expected 1 host registered; got %d", len(listed))
	}
	if listed[0].HostID != "studio.eval-rig" {
		t.Errorf("HostID = %q, want studio.eval-rig", listed[0].HostID)
	}
}

func TestHeartbeatWriter_DoubleStartIsNoOp(t *testing.T) {
	store := NewMemoryHeartbeatStore()
	source := &fakeHeartbeatSource{hostID: "h", tags: nil}
	w := NewHeartbeatWriter(store, source, 50*time.Millisecond, newSilentHBLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Start(ctx)
	w.Start(ctx) // second Start should be a no-op, not panic or leak a goroutine.
	cancel()
	w.Wait()
}

// fakeHeartbeatSource gives HeartbeatWriter a stable host/tag/task snapshot.
type fakeHeartbeatSource struct {
	mu          sync.Mutex
	hostID      string
	tags        []string
	activeTasks int
}

func (f *fakeHeartbeatSource) Snapshot() WorkerHeartbeat {
	f.mu.Lock()
	defer f.mu.Unlock()
	return WorkerHeartbeat{
		HostID:      f.hostID,
		Tags:        append([]string{}, f.tags...),
		ActiveTasks: f.activeTasks,
		LastSeen:    time.Now(),
		Version:     "test",
		UptimeSecs:  0,
		Type:        "bare-metal",
	}
}

func newSilentHBLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}

func TestHeartbeatWriter_LazySnapshot_PicksUpLateAgent(t *testing.T) {
	// Reproduces the M3-wiring bug discovered during integration testing:
	// daemon's startHeartbeat runs BEFORE Run() populates the agent
	// registry. The source MUST re-snapshot at each tick so tags become
	// visible once agents are registered.
	store := NewMemoryHeartbeatStore()
	src := &mutableSource{initial: WorkerHeartbeat{HostID: "before", LastSeen: time.Now()}}
	w := NewHeartbeatWriter(store, src, 30*time.Millisecond, newSilentHBLogger())
	ctx, cancel := context.WithCancel(context.Background())
	w.Start(ctx)

	// Wait for the initial write to land.
	time.Sleep(20 * time.Millisecond)

	// Simulate "agent registry just got populated" by changing the source's
	// returned value mid-flight.
	src.set(WorkerHeartbeat{
		HostID:   "studio.eval-rig",
		Tags:     []string{"ollama:gemma4-26b-ailang"},
		LastSeen: time.Now(),
		Type:     "bare-metal",
	})

	// Wait for at least one more tick at the 30ms interval.
	time.Sleep(80 * time.Millisecond)
	cancel()
	w.Wait()

	listed, _ := store.List(context.Background(), time.Hour)
	if len(listed) == 0 {
		t.Fatal("no heartbeats written; lazy snapshot path is broken")
	}
	// Last write wins (same HostID overwrite) — but here we changed HostID
	// AND wrote both, so 2 entries should be present.
	foundStudio := false
	for _, hb := range listed {
		if hb.HostID == "studio.eval-rig" && len(hb.Tags) > 0 {
			foundStudio = true
			break
		}
	}
	if !foundStudio {
		t.Errorf("expected studio.eval-rig heartbeat with tags after late agent registration; got %v", listed)
	}
}

// mutableSource lets the test swap the snapshot value mid-flight to simulate
// the agent registry being populated after the writer has already started.
type mutableSource struct {
	mu      sync.Mutex
	initial WorkerHeartbeat
	current *WorkerHeartbeat
}

func (s *mutableSource) Snapshot() WorkerHeartbeat {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current != nil {
		return *s.current
	}
	return s.initial
}

func (s *mutableSource) set(hb WorkerHeartbeat) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = &hb
}
