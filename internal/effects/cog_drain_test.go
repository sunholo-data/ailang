package effects

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval"
)

// fakeCallable is a stand-in for an AILANG closure (eval.Value) that
// records each invocation count + argument for assertions. The
// callback shape matches what ctx.FnCaller expects to dispatch.
type fakeCallable struct {
	calls   int32
	lastArg eval.Value
}

// Satisfy the eval.Value interface (closures are values).
func (f *fakeCallable) String() string { return "<fakeCallable>" }
func (f *fakeCallable) Type() string   { return "fakeCallable" }

// fakeFnCaller mimics the evaluator's FnCaller bridge. It type-asserts
// the closure to *fakeCallable + records the invocation.
func fakeFnCaller(fn eval.Value, arg eval.Value) (eval.Value, error) {
	fc, ok := fn.(*fakeCallable)
	if !ok {
		return nil, nil
	}
	atomic.AddInt32(&fc.calls, 1)
	fc.lastArg = arg
	return &eval.UnitValue{}, nil
}

func TestCogDrain_EmptyQueue_TimeoutZero_ReturnsZero(t *testing.T) {
	ctx := &EffContext{
		Cog:      NewCogContext(),
		FnCaller: fakeFnCaller,
	}
	res, err := cogDrain(ctx, []eval.Value{&eval.IntValue{Value: 0}})
	if err != nil {
		t.Fatalf("cogDrain: %v", err)
	}
	count := res.(*eval.IntValue).Value
	if count != 0 {
		t.Errorf("empty queue with timeout=0: got count=%d, want 0", count)
	}
}

func TestCogDrain_PendingItems_DispatchesAllAndReturnsCount(t *testing.T) {
	cog := NewCogContext()
	ctx := &EffContext{Cog: cog, FnCaller: fakeFnCaller}

	fc1 := &fakeCallable{}
	fc2 := &fakeCallable{}
	cog.Enqueue(fc1, &eval.StringValue{Value: "event-a"})
	cog.Enqueue(fc2, &eval.StringValue{Value: "event-b"})

	res, err := cogDrain(ctx, []eval.Value{&eval.IntValue{Value: 0}})
	if err != nil {
		t.Fatalf("cogDrain: %v", err)
	}
	count := res.(*eval.IntValue).Value
	if count != 2 {
		t.Errorf("drain count: got %d, want 2", count)
	}
	if atomic.LoadInt32(&fc1.calls) != 1 || atomic.LoadInt32(&fc2.calls) != 1 {
		t.Errorf("each callback should fire once, got %d / %d", fc1.calls, fc2.calls)
	}
	if s, ok := fc1.lastArg.(*eval.StringValue); !ok || s.Value != "event-a" {
		t.Errorf("fc1 should receive event-a, got %v", fc1.lastArg)
	}
}

func TestCogDrain_BlocksOnTimeout_UnblocksOnEnqueue(t *testing.T) {
	cog := NewCogContext()
	ctx := &EffContext{Cog: cog, FnCaller: fakeFnCaller}
	fc := &fakeCallable{}

	resultChan := make(chan int, 1)
	go func() {
		// 500ms timeout — we'll enqueue before that fires
		res, _ := cogDrain(ctx, []eval.Value{&eval.IntValue{Value: 500}})
		resultChan <- res.(*eval.IntValue).Value
	}()

	// Let drain park, then enqueue
	time.Sleep(50 * time.Millisecond)
	cog.Enqueue(fc, &eval.StringValue{Value: "wake"})

	select {
	case count := <-resultChan:
		if count != 1 {
			t.Errorf("expected 1 dispatch after wake, got %d", count)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("drain did not unblock after enqueue")
	}
}

func TestCogDrain_NoFnCaller_Errors(t *testing.T) {
	ctx := &EffContext{Cog: NewCogContext()}
	_, err := cogDrain(ctx, []eval.Value{&eval.IntValue{Value: 0}})
	if err == nil {
		t.Fatal("expected error when FnCaller is nil")
	}
}

func TestCogDrain_NoCogContext_Errors(t *testing.T) {
	ctx := &EffContext{FnCaller: fakeFnCaller}
	_, err := cogDrain(ctx, []eval.Value{&eval.IntValue{Value: 0}})
	if err == nil {
		t.Fatal("expected error when Cog context is nil")
	}
}

func TestCogContext_Enqueue_ConcurrentSafe(t *testing.T) {
	cog := NewCogContext()
	const goroutines = 20
	const perG = 50
	done := make(chan struct{}, goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			for i := 0; i < perG; i++ {
				cog.Enqueue(&fakeCallable{}, &eval.IntValue{Value: i})
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < goroutines; i++ {
		<-done
	}
	pending := cog.drainPending()
	if len(pending) != goroutines*perG {
		t.Errorf("expected %d pending items, got %d", goroutines*perG, len(pending))
	}
}

func TestDOMSubscribe_EnqueuesOnFireEvent(t *testing.T) {
	stub := NewStubDOMHandler()
	cog := NewCogContext()
	ctx := &EffContext{
		DOM:      NewDOMContext(stub),
		Cog:      cog,
		FnCaller: fakeFnCaller,
	}

	fc := &fakeCallable{}
	res, err := domSubscribe(ctx, []eval.Value{
		&eval.StringValue{Value: "agent_a"},
		&eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: "click"}}},
		fc,
	})
	if err != nil {
		t.Fatalf("domSubscribe: %v", err)
	}
	if _, ok := res.(*eval.UnitValue); !ok {
		t.Errorf("subscribe should return Unit, got %T", res)
	}

	// Fire an event via the stub — should enqueue into ctx.Cog
	stub.FireEvent("agent_a", EventClick{Node: "n_42"})

	// Drain — callback should fire
	drainRes, err := cogDrain(ctx, []eval.Value{&eval.IntValue{Value: 0}})
	if err != nil {
		t.Fatal(err)
	}
	if drainRes.(*eval.IntValue).Value != 1 {
		t.Errorf("expected 1 drained event, got %d", drainRes.(*eval.IntValue).Value)
	}
	if atomic.LoadInt32(&fc.calls) != 1 {
		t.Errorf("callback should fire once after drain, got %d", fc.calls)
	}
	rec, ok := fc.lastArg.(*eval.RecordValue)
	if !ok {
		t.Fatalf("callback arg should be RecordValue, got %T", fc.lastArg)
	}
	if k, _ := rec.Fields["kind"].(*eval.StringValue); k == nil || k.Value != "Click" {
		t.Errorf("expected kind=Click in callback arg, got %v", rec.Fields["kind"])
	}
}

func TestMsgSubscribe_EnqueuesOnFireMessage(t *testing.T) {
	stub := NewStubMsgHandler("node_a")
	cog := NewCogContext()
	ctx := &EffContext{
		Msg:      NewMsgContext(stub, "node_a"),
		Cog:      cog,
		FnCaller: fakeFnCaller,
	}

	fc := &fakeCallable{}
	if _, err := msgSubscribe(ctx, []eval.Value{
		&eval.StringValue{Value: "inbox_b"},
		fc,
	}); err != nil {
		t.Fatalf("msgSubscribe: %v", err)
	}

	// Send a message — stub delivers to subscribers
	if _, err := stub.Send("inbox_b", []byte("hello")); err != nil {
		t.Fatal(err)
	}

	// Drain
	drainRes, err := cogDrain(ctx, []eval.Value{&eval.IntValue{Value: 100}})
	if err != nil {
		t.Fatal(err)
	}
	if drainRes.(*eval.IntValue).Value != 1 {
		t.Errorf("expected 1 drained event, got %d", drainRes.(*eval.IntValue).Value)
	}
	rec := fc.lastArg.(*eval.RecordValue)
	if p, _ := rec.Fields["payload"].(*eval.StringValue); p == nil || p.Value != "hello" {
		t.Errorf("expected payload=hello, got %v", rec.Fields["payload"])
	}
}
