package effects

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
)

func TestInMemorySharedCache_Basic(t *testing.T) {
	cache := NewInMemorySharedCache()

	// Test Put and Get
	cache.Put("key1", []byte("value1"))
	val, ok := cache.Get("key1")
	if !ok {
		t.Fatal("expected key1 to exist")
	}
	if string(val) != "value1" {
		t.Errorf("expected value1, got %s", val)
	}

	// Test Get non-existent
	val, ok = cache.Get("nonexistent")
	if ok {
		t.Error("expected nonexistent key to not exist")
	}
	if val != nil {
		t.Error("expected nil value for non-existent key")
	}
}

func TestInMemorySharedCache_Delete(t *testing.T) {
	cache := NewInMemorySharedCache()

	cache.Put("key1", []byte("value1"))
	cache.Delete("key1")

	_, ok := cache.Get("key1")
	if ok {
		t.Error("expected key1 to be deleted")
	}

	// Delete non-existent should not panic
	cache.Delete("nonexistent")
}

func TestInMemorySharedCache_CAS(t *testing.T) {
	cache := NewInMemorySharedCache()

	// Test create-if-absent (oldValue = nil)
	ok := cache.CAS("key1", nil, []byte("initial"))
	if !ok {
		t.Error("CAS with nil oldValue should succeed for new key")
	}

	val, _ := cache.Get("key1")
	if string(val) != "initial" {
		t.Errorf("expected initial, got %s", val)
	}

	// Test create-if-absent fails when key exists
	ok = cache.CAS("key1", nil, []byte("should_not_work"))
	if ok {
		t.Error("CAS with nil oldValue should fail when key exists")
	}

	// Test successful CAS
	ok = cache.CAS("key1", []byte("initial"), []byte("updated"))
	if !ok {
		t.Error("CAS should succeed when oldValue matches")
	}

	val, _ = cache.Get("key1")
	if string(val) != "updated" {
		t.Errorf("expected updated, got %s", val)
	}

	// Test failed CAS (wrong oldValue)
	ok = cache.CAS("key1", []byte("wrong"), []byte("should_not_work"))
	if ok {
		t.Error("CAS should fail when oldValue doesn't match")
	}

	val, _ = cache.Get("key1")
	if string(val) != "updated" {
		t.Errorf("value should still be updated, got %s", val)
	}
}

func TestInMemorySharedCache_CopyOnRead(t *testing.T) {
	cache := NewInMemorySharedCache()

	original := []byte("original")
	cache.Put("key1", original)

	// Get a copy and modify it
	retrieved, _ := cache.Get("key1")
	retrieved[0] = 'X'

	// Original cache value should be unchanged
	cached, _ := cache.Get("key1")
	if string(cached) != "original" {
		t.Errorf("cache value was mutated by caller! got %s", cached)
	}
}

func TestInMemorySharedCache_CopyOnWrite(t *testing.T) {
	cache := NewInMemorySharedCache()

	original := []byte("original")
	cache.Put("key1", original)

	// Modify the original slice after Put
	original[0] = 'X'

	// Cached value should be unchanged
	cached, _ := cache.Get("key1")
	if string(cached) != "original" {
		t.Errorf("cache value was mutated by caller modifying Put input! got %s", cached)
	}
}

func TestInMemorySharedCache_KeysAndLen(t *testing.T) {
	cache := NewInMemorySharedCache()

	if cache.Len() != 0 {
		t.Errorf("expected len 0, got %d", cache.Len())
	}

	cache.Put("key1", []byte("v1"))
	cache.Put("key2", []byte("v2"))
	cache.Put("key3", []byte("v3"))

	if cache.Len() != 3 {
		t.Errorf("expected len 3, got %d", cache.Len())
	}

	keys := cache.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}

	// Verify all keys are present
	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}
	for _, expected := range []string{"key1", "key2", "key3"} {
		if !keySet[expected] {
			t.Errorf("missing key: %s", expected)
		}
	}
}

// Stress test for thread safety with concurrent writers
func TestInMemorySharedCache_ConcurrentWrite(t *testing.T) {
	cache := NewInMemorySharedCache()
	var wg sync.WaitGroup
	numGoroutines := 100
	numOps := 100

	// Launch concurrent writers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				value := []byte(fmt.Sprintf("value-%d-%d", id, j))
				cache.Put(key, value)
			}
		}(i)
	}

	wg.Wait()

	// Verify count
	expectedCount := numGoroutines * numOps
	if cache.Len() != expectedCount {
		t.Errorf("expected %d entries, got %d", expectedCount, cache.Len())
	}
}

// Stress test for thread safety with concurrent reads and writes
func TestInMemorySharedCache_ConcurrentReadWrite(t *testing.T) {
	cache := NewInMemorySharedCache()
	var wg sync.WaitGroup

	// Pre-populate
	for i := 0; i < 100; i++ {
		cache.Put(fmt.Sprintf("key-%d", i), []byte(fmt.Sprintf("value-%d", i)))
	}

	numReaders := 50
	numWriters := 50
	numOps := 100

	// Launch readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := fmt.Sprintf("key-%d", j%100)
				cache.Get(key)
			}
		}()
	}

	// Launch writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := fmt.Sprintf("key-%d", j%100)
				value := []byte(fmt.Sprintf("value-%d-%d", id, j))
				cache.Put(key, value)
			}
		}(i)
	}

	wg.Wait()
}

// Stress test for CAS under contention
func TestInMemorySharedCache_CASContention(t *testing.T) {
	cache := NewInMemorySharedCache()
	var wg sync.WaitGroup

	// Initialize counter
	cache.Put("counter", []byte("0"))

	numGoroutines := 100
	numIncrements := 10
	successCount := int64(0)
	var mu sync.Mutex

	// Each goroutine tries to increment the counter
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numIncrements; j++ {
				for {
					oldVal, _ := cache.Get("counter")
					// Parse counter
					counter := 0
					fmt.Sscanf(string(oldVal), "%d", &counter)
					newVal := []byte(fmt.Sprintf("%d", counter+1))

					if cache.CAS("counter", oldVal, newVal) {
						mu.Lock()
						successCount++
						mu.Unlock()
						break
					}
					// CAS failed, retry
				}
			}
		}()
	}

	wg.Wait()

	// Verify final counter value
	finalVal, _ := cache.Get("counter")
	var finalCounter int
	fmt.Sscanf(string(finalVal), "%d", &finalCounter)

	expectedCounter := numGoroutines * numIncrements
	if finalCounter != expectedCounter {
		t.Errorf("expected counter %d, got %d", expectedCounter, finalCounter)
	}

	if int(successCount) != expectedCounter {
		t.Errorf("expected %d successful CAS ops, got %d", expectedCounter, successCount)
	}
}

// Test that no torn writes occur under concurrent access
func TestInMemorySharedCache_NoTornWrites(t *testing.T) {
	cache := NewInMemorySharedCache()
	var wg sync.WaitGroup

	numWriters := 50
	numReaders := 50
	numOps := 100

	// Each writer writes a predictable pattern
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Create a value that has a checksum (first byte = length)
			for j := 0; j < numOps; j++ {
				length := byte((id*numOps + j) % 200)
				value := make([]byte, length+1)
				value[0] = length
				for k := byte(1); k <= length; k++ {
					value[k] = byte(id) // Fill with writer ID
				}
				cache.Put("shared", value)
			}
		}(i)
	}

	// Readers verify no torn writes (checksum matches)
	errorCount := int64(0)
	var errorMu sync.Mutex

	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				val, ok := cache.Get("shared")
				if !ok {
					continue // Key might not exist yet
				}
				if len(val) == 0 {
					continue
				}
				// Verify checksum: first byte should equal len-1
				expectedLen := int(val[0]) + 1
				if len(val) != expectedLen {
					errorMu.Lock()
					errorCount++
					errorMu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	if errorCount > 0 {
		t.Errorf("detected %d torn writes!", errorCount)
	}
}

// Test buffer mutation doesn't corrupt cache
func TestInMemorySharedCache_CallerBufferMutation(t *testing.T) {
	cache := NewInMemorySharedCache()
	var wg sync.WaitGroup

	numGoroutines := 50
	numOps := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := fmt.Sprintf("key-%d", id)

				// Put a value
				original := []byte(fmt.Sprintf("value-%d-%d", id, j))
				cache.Put(key, original)

				// Mutate the original slice (should not affect cache)
				for k := range original {
					original[k] = 'X'
				}

				// Get the value back
				retrieved, ok := cache.Get(key)
				if !ok {
					continue // Concurrent delete might have happened
				}

				// Mutate the retrieved slice (should not affect cache)
				for k := range retrieved {
					retrieved[k] = 'Y'
				}

				// Get again and verify it's intact
				final, ok := cache.Get(key)
				if !ok {
					continue
				}

				// Should start with "value-" not "YYYYY" or "XXXXX"
				if len(final) > 0 && !bytes.HasPrefix(final, []byte("value-")) {
					t.Errorf("cache value was corrupted: %s", final)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestSharedMemContext_Stats(t *testing.T) {
	cache := NewInMemorySharedCache()
	ctx := NewSharedMemContext(cache)

	ctx.IncrGetCount()
	ctx.IncrGetCount()
	ctx.IncrPutCount()
	ctx.IncrCASCount(true)
	ctx.IncrCASCount(false)

	gets, puts, cas, casSuccess := ctx.Stats()
	if gets != 2 {
		t.Errorf("expected 2 gets, got %d", gets)
	}
	if puts != 1 {
		t.Errorf("expected 1 put, got %d", puts)
	}
	if cas != 2 {
		t.Errorf("expected 2 CAS, got %d", cas)
	}
	if casSuccess != 1 {
		t.Errorf("expected 1 CAS success, got %d", casSuccess)
	}
}
