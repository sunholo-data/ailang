package trace

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCollectorConcurrentCleanupAndSnapshots(t *testing.T) {
	collector := NewCollector()
	collector.RecordModuleStart("run", []string{"IO", "Process"})
	var tasks sync.WaitGroup
	tasks.Add(3)
	go func() {
		defer tasks.Done()
		for i := 0; i < 400; i++ {
			collector.RecordFunctionEnter("eval", []string{"value"})
			collector.RecordEffect("IO", "println", nil, "()")
			collector.RecordFunctionExit("eval", "()")
		}
	}()
	go func() {
		defer tasks.Done()
		for i := 0; i < 400; i++ {
			collector.RecordEffect("Process", "shutdownWorkers", []string{"joined"}, "()")
			for _, event := range collector.Events() {
				if event.Effect != nil {
					_ = len(event.Effect.Args)
				}
				_ = event.SpanID
			}
		}
	}()
	go func() {
		defer tasks.Done()
		for i := 0; i < 400; i++ {
			collector.SetLimits(64, 4000)
			collector.SetValueMode(ValueMode(i % 2))
			collector.ValueBudget()
			collector.Enabled()
			collector.DroppedEvents()
			collector.RecordsFunctionCalls()
			collector.BaseTime()
		}
	}()
	tasks.Wait()
	collector.RecordModuleEnd("run", 0)
	if len(collector.Events()) == 0 {
		t.Fatal("no cleanup/evaluation trace retained")
	}
}

func TestCollectorPublishedPayloadsAreIndependent(t *testing.T) {
	collector := NewCollector()
	args := []string{"original"}
	route := &ResolvedRoute{ResolvedModel: "model", FallbackChain: []string{"provider"}}
	collector.RecordAIEffect("call", args, "result", route)
	args[0] = "changed input"
	route.ResolvedModel = "changed model"
	route.FallbackChain[0] = "changed chain"
	first := collector.Events()
	if first[0].Effect.Args[0] != "original" || first[0].Effect.Route.ResolvedModel != "model" || first[0].Effect.Route.FallbackChain[0] != "provider" {
		t.Fatal("record retained mutable caller data")
	}
	first[0].Effect.Args[0] = "changed snapshot"
	first[0].Effect.Route.FallbackChain[0] = "changed snapshot chain"
	first[0].Event = EventError
	second := collector.Events()
	if second[0].Event != EventEffect || second[0].Effect.Args[0] != "original" || second[0].Effect.Route.FallbackChain[0] != "provider" {
		t.Fatal("snapshot mutated collector state")
	}
}

func TestCollectorObserverCanReenterWithoutChangingRetainedPayload(t *testing.T) {
	collector := NewCollector()
	collector.OnEvent = func(event TraceEvent) {
		if event.Effect.OpName != "outer" {
			return
		}
		_ = collector.Events()
		event.Effect.Args[0] = "observer mutation"
		collector.RecordEffect("Process", "inner", nil, "joined")
	}
	finished := make(chan struct{})
	go func() { collector.RecordEffect("Process", "outer", []string{"original"}, "()"); close(finished) }()
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("observer reentry deadlocked collector")
	}
	events := collector.Events()
	if len(events) != 2 {
		t.Fatalf("observer reentry retained %d events", len(events))
	}
	for _, event := range events {
		if event.Effect.OpName == "outer" && event.Effect.Args[0] != "original" {
			t.Fatal("observer modified retained payload")
		}
	}
}

func TestCollectorObserverReplacementDuringCleanup(t *testing.T) {
	collector := NewCollector()
	var observed atomic.Int64
	observer := func(TraceEvent) {
		observed.Add(1)
		_ = collector.Events()
	}
	collector.SetOnEvent(observer)
	var tasks sync.WaitGroup
	tasks.Add(2)
	go func() {
		defer tasks.Done()
		for i := 0; i < 100; i++ {
			collector.RecordEffect("Process", "shutdownWorkers", nil, "joined")
		}
	}()
	go func() {
		defer tasks.Done()
		for i := 0; i < 100; i++ {
			collector.SetOnEvent(nil)
			collector.SetOnEvent(observer)
		}
	}()
	tasks.Wait()
	collector.SetOnEvent(observer)
	collector.RecordEffect("Process", "shutdownWorkers", nil, "joined")
	if observed.Load() == 0 || len(collector.Events()) != 101 {
		t.Fatal("observer replacement lost retained cleanup events")
	}
}
