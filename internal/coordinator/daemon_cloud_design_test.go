package coordinator

import (
	"context"
	"io"
	"log"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pubsub"
	"github.com/sunholo-data/ailang/internal/testutil"
)

type designDispatchRecorder struct {
	params    []DispatchParams
	published int
}

func (r *designDispatchRecorder) Dispatch(_ context.Context, p DispatchParams) error {
	r.params = append(r.params, p)
	return nil
}
func (r *designDispatchRecorder) PublishTask(context.Context, string, string, string, string) error {
	r.published++
	return nil
}
func (*designDispatchRecorder) PublishMessage(context.Context, string, pubsub.MessageAttributes) error {
	return nil
}
func (*designDispatchRecorder) PublishEvent(context.Context, []byte, string, string, string) error {
	return nil
}
func (*designDispatchRecorder) Stop() {}

// Exercise dispatchTasksCloud itself, not just the scope-reducing helper. The
// release probe caught an uninitialized agent config bypassing that helper.
func TestCloudDispatchDesignScopeAndOrdinaryControl(t *testing.T) {
	testutil.SetHomeDir(t, t.TempDir())
	for _, design := range []bool{false, true} {
		name := "ordinary"
		if design {
			name = "design"
		}
		t.Run(name, func(t *testing.T) {
			store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "coordinator.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			agent := &AgentConfig{ID: "pkg-test", Inbox: "pkg:test", Workspace: "example/design-fixture", Provider: "pi", Model: "existing-model", ExecutorVariant: "pi", Timeout: "2h", MergeBranch: "main", SkipApproval: true, Subdirectory: "packages/auth", DesignRequests: true, Invoke: &InvokeConfig{Type: "prompt", Template: "ordinary maintenance {{.Content}}"}}
			registry := NewAgentRegistry()
			if err := registry.Register(agent); err != nil {
				t.Fatal(err)
			}
			task := designRequestTask()
			task.AgentID = agent.ID
			task.Workspace = agent.Workspace
			task.Status = TaskStatusPending
			if !design {
				task.Content = "ordinary request"
			}
			ctx := context.Background()
			if err := store.CreateTask(ctx, task); err != nil {
				t.Fatal(err)
			}
			record := &designDispatchRecorder{}
			d := &Daemon{ctx: ctx, taskStore: store, agentRegistry: registry, pubsubPublisher: record, cloudDispatcher: record, logger: log.New(io.Discard, "", 0)}
			if err := d.dispatchTasksCloud(); err != nil {
				t.Fatal(err)
			}
			if record.published != 1 || len(record.params) != 1 {
				t.Fatalf("published=%d dispatched=%d", record.published, len(record.params))
			}
			p := record.params[0]
			if p.AgentID != agent.ID || p.Provider != "pi" || p.Model != agent.Model || p.ExecutorVariant != "pi" || p.Branch != "main" {
				t.Fatalf("lost configured route: %+v", p)
			}
			if design {
				if p.PushBranch != "" || p.Subdirectory != "" || p.Timeout != "20m" {
					t.Fatalf("retained implementation authority: %+v", p)
				}
				if !strings.Contains(p.Directive, "first read .claude/skills/design-doc-creator/SKILL.md") || !strings.Contains(p.Directive, task.Content) {
					t.Fatalf("lost design scope: %s", p.Directive)
				}
			} else {
				if p.PushBranch != "main" || p.Subdirectory != "packages/auth" || p.Timeout != "2h" || !strings.HasPrefix(p.Directive, "ordinary maintenance") {
					t.Fatalf("changed ordinary dispatch: %+v", p)
				}
			}
			if !agent.SkipApproval || agent.Subdirectory != "packages/auth" || agent.Timeout != "2h" {
				t.Fatal("mutated shared registry")
			}
		})
	}
}
