package effects

import (
	"sync"
	"time"
)

// ProcessContext provides configuration for Process effect security
//
// The process context holds security settings for command execution:
//   - Timeout enforcement (default: 30s)
//   - Output size limits (default: 10MB)
//   - Command allowlist with path-pinned resolution
//   - Working directory from sandbox
//   - Managed process tracking (M-ASYNC-IO Phase 3)
type ProcessContext struct {
	Timeout      time.Duration     // Maximum execution time
	MaxOutput    int64             // Maximum stdout/stderr bytes before kill
	Allowlist    map[string]string // Allowed commands: name → resolved absolute path (nil = all)
	HasAllowlist bool              // True if allowlist was explicitly set

	// Managed process tracking (M-ASYNC-IO Phase 3)
	mu            sync.Mutex
	managed       map[int]*managedProcess
	nextManagedID int
}

// NewProcessContext creates a new process context with secure defaults
func NewProcessContext() *ProcessContext {
	return &ProcessContext{
		Timeout:   30 * time.Second,
		MaxOutput: 10 * 1024 * 1024, // 10 MB
	}
}

// AcquireManagedProcess registers a managed process and returns its ID.
func (pc *ProcessContext) AcquireManagedProcess(mp *managedProcess) int {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.managed == nil {
		pc.managed = make(map[int]*managedProcess)
	}

	id := pc.nextManagedID
	pc.nextManagedID++
	mp.id = id
	pc.managed[id] = mp
	return id
}

// GetManagedProcess retrieves a managed process by ID.
func (pc *ProcessContext) GetManagedProcess(id int) (*managedProcess, bool) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	mp, ok := pc.managed[id]
	return mp, ok
}

// ReleaseManagedProcess removes a managed process from tracking.
func (pc *ProcessContext) ReleaseManagedProcess(id int) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	delete(pc.managed, id)
}

// CloseAllManaged kills all tracked managed processes. Used for graceful shutdown.
func (pc *ProcessContext) CloseAllManaged() {
	pc.mu.Lock()
	procs := make([]*managedProcess, 0, len(pc.managed))
	for _, mp := range pc.managed {
		procs = append(procs, mp)
	}
	pc.mu.Unlock()

	for _, mp := range procs {
		mp.Close()
	}
}
