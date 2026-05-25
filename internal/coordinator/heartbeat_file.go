package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// FileHeartbeatStore persists worker heartbeats to a single JSON file on
// disk. This is the single-host bridge between the daemon's in-process
// heartbeat goroutine and the CLI's `workers list` lookup — Firestore is on
// the v0.25 roadmap for cross-host visibility, but this gets the surface
// working today on a single machine.
//
// File format: `{"host_id": WorkerHeartbeat, ...}` — keyed by host_id so
// repeated writes from the same host overwrite the previous entry. The
// file is rewritten atomically (write to .tmp, rename) so a concurrent
// reader never sees a half-written JSON document.
//
// Concurrency: a single sync.Mutex guards the file. Multiple daemons on
// the same host pointing at the same file (an unusual setup) would
// serialize at the OS level via the rename; in normal single-daemon use
// this is uncontested.
type FileHeartbeatStore struct {
	path string
	mu   sync.Mutex
}

// NewFileHeartbeatStore returns a heartbeat store backed by the given JSON
// file. If the file does not yet exist, the first Put creates it. Parent
// directory must already exist; missing parents are a configuration error
// and surfaced from Put.
func NewFileHeartbeatStore(path string) *FileHeartbeatStore {
	return &FileHeartbeatStore{path: path}
}

func (s *FileHeartbeatStore) Put(_ context.Context, hb WorkerHeartbeat) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.readLocked()
	if err != nil {
		return err
	}
	if entries == nil {
		entries = make(map[string]WorkerHeartbeat)
	}
	entries[hb.HostID] = hb

	return s.writeLocked(entries)
}

func (s *FileHeartbeatStore) List(_ context.Context, maxAge time.Duration) ([]WorkerHeartbeat, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.readLocked()
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}

	now := time.Now()
	out := make([]WorkerHeartbeat, 0, len(entries))
	for _, hb := range entries {
		if maxAge > 0 && now.Sub(hb.LastSeen) > maxAge {
			continue
		}
		out = append(out, hb)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].HostID < out[j].HostID })
	return out, nil
}

// readLocked loads the heartbeat map from disk. Missing file → empty map
// (the first Put creates the file). Returns nil, nil for genuinely empty.
// The caller MUST hold s.mu.
func (s *FileHeartbeatStore) readLocked() (map[string]WorkerHeartbeat, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("heartbeat file read %s: %w", s.path, err)
	}
	if len(data) == 0 {
		return nil, nil
	}
	var entries map[string]WorkerHeartbeat
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("heartbeat file parse %s: %w", s.path, err)
	}
	return entries, nil
}

// writeLocked persists the heartbeat map atomically via write-then-rename.
// The caller MUST hold s.mu.
func (s *FileHeartbeatStore) writeLocked(entries map[string]WorkerHeartbeat) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("heartbeat file marshal: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("heartbeat file write tmp %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("heartbeat file rename %s -> %s: %w", tmp, s.path, err)
	}
	return nil
}

// DefaultHeartbeatPath returns the recommended path for the on-host
// heartbeat file. It lives under the user's AILANG state directory.
func DefaultHeartbeatPath(stateDir string) string {
	if stateDir == "" {
		home, _ := os.UserHomeDir()
		stateDir = filepath.Join(home, ".ailang", "state")
	}
	return filepath.Join(stateDir, "worker_heartbeats.json")
}
