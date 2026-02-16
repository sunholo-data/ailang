package server

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestResponseCache_HitAndMiss(t *testing.T) {
	cache := newResponseCache(1 * time.Second)

	var callCount atomic.Int32
	handler := func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}

	wrapped := cache.withCache(handler)

	// First request - cache miss
	req := httptest.NewRequest("GET", "/api/test?foo=bar", nil)
	rec := httptest.NewRecorder()
	wrapped(rec, req)

	if rec.Header().Get("X-Cache") == "HIT" {
		t.Error("first request should be a cache miss")
	}
	if callCount.Load() != 1 {
		t.Errorf("handler should be called once, got %d", callCount.Load())
	}
	if rec.Body.String() != `{"status":"ok"}` {
		t.Errorf("unexpected body: %s", rec.Body.String())
	}

	// Second request with same URL - cache hit
	req2 := httptest.NewRequest("GET", "/api/test?foo=bar", nil)
	rec2 := httptest.NewRecorder()
	wrapped(rec2, req2)

	if rec2.Header().Get("X-Cache") != "HIT" {
		t.Error("second request should be a cache hit")
	}
	if callCount.Load() != 1 {
		t.Errorf("handler should NOT be called again, got %d calls", callCount.Load())
	}
	if rec2.Body.String() != `{"status":"ok"}` {
		t.Errorf("cached body mismatch: %s", rec2.Body.String())
	}
}

func TestResponseCache_DifferentURLs(t *testing.T) {
	cache := newResponseCache(1 * time.Second)

	var callCount atomic.Int32
	handler := func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.Write([]byte(r.URL.String()))
	}

	wrapped := cache.withCache(handler)

	// Request 1
	req1 := httptest.NewRequest("GET", "/api/test?a=1", nil)
	rec1 := httptest.NewRecorder()
	wrapped(rec1, req1)

	// Request 2 with different query params
	req2 := httptest.NewRequest("GET", "/api/test?a=2", nil)
	rec2 := httptest.NewRecorder()
	wrapped(rec2, req2)

	if callCount.Load() != 2 {
		t.Errorf("different URLs should both call handler, got %d calls", callCount.Load())
	}
}

func TestResponseCache_Expiry(t *testing.T) {
	cache := newResponseCache(50 * time.Millisecond) // Very short TTL for testing

	var callCount atomic.Int32
	handler := func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.Write([]byte(`{"n":` + time.Now().String() + `}`))
	}

	wrapped := cache.withCache(handler)

	// First request
	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()
	wrapped(rec, req)
	if callCount.Load() != 1 {
		t.Fatalf("expected 1 call, got %d", callCount.Load())
	}

	// Wait for TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Second request - should be cache miss after expiry
	req2 := httptest.NewRequest("GET", "/api/test", nil)
	rec2 := httptest.NewRecorder()
	wrapped(rec2, req2)
	if callCount.Load() != 2 {
		t.Errorf("expected 2 calls after TTL expiry, got %d", callCount.Load())
	}
}

func TestResponseCache_NonSuccessNotCached(t *testing.T) {
	cache := newResponseCache(1 * time.Second)

	var callCount atomic.Int32
	handler := func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}

	wrapped := cache.withCache(handler)

	// First request - returns 500
	req1 := httptest.NewRequest("GET", "/api/test", nil)
	rec1 := httptest.NewRecorder()
	wrapped(rec1, req1)

	// Second request - should NOT be cached (500 response)
	req2 := httptest.NewRequest("GET", "/api/test", nil)
	rec2 := httptest.NewRecorder()
	wrapped(rec2, req2)

	if callCount.Load() != 2 {
		t.Errorf("error responses should not be cached, got %d calls", callCount.Load())
	}
}

func TestResponseCache_ConcurrentAccess(t *testing.T) {
	cache := newResponseCache(1 * time.Second)

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true}`))
	}

	wrapped := cache.withCache(handler)

	// Run 100 concurrent requests
	done := make(chan struct{}, 100)
	for i := 0; i < 100; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			req := httptest.NewRequest("GET", "/api/test", nil)
			rec := httptest.NewRecorder()
			wrapped(rec, req)
			if rec.Body.Len() == 0 {
				t.Error("empty response from concurrent request")
			}
		}()
	}

	// Wait for all to complete
	for i := 0; i < 100; i++ {
		<-done
	}
}
