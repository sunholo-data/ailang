package firestore

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// quotaOverrideCollection holds one document per subscription bucket (google, codex,
// anthropic). It lives in the plane's own project, beside the credential leases.
const quotaOverrideCollection = "quota_overrides"

// QuotaOverride is an operator's time-boxed lift of one bucket's cloud admission ration.
// It is written only by an attended `ailang mission quota --allow` (refused inside a
// mission iteration) and expires by itself: there is no way to set one without an end.
type QuotaOverride struct {
	Bucket    string    `firestore:"bucket"`
	Until     time.Time `firestore:"until"`
	GrantedAt time.Time `firestore:"granted_at"`
	GrantedBy string    `firestore:"granted_by"`
	Reason    string    `firestore:"reason"`
}

// Active reports whether the override still lifts the ration at now.
func (o *QuotaOverride) Active(now time.Time) bool {
	return o != nil && now.Before(o.Until)
}

// QuotaOverrideStore reads and writes quota overrides.
type QuotaOverrideStore struct{ client *Client }

// NewQuotaOverrideStore wraps an open client.
func NewQuotaOverrideStore(client *Client) *QuotaOverrideStore {
	return &QuotaOverrideStore{client: client}
}

// Get returns the bucket's override, or nil when there is none (expired ones included:
// the caller decides with Active).
func (s *QuotaOverrideStore) Get(ctx context.Context, bucket string) (*QuotaOverride, error) {
	if bucket == "" {
		return nil, fmt.Errorf("quota override bucket is required")
	}
	doc, err := s.client.Doc(quotaOverrideCollection, bucket).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var o QuotaOverride
	if err := doc.DataTo(&o); err != nil {
		return nil, err
	}
	return &o, nil
}

// Put writes the bucket's override, replacing any earlier one.
func (s *QuotaOverrideStore) Put(ctx context.Context, o QuotaOverride) error {
	if o.Bucket == "" || o.Until.IsZero() {
		return fmt.Errorf("quota override needs a bucket and an end time")
	}
	_, err := s.client.Doc(quotaOverrideCollection, o.Bucket).Set(ctx, o)
	return err
}

// Delete removes the bucket's override; removing a missing one is not an error.
func (s *QuotaOverrideStore) Delete(ctx context.Context, bucket string) error {
	_, err := s.client.Doc(quotaOverrideCollection, bucket).Delete(ctx)
	if status.Code(err) == codes.NotFound {
		return nil
	}
	return err
}
