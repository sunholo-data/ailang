package coordinator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"reflect"
	"testing"
)

type shadowRecordingDispatcher struct {
	calls []DispatchParams
	err   error
}

func (r *shadowRecordingDispatcher) Dispatch(_ context.Context, p DispatchParams) error {
	r.calls = append(r.calls, p)
	return r.err
}

func shadowTestDaemon(t *testing.T, disp CloudDispatcher, tasks ...*TaskRecord) (*Daemon, *SQLiteStore) {
	t.Helper()
	s, err := NewSQLiteStore(filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	ctx := context.Background()
	for _, task := range tasks {
		if err := s.CreateTask(ctx, task); err != nil {
			t.Fatal(err)
		}
	}
	return &Daemon{taskStore: s, cloudDispatcher: disp, ctx: ctx, logger: log.New(io.Discard, "", 0)}, s
}

// primaryParams is a primary dispatch with every push/PR field set, so the
// test sees the shadow clear them.
func primaryParams() DispatchParams {
	return DispatchParams{
		TaskID: "task-p", AgentID: "pkg-sunholo-ailang-parse", Provider: "pi", ExecutorVariant: "pi",
		Model: "openrouter/z-ai/glm-5.3-flash", Directive: "fix the parser", MaxCostUSD: 2,
		PushBranch: "dev", AutoMerge: true, AutoMergeCode: true, AutoMergeRequiredChecks: []string{"ci"},
		AutoMergeApproverSecret: "sec", AutoMergeApproverIdentity: "bot", PRLabels: []string{"needs-approval"},
		MergeStarts: []string{"next"}, AuthMode: "apikey", APIKey: "user-key",
		ToolPolicy: "ailang_only", PolicyTOML: "[policy]", Inputs: []TaskInput{{Repo: "a/b", Ref: "main", Path: "x"}},
	}
}

func primaryTask() *TaskRecord {
	return &TaskRecord{
		ID: "task-p", AgentID: "pkg-sunholo-ailang-parse", Title: "fix", Content: "fix the parser",
		MessageID: "msg-1", ThreadID: "thread-1", GithubIssue: 42, ChainID: "chain-1", StageID: "stage-1",
		Status: TaskStatusQueued, Type: TaskTypeBugFix,
	}
}

// An eligible task gives exactly one extra dispatch: the shadow, on the shadow
// harness, flagged, with every push/PR field cleared, and a child record that
// carries ShadowOf and none of the primary's pipeline links.
func TestDispatchShadowEligible(t *testing.T) {
	rec := &shadowRecordingDispatcher{}
	primary := primaryTask()
	d, s := shadowTestDaemon(t, rec, primary)
	agent := laneAgent(&ShadowConfig{Enabled: true, Provider: "motoko", ExecutorVariant: "motoko", SampleRate: 1, MaxCostUSD: 0.5})
	pp := primaryParams()
	before := primaryParams()

	d.dispatchShadow(primary, agent, pp)

	if !reflect.DeepEqual(pp, before) {
		t.Fatal("dispatchShadow changed the primary's dispatch parameters")
	}
	if len(rec.calls) != 1 {
		t.Fatalf("dispatches = %d, want 1 (the shadow)", len(rec.calls))
	}
	got := rec.calls[0]
	if got.TaskID != "task-p-shadow" || !got.Shadow || got.Provider != "motoko" || got.ExecutorVariant != "motoko" {
		t.Fatalf("shadow identity wrong: %+v", got)
	}
	if got.Model != pp.Model {
		t.Errorf("model = %q, want the primary's %q (D3)", got.Model, pp.Model)
	}
	if got.MaxCostUSD != 0.5 {
		t.Errorf("MaxCostUSD = %v, want the shadow cap 0.5", got.MaxCostUSD)
	}
	if got.PushBranch != "" || got.AutoMerge || got.AutoMergeCode || got.AutoMergeRequiredChecks != nil ||
		got.AutoMergeApproverSecret != "" || got.AutoMergeApproverIdentity != "" || got.PRLabels != nil || got.MergeStarts != nil {
		t.Errorf("shadow kept a push/PR field: %+v", got)
	}
	if got.APIKey != "" || got.AuthMode != "" {
		t.Errorf("shadow kept the user's key: auth=%q key set=%v", got.AuthMode, got.APIKey != "")
	}
	if got.Directive != pp.Directive || got.ToolPolicy != pp.ToolPolicy || got.PolicyTOML != pp.PolicyTOML {
		t.Errorf("shadow must run the same directive under the same policy: %+v", got)
	}

	child, err := s.GetTask(d.ctx, "task-p-shadow")
	if err != nil {
		t.Fatal(err)
	}
	if child.ShadowOf != "task-p" || child.Status != TaskStatusQueued {
		t.Errorf("child ShadowOf=%q status=%s; want task-p, queued", child.ShadowOf, child.Status)
	}
	if child.StageID != "" || child.ChainID != "" || child.MessageID != "" || child.ThreadID != "" || child.GithubIssue != 0 {
		t.Errorf("child carries a pipeline link: %+v", child)
	}
}

func TestDispatchShadowIneligible(t *testing.T) {
	enabled := &ShadowConfig{Enabled: true, Provider: "motoko", ExecutorVariant: "motoko", SampleRate: 1}
	cases := []struct {
		name    string
		agent   *AgentConfig
		primary func() *TaskRecord
	}{
		{"no shadow block", laneAgent(nil), primaryTask},
		{"disabled", laneAgent(&ShadowConfig{Provider: "motoko", ExecutorVariant: "motoko", SampleRate: 1}), primaryTask},
		{"sample miss", laneAgent(&ShadowConfig{Enabled: true, Provider: "motoko", ExecutorVariant: "motoko", SampleRate: 0}), primaryTask},
		{"nil agent", nil, primaryTask},
		{"shadow of a shadow", laneAgent(enabled), func() *TaskRecord {
			p := primaryTask()
			p.ShadowOf = "task-0"
			return p
		}},
		{"codex shadow leases the subscription lane", laneAgent(&ShadowConfig{
			Enabled: true, Provider: "codex", ExecutorVariant: "codex", SampleRate: 1}), primaryTask},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := &shadowRecordingDispatcher{}
			p := tc.primary()
			d, s := shadowTestDaemon(t, rec, p)
			d.dispatchShadow(p, tc.agent, primaryParams())
			if len(rec.calls) != 0 {
				t.Fatalf("dispatches = %d, want 0", len(rec.calls))
			}
			if _, err := s.GetTask(d.ctx, shadowTaskID(p.ID)); err == nil {
				t.Fatal("a shadow child was created")
			}
		})
	}
}

// upsertStore models Firestore's CreateTask, which is a Set: it overwrites an
// existing document instead of failing, as SQLite's primary key would.
type upsertStore struct{ *SQLiteStore }

func (u upsertStore) CreateTask(ctx context.Context, t *TaskRecord) error {
	_ = u.SQLiteStore.DeleteTask(ctx, t.ID)
	return u.SQLiteStore.CreateTask(ctx, t)
}

// A second dispatch of the same primary finds the child and does not dispatch
// twice, even on a store whose create overwrites (Firestore).
func TestDispatchShadowOncePerPrimary(t *testing.T) {
	rec := &shadowRecordingDispatcher{}
	primary := primaryTask()
	d, s := shadowTestDaemon(t, rec, primary)
	d.taskStore = upsertStore{s}
	agent := laneAgent(motokoShadow())
	d.dispatchShadow(primary, agent, primaryParams())
	d.dispatchShadow(primary, agent, primaryParams())
	if len(rec.calls) != 1 {
		t.Fatalf("dispatches = %d, want 1", len(rec.calls))
	}
}

// A failed shadow dispatch is marked failed and not retried; the primary is untouched.
func TestDispatchShadowFailureMarksChildFailed(t *testing.T) {
	rec := &shadowRecordingDispatcher{err: errors.New("quota")}
	primary := primaryTask()
	d, s := shadowTestDaemon(t, rec, primary)
	d.dispatchShadow(primary, laneAgent(motokoShadow()), primaryParams())
	child, err := s.GetTask(d.ctx, "task-p-shadow")
	if err != nil {
		t.Fatal(err)
	}
	if child.Status != TaskStatusFailed {
		t.Fatalf("child status = %s, want failed", child.Status)
	}
	if p, _ := s.GetTask(d.ctx, "task-p"); p.Status != TaskStatusQueued {
		t.Fatalf("primary status = %s, want queued (untouched)", p.Status)
	}
}

// An orphaned shadow child left pending must never be dispatched as an
// ordinary task: it is failed before any publish or dispatch is attempted (a
// nil publisher here would panic if it were reached).
func TestOrphanedShadowIsFailedNotDispatched(t *testing.T) {
	rec := &shadowRecordingDispatcher{}
	d, s := shadowTestDaemon(t, rec, &TaskRecord{
		ID: "task-p-shadow", ShadowOf: "task-p", AgentID: "pkg-sunholo-ailang-parse",
		Title: "fix", Content: "fix", Status: TaskStatusPending,
	})
	if err := d.dispatchTasksCloud(); err != nil {
		t.Fatal(err)
	}
	if len(rec.calls) != 0 {
		t.Fatalf("orphan dispatched %d times", len(rec.calls))
	}
	if got, _ := s.GetTask(d.ctx, "task-p-shadow"); got.Status != TaskStatusFailed {
		t.Fatalf("orphan status = %s, want failed", got.Status)
	}
}

// The sampling decision is a pure function of the task ID: stable, and close to
// the configured rate over many IDs.
func TestShadowSamplingStableAndCalibrated(t *testing.T) {
	for i := 0; i < 20; i++ {
		if shadowSampleFraction("task-abc123") != shadowSampleFraction("task-abc123") {
			t.Fatal("sampling fraction is not stable")
		}
	}
	hits := 0
	for i := 0; i < 10000; i++ {
		if shadowSampled(fmt.Sprintf("task-%08x", i*2654435761), 0.25) {
			hits++
		}
	}
	if hits < 2300 || hits > 2700 {
		t.Fatalf("rate 0.25 sampled %d/10000", hits)
	}
	if !shadowSampled("task-x", 1) || shadowSampled("task-x", 0) {
		t.Fatal("rate 1 must always sample and rate 0 never")
	}
}
