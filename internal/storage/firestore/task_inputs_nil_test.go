package firestore

import (
	"context"
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
)

func TestTaskInputsPutMessageIfAbsentNilGuard(t *testing.T) {
	s := &MessagingStore{}
	for _, msg := range []*messaging.InboxMessage{nil, {}} {
		created, err := s.PutMessageIfAbsent(context.Background(), msg)
		if err == nil || created {
			t.Fatal("invalid message must fail before backend access")
		}
	}
}
