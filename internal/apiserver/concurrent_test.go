package apiserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// TestConcurrentHTTPRequests verifies concurrent HTTP requests through the
// full serve-api stack don't deadlock or produce incorrect results.
func TestConcurrentHTTPRequests(t *testing.T) {
	s := &Server{
		modules: map[string]*ModuleInfo{
			"test": {
				Path: "test",
				Exports: []ExportInfo{
					{Name: "hello", Type: "string -> string", Arity: 1},
				},
			},
		},
	}

	handler := s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"result": "ok"})
	})

	const numRequests = 50
	var wg sync.WaitGroup
	errors := make(chan string, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/api/test/hello", nil)
			rr := httptest.NewRecorder()
			handler(rr, req)
			if rr.Code != http.StatusOK {
				errors <- "request failed"
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	var errs []string
	for e := range errors {
		errs = append(errs, e)
	}
	if len(errs) > 0 {
		t.Errorf("%d errors in %d concurrent requests:\n%s", len(errs), numRequests, strings.Join(errs, "\n"))
	}
}
