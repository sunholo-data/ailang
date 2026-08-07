package apiserver

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCallbackRunnerTimeoutAndStopsChain(t *testing.T) {
	runner, err := NewCallbackRunner(20*time.Millisecond, 1)
	if err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	defer close(release)
	start := time.Now()
	_, err = RunCallback(context.Background(), runner, func(context.Context) (int, error) {
		<-release
		return 1, nil
	})
	elapsed := time.Since(start)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want deadline exceeded", err)
	}
	if elapsed < 20*time.Millisecond || elapsed >= 250*time.Millisecond {
		t.Fatalf("elapsed = %v, want [20ms, 250ms)", elapsed)
	}
	// The chain-level "stop after a timeout" clause needs a chain, which arrives
	// with the M2 wrapper; guarding a follow-up call on `err == nil` here would be
	// dead code and the counter assertion would hold under any implementation.
	// What IS provable at the runner level is the stronger half of the same claim:
	// a timed-out callback keeps its slot until it actually RETURNS, so the next
	// call on a saturated runner is rejected without host code ever being entered.
	// This fails if the token is released when the handler stops waiting.
	var next atomic.Int32
	_, nextErr := RunCallback(context.Background(), runner, func(context.Context) (int, error) {
		next.Add(1)
		return 0, nil
	})
	if !errors.Is(nextErr, ErrCallbackCapacity) {
		t.Fatalf("follow-up call error = %v, want capacity exceeded", nextErr)
	}
	if next.Load() != 0 {
		t.Fatalf("next callback entered %d times", next.Load())
	}
}

func TestCallbackRunnerObservedDeadlineAndFastCall(t *testing.T) {
	runner, err := NewCallbackRunner(5*time.Second, 4)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	deadline, err := RunCallback(context.Background(), runner, func(ctx context.Context) (time.Time, error) {
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("callback context has no deadline")
		}
		return deadline, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if deadline.Before(now.Add(4*time.Second)) || deadline.After(now.Add(6*time.Second)) {
		t.Fatalf("observed deadline %v outside expected window from %v", deadline, now)
	}
	value, err := RunCallback(context.Background(), runner, func(context.Context) (int, error) { return 137, nil })
	if err != nil || value != 137 {
		t.Fatalf("fast callback = (%d, %v)", value, err)
	}
}

func TestCallbackRunnerBoundsConcurrencyAndRecovers(t *testing.T) {
	runner, err := NewCallbackRunner(30*time.Millisecond, 4)
	if err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	var starts atomic.Int32
	runBurst := func(count int) int {
		var overloads atomic.Int32
		var wg sync.WaitGroup
		for range count {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, callErr := RunCallback(context.Background(), runner, func(context.Context) (int, error) {
					starts.Add(1)
					<-release
					return 0, nil
				})
				if errors.Is(callErr, ErrCallbackCapacity) {
					overloads.Add(1)
				}
			}()
		}
		wg.Wait()
		return int(overloads.Load())
	}
	baseline := runtime.NumGoroutine()
	if overloads := runBurst(40); overloads != 36 {
		t.Fatalf("40-call overloads = %d, want 36", overloads)
	}
	if starts.Load() != 4 {
		t.Fatalf("callback starts = %d, want 4", starts.Load())
	}
	after40 := runtime.NumGoroutine()
	if overloads := runBurst(80); overloads != 80 {
		t.Fatalf("80-call overloads = %d, want 80", overloads)
	}
	after80 := runtime.NumGoroutine()
	if growth := after80 - after40; growth > 4 {
		t.Fatalf("goroutines grew with rejected call count: baseline=%d after40=%d after80=%d", baseline, after40, after80)
	}
	close(release)
	deadline := time.Now().Add(time.Second)
	for len(runner.slots) != 0 && time.Now().Before(deadline) {
		runtime.Gosched()
	}
	if len(runner.slots) != 0 {
		t.Fatal("capacity did not recover after callbacks returned")
	}

	fastRunner, _ := NewCallbackRunner(time.Second, 4)
	var fastOverloads atomic.Int32
	var wg sync.WaitGroup
	for range 40 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, callErr := RunCallback(context.Background(), fastRunner, func(context.Context) (int, error) { return 1, nil })
			if errors.Is(callErr, ErrCallbackCapacity) {
				fastOverloads.Add(1)
			}
		}()
	}
	wg.Wait()
	if fastOverloads.Load() != 0 {
		t.Fatalf("fast callbacks overloaded %d times", fastOverloads.Load())
	}
}
