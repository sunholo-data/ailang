package coordinator

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestApprovalCheckpoint_Basic(t *testing.T) {
	ac := NewApprovalCheckpoint(1 * time.Hour)

	// Create a request in a goroutine since RequestApproval blocks
	var wg sync.WaitGroup
	var status ApprovalStatus
	var err error

	wg.Add(1)
	go func() {
		defer wg.Done()
		status, err = ac.RequestApproval(context.Background(), &ApprovalRequest{
			ID:          "test-1",
			TaskID:      "task-123",
			Type:        ApprovalTypeMerge,
			Title:       "Merge changes",
			Description: "Please review and approve merge",
		})
	}()

	// Wait for request to be registered
	time.Sleep(50 * time.Millisecond)

	// Approve
	if err := ac.Approve("test-1", "user"); err != nil {
		t.Fatalf("failed to approve: %v", err)
	}

	wg.Wait()

	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if status != ApprovalStatusApproved {
		t.Errorf("expected approved, got %s", status)
	}
}

func TestApprovalCheckpoint_Reject(t *testing.T) {
	ac := NewApprovalCheckpoint(1 * time.Hour)

	var wg sync.WaitGroup
	var status ApprovalStatus

	wg.Add(1)
	go func() {
		defer wg.Done()
		status, _ = ac.RequestApproval(context.Background(), &ApprovalRequest{
			ID:     "test-1",
			TaskID: "task-123",
			Type:   ApprovalTypeMerge,
		})
	}()

	time.Sleep(50 * time.Millisecond)

	if err := ac.Reject("test-1", "user"); err != nil {
		t.Fatalf("failed to reject: %v", err)
	}

	wg.Wait()

	if status != ApprovalStatusRejected {
		t.Errorf("expected rejected, got %s", status)
	}
}

func TestApprovalCheckpoint_Timeout(t *testing.T) {
	ac := NewApprovalCheckpoint(100 * time.Millisecond)

	status, err := ac.RequestApproval(context.Background(), &ApprovalRequest{
		ID:     "test-1",
		TaskID: "task-123",
		Type:   ApprovalTypeMerge,
	})

	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if status != ApprovalStatusTimeout {
		t.Errorf("expected timeout, got %s", status)
	}
}

func TestApprovalCheckpoint_AutoReject(t *testing.T) {
	ac := NewApprovalCheckpoint(100 * time.Millisecond)

	status, err := ac.RequestApproval(context.Background(), &ApprovalRequest{
		ID:         "test-1",
		TaskID:     "task-123",
		Type:       ApprovalTypeMerge,
		AutoReject: true,
	})

	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if status != ApprovalStatusRejected {
		t.Errorf("expected rejected on timeout, got %s", status)
	}
}

func TestApprovalCheckpoint_ContextCancel(t *testing.T) {
	ac := NewApprovalCheckpoint(1 * time.Hour)

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	var status ApprovalStatus
	var err error

	wg.Add(1)
	go func() {
		defer wg.Done()
		status, err = ac.RequestApproval(ctx, &ApprovalRequest{
			ID:     "test-1",
			TaskID: "task-123",
			Type:   ApprovalTypeMerge,
		})
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	wg.Wait()

	if err != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
	if status != ApprovalStatusRejected {
		t.Errorf("expected rejected on cancel, got %s", status)
	}
}

func TestApprovalCheckpoint_GetPendingRequests(t *testing.T) {
	ac := NewApprovalCheckpoint(1 * time.Hour)

	// Start multiple requests
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ac.RequestApproval(context.Background(), &ApprovalRequest{
				ID:     "test-" + string(rune('0'+id)),
				TaskID: "task-" + string(rune('0'+id)),
				Type:   ApprovalTypeMerge,
			})
		}(i)
	}

	time.Sleep(50 * time.Millisecond)

	pending := ac.GetPendingRequests()
	if len(pending) != 3 {
		t.Errorf("expected 3 pending, got %d", len(pending))
	}

	// Approve one
	ac.Approve("test-0", "user")

	pending = ac.GetPendingRequests()
	if len(pending) != 2 {
		t.Errorf("expected 2 pending after approval, got %d", len(pending))
	}

	// Cleanup
	ac.Clear()
	wg.Wait()
}

func TestApprovalCheckpoint_ApproveByTask(t *testing.T) {
	ac := NewApprovalCheckpoint(1 * time.Hour)

	var wg sync.WaitGroup
	var status ApprovalStatus

	wg.Add(1)
	go func() {
		defer wg.Done()
		status, _ = ac.RequestApproval(context.Background(), &ApprovalRequest{
			ID:     "test-1",
			TaskID: "task-123",
			Type:   ApprovalTypeMerge,
		})
	}()

	time.Sleep(50 * time.Millisecond)

	// Approve by task ID
	if err := ac.ApproveByTask("task-123", "user"); err != nil {
		t.Fatalf("failed to approve by task: %v", err)
	}

	wg.Wait()

	if status != ApprovalStatusApproved {
		t.Errorf("expected approved, got %s", status)
	}
}

func TestApprovalCheckpoint_Callback(t *testing.T) {
	ac := NewApprovalCheckpoint(1 * time.Hour)

	var callbackCalled bool
	var callbackRequest *ApprovalRequest
	var mu sync.Mutex

	ac.SetCallback(func(req *ApprovalRequest) {
		mu.Lock()
		callbackCalled = true
		callbackRequest = req
		mu.Unlock()
	})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ac.RequestApproval(context.Background(), &ApprovalRequest{
			ID:     "test-1",
			TaskID: "task-123",
			Type:   ApprovalTypeMerge,
		})
	}()

	time.Sleep(50 * time.Millisecond)
	ac.Approve("test-1", "user")
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()

	if !callbackCalled {
		t.Error("callback was not called")
	}
	if callbackRequest == nil {
		t.Error("callback request was nil")
	}
	if callbackRequest != nil && callbackRequest.Status != ApprovalStatusApproved {
		t.Errorf("callback request status: expected approved, got %s", callbackRequest.Status)
	}
}

func TestApprovalCheckpoint_NotFound(t *testing.T) {
	ac := NewApprovalCheckpoint(1 * time.Hour)

	err := ac.Approve("nonexistent", "user")
	if err == nil {
		t.Error("expected error for nonexistent request")
	}

	err = ac.ApproveByTask("nonexistent-task", "user")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestApprovalCheckpoint_HasPendingApproval(t *testing.T) {
	ac := NewApprovalCheckpoint(1 * time.Hour)

	if ac.HasPendingApproval("task-123") {
		t.Error("expected no pending approval initially")
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ac.RequestApproval(context.Background(), &ApprovalRequest{
			ID:     "test-1",
			TaskID: "task-123",
			Type:   ApprovalTypeMerge,
		})
	}()

	time.Sleep(50 * time.Millisecond)

	if !ac.HasPendingApproval("task-123") {
		t.Error("expected pending approval after request")
	}

	ac.Approve("test-1", "user")
	wg.Wait()

	// After approval, still in requests but not pending
	if ac.HasPendingApproval("task-123") {
		t.Error("expected no pending approval after resolution")
	}
}

func TestApprovalCheckpoint_Count(t *testing.T) {
	ac := NewApprovalCheckpoint(1 * time.Hour)

	if ac.Count() != 0 {
		t.Error("expected count 0 initially")
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ac.RequestApproval(context.Background(), &ApprovalRequest{
				ID:     "test-" + string(rune('0'+id)),
				TaskID: "task-" + string(rune('0'+id)),
				Type:   ApprovalTypeMerge,
			})
		}(i)
	}

	time.Sleep(50 * time.Millisecond)

	if ac.Count() != 5 {
		t.Errorf("expected count 5, got %d", ac.Count())
	}

	ac.Clear()
	wg.Wait()

	if ac.Count() != 0 {
		t.Errorf("expected count 0 after clear, got %d", ac.Count())
	}
}

// TestStoreBackedApprovalCheckpoint tests the store-backed version
func TestStoreBackedApprovalCheckpoint(t *testing.T) {
	// Create a temp database
	tmpDir := t.TempDir()
	store, err := NewSQLiteStore(tmpDir + "/test.db")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	sac := NewStoreBackedApprovalCheckpoint(store, 1*time.Hour)

	// Request approval in background
	var wg sync.WaitGroup
	var status ApprovalStatus
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		status, err = sac.RequestApproval(context.Background(), &ApprovalRequest{
			ID:          "test-store-1",
			TaskID:      "task-store-1",
			Type:        ApprovalTypeMerge,
			Description: "Test approval",
		})
		if err != nil {
			t.Errorf("failed to request approval: %v", err)
		}
	}()

	// Wait for request to be stored
	time.Sleep(100 * time.Millisecond)

	// Verify it's in the store
	pending, err := store.ListPendingApprovals(context.Background())
	if err != nil {
		t.Fatalf("failed to list pending: %v", err)
	}
	if len(pending) != 1 {
		t.Errorf("expected 1 pending, got %d", len(pending))
	}

	// Approve via store (simulates CLI)
	err = store.ResolveApprovalRequest(context.Background(), "test-store-1", "approved", "test-user")
	if err != nil {
		t.Fatalf("failed to resolve: %v", err)
	}

	// Wait for polling to detect the change
	wg.Wait()

	if status != ApprovalStatusApproved {
		t.Errorf("expected approved, got %s", status)
	}
}

// TestStoreBackedApprovalCheckpoint_Rejection tests store-backed rejection
func TestStoreBackedApprovalCheckpoint_Rejection(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewSQLiteStore(tmpDir + "/test.db")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	sac := NewStoreBackedApprovalCheckpoint(store, 1*time.Hour)

	var wg sync.WaitGroup
	var status ApprovalStatus
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		status, err = sac.RequestApproval(context.Background(), &ApprovalRequest{
			ID:          "test-reject-1",
			TaskID:      "task-reject-1",
			Type:        ApprovalTypeDestroy,
			Description: "Destroy worktree",
		})
		if err != nil {
			t.Errorf("failed to request approval: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Reject via store
	err = store.ResolveApprovalRequest(context.Background(), "test-reject-1", "rejected", "test-user")
	if err != nil {
		t.Fatalf("failed to reject: %v", err)
	}

	wg.Wait()

	if status != ApprovalStatusRejected {
		t.Errorf("expected rejected, got %s", status)
	}
}
