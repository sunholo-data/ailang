package main

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestEvalRigGateDrainsAllTrialsBeforeYield(t *testing.T) {
	var pending atomic.Bool
	yielded := make(chan struct{})
	resume := make(chan struct{})
	g := &evalRigGate{pending: pending.Load, yield: func(context.Context) error { close(yielded); <-resume; return nil }}
	first, err := g.enter(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := g.enter(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	pending.Store(true)
	started := make(chan struct{})
	go func() {
		release, err := g.enter(context.Background())
		if err == nil {
			close(started)
			release()
		}
	}()
	first()
	select {
	case <-yielded:
		t.Fatal("yielded with second trial still running")
	case <-time.After(30 * time.Millisecond):
	}
	second()
	select {
	case <-yielded:
	case <-time.After(time.Second):
		t.Fatal("did not yield after drain")
	}
	select {
	case <-started:
		t.Fatal("new trial ran during priority work")
	default:
	}
	pending.Store(false)
	close(resume)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("did not resume")
	}
}

func TestEvalRigGateFailsClosedAfterHandoffError(t *testing.T) {
	want := errors.New("lost lock")
	var calls int
	g := &evalRigGate{pending: func() bool { return true }, yield: func(context.Context) error { calls++; return want }}
	for i := 0; i < 2; i++ {
		release, err := g.enter(context.Background())
		release()
		if !errors.Is(err, want) {
			t.Fatalf("got %v", err)
		}
	}
	if calls != 1 {
		t.Fatalf("retried failed handoff %d times", calls)
	}
}
