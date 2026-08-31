package coordinator

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// BackstopSweepMode selects what the sweep does with what it finds.
type BackstopSweepMode string

const (
	// BackstopReport logs what WOULD be recovered and changes nothing. Default.
	BackstopReport BackstopSweepMode = "report"
	// BackstopDispatch hands recovered messages to the normal drain.
	BackstopDispatch BackstopSweepMode = "dispatch"
	// BackstopOff disables the sweep entirely.
	BackstopOff BackstopSweepMode = "off"
)

// BackstopSweep recovers messages that reached Firestore but whose Pub/Sub
// notification never arrived.
//
// The cloud coordinator's intake is Pub/Sub ONLY: pollAndProcessTasksCloud
// drains the PubSubInboxAdapter and never queries Firestore. So a message whose
// notification was never published is invisible FOREVER, not merely late.
//
// Measured 2026-08-31: three pkg:sunholo/ailang_parse reports sat unread with a
// correctly registered agent and a healthy coordinator, because the SENDER's
// config carried no pubsub section. That send-side hole is closed separately;
// this is the receive-side floor, so the next hole of that shape costs latency
// instead of silence.
//
// Mode comes from AILANG_BACKSTOP_SWEEP and defaults to "report" deliberately.
// The sweep's own risk is double-dispatching work that push already did, so the
// first thing it has to earn is a measurement of how often it would fire at all.
// Promote to "dispatch" once the reported count is understood.
type BackstopSweep struct {
	msgStore      messaging.MessageStore
	agentRegistry *AgentRegistry
	adapter       *PubSubInboxAdapter
	logger        *log.Logger
	interval      time.Duration
	mode          BackstopSweepMode
}

// NewBackstopSweep builds the sweep. mode is read from AILANG_BACKSTOP_SWEEP.
func NewBackstopSweep(
	msgStore messaging.MessageStore,
	agentRegistry *AgentRegistry,
	adapter *PubSubInboxAdapter,
	logger *log.Logger,
) *BackstopSweep {
	return &BackstopSweep{
		msgStore:      msgStore,
		agentRegistry: agentRegistry,
		adapter:       adapter,
		logger:        logger,
		interval:      10 * time.Minute,
		mode:          backstopModeFromEnv(),
	}
}

// backstopModeFromEnv resolves the mode, defaulting to report. An unrecognised
// value is refused into "report" WITH a warning rather than silently treated as
// "dispatch" — the safe reading of an unclear instruction is the one that
// cannot double-run somebody's work.
func backstopModeFromEnv() BackstopSweepMode {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("AILANG_BACKSTOP_SWEEP"))) {
	case "dispatch":
		return BackstopDispatch
	case "off":
		return BackstopOff
	case "", "report":
		return BackstopReport
	default:
		return BackstopReport
	}
}

// Run starts the periodic sweep. Blocks until ctx is cancelled.
func (s *BackstopSweep) Run(ctx context.Context) {
	if s.mode == BackstopOff {
		s.logger.Println("backstop sweep: disabled (AILANG_BACKSTOP_SWEEP=off)")
		return
	}
	s.logger.Printf("backstop sweep: started (interval=%v, mode=%s)", s.interval, s.mode)

	// Sweep ONCE on startup, before the first tick.
	//
	// This is load-bearing in the deployment we actually have, not a nicety.
	// The cloud coordinator runs as a Cloud Run service with minScale=0, woken
	// by Pub/Sub push and killed again when idle. Measured in dev 2026-08-31,
	// straight after this shipped: consecutive instance lifetimes of 34s and
	// 12s. A 10-minute ticker inside a process that lives seconds would never
	// fire once — the sweep would be running, logging that it started, and
	// silently doing nothing. That is the exact failure shape this whole design
	// doc exists to remove, so it must not be reintroduced by the sweep itself.
	//
	// Startup is the one moment this process is reliably alive, so that is when
	// the work happens. The ticker remains for long-lived hosts (local mode, the
	// rig) where an instance does outlive the interval.
	s.SweepOnce(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Println("backstop sweep: stopped")
			return
		case <-ticker.C:
			s.SweepOnce(ctx)
		}
	}
}

// SweepOnce performs one pass. Exported so a test — and an operator — can run a
// single pass without waiting for a tick.
func (s *BackstopSweep) SweepOnce(_ context.Context) {
	if s.msgStore == nil || s.agentRegistry == nil {
		return
	}
	msgs, err := s.msgStore.ListInboxMessages(messaging.InboxListOptions{
		UnreadOnly: true,
		Collapsed:  true, // respect semantic dedup: never resurrect a known duplicate
	})
	if err != nil {
		s.logger.Printf("backstop sweep: list unread failed: %v", err)
		return
	}

	var recoverable []messaging.InboxMessage
	for _, m := range msgs {
		// A declared human-triage inbox is filed for a person on purpose, not a
		// dropped dispatch. It resolves to no agent anyway; saying so explicitly
		// keeps the intent readable if that ever stops being true.
		if s.agentRegistry.IsTriageOnly(m.ToInbox) {
			continue
		}
		// Only messages an agent would actually have taken. An unrouted inbox is
		// a triage question for a human, not lost work.
		if agent := s.agentRegistry.GetAgentForInbox(m.ToInbox); agent == nil {
			continue
		}
		recoverable = append(recoverable, m)
	}

	if len(recoverable) == 0 {
		return
	}

	// A non-zero count is itself the signal: in a healthy system push delivers
	// everything and this is always zero. Say so at every pass that finds work.
	s.logger.Printf("backstop sweep: found %d unread message(s) with a registered agent that push did not deliver (mode=%s)",
		len(recoverable), s.mode)

	for _, m := range recoverable {
		if s.mode == BackstopReport {
			s.logger.Printf("backstop sweep: WOULD recover %s (inbox=%s, from=%s, title=%q)",
				m.ID, m.ToInbox, m.FromAgent, truncateForLog(m.Title))
			continue
		}
		s.adapter.Enqueue(&Message{
			ID:      m.ID,
			From:    m.FromAgent,
			Title:   m.Title,
			Content: m.Payload,
			Inbox:   m.ToInbox,
			Type:    m.Category,
		})
		s.logger.Printf("backstop sweep: recovered %s (inbox=%s) into the normal drain", m.ID, m.ToInbox)
	}
}

// truncateForLog keeps a title readable in a log line.
func truncateForLog(s string) string {
	const max = 60
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
