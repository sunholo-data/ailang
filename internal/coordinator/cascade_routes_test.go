package coordinator

import (
	"context"
	"io"
	"log"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/pubsub"
)

// Route-level tests for M-CASCADE-ENVELOPE-HYDRATION: the three routes a
// cascade directive travels, and the gate that refuses an unauthorised one
// before a Cloud Run job exists.

// --- Route A: Pub/Sub push with legacy-shaped data (no embedded envelope) ---

// A cascade-topic notification whose data carries only a message id (an older
// publisher, or any non-envelope publisher) used to produce a task with ALL
// cascade fields empty. The stored payload always carried the envelope; it was
// simply never read. Now it is.
func TestPubSubAdapter_CascadeLegacyShape_HydratesFromStoredPayload(t *testing.T) {
	a := NewPubSubInboxAdapter(nil, "sub", "pkg:sunholo/docparse",
		&failingGetStore{msg: &messaging.InboxMessage{
			MessageID: "msg-cascade-1", ToInbox: "pkg:sunholo/docparse",
			FromAgent: "coordinator", MessageType: "upgrade-available",
			Payload: cascadePayload(t),
		}}, newSilentLogger())

	attrs := map[string]string{
		"inbox":       "pkg:sunholo/docparse",
		"from_agent":  "coordinator",
		"source":      "cascade",
		"message_type": "upgrade-available",
	}
	if err := a.HandleNotification(validNotification("msg-cascade-1"), attrs); err != nil {
		t.Fatalf("a hydratable cascade message must dispatch: %v", err)
	}
	if len(a.buffered) != 1 {
		t.Fatalf("buffered %d, want 1", len(a.buffered))
	}
	got := a.buffered[0]
	if got.Source != pubsub.SourceCascade {
		t.Errorf("Source = %q, want cascade (asserted by the topic)", got.Source)
	}
	if got.RootPackage != "sunholo/auth" || got.RootChangeClass != "C" ||
		got.FromVersion != "1.2.0" || got.ToVersion != "1.3.0" ||
		got.FromInterfaceHash != "iface-old" || got.ToInterfaceHash != "iface-new" {
		t.Errorf("envelope not hydrated from the payload: %+v", got)
	}
}

// A cascade-topic assertion with NOTHING durable behind it — no embedded
// envelope, no hydratable payload — is permanently unactionable. Ack loudly;
// redelivery cannot grow an envelope, and NACKing forever starves the
// subscription exactly the way the EMPTY-dispatch flood starved triage.
func TestPubSubAdapter_CascadeNoEnvelope_AcksLoudlyAndBuffersNothing(t *testing.T) {
	a := NewPubSubInboxAdapter(nil, "sub", "pkg:sunholo/docparse",
		&failingGetStore{msg: &messaging.InboxMessage{
			MessageID: "msg-cascade-2", ToInbox: "pkg:sunholo/docparse",
			Payload: "please bump the thing",
		}}, newSilentLogger())

	attrs := map[string]string{"inbox": "pkg:sunholo/docparse", "source": "cascade"}
	if err := a.HandleNotification(validNotification("msg-cascade-2"), attrs); err != nil {
		t.Errorf("permanently unactionable data must be acked, not retried forever: %v", err)
	}
	if n := len(a.buffered); n != 0 {
		t.Errorf("buffered %d task(s) from an envelope-less cascade assertion", n)
	}
}

// A package bump payload on a NON-cascade topic hydrates its envelope fields
// (they are inert data) but must NOT gain cascade authority.
func TestPubSubAdapter_PublicTopicPackageMessage_NeverGainsCascadeSource(t *testing.T) {
	a := NewPubSubInboxAdapter(nil, "sub", "pkg:sunholo/docparse",
		&failingGetStore{msg: &messaging.InboxMessage{
			MessageID: "msg-public-1", ToInbox: "pkg:sunholo/docparse",
			Payload: cascadePayload(t),
		}}, newSilentLogger())

	attrs := map[string]string{"inbox": "pkg:sunholo/docparse"}
	if err := a.HandleNotification(validNotification("msg-public-1"), attrs); err != nil {
		t.Fatalf("a readable message must dispatch: %v", err)
	}
	got := a.buffered[0]
	if got.RootPackage != "sunholo/auth" {
		t.Errorf("RootPackage = %q — payload hydration must not depend on the topic", got.RootPackage)
	}
	if got.Source == pubsub.SourceCascade {
		t.Error("a public-topic message must never carry the cascade authority stamp")
	}
}

// --- Route B: backstop sweep recovery ---

// The sweep recovers a bump whose payload carries the envelope. The envelope
// fields survive; Source stays unset (InboxMessage persists no Source, so
// authority cannot be verified on this route) and the dispatch gate refuses it
// loudly rather than the old silent-empty dispatch.
func TestBackstopSweep_PreservesCascadeEnvelopeWithoutStampingAuthority(t *testing.T) {
	reg := NewAgentRegistry()
	mustRegister(t, reg, "pkg-sunholo-docparse", "pkg:sunholo/docparse")

	store := &sweepStore{msgs: []messaging.InboxMessage{{
		ID: "inbox_cascade", ToInbox: "pkg:sunholo/docparse", FromAgent: "coordinator",
		MessageType: "upgrade-available", Title: "upgrade available: sunholo/auth",
		Payload: cascadePayload(t),
	}}}

	var logBuf strings.Builder
	adapter := &PubSubInboxAdapter{logger: log.New(io.Discard, "", 0)}
	s := NewBackstopSweep(store, reg, adapter, log.New(&logBuf, "", 0))
	s.mode = BackstopDispatch
	s.SweepOnce(context.Background())

	if len(adapter.buffered) != 1 {
		t.Fatalf("recovered %d message(s), want 1", len(adapter.buffered))
	}
	got := adapter.buffered[0]
	if got.RootPackage != "sunholo/auth" || got.ToVersion != "1.3.0" || got.RootChangeClass != "C" {
		t.Errorf("sweep dropped the envelope the payload carried: %+v", got)
	}
	if got.Source != "" {
		t.Errorf("Source = %q — the sweep has no way to verify cascade authority and must not stamp it", got.Source)
	}
	if !strings.Contains(logBuf.String(), "no verifiable cascade source") {
		t.Errorf("the sweep must say loudly that authority could not be verified, got:\n%s", logBuf.String())
	}
}

// --- Route C: the dispatch gate ---

// A cascade-shaped task with an empty envelope is BLOCKED before any job is
// created — the measured 173 EMPTY-dispatch reports each cost a full agent run
// to conclude "nothing to repair". Refusal must be durable (status + error on
// the task record, naming the source message id) and cheap (no job, no agent).
func TestDispatchGate_BlocksUnauthorisedCascadeTask(t *testing.T) {
	s, err := NewSQLiteStore(filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	task := &TaskRecord{
		ID: "task-emptycascade", AgentID: "pkg-sunholo-docparse",
		Title: "upgrade available", Content: cascadePayload(t),
		Kind: "upgrade-available", Status: TaskStatusPending,
		MessageID: "inbox_1788179644941_9d0b57b3",
	}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}

	d := &Daemon{taskStore: s, agentRegistry: NewAgentRegistry(), ctx: ctx, logger: log.New(io.Discard, "", 0)}
	// A nil publisher would panic if the task reached dispatch — which is
	// exactly the assertion: the gate must refuse it first.
	if err := d.dispatchTasksCloud(); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != TaskStatusBlocked {
		t.Errorf("status = %q, want blocked — the task must be refused, not dispatched", got.Status)
	}
	if !strings.Contains(got.Error, "inbox_1788179644941_9d0b57b3") {
		t.Errorf("the refusal must name the source message id for forensics, got: %q", got.Error)
	}
	if !strings.Contains(got.Error, "source=") {
		t.Errorf("the refusal must state what the task actually carried, got: %q", got.Error)
	}
}

// An authorised cascade task (verified source + populated envelope) passes the
// gate — the gate narrows dispatch, it does not close it.
func TestDispatchGate_AllowsAuthorisedCascadeTask(t *testing.T) {
	if !cascadeDispatchAuthorised(&TaskRecord{
		Source: pubsub.SourceCascade, RootPackage: "sunholo/auth", ToVersion: "1.3.0",
		Kind: "upgrade-available",
	}) {
		t.Fatal("precondition: a fully populated cascade task must pass the gate predicate")
	}
}
