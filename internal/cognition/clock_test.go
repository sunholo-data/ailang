package cognition

import (
	"sync"
	"testing"
)

// ============================================================================
// Tick — monotonicity invariant
// ============================================================================

func TestClock_Tick_Monotonic(t *testing.T) {
	c := NewClock()
	prev := LamportValue(0)
	for i := 0; i < 100; i++ {
		got := c.Tick()
		if got <= prev {
			t.Fatalf("Tick should be strictly increasing: got %d after %d", got, prev)
		}
		prev = got
	}
	if c.Read() != 100 {
		t.Errorf("Read after 100 Ticks: got %d, want 100", c.Read())
	}
}

func TestClock_Tick_StartsAtOne(t *testing.T) {
	c := NewClock()
	if got := c.Tick(); got != 1 {
		t.Errorf("first Tick: got %d, want 1", got)
	}
}

func TestClock_NewClockAt(t *testing.T) {
	c := NewClockAt(42)
	if got := c.Tick(); got != 43 {
		t.Errorf("first Tick after NewClockAt(42): got %d, want 43", got)
	}
}

// ============================================================================
// Update — happens-before establishment
// ============================================================================

func TestClock_Update_FromRemote_AdvancesPastRemote(t *testing.T) {
	c := NewClock() // local at 0
	c.Tick()        // local at 1
	c.Tick()        // local at 2

	// Remote sends with clock 5
	got := c.Update(5)
	if got != 6 {
		t.Errorf("Update(5) on local=2: got %d, want 6 (max+1)", got)
	}
	if c.Read() != 6 {
		t.Errorf("Read after Update: got %d, want 6", c.Read())
	}
}

func TestClock_Update_FromRemote_AdvancesLocalEvenIfLocalHigher(t *testing.T) {
	c := NewClock()
	for i := 0; i < 10; i++ {
		c.Tick() // local at 10
	}

	// Remote sends with clock 3 (less than local)
	got := c.Update(3)
	if got != 11 {
		t.Errorf("Update(3) on local=10: got %d, want 11 (local+1)", got)
	}
}

func TestClock_Update_NextTickHigher(t *testing.T) {
	c := NewClock()
	c.Update(5)
	if got := c.Tick(); got != 7 {
		t.Errorf("Tick after Update(5): got %d, want 7", got)
	}
}

// ============================================================================
// Read — non-mutating peek
// ============================================================================

func TestClock_Read_DoesNotAdvance(t *testing.T) {
	c := NewClockAt(42)
	for i := 0; i < 100; i++ {
		if v := c.Read(); v != 42 {
			t.Fatalf("Read should not advance, got %d after %d Reads", v, i)
		}
	}
}

// ============================================================================
// Concurrency — atomic correctness under contention
// ============================================================================

func TestClock_Tick_ConcurrentMonotonic(t *testing.T) {
	c := NewClock()
	const goroutines = 50
	const ticksPerG = 100

	var wg sync.WaitGroup
	results := make([][]LamportValue, goroutines)
	for g := 0; g < goroutines; g++ {
		g := g
		wg.Add(1)
		go func() {
			defer wg.Done()
			seen := make([]LamportValue, ticksPerG)
			for i := 0; i < ticksPerG; i++ {
				seen[i] = c.Tick()
			}
			results[g] = seen
		}()
	}
	wg.Wait()

	// Every value across all goroutines should be unique.
	all := map[LamportValue]bool{}
	for _, seen := range results {
		for _, v := range seen {
			if all[v] {
				t.Errorf("duplicate Lamport value across goroutines: %d", v)
			}
			all[v] = true
		}
	}

	if len(all) != goroutines*ticksPerG {
		t.Errorf("expected %d unique values, got %d", goroutines*ticksPerG, len(all))
	}
	if c.Read() != LamportValue(goroutines*ticksPerG) {
		t.Errorf("final value: got %d, want %d", c.Read(), goroutines*ticksPerG)
	}
}

func TestClock_Update_ConcurrentSafe(t *testing.T) {
	c := NewClock()
	const goroutines = 20
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				c.Tick()
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				c.Update(LamportValue(i * 5))
			}
		}()
	}
	wg.Wait()
	// Final clock value should be >= max(local Ticks, max Update remote)
	if c.Read() < 100 {
		t.Errorf("expected clock to advance past concurrent contention, got %d", c.Read())
	}
}

// ============================================================================
// CompareEvents — sender tiebreaker
// ============================================================================

func TestCompareEvents_ByLamportClock(t *testing.T) {
	if got := CompareEvents(5, "a", 6, "a"); got >= 0 {
		t.Errorf("clock 5 should precede clock 6, got %d", got)
	}
	if got := CompareEvents(10, "a", 5, "a"); got <= 0 {
		t.Errorf("clock 10 should follow clock 5, got %d", got)
	}
}

func TestCompareEvents_BySenderWhenClockTies(t *testing.T) {
	if got := CompareEvents(5, "a", 5, "b"); got >= 0 {
		t.Errorf("sender 'a' should precede 'b' at same clock, got %d", got)
	}
	if got := CompareEvents(5, "z", 5, "a"); got <= 0 {
		t.Errorf("sender 'z' should follow 'a' at same clock, got %d", got)
	}
}

func TestCompareEvents_Equal(t *testing.T) {
	if got := CompareEvents(5, "a", 5, "a"); got != 0 {
		t.Errorf("identical events should compare equal, got %d", got)
	}
}

// ============================================================================
// ClockRegistry — per-node clock management
// ============================================================================

func TestClockRegistry_LazyCreation(t *testing.T) {
	r := NewClockRegistry()
	c1 := r.Get("node_a")
	c1.Tick()

	c2 := r.Get("node_a") // same node — same clock
	if c2.Read() != 1 {
		t.Errorf("expected shared clock to be 1, got %d", c2.Read())
	}

	c3 := r.Get("node_b") // different node — independent clock
	if c3.Read() != 0 {
		t.Errorf("expected fresh clock for node_b to be 0, got %d", c3.Read())
	}
}

func TestClockRegistry_Snapshot(t *testing.T) {
	r := NewClockRegistry()
	r.Get("a").Tick()
	r.Get("a").Tick()
	r.Get("b").Tick()
	r.Get("c").Update(42)

	snap := r.Snapshot()
	if snap["a"] != 2 {
		t.Errorf("snap[a]: got %d, want 2", snap["a"])
	}
	if snap["b"] != 1 {
		t.Errorf("snap[b]: got %d, want 1", snap["b"])
	}
	if snap["c"] != 43 {
		t.Errorf("snap[c]: got %d, want 43", snap["c"])
	}
}

func TestClockRegistry_Snapshot_DefensiveCopy(t *testing.T) {
	r := NewClockRegistry()
	r.Get("a").Tick()
	snap := r.Snapshot()
	snap["a"] = 999 // mutate caller's copy

	if r.Get("a").Read() != 1 {
		t.Errorf("registry should not be affected by Snapshot mutation, got %d", r.Get("a").Read())
	}
}

func TestClockRegistry_ConcurrentLazyCreation(t *testing.T) {
	r := NewClockRegistry()
	var wg sync.WaitGroup
	const n = 100

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Get("shared").Tick()
		}()
	}
	wg.Wait()

	if v := r.Get("shared").Read(); v != n {
		t.Errorf("concurrent Get/Tick on same node: got %d, want %d", v, n)
	}
}
