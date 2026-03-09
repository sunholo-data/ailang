//go:build !js

package effects

import (
	"sync"
)

// managedState holds managed process tracking state (not available on js/wasm).
type managedState struct {
	mu            sync.Mutex
	managed       map[int]*managedProcess
	nextManagedID int
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
