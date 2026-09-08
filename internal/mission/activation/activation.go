// Package activation owns recoverable, host-local Docs canary installation.
// It never starts/stops processes: callers must verify the process boundary.
package activation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Manager.Dir must be the one canonical host activation directory for all callers.
// The durable owner survives process exit; a short OS lock serializes all writes.
type Manager struct {
	Dir        string
	checkpoint func(string)
}

// Request supports only the local Docs marker and strict runtime binding format.
// The CLI must resolve these paths from host placement, not accept arbitrary targets.
type Request struct {
	OperationID string
	MissionID   string
	WorkItemID  string
	MarkerPath  string
	BindingPath string
	Binding     []byte
}

// File records both sides of an atomic mutation before the mutation occurs.
type File struct {
	Path          string `json:"path"`
	Existed       bool   `json:"existed"`
	Prior         []byte `json:"prior,omitempty"`
	PriorHash     string `json:"prior_hash"`
	PriorMode     uint32 `json:"prior_mode"`
	Installed     []byte `json:"installed"`
	InstalledHash string `json:"installed_hash"`
}

// Record is retained after cleanup, including the canary binding for read-only access.
type Record struct {
	Version        int    `json:"version"`
	OperationID    string `json:"operation_id"`
	MissionID      string `json:"mission_id"`
	WorkItemID     string `json:"work_item_id"`
	Phase          string `json:"phase"`
	CleanupPending string `json:"cleanup_pending,omitempty"`
	Marker         File   `json:"marker"`
	Binding        File   `json:"binding"`
}

// Verify must establish process identity and absence of running or ambiguous work.
// A terminal DB label alone is insufficient. Nil always refuses mutation/cleanup.
type Verify func(context.Context, Record) error

var identifier = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,127}$`)

func (m *Manager) point(name string) {
	if m.checkpoint != nil {
		m.checkpoint(name)
	}
}

// Activate installs the pause before changing the binding. Any partial failure
// retains ownership for Recover; it never assumes a failure means processes stopped.
func (m *Manager) Activate(ctx context.Context, q Request, verifyIdle Verify) (r Record, err error) {
	if !identifier.MatchString(q.OperationID) || q.MissionID != "docs" || q.WorkItemID == "" {
		return r, errors.New("activation requires valid operation ID, docs mission and work item")
	}
	if err = validateBinding(q.Binding); err != nil {
		return r, err
	}
	unlock, err := m.lock()
	if err != nil {
		return r, err
	}
	defer unlock()
	if err = m.available(); err != nil {
		return r, err
	}
	if _, err = os.Lstat(m.recordPath(q.OperationID)); !os.IsNotExist(err) {
		return r, errors.New("operation record already exists or cannot be inspected; recover it or use a new ID")
	}
	marker, err := capture(q.MarkerPath, []byte("ailang mission activation "+q.OperationID+"\n"))
	if err != nil {
		return r, err
	}
	if marker.Existed {
		return r, errors.New("foreign Docs disable marker exists; activation refused")
	}
	binding, err := capture(q.BindingPath, q.Binding)
	if err != nil {
		return r, err
	}
	if marker.Path == binding.Path {
		return r, errors.New("marker and binding must be different files")
	}
	dir, err := filepath.EvalSymlinks(m.Dir)
	if err != nil {
		return r, err
	}
	for _, p := range []string{marker.Path, binding.Path} {
		rel, e := filepath.Rel(dir, p)
		if e != nil || rel == "." || !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
			return r, errors.New("activation targets must be outside record directory")
		}
	}
	if binding.Existed {
		if err = validateBinding(binding.Prior); err != nil {
			return r, fmt.Errorf("prior binding cannot be safely recorded: %w", err)
		}
	}
	r = Record{Version: 1, OperationID: q.OperationID, MissionID: q.MissionID, WorkItemID: q.WorkItemID, Phase: "prepared", Marker: marker, Binding: binding}
	if err = verify(ctx, r, verifyIdle); err != nil {
		return r, err
	}
	if err = m.save(r); err != nil {
		return r, err
	}
	m.point("record_prepared")
	if err = atomicWrite(filepath.Join(m.Dir, "active"), []byte(q.OperationID), 0600); err != nil {
		return r, err
	}
	m.point("owner_persisted")
	for _, item := range []struct {
		name string
		file File
	}{{"marker", marker}, {"binding", binding}} {
		m.point(item.name + "_before")
		if err = matchesPrior(item.file); err != nil {
			return m.pending(r, err)
		}
		if err = atomicWrite(item.file.Path, item.file.Installed, 0600); err != nil {
			return m.pending(r, err)
		}
		m.point(item.name + "_after")
		// Closing the scheduler gate and checking idle are separate operations.
		// Recheck after the pause so a legacy start in that window cannot see
		// the temporary runtime binding.
		if item.name == "marker" {
			if err = verify(ctx, r, verifyIdle); err != nil {
				return m.pending(r, err)
			}
		}
	}
	r.Phase = "active"
	if err = m.save(r); err != nil {
		return r, err
	}
	m.point("active")
	return r, nil
}

// Recover restores only unchanged owned files, after explicit verified stop.
// It is also safe after any activation/cleanup crash, including before ownership.
func (m *Manager) Recover(ctx context.Context, id string, verifyStopped Verify) (r Record, err error) {
	unlock, err := m.lock()
	if err != nil {
		return r, err
	}
	defer unlock()
	r, err = m.Inspect(id)
	if err != nil {
		return r, err
	}
	owner, err := m.owner()
	if err != nil {
		return r, err
	}
	if owner != "" && owner != id {
		return r, fmt.Errorf("host activation owned by %s; refusing recovery", owner)
	}
	if r.Phase == "restored" {
		if owner == id {
			err = m.release()
		}
		return r, err
	}
	if owner == "" {
		// The only unowned incomplete record is a pre-pointer preparation. No
		// mutation was authorized yet; installed-looking bytes are foreign evidence.
		if r.Phase != "prepared" {
			return m.pending(r, errors.New("cleanup_pending: active ownership record missing; investigate before restoring"))
		}
		for _, f := range []File{r.Marker, r.Binding} {
			if err = matchesPrior(f); err != nil {
				return m.pending(r, err)
			}
		}
	}
	if err = verify(ctx, r, verifyStopped); err != nil {
		return m.pending(r, err)
	}
	// Preflight both files before changing either. A changed binding keeps the pause.
	for _, f := range []File{r.Binding, r.Marker} {
		if _, err = restorable(f); err != nil {
			return m.pending(r, err)
		}
	}
	r.Phase = "restoring"
	r.CleanupPending = ""
	if err = m.save(r); err != nil {
		return r, err
	}
	for _, item := range []struct {
		name string
		file File
	}{{"binding", r.Binding}, {"marker", r.Marker}} {
		m.point("restore_" + item.name + "_before")
		prior, checkErr := restorable(item.file)
		if checkErr != nil {
			return m.pending(r, checkErr)
		}
		if !prior {
			if item.file.Existed {
				err = atomicWrite(item.file.Path, item.file.Prior, os.FileMode(item.file.PriorMode))
			} else {
				err = os.Remove(item.file.Path)
				if err == nil {
					err = syncDir(filepath.Dir(item.file.Path))
				}
			}
			if err != nil {
				return m.pending(r, err)
			}
		}
		m.point("restore_" + item.name + "_after")
	}
	r.Phase = "restored"
	r.CleanupPending = ""
	if err = m.save(r); err != nil {
		return r, err
	}
	m.point("restored")
	if owner == id {
		err = m.release()
	}
	m.point("owner_released")
	return r, err
}
func verify(ctx context.Context, r Record, fn Verify) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if fn == nil {
		return errors.New("process-stop verifier required; inspect owned process and ambiguous work before recovery")
	}
	return fn(ctx, r)
}
func (m *Manager) pending(r Record, cause error) (Record, error) {
	r.CleanupPending = cause.Error()
	if err := m.save(r); err != nil {
		return r, errors.Join(cause, err)
	}
	return r, cause
}
func (m *Manager) release() error {
	if err := os.Remove(filepath.Join(m.Dir, "active")); err != nil {
		return err
	}
	return syncDir(m.Dir)
}
func (m *Manager) available() error {
	owner, err := m.owner()
	if err != nil {
		return err
	}
	if owner == "" {
		return nil
	}
	r, err := m.Inspect(owner)
	if err != nil {
		return err
	}
	if r.Phase == "restored" {
		return m.release()
	}
	return fmt.Errorf("host activation owned by %s (%s); inspect/recover it first", owner, r.Phase)
}
