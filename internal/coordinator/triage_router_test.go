package coordinator

import (
	"context"
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// fakeTriageStore is a minimal in-memory triageStore for tickOnce tests.
type fakeTriageStore struct {
	byInbox  map[string][]messaging.InboxMessage
	forwards map[string]string // message ID -> destination inbox
}

func newFakeTriageStore() *fakeTriageStore {
	return &fakeTriageStore{
		byInbox:  map[string][]messaging.InboxMessage{},
		forwards: map[string]string{},
	}
}

func (f *fakeTriageStore) ListInboxMessages(opts messaging.InboxListOptions) ([]messaging.InboxMessage, error) {
	var out []messaging.InboxMessage
	for _, m := range f.byInbox[opts.Inbox] {
		if _, forwarded := f.forwards[m.ID]; forwarded {
			continue // already moved to another inbox
		}
		if opts.Collapsed && m.DupOf != "" {
			continue // store hides duplicates when Collapsed
		}
		out = append(out, m)
	}
	return out, nil
}

func (f *fakeTriageStore) ForwardInboxMessage(id, toInbox string) error {
	f.forwards[id] = toInbox
	return nil
}

func TestTriageRouterTickOnce(t *testing.T) {
	store := newFakeTriageStore()
	store.byInbox["user"] = []messaging.InboxMessage{
		{ID: "b1", Category: "bug", FromAgent: "cli", Title: "real bug"},
		{ID: "g1", Category: "general", FromAgent: "cli", Title: "vague note"},
		{ID: "bd1", Category: "bug", FromAgent: "cli", Title: "dup bug", DupOf: "b1"},
		{ID: "n1", Category: "general", FromAgent: "eval-suite", Title: "eval run done"},
	}

	cfg := TriageConfig{IntakeInboxes: []string{"user"}} // rest defaulted via normalized()
	router := NewTriageRouter(store, cfg, nil)

	promoted := router.tickOnce(context.Background())
	if promoted != 1 {
		t.Fatalf("expected 1 promotion, got %d", promoted)
	}
	if store.forwards["b1"] != "design-doc-creator" {
		t.Errorf("bug b1 should be forwarded to design-doc-creator, got %q", store.forwards["b1"])
	}
	if _, ok := store.forwards["g1"]; ok {
		t.Error("general g1 should be held, not forwarded")
	}
	if _, ok := store.forwards["bd1"]; ok {
		t.Error("duplicate bd1 should be hidden by Collapsed, not forwarded")
	}
	if _, ok := store.forwards["n1"]; ok {
		t.Error("eval-suite noise n1 should be dropped, not forwarded")
	}

	// Idempotency: a second tick promotes nothing (b1 already left the inbox).
	if again := router.tickOnce(context.Background()); again != 0 {
		t.Fatalf("expected 0 promotions on second tick, got %d", again)
	}
}

func TestClassify(t *testing.T) {
	cfg := TriageConfig{}.normalized() // defaults: promote {bug,feature}, noise {eval-suite}

	tests := []struct {
		name     string
		category string
		from     string
		dupOf    string
		want     Decision
	}{
		{"bug promotes", "bug", "cli", "", DecisionPromote},
		{"feature promotes", "feature", "user", "", DecisionPromote},
		{"general holds", "general", "cli", "", DecisionHold},
		{"docs holds", "docs", "cli", "", DecisionHold},
		{"research holds", "research", "cli", "", DecisionHold},
		{"refactor holds", "refactor", "cli", "", DecisionHold},
		{"test holds", "test", "cli", "", DecisionHold},
		{"empty category holds", "", "cli", "", DecisionHold},
		{"bug from noise agent still promotes (category wins)", "bug", "eval-suite", "", DecisionPromote},
		{"feature from noise agent still promotes", "feature", "eval-suite", "", DecisionPromote},
		{"general from noise agent drops", "general", "eval-suite", "", DecisionDrop},
		{"empty category from noise agent drops", "", "eval-suite", "", DecisionDrop},
		{"bug duplicate holds (original handles it)", "bug", "cli", "orig-123", DecisionHold},
		{"feature duplicate holds", "feature", "cli", "orig-456", DecisionHold},
		{"duplicate from noise agent drops", "", "eval-suite", "orig-789", DecisionDrop},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := messaging.InboxMessage{
				Category:  tt.category,
				FromAgent: tt.from,
				DupOf:     tt.dupOf,
			}
			if got := classify(msg, cfg); got != tt.want {
				t.Fatalf("classify(category=%q, from=%q, dupOf=%q) = %s, want %s",
					tt.category, tt.from, tt.dupOf, got, tt.want)
			}
		})
	}
}

func TestTriageConfigNormalized(t *testing.T) {
	got := TriageConfig{}.normalized()
	if got.PromoteInbox != "design-doc-creator" {
		t.Errorf("PromoteInbox default = %q, want design-doc-creator", got.PromoteInbox)
	}
	if got.ClusterSlot != "intent" {
		t.Errorf("ClusterSlot default = %q, want intent", got.ClusterSlot)
	}
	if got.SimilarityThreshold != 0.75 {
		t.Errorf("SimilarityThreshold default = %v, want 0.75", got.SimilarityThreshold)
	}
	if got.PollIntervalSecs != 120 {
		t.Errorf("PollIntervalSecs default = %d, want 120", got.PollIntervalSecs)
	}
	if len(got.IntakeInboxes) != 2 {
		t.Errorf("IntakeInboxes default = %v, want [user claude-code]", got.IntakeInboxes)
	}
	if !stringInSlice(got.PromoteCategories, "bug") || !stringInSlice(got.PromoteCategories, "feature") {
		t.Errorf("PromoteCategories default = %v, want [bug feature]", got.PromoteCategories)
	}
	if !stringInSlice(got.NoiseAgents, "eval-suite") {
		t.Errorf("NoiseAgents default = %v, want [eval-suite]", got.NoiseAgents)
	}
}

func TestTriageConfigNormalizedPreservesExplicit(t *testing.T) {
	in := TriageConfig{
		PromoteInbox:        "custom-inbox",
		PromoteCategories:   []string{"bug"},
		SimilarityThreshold: 0.9,
		PollIntervalSecs:    30,
	}
	got := in.normalized()
	if got.PromoteInbox != "custom-inbox" {
		t.Errorf("explicit PromoteInbox overwritten: %q", got.PromoteInbox)
	}
	if len(got.PromoteCategories) != 1 || got.PromoteCategories[0] != "bug" {
		t.Errorf("explicit PromoteCategories overwritten: %v", got.PromoteCategories)
	}
	if got.SimilarityThreshold != 0.9 {
		t.Errorf("explicit SimilarityThreshold overwritten: %v", got.SimilarityThreshold)
	}
	if got.PollIntervalSecs != 30 {
		t.Errorf("explicit PollIntervalSecs overwritten: %d", got.PollIntervalSecs)
	}
}
