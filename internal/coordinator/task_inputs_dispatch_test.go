package coordinator

import (
	"context"
	"errors"
	"io"
	"log"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestTaskInputsGrants(t *testing.T) {
	inputs := []TaskInput{{Repo: "a/b", Ref: "incoming/poster", Path: "poster.png"}}
	for _, agent := range []*AgentConfig{nil, {ID: "site"}, {ID: "site", InputsAllow: []string{"a/*"}}, {ID: "site", InputsAllow: []string{"A/b"}}} {
		err := ValidateInputsForAgent(inputs, agent)
		if !errors.Is(err, ErrDispatchPermanent) {
			t.Fatalf("grant %#v: %v", agent, err)
		}
	}
	if err := ValidateInputsForAgent(inputs, &AgentConfig{ID: "site", InputsAllow: []string{"a/b"}}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateInputsForAgent(nil, nil); err != nil {
		t.Fatal(err)
	}
}
func TestTaskInputsTaskSQLite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.db")
	s, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	task := &TaskRecord{ID: "inputs", Title: "poster", Content: "publish", GithubIssue: 1600, Stage: TaskStageImplementation, Inputs: []TaskInput{{Repo: "a/b", Ref: "main", Path: "poster"}}}
	ctx := context.Background()
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	if err := s.SetTaskStage(ctx, task.ID, TaskStageImplementation); err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = NewSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.GetTask(ctx, task.ID)
	if err != nil || !reflect.DeepEqual(task.Inputs, got.Inputs) {
		t.Fatalf("get: %v %v", got, err)
	}
	lists := []func() ([]*TaskRecord, error){func() ([]*TaskRecord, error) { return s.ListTasks(ctx, &TaskFilter{}) }, func() ([]*TaskRecord, error) { return s.GetTasksByGithubIssue(ctx, 1600) }, func() ([]*TaskRecord, error) { return s.GetTasksByStage(ctx, TaskStageImplementation) }}
	for _, list := range lists {
		rows, err := list()
		if err != nil || len(rows) != 1 || !reflect.DeepEqual(rows[0].Inputs, task.Inputs) {
			t.Fatalf("list: %v %v", rows, err)
		}
	}
	if _, err := s.db.Exec(`UPDATE tasks SET inputs='broken' WHERE id=?`, task.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetTask(ctx, task.ID); err == nil {
		t.Fatal("dropped invalid stored task inputs")
	}
}

func TestTaskInputsTaskSQLiteLegacyMigration(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy-tasks.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := s.CreateTask(ctx, &TaskRecord{ID: "legacy", Title: "existing task", Content: "preserve"}); err != nil {
		t.Fatal(err)
	}
	// Model the immediately preceding task schema, then use normal startup's
	// additive migration rather than constructing a replacement task row.
	if _, err := s.db.Exec(`ALTER TABLE tasks DROP COLUMN inputs`); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.GetTask(ctx, "legacy")
	if err != nil || got.Title != "existing task" || got.Content != "preserve" || len(got.Inputs) != 0 {
		t.Fatalf("legacy task changed: %+v %v", got, err)
	}
}
func TestTaskInputsDeniedFailsBeforePublishing(t *testing.T) {
	s, err := NewSQLiteStore(filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	task := &TaskRecord{ID: "denied", AgentID: "site", Title: "poster", Content: "publish", Status: TaskStatusPending, Inputs: []TaskInput{{Repo: "private/infra", Ref: "main", Path: "secret"}}}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	registry := NewAgentRegistry()
	if err := registry.Register(&AgentConfig{ID: "site", Inbox: "site", Workspace: "a/b", Capabilities: []string{"docs"}, InputsAllow: []string{"a/b"}}); err != nil {
		t.Fatal(err)
	}
	d := &Daemon{taskStore: s, agentRegistry: registry, ctx: ctx, logger: log.New(io.Discard, "", 0)}
	// A nil publisher would panic if the unauthorized task reached dispatch.
	if err := d.dispatchTasksCloud(); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != TaskStatusFailed || !strings.Contains(got.Error, "private/infra") || !strings.Contains(got.Error, "site") {
		t.Fatalf("not permanently refused: %#v", got)
	}
}
func TestTaskInputsLocalRefusal(t *testing.T) {
	d := &Daemon{}
	err := d.executeTask(&TaskRecord{ID: "local", Inputs: []TaskInput{{Repo: "a/b", Ref: "main"}}})
	if err == nil || !strings.Contains(err.Error(), "cloud") {
		t.Fatalf("local inputs were not rejected before startup: %v", err)
	}
}
