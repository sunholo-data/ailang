package apiserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestAuthMiddleware_NoAuth(t *testing.T) {
	s := &Server{}
	handler := s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/test/func", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 with no auth configured, got %d", rr.Code)
	}
}

func TestAuthMiddleware_ValidKey(t *testing.T) {
	os.Setenv("TEST_API_KEY", "secret123")
	defer os.Unsetenv("TEST_API_KEY")

	s := &Server{apiKeyHeader: "x-api-key", apiKeyEnv: "TEST_API_KEY"}
	handler := s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/api/test/func", nil)
	req.Header.Set("x-api-key", "secret123")
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 with valid key, got %d", rr.Code)
	}
}

func TestAuthMiddleware_InvalidKey(t *testing.T) {
	os.Setenv("TEST_API_KEY", "secret123")
	defer os.Unsetenv("TEST_API_KEY")

	s := &Server{apiKeyHeader: "x-api-key", apiKeyEnv: "TEST_API_KEY"}
	handler := s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/api/test/func", nil)
	req.Header.Set("x-api-key", "wrongkey")
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with invalid key, got %d", rr.Code)
	}
}

func TestAuthMiddleware_MissingKey(t *testing.T) {
	os.Setenv("TEST_API_KEY", "secret123")
	defer os.Unsetenv("TEST_API_KEY")

	s := &Server{apiKeyHeader: "x-api-key", apiKeyEnv: "TEST_API_KEY"}
	handler := s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/api/test/func", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with missing key, got %d", rr.Code)
	}
}

func TestAuthMiddleware_BearerToken(t *testing.T) {
	os.Setenv("TEST_API_KEY", "secret123")
	defer os.Unsetenv("TEST_API_KEY")

	s := &Server{apiKeyHeader: "x-api-key", apiKeyEnv: "TEST_API_KEY"}
	handler := s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/api/test/func", nil)
	req.Header.Set("Authorization", "Bearer secret123")
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 with Bearer token, got %d", rr.Code)
	}
}

func TestAuthMiddleware_MetaEndpointBypass(t *testing.T) {
	os.Setenv("TEST_API_KEY", "secret123")
	defer os.Unsetenv("TEST_API_KEY")

	s := &Server{apiKeyHeader: "x-api-key", apiKeyEnv: "TEST_API_KEY"}
	handler := s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Meta endpoints bypass auth
	paths := []string{"/api/_health", "/api/_meta/modules", "/mcp/", "/a2a/", "/.well-known/agent.json"}
	for _, path := range paths {
		req := httptest.NewRequest("GET", path, nil)
		rr := httptest.NewRecorder()
		handler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200 for %s (auth bypass), got %d", path, rr.Code)
		}
	}
}

func TestWriteRawResponse(t *testing.T) {
	// Test that writeRawResponse handles BytesValue correctly
	// (can't easily test without eval imports, covered by integration tests)
}
