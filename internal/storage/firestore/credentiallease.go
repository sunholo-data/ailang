package firestore

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/sunholo-data/ailang/internal/credentiallease"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CredentialLeaseStore uses the same Firestore client/transaction boundary as
// other cloud stores. Documents must NEVER have a Firestore TTL policy: crashed
// owners require attended verification of execution termination before recovery.
type CredentialLeaseStore struct{ client *Client }

func NewCredentialLeaseStore(client *Client) *CredentialLeaseStore {
	return &CredentialLeaseStore{client: client}
}
func (s *CredentialLeaseStore) Change(ctx context.Context, key, owner, action string) error {
	if key == "" {
		return fmt.Errorf("credential lease key is required")
	}
	id := fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
	ref := s.client.Doc("credential_leases", id)
	return s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		r := credentiallease.CredentialLeaseRecord{}
		doc, err := tx.Get(ref)
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}
		if err == nil {
			if err := doc.DataTo(&r); err != nil {
				return err
			}
		}
		next, err := credentiallease.TransitionCredentialLease(r, owner, action, time.Now().UTC())
		if err != nil {
			return err
		}
		return tx.Set(ref, map[string]interface{}{"owner": next.Owner, "heartbeat": next.Heartbeat, "quarantined": next.Quarantined, "credential_resource": key})
	})
}
