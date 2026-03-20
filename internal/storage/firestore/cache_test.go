package firestore

import (
	"testing"
	"time"
)

func TestTTLCacheGetSetInvalidate(t *testing.T) {
	c := newTTLCache[string](1 * time.Hour)

	// Miss on empty cache.
	if _, ok := c.get(); ok {
		t.Error("expected cache miss on empty cache")
	}

	// Set and hit.
	c.set("hello")
	val, ok := c.get()
	if !ok || val != "hello" {
		t.Errorf("expected cache hit with 'hello', got %q ok=%v", val, ok)
	}

	// Invalidate.
	c.invalidate()
	if _, ok := c.get(); ok {
		t.Error("expected cache miss after invalidation")
	}
}

func TestTTLCacheExpiry(t *testing.T) {
	c := newTTLCache[int](1 * time.Millisecond)

	c.set(42)
	time.Sleep(5 * time.Millisecond)

	if _, ok := c.get(); ok {
		t.Error("expected cache miss after TTL expiry")
	}
}

func TestTTLCacheConcurrency(t *testing.T) {
	c := newTTLCache[int](1 * time.Hour)

	done := make(chan struct{})
	for i := 0; i < 100; i++ {
		go func(v int) {
			c.set(v)
			done <- struct{}{}
		}(i)
		go func() {
			c.get()
			done <- struct{}{}
		}()
	}
	for i := 0; i < 200; i++ {
		<-done
	}

	// Should have some value cached.
	val, ok := c.get()
	if !ok {
		t.Error("expected cache hit after concurrent writes")
	}
	if val < 0 || val >= 100 {
		t.Errorf("unexpected value: %d", val)
	}
}
