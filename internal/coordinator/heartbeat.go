package coordinator

import (
	"context"
	"log"
	"sort"
	"sync"
	"time"
)

// WorkerHeartbeat is the per-host status record written periodically by the
// coordinator daemon and surfaced by `ailang coordinator workers list`
// (M-COORD-MULTI-HOST-WORKERS, v0.24.0).
type WorkerHeartbeat struct {
	HostID      string    `json:"host_id"`
	Tags        []string  `json:"tags,omitempty"`
	ActiveTasks int       `json:"active_tasks"`
	LastSeen    time.Time `json:"last_seen"`
	Version     string    `json:"version,omitempty"`
	UptimeSecs  int64     `json:"uptime_secs,omitempty"`
	// Type categorises the worker for unified listing. Values:
	//   "bare-metal" — long-running coordinator on a developer machine / rig host
	//   "cloud-run"  — ephemeral Cloud Run Job, reconstructed from task history
	Type string `json:"type,omitempty"`
}

// HeartbeatStore is the persistence interface for worker heartbeats. Production
// backends are Firestore-backed (so `workers list` from anywhere sees every
// host); in-memory is for tests and local-only setups.
type HeartbeatStore interface {
	Put(ctx context.Context, hb WorkerHeartbeat) error
	List(ctx context.Context, maxAge time.Duration) ([]WorkerHeartbeat, error)
}

// HeartbeatSource snapshots the current host state for the writer to publish.
// Implementations should make Snapshot cheap and lock-clean.
type HeartbeatSource interface {
	Snapshot() WorkerHeartbeat
}

// MemoryHeartbeatStore is an in-process HeartbeatStore. Useful for tests and
// for `ailang coordinator workers list` when no Firestore backend is wired —
// you'll only see the local host, but at least the surface area works.
type MemoryHeartbeatStore struct {
	mu sync.Mutex
	// keyed by HostID; latest Put wins.
	entries map[string]WorkerHeartbeat
}

func NewMemoryHeartbeatStore() *MemoryHeartbeatStore {
	return &MemoryHeartbeatStore{entries: make(map[string]WorkerHeartbeat)}
}

func (m *MemoryHeartbeatStore) Put(_ context.Context, hb WorkerHeartbeat) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[hb.HostID] = hb
	return nil
}

func (m *MemoryHeartbeatStore) List(_ context.Context, maxAge time.Duration) ([]WorkerHeartbeat, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	out := make([]WorkerHeartbeat, 0, len(m.entries))
	for _, hb := range m.entries {
		// maxAge == 0 means "no filter"; otherwise drop stale entries.
		if maxAge > 0 && now.Sub(hb.LastSeen) > maxAge {
			continue
		}
		out = append(out, hb)
	}
	// Stable, deterministic order for display + tests: by HostID ascending.
	sort.Slice(out, func(i, j int) bool { return out[i].HostID < out[j].HostID })
	return out, nil
}

// HeartbeatWriter runs a background goroutine that calls source.Snapshot()
// at the configured interval and writes the result to store. Start() is
// safe to call multiple times — subsequent calls are no-ops. The writer
// stops when the context passed to Start is cancelled; Wait() blocks until
// the goroutine has fully exited.
type HeartbeatWriter struct {
	store    HeartbeatStore
	source   HeartbeatSource
	interval time.Duration
	logger   *log.Logger

	mu      sync.Mutex
	running bool
	done    chan struct{}
}

func NewHeartbeatWriter(store HeartbeatStore, source HeartbeatSource, interval time.Duration, logger *log.Logger) *HeartbeatWriter {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	return &HeartbeatWriter{
		store:    store,
		source:   source,
		interval: interval,
		logger:   logger,
	}
}

func (w *HeartbeatWriter) Start(ctx context.Context) {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.done = make(chan struct{})
	w.mu.Unlock()

	go func() {
		defer close(w.done)

		// Initial heartbeat without waiting a full tick — makes the host
		// visible in `workers list` immediately after the daemon starts.
		w.writeOnce(ctx)

		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.writeOnce(ctx)
			}
		}
	}()
}

func (w *HeartbeatWriter) writeOnce(ctx context.Context) {
	hb := w.source.Snapshot()
	if err := w.store.Put(ctx, hb); err != nil && w.logger != nil {
		w.logger.Printf("heartbeat: write failed for host=%s: %v", hb.HostID, err)
	}
}

// Wait blocks until the background goroutine has exited (after the context
// passed to Start was cancelled). If Start has not been called, Wait is a no-op.
func (w *HeartbeatWriter) Wait() {
	w.mu.Lock()
	done := w.done
	w.mu.Unlock()
	if done == nil {
		return
	}
	<-done
}
