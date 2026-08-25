package auth

import (
	"context"
	"sync"
	"time"
)

// AuditDecision is the outcome recorded for one profile operation.
type AuditDecision string

const (
	DecisionAllowed       AuditDecision = "allowed"
	DecisionDenied        AuditDecision = "denied"
	DecisionReleased      AuditDecision = "released"
	DecisionRevoked       AuditDecision = "revoked"
	DecisionPublished     AuditDecision = "published"
	DecisionOrphanRemoved AuditDecision = "orphan_removed"
	DecisionCleanupFailed AuditDecision = "cleanup_failed"
)

// AuditEvent is the durable record of who used which profile version, under what
// authority, and with what outcome.
//
// Every field is safe by construction: the type has no field that can hold
// material, a context ID, or a credential, so an audit sink cannot leak one by
// writing the struct out. The redaction guarantee here is structural, not a
// filter that has to be maintained.
type AuditEvent struct {
	Op              string          `json:"op"`
	Alias           string          `json:"alias"`
	Version         string          `json:"version"`
	ProfileHash     string          `json:"profile_hash,omitempty"`
	LeaseID         string          `json:"lease_id,omitempty"`
	Mode            LeaseMode       `json:"mode,omitempty"`
	Run             RunIdentity     `json:"run"`
	AllowedOrigins  []string        `json:"allowed_origins,omitempty"`
	Provider        string          `json:"provider,omitempty"`
	AccountClass    AccountClass    `json:"account_class,omitempty"`
	Decision        AuditDecision   `json:"decision"`
	FailureCategory FailureCategory `json:"failure_category,omitempty"`
	Reason          string          `json:"reason,omitempty"`
	At              time.Time       `json:"at"`
}

// NewAuditEvent builds an allowed/denied event from a profile and a lease. Pass
// a zero lease for operations that were denied before any lease existed.
func NewAuditEvent(op string, profile SafeProfile, lease ProfileLease, runID RunIdentity, at time.Time, err error) AuditEvent {
	event := AuditEvent{
		Op:             op,
		Alias:          profile.Alias,
		Version:        profile.Version,
		ProfileHash:    profile.ProfileHash,
		LeaseID:        lease.SafeID,
		Mode:           lease.Mode,
		Run:            runID,
		AllowedOrigins: append([]string(nil), profile.Policy.AllowedOrigins...),
		Provider:       profile.Provider,
		AccountClass:   profile.Policy.AccountClass,
		Decision:       DecisionAllowed,
		At:             at.UTC(),
	}
	if err != nil {
		event.Decision = DecisionDenied
		if failure, ok := AsFailure(err); ok {
			event.FailureCategory = failure.Category
			event.Reason = failure.Reason
		}
	}
	return event
}

// AuditSink receives profile audit events. Recording must never fail the
// operation being audited; callers log a sink error and continue.
type AuditSink interface {
	Record(ctx context.Context, event AuditEvent) error
}

// MemoryAuditSink is the reference sink and the test double.
type MemoryAuditSink struct {
	mu     sync.Mutex
	events []AuditEvent
}

func NewMemoryAuditSink() *MemoryAuditSink { return &MemoryAuditSink{} }

func (s *MemoryAuditSink) Record(_ context.Context, event AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *MemoryAuditSink) Events() []AuditEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]AuditEvent(nil), s.events...)
}

var _ AuditSink = (*MemoryAuditSink)(nil)
