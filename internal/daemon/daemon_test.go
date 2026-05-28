package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/notify"
	"github.com/sunholo-data/ailang/internal/pubsub"
)

// fakeSubscriber simulates Pub/Sub: tests push events into channels, the
// fake delivers them to the registered handlers.
type fakeSubscriber struct {
	mu       sync.Mutex
	handlers map[string]MessageHandler
}

func newFakeSubscriber() *fakeSubscriber {
	return &fakeSubscriber{handlers: make(map[string]MessageHandler)}
}

func (f *fakeSubscriber) Subscribe(ctx context.Context, subName string, handler MessageHandler) error {
	f.mu.Lock()
	f.handlers[subName] = handler
	f.mu.Unlock()
	<-ctx.Done()
	return nil
}

func (f *fakeSubscriber) deliver(t *testing.T, subName string, data []byte, attrs map[string]string) error {
	t.Helper()
	f.mu.Lock()
	h, ok := f.handlers[subName]
	f.mu.Unlock()
	if !ok {
		t.Fatalf("no handler registered for %q", subName)
	}
	return h(context.Background(), data, attrs)
}

// fakeFetcher resolves MessageIDs to InboxMessages for tests.
type fakeFetcher struct {
	msgs map[string]*messaging.InboxMessage
}

func (f *fakeFetcher) Fetch(_ context.Context, messageID string) (*messaging.InboxMessage, error) {
	if f.msgs == nil {
		return nil, nil
	}
	return f.msgs[messageID], nil
}

// recordingNotifier captures Notification calls.
type recordingNotifier struct {
	mu   sync.Mutex
	out  []notify.Notification
	fail error
}

func (r *recordingNotifier) notify(n notify.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fail != nil {
		return r.fail
	}
	r.out = append(r.out, n)
	return nil
}

func (r *recordingNotifier) calls() []notify.Notification {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]notify.Notification, len(r.out))
	copy(out, r.out)
	return out
}

func newTestDaemon(t *testing.T) (*Daemon, *fakeSubscriber, *fakeFetcher, *recordingNotifier) {
	t.Helper()
	sub := newFakeSubscriber()
	fetch := &fakeFetcher{msgs: map[string]*messaging.InboxMessage{}}
	rec := &recordingNotifier{}
	d := New(Config{
		EventsSub:   "events-laptop",
		MessagesSub: "messages-laptop",
		TaskWindow:  60 * time.Second,
		MsgWindow:   5 * time.Minute,
	}, sub, fetch, rec.notify)
	return d, sub, fetch, rec
}

func taskCompletionJSON(t *testing.T, c pubsub.TaskCompletion) []byte {
	t.Helper()
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func msgNotificationJSON(t *testing.T, n pubsub.MessageNotification) []byte {
	t.Helper()
	b, err := json.Marshal(n)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestDaemon_TaskEventFiresNotification(t *testing.T) {
	d, sub, _, rec := newTestDaemon(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = d.Run(ctx) }()
	time.Sleep(20 * time.Millisecond) // let goroutines register handlers

	data := taskCompletionJSON(t, pubsub.TaskCompletion{
		TaskID: "t1", AgentID: "design-doc-creator", Status: "pending_approval",
	})
	if err := sub.deliver(t, "events-laptop", data, nil); err != nil {
		t.Fatalf("deliver: %v", err)
	}
	calls := rec.calls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(calls))
	}
	if calls[0].Title != "⏳ Approval needed" {
		t.Errorf("title = %q", calls[0].Title)
	}
}

func TestDaemon_PublicFeedbackEventFiresDedicatedNotification(t *testing.T) {
	d, sub, fetch, rec := newTestDaemon(t)
	fetch.msgs["msg_pf_1"] = &messaging.InboxMessage{
		MessageID: "msg_pf_1",
		ToInbox:   "public-feedback",
		FromAgent: "mcp-public",
		Title:     "Effect row error needs call site",
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = d.Run(ctx) }()
	time.Sleep(20 * time.Millisecond)

	data := msgNotificationJSON(t, pubsub.MessageNotification{MessageID: "msg_pf_1"})
	if err := sub.deliver(t, "messages-laptop", data, map[string]string{"inbox": "public-feedback"}); err != nil {
		t.Fatalf("deliver: %v", err)
	}
	calls := rec.calls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(calls))
	}
	if calls[0].Title != "🌐 External feedback" {
		t.Errorf("title = %q", calls[0].Title)
	}
}

func TestDaemon_TaskDedupSuppressesRepeats(t *testing.T) {
	d, sub, _, rec := newTestDaemon(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = d.Run(ctx) }()
	time.Sleep(20 * time.Millisecond)

	data := taskCompletionJSON(t, pubsub.TaskCompletion{TaskID: "t1", AgentID: "x", Status: "completed"})
	for i := 0; i < 3; i++ {
		if err := sub.deliver(t, "events-laptop", data, nil); err != nil {
			t.Fatalf("deliver %d: %v", i, err)
		}
	}
	if got := len(rec.calls()); got != 1 {
		t.Errorf("expected 1 notification under dedup, got %d", got)
	}
}

func TestDaemon_DryRunDoesNotInvokeNotifier(t *testing.T) {
	sub := newFakeSubscriber()
	fetch := &fakeFetcher{}
	rec := &recordingNotifier{}
	d := New(Config{
		EventsSub: "events-laptop", MessagesSub: "messages-laptop",
		TaskWindow: 60 * time.Second, MsgWindow: 5 * time.Minute,
		DryRun: true,
	}, sub, fetch, rec.notify)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = d.Run(ctx) }()
	time.Sleep(20 * time.Millisecond)

	data := taskCompletionJSON(t, pubsub.TaskCompletion{TaskID: "t1", AgentID: "x", Status: "completed"})
	if err := sub.deliver(t, "events-laptop", data, nil); err != nil {
		t.Fatalf("deliver: %v", err)
	}
	if got := len(rec.calls()); got != 0 {
		t.Errorf("expected 0 notifier calls in dry-run, got %d", got)
	}
}

func TestDaemon_AckOnlyAfterNotifySuccess(t *testing.T) {
	// When notify fails, the handler must return error so Pub/Sub nacks (retries).
	sub := newFakeSubscriber()
	fetch := &fakeFetcher{}
	notifyErr := errors.New("notify boom")
	rec := &recordingNotifier{fail: notifyErr}
	d := New(Config{
		EventsSub: "events-laptop", MessagesSub: "messages-laptop",
		TaskWindow: 60 * time.Second, MsgWindow: 5 * time.Minute,
	}, sub, fetch, rec.notify)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = d.Run(ctx) }()
	time.Sleep(20 * time.Millisecond)

	data := taskCompletionJSON(t, pubsub.TaskCompletion{TaskID: "t1", AgentID: "x", Status: "completed"})
	err := sub.deliver(t, "events-laptop", data, nil)
	if !errors.Is(err, notifyErr) {
		t.Errorf("expected nack with notify error, got %v", err)
	}
}

// flakyNotifier fails its first failCalls invocations, then succeeds.
type flakyNotifier struct {
	mu        sync.Mutex
	failCalls int
	n         int
	out       []notify.Notification
}

func (f *flakyNotifier) notify(n notify.Notification) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.n++
	if f.n <= f.failCalls {
		return errors.New("transient")
	}
	f.out = append(f.out, n)
	return nil
}

func TestDaemon_NackedEventRetriesAfterForget(t *testing.T) {
	// A delivery whose notify fails must be forgotten from the dedup window so a
	// Pub/Sub redelivery actually re-fires instead of being suppressed.
	sub := newFakeSubscriber()
	fetch := &fakeFetcher{}
	fl := &flakyNotifier{failCalls: 1}
	d := New(Config{
		EventsSub: "events-laptop", MessagesSub: "messages-laptop",
		TaskWindow: 60 * time.Second, MsgWindow: 5 * time.Minute,
	}, sub, fetch, fl.notify)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = d.Run(ctx) }()
	time.Sleep(20 * time.Millisecond)

	data := taskCompletionJSON(t, pubsub.TaskCompletion{TaskID: "t1", AgentID: "x", Status: "completed"})

	// First delivery fails -> nack (error) and the dedup key is forgotten.
	if err := sub.deliver(t, "events-laptop", data, nil); err == nil {
		t.Fatal("expected nack on first (failing) delivery")
	}
	// Redelivery must NOT be deduped — it re-fires and succeeds.
	if err := sub.deliver(t, "events-laptop", data, nil); err != nil {
		t.Fatalf("redelivery should succeed, got %v", err)
	}
	if got := len(fl.out); got != 1 {
		t.Fatalf("expected exactly 1 successful notification after retry, got %d", got)
	}
}

func TestDaemon_ExcludesMatchedNotificationsSkipped(t *testing.T) {
	sub := newFakeSubscriber()
	fetch := &fakeFetcher{}
	rec := &recordingNotifier{}
	d := New(Config{
		EventsSub: "events-laptop", MessagesSub: "messages-laptop",
		TaskWindow: 60 * time.Second, MsgWindow: 5 * time.Minute,
		Excludes: []string{"Approval needed"},
	}, sub, fetch, rec.notify)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = d.Run(ctx) }()
	time.Sleep(20 * time.Millisecond)

	data := taskCompletionJSON(t, pubsub.TaskCompletion{TaskID: "t1", AgentID: "x", Status: "pending_approval"})
	if err := sub.deliver(t, "events-laptop", data, nil); err != nil {
		t.Fatal(err)
	}
	if got := len(rec.calls()); got != 0 {
		t.Errorf("expected 0 calls (excluded), got %d", got)
	}
}

func TestDaemon_GracefulShutdown(t *testing.T) {
	d, _, _, _ := newTestDaemon(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- d.Run(ctx) }()
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("expected nil or context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("daemon did not shut down within 2s of context cancel")
	}
}
