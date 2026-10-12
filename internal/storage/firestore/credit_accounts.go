package firestore

import (
	"context"
	"fmt"
	"sort"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sunholo-data/ailang/internal/creditbudget"
)

// CreditStore is the canonical durable authority shared by every environment
// spending an allocation. Use the same explicit project in dev and prod.
// Requests, tasks, grants and days are subcollections, so no request history is
// accumulated in a growing account document.
// Executors should use the gateway; Firestore IAM is for the authority service.
type CreditStore struct{ client *Client }

func NewCreditStore(client *Client) *CreditStore { return &CreditStore{client: client} }
func NewCreditAuthority(client *Client) *creditbudget.Engine {
	return creditbudget.New(NewCreditStore(client), nil)
}
func (s *CreditStore) Run(ctx context.Context, accountID string, fn func(creditbudget.Transaction) error) error {
	return s.client.RunTransaction(ctx, func(_ context.Context, tx *firestore.Transaction) error {
		return fn(&creditTransaction{tx: tx, account: s.client.Doc("credit_accounts", accountID)})
	})
}

type creditTransaction struct {
	tx      *firestore.Transaction
	account *firestore.DocumentRef
}

func (t *creditTransaction) doc(collection, id string) *firestore.DocumentRef {
	if collection == "account" && id == "current" {
		return t.account
	}
	return t.account.Collection(collection).Doc(id)
}
func (t *creditTransaction) Get(collection, id string, into any) (bool, error) {
	snapshot, err := t.tx.Get(t.doc(collection, id))
	if status.Code(err) == codes.NotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, snapshot.DataTo(into)
}
func (t *creditTransaction) Put(collection, id string, value any) error {
	return t.tx.Set(t.doc(collection, id), value)
}

func (s *CreditStore) ListRequests(ctx context.Context, accountID, taskID string, limit int) ([]creditbudget.Request, error) {
	if limit < 1 || limit > creditbudget.RequestInspectionLimit {
		return nil, fmt.Errorf("invalid request inspection bound")
	}
	iter := s.client.Doc("credit_accounts", accountID).Collection("requests").Where("TaskID", "==", taskID).Limit(limit + 1).Documents(ctx)
	defer iter.Stop()
	rows := []creditbudget.Request{}
	for {
		snapshot, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(rows) == limit {
			return nil, fmt.Errorf("request inspection truncated: more than %d requests", limit)
		}
		var r creditbudget.Request
		if err := snapshot.DataTo(&r); err != nil {
			return nil, err
		}
		rows = append(rows, r)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].RequestID < rows[j].RequestID })
	return rows, nil
}
