package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// FileAuditSink appends events as JSON Lines. Profile use must leave a durable
// trail that survives the process, and append-only JSONL is the format an
// incident review can read without tooling.
type FileAuditSink struct {
	mu   sync.Mutex
	path string
}

func NewFileAuditSink(path string) (*FileAuditSink, error) {
	if path == "" {
		return nil, errors.New("audit sink requires a path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	return &FileAuditSink{path: path}, nil
}

func (s *FileAuditSink) Path() string { return s.path }

func (s *FileAuditSink) Record(_ context.Context, event AuditEvent) error {
	encoded, err := json.Marshal(event)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	_, err = file.Write(append(encoded, '\n'))
	return err
}

// ReadAuditLog parses a JSONL audit file. A malformed line fails loudly rather
// than being skipped: a silently truncated audit trail is worse than none.
func ReadAuditLog(path string) ([]AuditEvent, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var events []AuditEvent
	for index, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var event AuditEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("audit log line %d is unreadable: %w", index+1, err)
		}
		events = append(events, event)
	}
	return events, nil
}

var _ AuditSink = (*FileAuditSink)(nil)
