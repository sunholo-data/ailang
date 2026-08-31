package coordinator

import (
	"context"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// sweepStore serves a fixed unread set. Embedding the interface means any method
// the sweep reaches for beyond ListInboxMessages panics loudly rather than
// silently returning a zero value.
type sweepStore struct {
	messaging.MessageStore
	msgs []messaging.InboxMessage
	err  error
	opts messaging.InboxListOptions
}

func (s *sweepStore) ListInboxMessages(opts messaging.InboxListOptions) ([]messaging.InboxMessage, error) {
	s.opts = opts
	return s.msgs, s.err
}

func sweepFixture(t *testing.T, mode BackstopSweepMode) (*BackstopSweep, *sweepStore, *PubSubInboxAdapter, *strings.Builder) {
	t.Helper()
	reg := NewAgentRegistry()
	mustRegister(t, reg, "pkg-sunholo-ailang-parse", "pkg:sunholo/ailang_parse")
	reg.SetTriageOnlyInboxes([]string{"public-feedback"})

	store := &sweepStore{msgs: []messaging.InboxMessage{
		{ID: "inbox_1", ToInbox: "pkg:sunholo/ailang_parse", FromAgent: "aitana-platform", Title: "docx list coalescing"},
		{ID: "inbox_2", ToInbox: "public-feedback", FromAgent: "mcp-public", Title: "triage-only, must be skipped"},
		{ID: "inbox_3", ToInbox: "nobody-watches-this", FromAgent: "x", Title: "unrouted, must be skipped"},
	}}

	var logBuf strings.Builder
	adapter := &PubSubInboxAdapter{logger: log.New(io.Discard, "", 0)}
	s := NewBackstopSweep(store, reg, adapter, log.New(&logBuf, "", 0))
	s.mode = mode
	return s, store, adapter, &logBuf
}

// TestBackstopSweepReportModeChangesNothing: report is the default because the
// sweep's own risk is double-dispatching work push already did. It must observe
// and not act.
func TestBackstopSweepReportModeChangesNothing(t *testing.T) {
	s, _, adapter, logBuf := sweepFixture(t, BackstopReport)
	s.SweepOnce(context.Background())

	if got := len(adapter.buffered); got != 0 {
		t.Errorf("report mode enqueued %d message(s); it must change nothing", got)
	}
	out := logBuf.String()
	if !strings.Contains(out, "WOULD recover") || !strings.Contains(out, "inbox_1") {
		t.Errorf("report mode must name what it would recover, got:\n%s", out)
	}
}

// TestBackstopSweepDispatchModeRecoversOnlyRoutable pins the filtering: a
// triage-only inbox is filed for a human on purpose, and an unrouted inbox is a
// config question — neither is dropped work.
func TestBackstopSweepDispatchModeRecoversOnlyRoutable(t *testing.T) {
	s, _, adapter, _ := sweepFixture(t, BackstopDispatch)
	s.SweepOnce(context.Background())

	if len(adapter.buffered) != 1 {
		t.Fatalf("recovered %d message(s), want exactly 1 (the routable one)", len(adapter.buffered))
	}
	got := adapter.buffered[0]
	if got.ID != "inbox_1" {
		t.Errorf("recovered %q, want inbox_1", got.ID)
	}
	if got.Inbox != "pkg:sunholo/ailang_parse" || got.From != "aitana-platform" {
		t.Errorf("recovered message lost its routing fields: %+v", got)
	}
}

// TestBackstopSweepRespectsSemanticDedup: resurrecting a known duplicate is the
// sweep turning into a work amplifier.
func TestBackstopSweepRespectsSemanticDedup(t *testing.T) {
	s, store, _, _ := sweepFixture(t, BackstopReport)
	s.SweepOnce(context.Background())

	if !store.opts.UnreadOnly {
		t.Error("sweep must query UnreadOnly")
	}
	if !store.opts.Collapsed {
		t.Error("sweep must set Collapsed so messages already marked dup_of are not resurrected")
	}
}

// TestBackstopModeFromEnv: an unrecognised value must land on the mode that
// cannot double-run somebody's work.
func TestBackstopModeFromEnv(t *testing.T) {
	for _, tc := range []struct {
		env  string
		want BackstopSweepMode
	}{
		{"", BackstopReport},
		{"report", BackstopReport},
		{"dispatch", BackstopDispatch},
		{"DISPATCH", BackstopDispatch},
		{" off ", BackstopOff},
		{"banana", BackstopReport},
	} {
		t.Run(tc.env, func(t *testing.T) {
			t.Setenv("AILANG_BACKSTOP_SWEEP", tc.env)
			if got := backstopModeFromEnv(); got != tc.want {
				t.Errorf("mode for %q = %q, want %q", tc.env, got, tc.want)
			}
		})
	}
}

// TestBackstopSweepSilentWhenNothingToRecover: in a healthy system push delivers
// everything and this must be quiet, or the signal is worthless.
func TestBackstopSweepSilentWhenNothingToRecover(t *testing.T) {
	s, store, _, logBuf := sweepFixture(t, BackstopReport)
	store.msgs = nil
	s.SweepOnce(context.Background())
	if logBuf.String() != "" {
		t.Errorf("sweep must be silent with nothing to recover, got:\n%s", logBuf.String())
	}
}

// TestBackstopSweepRunsOnStartup pins the fix for a defect this sweep shipped
// with and which its own deployment exposed.
//
// The cloud coordinator is a Cloud Run service with minScale=0: Pub/Sub push
// wakes it, idleness kills it. Measured in dev on 2026-08-31, minutes after the
// sweep first deployed, consecutive instance lifetimes were 34s and 12s. A
// 10-minute ticker inside a process that lives seconds fires zero times — the
// sweep would log "started", do nothing, and be indistinguishable from working.
// Startup is the only moment the process is reliably alive.
func TestBackstopSweepRunsOnStartup(t *testing.T) {
	s, store, adapter, _ := sweepFixture(t, BackstopDispatch)
	s.interval = time.Hour // guarantee no tick can fire during this test

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { s.Run(ctx); close(done) }()

	// Wait for the startup pass rather than sleeping a fixed amount.
	deadline := time.After(2 * time.Second)
	for {
		adapter.mu.Lock()
		n := len(adapter.buffered)
		adapter.mu.Unlock()
		if n > 0 {
			break
		}
		select {
		case <-deadline:
			cancel()
			<-done
			t.Fatal("sweep did nothing before the first tick: on a scale-to-zero host it would never run at all")
		case <-time.After(10 * time.Millisecond):
		}
	}
	cancel()
	<-done

	if store.opts.UnreadOnly != true {
		t.Error("the startup pass must be a real sweep, not a no-op")
	}
}

// TestBackstopSweepOffDoesNotSweepOnStartup: "off" must mean off, including the
// new startup pass.
func TestBackstopSweepOffDoesNotSweepOnStartup(t *testing.T) {
	s, _, adapter, _ := sweepFixture(t, BackstopOff)
	s.interval = time.Hour

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { s.Run(ctx); close(done) }()
	select {
	case <-done: // Run returns immediately when off
	case <-time.After(2 * time.Second):
		cancel()
		<-done
		t.Fatal("Run should return immediately when the sweep is off")
	}
	cancel()

	adapter.mu.Lock()
	n := len(adapter.buffered)
	adapter.mu.Unlock()
	if n != 0 {
		t.Errorf("off mode swept anyway and enqueued %d", n)
	}
}

// TestBackstopSweepCarriesDispatchSemantics is the regression arm for the prod
// loop of 2026-08-31.
//
// The sweep built its Message from six fields and dropped two that decide what
// happens next: Kind (isOutcomeNotice reads it) and CreatedAt (the task inherits
// it and the stale detector ages the task from it). Both losses are silent — the
// message is enqueued, dispatch "succeeds", and the damage shows up one hop away.
//
// MU: delete either field from the Enqueue in backstop_sweep.go and this fails.
func TestBackstopSweepCarriesDispatchSemantics(t *testing.T) {
	reg := NewAgentRegistry()
	mustRegister(t, reg, "docparse", "docparse")

	created := time.Date(2026, 8, 31, 10, 20, 58, 0, time.UTC)
	store := &sweepStore{msgs: []messaging.InboxMessage{
		{
			ID:          "inbox_req",
			ToInbox:     "docparse",
			FromAgent:   "ailang-parse-c",
			Title:       "Redeploy needed",
			MessageType: "request",
			CreatedAt:   created,
		},
	}}

	adapter := &PubSubInboxAdapter{logger: log.New(io.Discard, "", 0)}
	s := NewBackstopSweep(store, reg, adapter, log.New(io.Discard, "", 0))
	s.mode = BackstopDispatch
	s.SweepOnce(context.Background())

	if len(adapter.buffered) != 1 {
		t.Fatalf("recovered %d message(s), want 1", len(adapter.buffered))
	}
	got := adapter.buffered[0]

	if got.Kind != "request" {
		t.Errorf("Kind = %q, want %q — dropping Kind makes every completion notice "+
			"read as a request for work", got.Kind, "request")
	}
	if !got.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt = %v, want %v — a zero CreatedAt reaches the task record, "+
			"and time.Since(zero) is ~292 years, so the stale detector kills the task "+
			"on the first tick after dispatch", got.CreatedAt, created)
	}
}

// TestBackstopSweepIgnoresOutcomeNotices: a completion notice is posted INTO the
// inbox of the agent that ran the task, so it is unread and routable by
// construction. Counting it as "push did not deliver this" reports a permanent
// backlog for a healthy plane, and recovering it dispatches a report as work —
// whose failure posts another report.
//
// MU: remove the isOutcomeNotice guard from the filter loop and this fails.
func TestBackstopSweepIgnoresOutcomeNotices(t *testing.T) {
	reg := NewAgentRegistry()
	mustRegister(t, reg, "docparse", "docparse")

	store := &sweepStore{msgs: []messaging.InboxMessage{
		{ID: "inbox_notice", ToInbox: "docparse", FromAgent: "docparse",
			Title: "Task task-a855b349: failed (timeout)", MessageType: "completion",
			CreatedAt: time.Now()},
		{ID: "inbox_real", ToInbox: "docparse", FromAgent: "ailang-parse-c",
			Title: "Redeploy needed", MessageType: "request", CreatedAt: time.Now()},
	}}

	var logBuf strings.Builder
	adapter := &PubSubInboxAdapter{logger: log.New(io.Discard, "", 0)}
	s := NewBackstopSweep(store, reg, adapter, log.New(&logBuf, "", 0))
	s.mode = BackstopDispatch
	s.SweepOnce(context.Background())

	if len(adapter.buffered) != 1 {
		t.Fatalf("recovered %d message(s), want exactly 1 (the request, not the notice)",
			len(adapter.buffered))
	}
	if got := adapter.buffered[0].ID; got != "inbox_real" {
		t.Errorf("recovered %q, want inbox_real — a completion notice is not lost work", got)
	}
	// The count in the log is the operator-facing signal; a notice must not inflate it.
	if strings.Contains(logBuf.String(), "found 2 unread") {
		t.Errorf("sweep counted the completion notice as undelivered work:\n%s", logBuf.String())
	}
}
