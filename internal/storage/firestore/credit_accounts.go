package firestore

import (
	"context"

	"cloud.google.com/go/firestore"
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
