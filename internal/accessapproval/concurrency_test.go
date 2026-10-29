package accessapproval

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

func TestConcurrentExecuteProducesSingleDurableReceipt(t *testing.T) {
	dir := t.TempDir()
	a := service(t, dir)
	b := service(t, dir)
	r := approved(t, a)
	in := binding(r)
	r = record(t, a, "claim", in)
	in.ClaimID = r.Execution.ClaimID
	startRevision := r.Revision
	var calls atomic.Int32
	countDummy := func(r *Request) map[string]any { calls.Add(1); return dummyOperation(r) }
	a.runDummy = countDummy
	b.runDummy = countDummy
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s := a
			if i%2 == 1 {
				s = b
			}
			if _, err := s.Handle(context.Background(), "execute", in); err != nil {
				t.Errorf("execute: %v", err)
			}
		}(i)
	}
	wg.Wait()
	r = record(t, a, "receipt", Input{ID: r.ID})
	if r.Execution.State != "completed" || r.Revision != startRevision+2 || calls.Load() != 1 {
		t.Fatalf("expected one start and one receipt, got %+v", r)
	}
}

func TestPendingCapacityAtomicAndLegacyIsolation(t *testing.T) {
	dir := t.TempDir()
	s := service(t, dir)
	other := service(t, dir)
	ctx := context.Background()
	first := record(t, s, "request", input())
	for i := 1; i < 99; i++ {
		in := input()
		in.Attempt = string(rune(1000 + i))
		record(t, s, "request", in)
	}
	var wins atomic.Int32
	var wg sync.WaitGroup
	for i, service := range []*Service{s, other} {
		wg.Add(1)
		go func(i int, service *Service) {
			defer wg.Done()
			in := input()
			in.Attempt = []string{"last-slot-a", "last-slot-b"}[i]
			if _, err := service.Handle(ctx, "request", in); err == nil {
				wins.Add(1)
			}
		}(i, service)
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("capacity race admitted %d requests into last slot", wins.Load())
	}
	// At capacity an identical delivery still returns the existing request.
	if record(t, s, "request", input()).ID != first.ID {
		t.Fatal("replay rejected at capacity")
	}
	in := input()
	in.Attempt = "overflow"
	if _, err := s.Handle(ctx, "request", in); err == nil {
		t.Fatal("pending capacity exceeded")
	}
	legacy, err := s.store.ListPendingApprovals(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(legacy) != 0 {
		t.Fatalf("durable requests reached legacy watcher: %d", len(legacy))
	}
	record(t, s, "decide", Input{ID: first.ID, Actor: "mark", Version: 1, Decision: "declined", IdempotencyKey: "free-slot"})
	record(t, s, "request", in)
}

func TestCancelAndExpiredClaimCannotRun(t *testing.T) {
	s := service(t, t.TempDir())
	now := s.now()
	s.now = func() int64 { return now }
	r := approved(t, s)
	in := binding(r)
	r = record(t, s, "claim", in)
	in.ClaimID = r.Execution.ClaimID
	now += 301
	r = record(t, s, "execute", in)
	if r.Execution.State != "failed" || r.Execution.StartedAt != 0 {
		t.Fatalf("expired claim executed: %+v", r)
	}
	other := input()
	other.Attempt = "cancel-started"
	r = record(t, s, "request", other)
	r = record(t, s, "decide", Input{ID: r.ID, Actor: "mark", Version: 1, Decision: "approved", IdempotencyKey: "approve"})
	in = binding(r)
	r = record(t, s, "claim", in)
	in.ClaimID = r.Execution.ClaimID
	if _, err := s.mutate(context.Background(), r.ID, func(r *Request) error { r.Execution.State = "started"; return nil }); err != nil {
		t.Fatal(err)
	}
	r = record(t, s, "cancel", Input{ID: r.ID, Actor: "mark", Version: 1, Reason: "task cancelled"})
	if r.Execution.State != "unknown" {
		t.Fatal(r)
	}
	if _, err := s.Handle(context.Background(), "execute", in); err == nil {
		t.Fatal("cancelled claim accepted")
	}
}

func TestExpiredExplicitRequestReplayAndBoundedList(t *testing.T) {
	s := service(t, t.TempDir())
	now := s.now()
	s.now = func() int64 { return now }
	in := input()
	in.ExpiresAt = now + 5
	r := record(t, s, "request", in)
	now += 6
	replay := record(t, s, "request", in)
	if replay.ID != r.ID || replay.Decision != "expired" {
		t.Fatal(replay)
	}
	for i := 0; i < 3; i++ {
		next := input()
		next.Attempt = string(rune(2000 + i))
		record(t, s, "request", next)
	}
	out, err := s.Handle(context.Background(), "list", Input{Owner: "mark", Status: "pending", Limit: 2, Offset: 1})
	if err != nil {
		t.Fatal(err)
	}
	page := out.(*List)
	if page.Total != 3 || len(page.Requests) != 2 {
		t.Fatalf("unexpected bounded pending page: %+v", page)
	}
	out, err = s.Handle(context.Background(), "list", Input{Owner: "mark", Status: "history", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	page = out.(*List)
	if page.Total != 1 || len(page.Requests) != 1 || page.Requests[0].Decision != "expired" {
		t.Fatal(page)
	}
}

func TestPendingPageOldestFirst(t *testing.T) {
	s := service(t, t.TempDir())
	now := s.now()
	s.now = func() int64 { return now }
	ids := []string{}
	for i := 0; i < 3; i++ {
		in := input()
		in.Attempt = []string{"oldest", "middle", "newest"}[i]
		ids = append(ids, record(t, s, "request", in).ID)
		now++
	}
	out, err := s.Handle(context.Background(), "list", Input{Owner: "mark", Status: "pending", Limit: 1, Offset: 1})
	if err != nil {
		t.Fatal(err)
	}
	page := out.(*List)
	if page.Total != 3 || len(page.Requests) != 1 || page.Requests[0].ID != ids[1] {
		t.Fatalf("pending queue page lost oldest order: %+v", page)
	}
	out, err = s.Handle(context.Background(), "list", Input{Owner: "mark", Status: "pending", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if out.(*List).Requests[0].ID != ids[0] {
		t.Fatal("oldest pending request starved")
	}
}

func TestCancellationDuringExecutionKeepsUnknownReceipt(t *testing.T) {
	dir := t.TempDir()
	s := service(t, dir)
	operator := service(t, dir)
	r := approved(t, s)
	in := binding(r)
	r = record(t, s, "claim", in)
	in.ClaimID = r.Execution.ClaimID
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	s.runDummy = func(r *Request) map[string]any { close(started); <-release; return dummyOperation(r) }
	type outcome struct {
		record any
		err    error
	}
	done := make(chan outcome, 1)
	go func() { out, err := s.Handle(context.Background(), "execute", in); done <- outcome{out, err} }()
	<-started
	cancelled := record(t, operator, "cancel", Input{ID: r.ID, Actor: "mark", Version: 1, Reason: "task ended during execution"})
	if cancelled.Execution.State != "unknown" {
		t.Fatal(cancelled)
	}
	unblock()
	out := <-done
	if out.err != nil {
		t.Fatal(out.err)
	}
	final := out.record.(*Request)
	if final.Decision != "cancelled" || final.Execution.State != "unknown" || final.Execution.Result != nil {
		t.Fatalf("late completion overwrote cancellation: %+v", final)
	}
}
