package coordinator

import (
	"context"
	"io"
	"log"
	"strings"
	"testing"

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
