package coordinator

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestApprovalHandoffTargets_ExcludesAutoEdges(t *testing.T) {
	tests := []struct {
		name  string
		agent *AgentConfig
		want  []string
	}{
		{
			name:  "nil agent",
			agent: nil,
			want:  nil,
		},
		{
			name:  "no handoffs configured",
			agent: &AgentConfig{ID: "a"},
			want:  nil,
		},
		{
			name: "all auto — everything already fired at completion",
			agent: &AgentConfig{
				ID: "a", TriggerOnComplete: []string{"b", "c"}, AutoApproveHandoffs: true,
			},
			want: nil,
		},
		{
			name: "none auto — the pipeline shape",
			agent: &AgentConfig{
				ID: "design-doc-creator", TriggerOnComplete: []string{"sprint-planner"},
			},
			want: []string{"sprint-planner"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := approvalHandoffTargets(tc.agent)
			if len(got) != len(tc.want) {
				t.Fatalf("approvalHandoffTargets() = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("approvalHandoffTargets() = %v, want %v", got, tc.want)
				}
			}
		})
	}
}

// TestHandoffTargetsPartition is the property that stops double-dispatch: every
// configured target fires exactly once, either at completion (auto) or at
// approval (non-auto) — never both, never neither.
func TestHandoffTargetsPartition(t *testing.T) {
	agent := &AgentConfig{
		ID:                  "a",
		TriggerOnComplete:   []string{"b", "c", "d"},
		AutoApproveHandoffs: false,
	}

	f := &finalizer{
		deps: &FinalizeDeps{AgentRegistry: registryWith(agent)},
		in:   FinalizeInput{Task: &TaskRecord{ID: "task-1", AgentID: "a"}},
	}
	auto := f.autoHandoffTargets()
	onApproval := approvalHandoffTargets(agent)

	seen := map[string]int{}
	for _, x := range auto {
		seen[x]++
	}
	for _, x := range onApproval {
		seen[x]++
	}
	for _, target := range agent.TriggerOnComplete {
		switch seen[target] {
		case 0:
			t.Errorf("target %q fires in NEITHER path — the handoff would be lost", target)
		case 1: // correct
		default:
			t.Errorf("target %q fires in BOTH paths — duplicate dispatch", target)
		}
	}
}

func registryWith(agents ...*AgentConfig) *AgentRegistry {
	r := NewAgentRegistry()
	for _, a := range agents {
		if a.Inbox == "" {
			a.Inbox = a.ID
		}
		_ = r.Register(a)
	}
	return r
}

// TestDispatchApprovalHandoffs_LoudWhenAgentUnknown pins the refusal that the
// twelve-day silent failure needed: a registry that cannot see the task's agent
// must not report "nothing to do".
func TestDispatchApprovalHandoffs_LoudWhenAgentUnknown(t *testing.T) {
	task := &TaskRecord{ID: "task-x", AgentID: "ghost"}
	_, err := dispatchApprovalHandoffs(context.Background(), NewAgentRegistry(), nil, nil, task)
	if err == nil {
		t.Fatal("expected an error when the registry does not know the task's agent")
	}
}

// TestDispatchApprovalHandoffs_NoMessageStoreIsLoud: owing a handoff with no way
// to deliver it must not look like success.
func TestDispatchApprovalHandoffs_NoMessageStoreIsLoud(t *testing.T) {
	agent := &AgentConfig{ID: "a", Inbox: "a", TriggerOnComplete: []string{"b"}}
	target := &AgentConfig{ID: "b", Inbox: "b"}
	task := &TaskRecord{ID: "task-y", AgentID: "a"}

	_, err := dispatchApprovalHandoffs(context.Background(), registryWith(agent, target), nil, nil, task)
	if err == nil {
		t.Fatal("expected an error: a handoff is owed but nothing can deliver it")
	}
}

// TestHandoffContent_NamesTheArtifact is the fact the next stage actually needs.
//
// design-doc-creator declares an output_marker of DESIGN_DOC_PATH:, the
// finalizer stores it on the task, and the handoff dropped it — so
// sprint-planner was asked to plan a design doc whose path it was never told,
// from a copy of the original request.
func TestHandoffContent_NamesTheArtifact(t *testing.T) {
	src := &AgentConfig{ID: "design-doc-creator", Label: "Design Doc Creator"}
	task := &TaskRecord{
		ID:            "task-08032ebc",
		Content:       "Design a secondary-model fallback for cloud executor agents",
		DesignDocPath: "design_docs/planned/m-secondary-model-fallback.md",
		BaseBranch:    "dev",
	}

	got := handoffContent(src, task, 0, nil)

	if !strings.Contains(got, "design_docs/planned/m-secondary-model-fallback.md") {
		t.Errorf("the handoff must name the artifact the previous stage produced:\n%s", got)
	}
	if !strings.Contains(got, "Design Doc Creator") || !strings.Contains(got, "task-08032ebc") {
		t.Errorf("the handoff lost its provenance:\n%s", got)
	}
	if !strings.Contains(got, task.Content) {
		t.Errorf("the original request is still context the next stage needs:\n%s", got)
	}
}

// A cloud task has no GitHub issue, and "#0" is a reference to nothing that
// reads exactly like a real one.
func TestHandoffContent_OmitsAbsentIssueNumber(t *testing.T) {
	src := &AgentConfig{ID: "a", Label: "A"}
	task := &TaskRecord{ID: "task-x", Content: "do the thing"}

	if got := handoffContent(src, task, 0, nil); strings.Contains(got, "#0") {
		t.Errorf("an absent issue must be omitted, not rendered as #0:\n%s", got)
	}
	if got := handoffContent(src, task, 1170, nil); !strings.Contains(got, "#1170") {
		t.Errorf("a real issue number must still appear:\n%s", got)
	}
}

// A task with no artifact still produces a usable handoff — the fields are
// additive, not required.
func TestHandoffContent_SurvivesAnEmptyTask(t *testing.T) {
	got := handoffContent(&AgentConfig{ID: "a", Label: "A"}, &TaskRecord{ID: "task-y"}, 0, nil)
	if !strings.Contains(got, "Please continue") {
		t.Errorf("a bare task must still hand off:\n%s", got)
	}
	for _, unwanted := range []string{"Design doc:", "Sprint plan:", "Branch:"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("an unset field must be omitted, not rendered empty (%s):\n%s", unwanted, got)
		}
	}
}

// TestResolveHandoffArtifacts_RecoversAPathNoMarkerRecorded is the backfill.
//
// DesignDocPath comes from an output marker the agent PRINTS, so a run that did
// the work and forgot the marker records nothing — and the handoff then says
// "continue" without saying from what. Every design-doc task created before
// 2026-09-14 is in that state. The approval's changed_files answers the same
// question mechanically, from the diff rather than from a model.
func TestResolveHandoffArtifacts_RecoversAPathNoMarkerRecorded(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	task := &TaskRecord{ID: "task-080f4657", AgentID: "design-doc-creator", Content: "x"}
	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := store.CreateApprovalRequest(ctx, &ApprovalRequestRecord{
		ID: ApprovalIDForTask(task.ID), TaskID: task.ID, Type: string(ApprovalTypeMerge),
		Status: "pending", CreatedAt: time.Now(),
		ContextJSON: `{"changed_files":["design_docs/planned/v0_38_0/m-openrouter-eu-routing.md","README.md"]}`,
	}); err != nil {
		t.Fatalf("approval: %v", err)
	}
	agent := &AgentConfig{ID: "design-doc-creator", ArtifactPatterns: []string{"design_docs/**/*.md"}}

	got := resolveHandoffArtifacts(ctx, store, task, agent)
	if len(got) != 1 || got[0] != "design_docs/planned/v0_38_0/m-openrouter-eu-routing.md" {
		t.Fatalf("got %v — want only the file inside the agent's declared artifact scope", got)
	}

	// And it reaches the message the next stage reads.
	body := handoffContent(&AgentConfig{ID: "design-doc-creator", Label: "Design Doc Creator"}, task, 0, got)
	if !strings.Contains(body, "m-openrouter-eu-routing.md") {
		t.Errorf("the recovered artifact must appear in the handoff:\n%s", body)
	}
}

// The marker WINS when present: it is the agent naming its own primary output,
// which a file list cannot distinguish among several.
func TestResolveHandoffArtifacts_MarkerWins(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	task := &TaskRecord{ID: "task-m", AgentID: "a", DesignDocPath: "design_docs/planned/the-one.md"}
	agent := &AgentConfig{ID: "a", ArtifactPatterns: []string{"design_docs/**/*.md"}}
	if got := resolveHandoffArtifacts(context.Background(), store, task, agent); got != nil {
		t.Errorf("a recorded marker needs no recovery, got %v", got)
	}
}

// artifact_patterns is the bound. Without a declared list the default is `**/*`,
// which would put every touched file in the handoff — the same reason auto-merge
// refuses on an undeclared list.
func TestResolveHandoffArtifacts_RefusesWithoutDeclaredPatterns(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	task := &TaskRecord{ID: "task-n", AgentID: "a"}
	if got := resolveHandoffArtifacts(context.Background(), store, task, &AgentConfig{ID: "a"}); got != nil {
		t.Errorf("no declared patterns means no bound, so no recovery: %v", got)
	}
}

func TestResolveHandoffArtifacts_ToleratesMissingOrJunkContext(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()
	agent := &AgentConfig{ID: "a", ArtifactPatterns: []string{"design_docs/**/*.md"}}

	// No approval record at all.
	if got := resolveHandoffArtifacts(ctx, store, &TaskRecord{ID: "task-none", AgentID: "a"}, agent); got != nil {
		t.Errorf("no approval record = nothing to recover, got %v", got)
	}
	// An approval whose context is not the shape we expect.
	task := &TaskRecord{ID: "task-junk", AgentID: "a"}
	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := store.CreateApprovalRequest(ctx, &ApprovalRequestRecord{
		ID: ApprovalIDForTask(task.ID), TaskID: task.ID, Status: "pending",
		CreatedAt: time.Now(), ContextJSON: `not json`,
	}); err != nil {
		t.Fatalf("approval: %v", err)
	}
	if got := resolveHandoffArtifacts(ctx, store, task, agent); got != nil {
		t.Errorf("unparseable context must degrade to no artifacts, not a crash: %v", got)
	}
	// Nil store must not panic — OnAgentApproved can be constructed without one.
	if got := resolveHandoffArtifacts(ctx, nil, task, agent); got != nil {
		t.Errorf("nil store = no recovery, got %v", got)
	}
}
