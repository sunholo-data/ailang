package coordinator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sunholo/ailang/internal/observatory"
)

// newTestDaemonT creates a minimal Daemon for HTTP handler testing.
func newTestDaemonT(t *testing.T) *Daemon {
	t.Helper()
	tmpDir := t.TempDir()
	return &Daemon{
		startedAt: time.Now().Add(-5 * time.Minute),
		config: &Config{
			PIDFile:  tmpDir + "/coordinator.pid",
			StateDir: tmpDir,
		},
	}
}

func TestHandleHealth(t *testing.T) {
	d := newTestDaemonT(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	d.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %v", resp["status"])
	}
	if resp["component"] != "coordinator" {
		t.Errorf("expected component coordinator, got %v", resp["component"])
	}
	if resp["uptime"] == nil || resp["uptime"] == "" {
		t.Error("expected non-empty uptime")
	}
}

func TestHandleStatus_NoStore(t *testing.T) {
	d := newTestDaemonT(t)

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()

	d.handleStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp Status
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
}

func TestHandleChainsActive_NilBackend(t *testing.T) {
	d := newTestDaemonT(t)

	req := httptest.NewRequest(http.MethodGet, "/chains/active", nil)
	rec := httptest.NewRecorder()

	d.handleChainsActive(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp []interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp) != 0 {
		t.Errorf("expected empty array, got %d items", len(resp))
	}
}

func TestHandleChainsStats_NilBackend(t *testing.T) {
	d := newTestDaemonT(t)

	req := httptest.NewRequest(http.MethodGet, "/chains/stats?hours=24", nil)
	rec := httptest.NewRecorder()

	d.handleChainsStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp observatory.ChainStatusCounts
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Total != 0 {
		t.Errorf("expected 0 total, got %d", resp.Total)
	}
}

func TestHandlePending_NilBackend(t *testing.T) {
	d := newTestDaemonT(t)

	req := httptest.NewRequest(http.MethodGet, "/pending", nil)
	rec := httptest.NewRecorder()

	d.handlePending(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp []interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp) != 0 {
		t.Errorf("expected empty array, got %d items", len(resp))
	}
}

func TestHandleChainsStats_DefaultHours(t *testing.T) {
	d := newTestDaemonT(t)

	req := httptest.NewRequest(http.MethodGet, "/chains/stats", nil)
	rec := httptest.NewRecorder()

	d.handleChainsStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHandleChainsStats_InvalidHours(t *testing.T) {
	d := newTestDaemonT(t)

	req := httptest.NewRequest(http.MethodGet, "/chains/stats?hours=abc", nil)
	rec := httptest.NewRecorder()

	d.handleChainsStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

// mockTaskStore implements just GetTaskStats for handler testing.
type mockTaskStore struct {
	Store // embed to satisfy interface
	stats *TaskStats
	err   error
}

func (m *mockTaskStore) GetTaskStats(ctx context.Context) (*TaskStats, error) {
	return m.stats, m.err
}

func TestHandleStatus_WithStore(t *testing.T) {
	d := newTestDaemonT(t)
	d.taskStore = &mockTaskStore{
		stats: &TaskStats{
			CompletedTasks:   10,
			PendingTasks:     2,
			RunningTasks:     1,
			PendingApprovals: 3,
			FailedTasks:      0,
			TotalCost:        1.50,
			TotalTokens:      5000,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()

	d.handleStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp Status
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if resp.TasksRun != 10 {
		t.Errorf("expected 10 tasks run, got %d", resp.TasksRun)
	}
	if resp.PendingTasks != 2 {
		t.Errorf("expected 2 pending, got %d", resp.PendingTasks)
	}
	if resp.TotalCost != 1.50 {
		t.Errorf("expected cost 1.50, got %f", resp.TotalCost)
	}
}

// M-CLOUD-ENDPOINT-AUTH: API key middleware tests

func TestRequireAPIKey_NoKeyConfigured(t *testing.T) {
	// When COORDINATOR_API_KEY is unset, requests pass through (local mode)
	d := newTestDaemonT(t)
	t.Setenv("COORDINATOR_API_KEY", "")

	handler := d.requireAPIKey(d.handleHealth)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (no key = open), got %d", rec.Code)
	}
}

func TestRequireAPIKey_ValidToken(t *testing.T) {
	d := newTestDaemonT(t)
	t.Setenv("COORDINATOR_API_KEY", "test-secret-key")

	handler := d.requireAPIKey(d.handleHealth)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Authorization", "Bearer test-secret-key")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid token, got %d", rec.Code)
	}
}

func TestRequireAPIKey_InvalidToken(t *testing.T) {
	d := newTestDaemonT(t)
	t.Setenv("COORDINATOR_API_KEY", "test-secret-key")

	handler := d.requireAPIKey(d.handleHealth)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with invalid token, got %d", rec.Code)
	}
}

func TestRequireAPIKey_MissingHeader(t *testing.T) {
	d := newTestDaemonT(t)
	t.Setenv("COORDINATOR_API_KEY", "test-secret-key")

	handler := d.requireAPIKey(d.handleHealth)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	// No Authorization header
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with missing header, got %d", rec.Code)
	}
}

func TestRequireAPIKey_WrongScheme(t *testing.T) {
	d := newTestDaemonT(t)
	t.Setenv("COORDINATOR_API_KEY", "test-secret-key")

	handler := d.requireAPIKey(d.handleHealth)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Authorization", "Basic test-secret-key")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong scheme, got %d", rec.Code)
	}
}

func TestStatusEndpoint_RequiresAPIKey(t *testing.T) {
	// Verify /status is protected when key is configured
	d := newTestDaemonT(t)
	t.Setenv("COORDINATOR_API_KEY", "my-key")

	handler := d.requireAPIKey(d.handleStatus)

	// Without token → 401
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}

	// With token → 200
	req = httptest.NewRequest(http.MethodGet, "/status", nil)
	req.Header.Set("Authorization", "Bearer my-key")
	rec = httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with token, got %d", rec.Code)
	}
}

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, map[string]string{"key": "value"})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["key"] != "value" {
		t.Errorf("expected value, got %s", resp["key"])
	}
}
