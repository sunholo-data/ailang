package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/mission/activation"
	"github.com/sunholo-data/ailang/internal/mission/iteration"
)

func activationCLIFixture(t *testing.T) (missionActivationDeps, []string, string) {
	t.Helper()
	home := t.TempDir()
	for _, p := range []string{filepath.Join(home, ".ailang", "state"), filepath.Join(home, ".config", "ailang")} {
		if err := os.MkdirAll(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	db := filepath.Join(home, "runtime.db")
	binding := filepath.Join(home, "canary.toml")
	if err := os.WriteFile(binding, []byte(fmt.Sprintf("version=1\nstate_db=%q\nworkspace_root=%q\n", db, filepath.Join(home, "work"))), 0600); err != nil {
		t.Fatal(err)
	}
	limits := iteration.Limits{TimeoutSeconds: 30, MaxTokens: 1000, MaxCostUSD: 0.1}
	spec := iteration.Spec{Version: 1, MissionID: "docs", WorkItemID: "work", Repository: "github.com/example/docs", BaseRevision: strings.Repeat("a", 40), Brief: "Approved work", AllowedPaths: []string{"docs/"}, Workflow: "full-v1", Limits: iteration.Limits{TimeoutSeconds: 120, MaxTokens: 4000, MaxCostUSD: 0.4}, AcceptanceCriteria: []iteration.Criterion{{ID: "clear", Text: "Clear docs"}}, Verification: []iteration.Verification{{ID: "check", Argv: []string{"git", "diff", "--check"}, Cwd: ".", TimeoutSeconds: 10}}}
	for _, role := range []string{"designer", "planner", "executor", "evaluator"} {
		spec.Stages = append(spec.Stages, iteration.Stage{ID: role, Role: role, Instructions: "Do approved work", RequiredArtifacts: []string{"docs/result.md"}, Limits: limits})
	}
	body, _ := json.Marshal(spec)
	work := filepath.Join(home, "work.json")
	if err := os.WriteFile(work, body, 0600); err != nil {
		t.Fatal(err)
	}
	d := missionActivationDeps{Home: home, LegacyIdle: func(context.Context, string) error { return nil }, ProcessAlive: func(int) (bool, error) { return false, nil }, SessionStopped: func(context.Context, int) error { return nil }}
	return d, []string{"run", "docs", "--operation", "test-op", "--work-item", work, "--binding", binding}, db
}
func activationFixtureTerminal(t *testing.T, db string, terminal bool) {
	t.Helper()
	ctx := context.Background()
	s, err := coordinator.NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	key := coordinator.MissionWorkItemKey{MissionID: "docs", WorkItemID: "work"}
	raw := "{}"
	item, err := s.ClaimMissionWorkItem(ctx, coordinator.MissionWorkItemSpec{MissionWorkItemKey: key, SpecJSON: raw, SpecDigest: fmt.Sprintf("%x", sha256.Sum256([]byte(raw))), StageIDs: []string{"review"}, TimeoutSeconds: 60}, 30)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.ClaimMissionChild(ctx, key, item.OwnerToken, coordinator.MissionAttemptSpec{MissionID: "docs", WorkItemID: "work", StageID: "review", AttemptID: "attempt", RequestJSON: raw, RequestDigest: fmt.Sprintf("%x", sha256.Sum256([]byte(raw)))}, 30)
	if err != nil {
		t.Fatal(err)
	}
	if terminal {
		item, err = s.GetMissionWorkItem(ctx, key)
		if err != nil {
			t.Fatal(err)
		}
		if err = s.CancelMissionWorkItem(ctx, key, item.Version); err != nil {
			t.Fatal(err)
		}
	}
}
func TestActivationCLIConfirmedCancellationRestores(t *testing.T) {
	d, args, db := activationCLIFixture(t)
	runErr := errors.New("child cancelled")
	d.RunChild = func(_ context.Context, dir string, p *activationProcess, _ io.Writer) error {
		p.SessionID = 12345
		p.Phase = "exited"
		if err := saveActivationProcess(dir, *p); err != nil {
			return err
		}
		activationFixtureTerminal(t, db, true)
		return runErr
	}
	err := runMissionActivation(context.Background(), args, &bytes.Buffer{}, d)
	if !errors.Is(err, runErr) {
		t.Fatal(err)
	}
	m := activation.Manager{Dir: filepath.Join(d.Home, ".ailang", "state", "mission-activations")}
	r, err := m.Inspect("test-op")
	if err != nil || r.Phase != "restored" {
		t.Fatalf("terminal cancellation not restored: %+v %v", r, err)
	}
	for _, p := range []string{r.Marker.Path, r.Binding.Path} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("retained installation %s", p)
		}
	}
}
func TestActivationCLIUnknownWorkAndLiveSessionHeld(t *testing.T) {
	for _, live := range []bool{false, true} {
		t.Run(fmt.Sprint(live), func(t *testing.T) {
			d, args, db := activationCLIFixture(t)
			d.RunChild = func(_ context.Context, dir string, p *activationProcess, _ io.Writer) error {
				p.SessionID = 12345
				p.Phase = "exited"
				if err := saveActivationProcess(dir, *p); err != nil {
					return err
				}
				activationFixtureTerminal(t, db, live)
				return nil
			}
			if live {
				d.SessionStopped = func(context.Context, int) error { return errors.New("owned child remains alive") }
			}
			if err := runMissionActivation(context.Background(), args, &bytes.Buffer{}, d); err == nil {
				t.Fatal("unsafe cleanup succeeded")
			}
			m := activation.Manager{Dir: filepath.Join(d.Home, ".ailang", "state", "mission-activations")}
			r, err := m.Inspect("test-op")
			if err != nil || r.CleanupPending == "" || r.Phase == "restored" {
				t.Fatalf("missing hold: %+v %v", r, err)
			}
			if _, err := os.Stat(r.Marker.Path); err != nil {
				t.Fatal("unsafe pause removal")
			}
		})
	}
}
func TestActivationInspectMissingAndFlagsNeverWrite(t *testing.T) {
	d := missionActivationDeps{Home: t.TempDir()}
	for _, args := range [][]string{{"inspect", "docs", "--operation", "missing"}, {"recover", "other", "--operation", "x"}, {"run", "docs", "--operation", "x", "--operation", "y"}, {"run", "docs", "--operation", "../x"}, {"inspect", "docs", "--operation", "x", "--binding", "unexpected"}} {
		if err := runMissionActivation(context.Background(), args, &bytes.Buffer{}, d); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	if _, err := os.Stat(filepath.Join(d.Home, ".ailang")); !os.IsNotExist(err) {
		t.Fatal("read-only/invalid command created state")
	}
}

func TestActivationFreezesSourceBeforeRun(t *testing.T) {
	d, args, _ := activationCLIFixture(t)
	source := args[5]
	original, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	d.RunChild = func(_ context.Context, _ string, p *activationProcess, _ io.Writer) error {
		if err := os.WriteFile(source, []byte("replaced source"), 0600); err != nil {
			return err
		}
		frozen, err := os.ReadFile(p.WorkFile)
		if err != nil {
			return err
		}
		if !bytes.Equal(frozen, original) {
			t.Error("dispatch depends on mutable source")
		}
		spec, err := iteration.Decode(bytes.NewReader(frozen))
		if err != nil {
			return err
		}
		if spec.Digest() != p.SpecDigest {
			t.Error("frozen digest missing")
		}
		return errors.New("fixture no dispatch")
	}
	if err := runMissionActivation(context.Background(), args, &bytes.Buffer{}, d); err == nil {
		t.Fatal("expected fixture exit")
	}
}

func TestActivationMissingDBRequiresNoDispatchReceipt(t *testing.T) {
	for _, receipt := range []bool{false, true} {
		t.Run(fmt.Sprint(receipt), func(t *testing.T) {
			d, args, _ := activationCLIFixture(t)
			d.RunChild = func(_ context.Context, dir string, p *activationProcess, _ io.Writer) error {
				p.SessionID = 12345
				p.Phase = "exited"
				p.NoDispatch = receipt
				return saveActivationProcess(dir, *p)
			}
			err := runMissionActivation(context.Background(), args, &bytes.Buffer{}, d)
			if receipt && err != nil {
				t.Fatal(err)
			}
			if !receipt && err == nil {
				t.Fatal("missing execution evidence was treated as no dispatch")
			}
			m := activation.Manager{Dir: filepath.Join(d.Home, ".ailang", "state", "mission-activations")}
			r, e := m.Inspect("test-op")
			if e != nil {
				t.Fatal(e)
			}
			if (r.Phase == "restored") != receipt {
				t.Fatalf("unexpected restoration: %+v", r)
			}
		})
	}
}
