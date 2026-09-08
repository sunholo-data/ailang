package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/mission/activation"
	"github.com/sunholo-data/ailang/internal/mission/iteration"
)

const activationCheckTimeout = 15 * time.Second

type activationProcess struct {
	Version     int    `json:"version"`
	OperationID string `json:"operation_id"`
	WorkItemID  string `json:"work_item_id"`
	WorkFile    string `json:"work_file"`
	SpecDigest  string `json:"spec_digest"`
	LauncherPID int    `json:"launcher_pid"`
	SessionID   int    `json:"session_id"`
	NoDispatch  bool   `json:"no_dispatch,omitempty"`
	Phase       string `json:"phase"`
}

func activationProcessPath(dir, id string) string { return filepath.Join(dir, id+".process.json") }
func createActivationProcess(dir string, p activationProcess) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(activationProcessPath(dir, p.OperationID), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	err = errors.Join(err, f.Sync(), f.Close())
	if err != nil {
		return err
	}
	return activationSyncDir(dir)
}
func saveActivationProcess(dir string, p activationProcess) error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".activation-process-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, err = f.Write(b)
	err = errors.Join(err, f.Sync(), f.Close())
	if err != nil {
		return err
	}
	if err = os.Rename(f.Name(), activationProcessPath(dir, p.OperationID)); err != nil {
		return err
	}
	return activationSyncDir(dir)
}
func activationSyncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}
func readActivationProcess(dir, id string) (p activationProcess, err error) {
	path := activationProcessPath(dir, id)
	info, err := os.Lstat(path)
	if err != nil {
		return p, err
	}
	if !info.Mode().IsRegular() || info.Size() > 65536 {
		return p, errors.New("invalid activation process evidence file")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return p, err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err = dec.Decode(&p); err != nil {
		return p, err
	}
	if err = dec.Decode(new(any)); err != io.EOF {
		return p, errors.New("trailing process record content")
	}
	if p.Phase == "launching" && p.SessionID != 0 {
		return p, errors.New("launching process record has inconsistent session identity")
	}
	if p.Version != 1 || p.OperationID != id || !activationID.MatchString(id) || p.WorkItemID == "" || p.LauncherPID < 1 || p.WorkFile != filepath.Join(dir, id+".work-item.json") || len(p.SpecDigest) != 64 || p.Phase != "launching" && p.Phase != "armed" && p.Phase != "exited" || p.Phase != "launching" && p.SessionID < 1 {
		return p, errors.New("invalid activation process ownership")
	}
	return p, nil
}
func activationStopVerifier(dir string, d missionActivationDeps, attended bool) activation.Verify {
	return func(ctx context.Context, r activation.Record) error {
		if err := d.LegacyIdle(ctx, d.Home); err != nil {
			return err
		}
		p, err := readActivationProcess(dir, r.OperationID)
		if err != nil {
			return fmt.Errorf("cleanup_pending: missing process ownership: %w", err)
		}
		if p.WorkItemID != r.WorkItemID {
			return errors.New("cleanup_pending: process work item does not match activation")
		}
		if !attended || p.LauncherPID != os.Getpid() {
			alive, e := d.ProcessAlive(p.LauncherPID)
			if e != nil {
				return e
			}
			if alive {
				return errors.New("cleanup_pending: activation supervisor is still alive")
			}
		}
		if p.SessionID > 0 {
			if err = d.SessionStopped(ctx, p.SessionID); err != nil {
				return err
			}
		} else if p.Phase != "launching" {
			return errors.New("cleanup_pending: unknown child session")
		}
		// Even an empty process session cannot resolve unknown successful effects.
		var b iteration.Binding
		if _, err = toml.Decode(string(r.Binding.Installed), &b); err != nil {
			return err
		}
		store, err := coordinator.OpenMissionReadOnlyStore(b.StateDB)
		if errors.Is(err, os.ErrNotExist) && (p.Phase == "launching" || p.NoDispatch) {
			return nil
		}
		if err != nil {
			return err
		}
		defer store.Close()
		item, err := store.GetMissionWorkItem(ctx, coordinator.MissionWorkItemKey{MissionID: r.MissionID, WorkItemID: r.WorkItemID})
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if item.State != "completed" && item.State != "failed" && item.State != "cancelled" {
			return fmt.Errorf("cleanup_pending: work item is %s; retain binding for reconciliation", item.State)
		}
		for _, stage := range item.StageIDs {
			a, e := store.GetMissionAttempt(ctx, coordinator.MissionAttemptKey{MissionID: r.MissionID, WorkItemID: r.WorkItemID, StageID: stage})
			if errors.Is(e, sql.ErrNoRows) {
				continue
			}
			if e != nil {
				return e
			}
			if a.State != "execution_completed" && a.State != "execution_failed" && a.State != "cancelled" {
				return fmt.Errorf("cleanup_pending: stage %s is %s", stage, a.State)
			}
		}
		return nil
	}
}
func runActivationChild(ctx context.Context, dir, id string, out io.Writer) error {
	// Parent writes the release byte only after durably recording the session ID.
	var gate [1]byte
	if _, err := io.ReadFull(os.Stdin, gate[:]); err != nil {
		return errors.New("activation parent exited before dispatch release")
	}
	if gate[0] != 'R' {
		return errors.New("invalid activation dispatch release")
	}
	p, err := readActivationProcess(dir, id)
	if err != nil {
		return err
	}
	if p.Phase != "armed" || p.SessionID != os.Getpid() {
		return errors.New("activation child does not own recorded session")
	}
	m := activation.Manager{Dir: dir}
	r, err := m.Inspect(id)
	if err != nil {
		return err
	}
	if r.Phase != "active" || r.WorkItemID != p.WorkItemID {
		return errors.New("activation is not active for this work item")
	}
	current, err := os.ReadFile(r.Binding.Path)
	if err != nil {
		return err
	}
	if !bytes.Equal(current, r.Binding.Installed) {
		return errors.New("activation binding changed before child dispatch")
	}
	f, err := os.Open(p.WorkFile)
	if err != nil {
		return err
	}
	spec, err := iteration.Decode(f)
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	}
	if spec.Digest() != p.SpecDigest || spec.MissionID != r.MissionID || spec.WorkItemID != p.WorkItemID {
		return errors.New("frozen activation work item changed")
	}
	binding, err := iteration.LoadBinding(r.Binding.Path)
	if err != nil {
		return err
	}
	_, beforeErr := os.Stat(binding.StateDB)
	runErr := runMissionIteration(ctx, "iterate", []string{"--work-item", p.WorkFile}, out, missionIterationDeps{BindingPath: r.Binding.Path})
	// CLI exit 2 is a typed pre-dispatch/configuration failure, emitted before
	// Service.Run. Preserve this receipt so a never-created DB is not mistaken
	// for missing execution evidence. Runtime/provider failures return other codes.
	if os.IsNotExist(beforeErr) && missionErrorExitCode(runErr) == 2 {
		p.NoDispatch = true
		return errors.Join(runErr, saveActivationProcess(dir, p))
	}
	return runErr
}
