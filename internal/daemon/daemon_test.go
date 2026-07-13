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

// prodOnlyFetcher resolves a message ID to a message ONLY if the ID belongs to
// the prod set — proving in-test that a source's fetcher is scoped to its own
// project's store (a dev fetcher would return nil for a prod ID and vice versa).
type prodOnlyFetcher struct {
	prodMsgs map[string]*messaging.InboxMessage
}

func (f *prodOnlyFetcher) Fetch(_ context.Context, messageID string) (*messaging.InboxMessage, error) {
	if f.prodMsgs == nil {
		return nil, nil
	}
	return f.prodMsgs[messageID], nil // nil for any non-prod id
}

// TestDaemon_DualProjectMessageFiresOnce proves that with a prod message source
// added, a message delivered on the prod source fires exactly one notification,
// resolved via the prod-scoped fetcher.
func TestDaemon_DualProjectMessageFiresOnce(t *testing.T) {
	// Primary (dev) source.
	devSub := newFakeSubscriber()
	devFetch := &fakeFetcher{msgs: map[string]*messaging.InboxMessage{}}
	rec := &recordingNotifier{}
	d := New(Config{
		EventsSub:   "events-laptop",
		MessagesSub: "messages-laptop",
		TaskWindow:  60 * time.Second,
		MsgWindow:   5 * time.Minute,
	}, devSub, devFetch, rec.notify)

	// Prod source: its OWN subscriber + a prod-scoped fetcher that only knows
	// prod message IDs.
	prodSub := newFakeSubscriber()
	prodFetch := &prodOnlyFetcher{prodMsgs: map[string]*messaging.InboxMessage{
		"fb_prod_1": {
			MessageID: "fb_prod_1",
			ToInbox:   "public-feedback",
			FromAgent: "mcp-public",
			Title:     "prod feedback from a real user",
		},
	}}
	d.AddMessageSource(MessageSource{
		Sub:     prodSub,
		Fetcher: prodFetch,
		SubName: "messages-laptop",
		Label:   "prod",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = d.Run(ctx) }()
	time.Sleep(30 * time.Millisecond) // let all sources register handlers

	data := msgNotificationJSON(t, pubsub.MessageNotification{MessageID: "fb_prod_1"})
	if err := prodSub.deliver(t, "messages-laptop", data, map[string]string{"inbox": "public-feedback"}); err != nil {
		t.Fatalf("prod deliver: %v", err)
	}

	calls := rec.calls()
	if len(calls) != 1 {
		t.Fatalf("expected exactly 1 notification from prod source, got %d", len(calls))
	}
	if calls[0].Title != "🌐 External feedback" {
		t.Errorf("title = %q, want 🌐 External feedback", calls[0].Title)
	}
	if calls[0].EventType != "public-feedback" {
		t.Errorf("EventType = %q, want public-feedback", calls[0].EventType)
	}

	// Dedup: a duplicate prod delivery is suppressed.
	if err := prodSub.deliver(t, "messages-laptop", data, nil); err != nil {
		t.Fatalf("prod redeliver: %v", err)
	}
	if got := len(rec.calls()); got != 1 {
		t.Errorf("expected dedup to suppress duplicate prod message, got %d calls", got)
	}
}

// TestDaemon_ProdFetcherScopedToProdStore proves the prod source resolves
// against its OWN store: a prod message ID resolves via the prod fetcher, but
// the SAME id delivered on the dev source (whose fetcher does not know it)
// nacks (not-yet-visible) rather than firing — the fetchers do not collide.
func TestDaemon_ProdFetcherScopedToProdStore(t *testing.T) {
	devSub := newFakeSubscriber()
	devFetch := &fakeFetcher{msgs: map[string]*messaging.InboxMessage{}} // dev knows nothing
	rec := &recordingNotifier{}
	d := New(Config{
		EventsSub: "events-laptop", MessagesSub: "messages-laptop",
		TaskWindow: 60 * time.Second, MsgWindow: 5 * time.Minute,
	}, devSub, devFetch, rec.notify)

	prodSub := newFakeSubscriber()
	prodFetch := &prodOnlyFetcher{prodMsgs: map[string]*messaging.InboxMessage{
		"fb_prod_2": {MessageID: "fb_prod_2", ToInbox: "public-feedback", FromAgent: "mcp-public", Title: "prod"},
	}}
	d.AddMessageSource(MessageSource{Sub: prodSub, Fetcher: prodFetch, SubName: "messages-laptop", Label: "prod"})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = d.Run(ctx) }()
	time.Sleep(30 * time.Millisecond)

	data := msgNotificationJSON(t, pubsub.MessageNotification{MessageID: "fb_prod_2"})

	// Delivered on the DEV source: dev fetcher returns nil -> handler nacks.
	if err := devSub.deliver(t, "messages-laptop", data, nil); err == nil {
		t.Fatal("expected dev source to nack a prod-only message id (fetcher scoping)")
	}
	if got := len(rec.calls()); got != 0 {
		t.Fatalf("dev source must not fire for a prod id, got %d", got)
	}

	// Delivered on the PROD source: prod fetcher resolves it -> fires once.
	if err := prodSub.deliver(t, "messages-laptop", data, nil); err != nil {
		t.Fatalf("prod deliver: %v", err)
	}
	if got := len(rec.calls()); got != 1 {
		t.Fatalf("expected prod source to fire once, got %d", got)
	}
}

// TestDaemon_SingleProjectUnchanged proves that a daemon built with New (no
// added sources) behaves exactly as before: the single messages-laptop source
// fires from the primary subscriber, and task events still flow.
func TestDaemon_SingleProjectUnchanged(t *testing.T) {
	d, sub, fetch, rec := newTestDaemon(t)
	fetch.msgs["msg_single_1"] = &messaging.InboxMessage{
		MessageID: "msg_single_1",
		ToInbox:   "public-feedback",
		FromAgent: "nightly-eval",
		Title:     "regression fade",
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = d.Run(ctx) }()
	time.Sleep(20 * time.Millisecond)

	// Message on the single (primary) source fires.
	data := msgNotificationJSON(t, pubsub.MessageNotification{MessageID: "msg_single_1"})
	if err := sub.deliver(t, "messages-laptop", data, nil); err != nil {
		t.Fatalf("deliver: %v", err)
	}
	if got := len(rec.calls()); got != 1 {
		t.Fatalf("expected 1 notification, got %d", got)
	}

	// Task events still flow on the same primary subscriber.
	tdata := taskCompletionJSON(t, pubsub.TaskCompletion{TaskID: "t1", AgentID: "x", Status: "pending_approval"})
	if err := sub.deliver(t, "events-laptop", tdata, nil); err != nil {
		t.Fatalf("task deliver: %v", err)
	}
	if got := len(rec.calls()); got != 2 {
		t.Fatalf("expected 2 notifications (msg+task), got %d", got)
	}
}

func TestResolveExtraMessageSources(t *testing.T) {
	// Basic: env=dev + extra prod -> one prod source.
	got, err := ResolveExtraMessageSources("dev", []string{"prod"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Env != "prod" || got[0].Project != "ailang-multivac" ||
		got[0].Prefix != "ailang" || got[0].MessagesSub != "messages-laptop" {
		t.Fatalf("got %+v", got)
	}

	// Dedup against primary: env=prod + extra prod -> no extra source.
	got, err = ResolveExtraMessageSources("prod", []string{"prod"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected extra prod to dedup against primary prod, got %+v", got)
	}

	// Dedup against itself + blank-skip: [prod, , prod] -> one prod.
	got, err = ResolveExtraMessageSources("dev", []string{"prod", "", "prod"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected self-dedup to yield 1 source, got %+v", got)
	}

	// Unknown env fails loudly.
	if _, err := ResolveExtraMessageSources("dev", []string{"staging"}); err == nil {
		t.Error("expected error for unknown env label")
	}

	// Nil/empty extras -> nil, backward compatible.
	got, err = ResolveExtraMessageSources("dev", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected no extra sources for nil, got %+v", got)
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
