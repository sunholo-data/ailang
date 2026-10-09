// Package credentiallease defines exclusive ownership of rotating credentials.
// Heartbeats are diagnostic, never permission to take over an old owner: a live
// stale worker cannot be fenced by a database expiry from an OAuth endpoint.
package credentiallease

import (
	"context"
	"fmt"
	"time"
)

type Store interface {
	Change(context.Context, string, string, string) error
}

type CredentialLeaseRecord struct {
	Owner       string    `firestore:"owner"`
	Heartbeat   time.Time `firestore:"heartbeat"`
	Quarantined bool      `firestore:"quarantined"`
}

// TransitionCredentialLease is applied transactionally by the storage backend.
// Ownership requires a unique process generation, not just a reusable task ID.
// Crashed owners remain blocked until attended recovery verifies termination.
func TransitionCredentialLease(r CredentialLeaseRecord, owner, action string, now time.Time) (CredentialLeaseRecord, error) {
	if owner == "" {
		return r, fmt.Errorf("credential lease requires an owner")
	}
	if r.Quarantined {
		return r, fmt.Errorf("credential quarantined: attended recovery required")
	}
	if action == "acquire" {
		if r.Owner != "" {
			return r, fmt.Errorf("credential already owned: attended recovery required if owner has terminated")
		}
		return CredentialLeaseRecord{Owner: owner, Heartbeat: now}, nil
	}
	if r.Owner != owner {
		return r, fmt.Errorf("credential lease ownership lost")
	}
	switch action {
	case "heartbeat":
		r.Heartbeat = now
	case "release":
		r.Owner = ""
		r.Heartbeat = now
	case "quarantine":
		r.Quarantined = true
		r.Heartbeat = now
	default:
		return r, fmt.Errorf("unknown credential lease action %q", action)
	}
	return r, nil
}
