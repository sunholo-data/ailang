package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/mission/dispatch"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

// runDurableMissionRole fences admission before creating a filesystem receipt.
// Only callers sharing this database share the fence; this is not distributed locking.
func runDurableMissionRole(ctx context.Context, req dispatch.Request, receipt, dbPath string, models *modelreg.ModelsConfig, factory dispatch.Factory) (*dispatch.Report, error) {
	return runMissionRoleWithHeartbeat(ctx, req, receipt, dbPath, models, factory, 10*time.Second)
}

func runMissionRoleWithHeartbeat(ctx context.Context, req dispatch.Request, receipt, dbPath string, models *modelreg.ModelsConfig, factory dispatch.Factory, heartbeatInterval time.Duration) (*dispatch.Report, error) {
	store, err := coordinator.NewSQLiteStore(dbPath)
	if err != nil {
		return nil, err
	}
	defer store.Close()
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	spec := coordinator.MissionAttemptSpec{MissionID: req.MissionID, WorkItemID: req.WorkItemID, StageID: req.StageID, AttemptID: req.AttemptID, RequestDigest: req.Digest(), RequestJSON: string(body)}
	attempt, err := store.ClaimMissionAttempt(ctx, spec, 30)
	if err != nil {
		return nil, err
	}
	journal, err := dispatch.OpenJournal(receipt)
	if err != nil {
		outcome, _ := json.Marshal(map[string]string{"admission_error": err.Error()})
		finishErr := store.CompleteMissionAttempt(ctx, spec.Key(), attempt.OwnerToken, "execution_failed", string(outcome))
		return nil, errors.Join(err, finishErr)
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	stop := make(chan struct{})
	done := make(chan struct{})
	var once sync.Once
	var heartbeatErr error // written by goroutine, read only after done closes
	go func() {
		defer close(done)
		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-runCtx.Done():
				return
			case <-ticker.C:
				heartbeatCtx, c := context.WithTimeout(runCtx, 5*time.Second)
				e := store.RenewMissionAttempt(heartbeatCtx, spec.Key(), attempt.OwnerToken, 30)
				c()
				if e != nil {
					heartbeatErr = fmt.Errorf("mission lease lost: %w", e)
					cancel()
					return
				}
			}
		}
	}()
	stopHeartbeat := func() { once.Do(func() { close(stop) }); <-done }
	defer stopHeartbeat()
	runner := dispatch.Runner{Models: models, Executors: factory, Record: func(event dispatch.Event) error {
		if err := journal.Record(event); err != nil {
			return err
		}
		switch event.Kind {
		case "dispatching":
			return store.StartMissionAttempt(runCtx, spec.Key(), attempt.OwnerToken)
		case "finished":
			stopHeartbeat()
			if event.Report == nil {
				return fmt.Errorf("missing completion report")
			}
			outcome, e := json.Marshal(event.Report)
			if e != nil {
				return e
			}
			state := event.Report.Status
			if state == "blocked" {
				state = "execution_failed"
			}
			// Record cancellation/failure even when the provider context has ended.
			finishCtx, c := context.WithTimeout(context.Background(), 5*time.Second)
			defer c()
			return store.CompleteMissionAttempt(finishCtx, spec.Key(), attempt.OwnerToken, state, string(outcome))
		}
		return nil
	}}
	report, runErr := runner.Run(runCtx, req)
	stopHeartbeat()
	return report, errors.Join(runErr, heartbeatErr, journal.Close())
}
