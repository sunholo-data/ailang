package iteration

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/mission/dispatch"
)

// Crashes use the real Service/dispatch/receipt transaction sequence, not just
// store methods. Fixture-only SQL expires leases to avoid thirty-second sleeps.
func TestIterationAbruptBoundaries(t *testing.T) {
	if input := os.Getenv("AILANG_TEST_ITERATION_SNAPSHOT"); input != "" {
		body, err := os.ReadFile(input)
		if err != nil {
			os.Exit(24)
		}
		var snap Snapshot
		if json.Unmarshal(body, &snap) != nil {
			os.Exit(25)
		}
		store, err := coordinator.NewSQLiteStore(os.Getenv("AILANG_TEST_ITERATION_DB"))
		if err != nil {
			os.Exit(26)
		}
		fake := &iterationExecutor{t: t}
		s := &Service{Store: store, Models: snap.Models, Factory: fake, RepositoryPath: snap.RepositoryPath, WorkspaceRoot: snap.WorkspaceRoot, Admit: func(context.Context, dispatch.Candidate) (dispatch.Admission, error) {
			return dispatch.Admission{Allowed: true, Policy: "fixture", ObservedAt: time.Now()}, nil
		}}
		s.checkpoint = func(phase string) {
			if phase == os.Getenv("AILANG_TEST_ITERATION_PHASE") {
				os.Exit(23)
			}
		}
		_, _ = s.Run(context.Background(), snap.Spec)
		os.Exit(27)
	}
	for _, phase := range []string{"prepared", "dispatching", "receipt_finished", "execution_completed", "accepted"} {
		t.Run(phase, func(t *testing.T) {
			s, spec, fake := runtimeFixture(t)
			db := filepath.Join(t.TempDir(), "crash.db")
			store, err := coordinator.NewSQLiteStore(db)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			s.Store = store
			snap, err := s.Plan(context.Background(), spec)
			if err != nil {
				t.Fatal(err)
			}
			body, _ := json.Marshal(snap)
			path := filepath.Join(t.TempDir(), "snapshot.json")
			if err = os.WriteFile(path, body, 0600); err != nil {
				t.Fatal(err)
			}
			ctx, c := context.WithTimeout(context.Background(), 20*time.Second)
			defer c()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestIterationAbruptBoundaries$")
			cmd.Env = append(os.Environ(), "AILANG_TEST_ITERATION_SNAPSHOT="+path, "AILANG_TEST_ITERATION_DB="+db, "AILANG_TEST_ITERATION_PHASE="+phase)
			out, err := cmd.CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 23 {
				t.Fatalf("crash boundary missed %v %s", err, out)
			}
			connection, err := sql.Open("sqlite3", db)
			if err != nil {
				t.Fatal(err)
			}
			defer connection.Close()
			for _, q := range []string{"UPDATE mission_work_items SET lease_until=0", "UPDATE mission_attempts SET lease_until=0"} {
				if _, err = connection.Exec(q); err != nil {
					t.Fatal(err)
				}
			}
			item, err := s.Run(context.Background(), spec)
			if err != nil {
				t.Fatalf("recover %+v %v", item, err)
			}
			switch phase {
			case "dispatching", "receipt_finished":
				if item.State != "needs_reconciliation" || fake.calls != 0 {
					t.Fatalf("ambiguous work reran %+v calls=%d", item, fake.calls)
				}
			default:
				want := 1
				if phase == "prepared" {
					want = 2
				}
				if item.State != "completed" || fake.calls != want {
					t.Fatalf("recovery %+v calls=%d want=%d", item, fake.calls, want)
				}
			}
		})
	}
}
