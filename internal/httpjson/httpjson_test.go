package httpjson_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sunholo-data/ailang/internal/httpjson"
)

func TestWriteSetsHeaderStatusAndBody(t *testing.T) {
	rec := httptest.NewRecorder()
	httpjson.Write(rec, http.StatusCreated, map[string]int{"n": 1})
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
	if body := rec.Body.String(); body != "{\"n\":1}\n" {
		t.Errorf("body = %q", body)
	}
}

func TestErrorShape(t *testing.T) {
	rec := httptest.NewRecorder()
	httpjson.Error(rec, http.StatusNotFound, "no such task")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if body := rec.Body.String(); body != "{\"error\":\"no such task\"}\n" {
		t.Errorf("body = %q", body)
	}
}

// An unencodable value must not produce a second status line: the first
// one has already gone to the client.
func TestEncodeFailureDoesNotWriteTwice(t *testing.T) {
	rec := httptest.NewRecorder()
	httpjson.Write(rec, http.StatusOK, make(chan int))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want the original 200", rec.Code)
	}
}
