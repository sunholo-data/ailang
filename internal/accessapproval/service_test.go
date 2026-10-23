package accessapproval

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func service(t *testing.T, dir string) *Service {
	t.Helper()
	s, err := Open(filepath.Join(dir, "runtime"), "daneel-local")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}
func input() Input {
	return Input{Owner: "mark", Worker: "machine:daneel", Task: "task-1", Attempt: "attempt-1", Operation: "dummy", ResourceAlias: "canary", Purpose: "Test remote approval", Arguments: map[string]any{"message": "hello"}, PolicyVersion: "1", Version: 1}
}
func record(t *testing.T, s *Service, cmd string, in Input) *Request {
	t.Helper()
	out, err := s.Handle(context.Background(), cmd, in)
	if err != nil {
		t.Fatalf("%s: %v", cmd, err)
	}
	return out.(*Request)
}
func binding(r *Request) Input {
	return Input{ID: r.ID, Worker: r.Worker, Task: r.Task, Attempt: r.Attempt, Operation: r.Operation, ArgumentsDigest: r.ArgumentsDigest, PolicyVersion: r.PolicyVersion, Version: r.Version, IdempotencyKey: "claim-1"}
}
func approved(t *testing.T, s *Service) *Request {
	r := record(t, s, "request", input())
	return record(t, s, "decide", Input{ID: r.ID, Actor: "mark", Version: 1, Decision: "approved", IdempotencyKey: "decision-1"})
}
func TestDurableLifecycleReopenAndReplay(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "operator")
	s := service(t, dir)
	r := approved(t, s)
	in := binding(r)
	claimed := record(t, s, "claim", in)
	if claimed.Execution.State != "claimed" || claimed.Execution.ClaimID == "" {
		t.Fatalf("bad claim: %+v", claimed)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s = service(t, dir)
	in.ClaimID = claimed.Execution.ClaimID
	done := record(t, s, "execute", in)
	if done.Execution.State != "completed" || done.Execution.Result["message"] != "hello" {
		t.Fatalf("bad receipt: %+v", done)
	}
	again := record(t, s, "execute", in)
	if again.Revision != done.Revision {
		t.Fatal("execution replay changed receipt")
	}
	receipt := record(t, s, "receipt", Input{ID: r.ID})
	if receipt.Execution.State != "completed" {
		t.Fatal(receipt)
	}
}
func TestRequestReplayAndBinding(t *testing.T) {
	s := service(t, t.TempDir())
	in := input()
	r := record(t, s, "request", in)
	if record(t, s, "request", in).ID != r.ID {
		t.Fatal("request not idempotent")
	}
	in.Arguments["message"] = "changed"
	if _, err := s.Handle(context.Background(), "request", in); err == nil {
		t.Fatal("changed arguments replay accepted")
	}
	r = record(t, s, "decide", Input{ID: r.ID, Actor: "mark", Version: 1, Decision: "approved", IdempotencyKey: "d"})
	b := binding(r)
	b.Task = "other"
	if _, err := s.Handle(context.Background(), "claim", b); err == nil {
		t.Fatal("changed task accepted")
	}
	b = binding(r)
	b.Version = 2
	if _, err := s.Handle(context.Background(), "claim", b); err == nil {
		t.Fatal("changed version accepted")
	}
	if _, err := s.Handle(context.Background(), "decide", Input{ID: r.ID, Actor: "other", Version: 1, Decision: "approved", IdempotencyKey: "d"}); err == nil {
		t.Fatal("wrong owner accepted")
	}
}
func TestConcurrentDecisionAndClaim(t *testing.T) {
	dir := t.TempDir()
	a := service(t, dir)
	b := service(t, dir)
	r := record(t, a, "request", input())
	var wins atomic.Int32
	var wg sync.WaitGroup
	for i, s := range []*Service{a, b} {
		wg.Add(1)
		go func(i int, s *Service) {
			defer wg.Done()
			d := "approved"
			if i == 1 {
				d = "declined"
			}
			_, err := s.Handle(context.Background(), "decide", Input{ID: r.ID, Actor: "mark", Version: 1, Decision: d, IdempotencyKey: d})
			if err == nil {
				wins.Add(1)
			}
		}(i, s)
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("decision winners=%d", wins.Load())
	}
	// Use a fresh attempt to ensure approved state regardless of the race winner.
	in := input()
	in.Attempt = "attempt-2"
	r = record(t, a, "request", in)
	r = record(t, a, "decide", Input{ID: r.ID, Actor: "mark", Version: 1, Decision: "approved", IdempotencyKey: "d"})
	wins.Store(0)
	for i, s := range []*Service{a, b} {
		wg.Add(1)
		go func(i int, s *Service) {
			defer wg.Done()
			in := binding(r)
			if i == 1 {
				in.IdempotencyKey = "other"
			}
			_, err := s.Handle(context.Background(), "claim", in)
			if err == nil {
				wins.Add(1)
			}
		}(i, s)
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("claim winners=%d", wins.Load())
	}
}
func TestExpiryCancellationAndUnknownNeverRetry(t *testing.T) {
	s := service(t, t.TempDir())
	now := time.Now().Unix()
	s.now = func() int64 { return now }
	r := approved(t, s)
	b := binding(r)
	r = record(t, s, "claim", b)
	b.ClaimID = r.Execution.ClaimID
	// A persisted start with no receipt represents process death or uncertain effect.
	_, err := s.mutate(context.Background(), r.ID, func(r *Request) error { r.Execution.State = "started"; r.Execution.StartedAt = now; return nil })
	if err != nil {
		t.Fatal(err)
	}
	now += 301
	r = record(t, s, "execute", b)
	if r.Execution.State != "unknown" {
		t.Fatalf("uncertain effect retried: %+v", r)
	}
	if record(t, s, "execute", b).Execution.State != "unknown" {
		t.Fatal("unknown became executable")
	}
	in := input()
	in.Attempt = "expiry"
	in.ExpiresAt = now + 10
	r = record(t, s, "request", in)
	now += 11
	r = record(t, s, "get", Input{ID: r.ID})
	if r.Decision != "expired" {
		t.Fatal(r)
	}
	in = input()
	in.Attempt = "cancel"
	r = record(t, s, "request", in)
	r = record(t, s, "cancel", Input{ID: r.ID, Actor: r.Worker, Version: 1, Reason: "task ended"})
	if r.Decision != "cancelled" {
		t.Fatal(r)
	}
	if _, err := s.Handle(context.Background(), "decide", Input{ID: r.ID, Actor: "mark", Version: 1, Decision: "approved", IdempotencyKey: "late"}); err == nil {
		t.Fatal("cancelled decision accepted")
	}
}
func TestListScopeAndFixedOperation(t *testing.T) {
	s := service(t, t.TempDir())
	r := record(t, s, "request", input())
	out, err := s.Handle(context.Background(), "list", Input{Owner: "someone-else", Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if out.(*List).Total != 0 {
		t.Fatal("owner scope leaked")
	}
	out, err = s.Handle(context.Background(), "list", Input{Owner: "mark", Worker: r.Worker, Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if out.(*List).Total != 1 {
		t.Fatal(out)
	}
	for _, op := range []string{"shell", "op-read", "item-read"} {
		in := input()
		in.Operation = op
		if _, err := s.Handle(context.Background(), "request", in); err == nil {
			t.Fatalf("operation %s enabled", op)
		}
	}
	in := input()
	in.Arguments["ref"] = "op://vault/item/password"
	if _, err := s.Handle(context.Background(), "request", in); err == nil {
		t.Fatal("unregistered argument accepted")
	}
	raw, _ := json.Marshal(r)
	if len(raw) == 0 {
		t.Fatal("empty canonical record")
	}
}
func TestExplicitLocalStoreRequired(t *testing.T) {
	for _, tc := range [][2]string{{"", "daneel-local"}, {"relative", "daneel-local"}, {t.TempDir(), "gcp"}, {t.TempDir(), ""}} {
		s, err := Open(tc[0], tc[1])
		if s != nil {
			_ = s.Close()
		}
		if err == nil {
			t.Fatalf("unsafe store accepted: %v", tc)
		}
	}
}
