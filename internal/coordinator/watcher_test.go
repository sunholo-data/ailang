package coordinator

import (
	"context"
	"testing"
	"time"
)

func TestNewMessageWatcher(t *testing.T) {
	store := NewMockMessageStore()
	watcher := NewMessageWatcher(store, time.Second)

	if watcher == nil {
		t.Fatal("watcher is nil")
	}

	if watcher.store == nil {
		t.Error("store is nil")
	}

	if watcher.pollInterval != time.Second {
		t.Errorf("expected poll interval 1s, got %v", watcher.pollInterval)
	}
}

func TestNewMessageWatcherDefaultInterval(t *testing.T) {
	store := NewMockMessageStore()
	watcher := NewMessageWatcher(store, 0)

	if watcher.pollInterval != 30*time.Second {
		t.Errorf("expected default poll interval 30s, got %v", watcher.pollInterval)
	}
}

func TestMessageWatcherPoll(t *testing.T) {
	store := NewMockMessageStore()
	store.AddMessage(&Message{
		ID:        "msg-1",
		Title:     "Test Bug",
		Content:   "Fix this bug",
		Type:      "bug",
		CreatedAt: time.Now(),
	})

	watcher := NewMessageWatcher(store, time.Second)

	// Poll should create a task
	err := watcher.poll()
	if err != nil {
		t.Fatalf("poll failed: %v", err)
	}

	// Should have seen the message
	if watcher.SeenCount() != 1 {
		t.Errorf("expected 1 seen message, got %d", watcher.SeenCount())
	}

	// Task should be in channel
	select {
	case task := <-watcher.tasksChan:
		if task.MessageID != "msg-1" {
			t.Errorf("expected message ID 'msg-1', got %q", task.MessageID)
		}
	default:
		t.Error("expected task in channel")
	}
}

func TestMessageWatcherDeduplication(t *testing.T) {
	store := NewMockMessageStore()
	store.AddMessage(&Message{
		ID:        "msg-1",
		Title:     "Test Bug",
		Content:   "Fix this bug",
		Type:      "bug",
		CreatedAt: time.Now(),
	})

	watcher := NewMessageWatcher(store, time.Second)

	// First poll
	if err := watcher.poll(); err != nil {
		t.Fatalf("first poll failed: %v", err)
	}

	// Drain the channel
	<-watcher.tasksChan

	// Second poll should not create duplicate
	if err := watcher.poll(); err != nil {
		t.Fatalf("second poll failed: %v", err)
	}

	// Should still have 1 seen message
	if watcher.SeenCount() != 1 {
		t.Errorf("expected 1 seen message after second poll, got %d", watcher.SeenCount())
	}

	// Channel should be empty
	select {
	case <-watcher.tasksChan:
		t.Error("should not have duplicate task")
	default:
		// Expected
	}
}

func TestMessageWatcherStart(t *testing.T) {
	store := NewMockMessageStore()
	store.AddMessage(&Message{
		ID:        "msg-1",
		Title:     "Test",
		Content:   "Test content",
		Type:      "feature",
		CreatedAt: time.Now(),
	})

	watcher := NewMessageWatcher(store, 100*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Start in goroutine
	done := make(chan error, 1)
	go func() {
		done <- watcher.Start(ctx)
	}()

	// Should receive a task
	select {
	case task := <-watcher.Tasks():
		if task == nil {
			t.Error("received nil task")
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for task")
	}

	// Wait for watcher to stop
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("watcher returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for watcher to stop")
	}
}

func TestClassifyPriority(t *testing.T) {
	tests := []struct {
		name     string
		msg      *Message
		wantPrio int
	}{
		{
			name:     "high priority explicit",
			msg:      &Message{Priority: "high"},
			wantPrio: 1,
		},
		{
			name:     "urgent priority",
			msg:      &Message{Priority: "urgent"},
			wantPrio: 1,
		},
		{
			name:     "medium priority",
			msg:      &Message{Priority: "medium"},
			wantPrio: 5,
		},
		{
			name:     "low priority",
			msg:      &Message{Priority: "low"},
			wantPrio: 10,
		},
		{
			name:     "bug type",
			msg:      &Message{Type: "bug"},
			wantPrio: 2,
		},
		{
			name:     "feature type",
			msg:      &Message{Type: "feature"},
			wantPrio: 5,
		},
		{
			name:     "docs type",
			msg:      &Message{Type: "docs"},
			wantPrio: 8,
		},
		{
			name:     "research type",
			msg:      &Message{Type: "research"},
			wantPrio: 9,
		},
		{
			name:     "default priority",
			msg:      &Message{},
			wantPrio: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyPriority(tt.msg)
			if got != tt.wantPrio {
				t.Errorf("classifyPriority() = %d, want %d", got, tt.wantPrio)
			}
		})
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name string
		msg  *Message
		want string
	}{
		{
			name: "use explicit title",
			msg:  &Message{Title: "Explicit Title", Content: "Some content"},
			want: "Explicit Title",
		},
		{
			name: "use first line",
			msg:  &Message{Title: "", Content: "First Line\nSecond Line"},
			want: "First Line",
		},
		{
			name: "short content no newline",
			msg:  &Message{Title: "", Content: "Short content"},
			want: "Short content",
		},
		{
			name: "truncate long content",
			msg: &Message{
				Title:   "",
				Content: "This is a very long content that should be truncated because it exceeds the maximum length of one hundred characters that we allow for titles",
			},
			want: "This is a very long content that should be truncated because it exceeds the maximum length of one hu...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitle(tt.msg)
			if got != tt.want {
				t.Errorf("extractTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMockMessageStore(t *testing.T) {
	store := NewMockMessageStore()

	// Initially empty
	unread, err := store.ListUnread()
	if err != nil {
		t.Fatalf("ListUnread failed: %v", err)
	}
	if len(unread) != 0 {
		t.Errorf("expected 0 unread, got %d", len(unread))
	}

	// Add a message
	store.AddMessage(&Message{ID: "msg-1", Content: "Test"})

	// Should have 1 unread
	unread, err = store.ListUnread()
	if err != nil {
		t.Fatalf("ListUnread failed: %v", err)
	}
	if len(unread) != 1 {
		t.Errorf("expected 1 unread, got %d", len(unread))
	}

	// Mark as read
	if err := store.MarkAsRead("msg-1"); err != nil {
		t.Fatalf("MarkAsRead failed: %v", err)
	}

	// Should have 0 unread
	unread, err = store.ListUnread()
	if err != nil {
		t.Fatalf("ListUnread failed: %v", err)
	}
	if len(unread) != 0 {
		t.Errorf("expected 0 unread after marking as read, got %d", len(unread))
	}
}

func TestMessageWatcherClearSeen(t *testing.T) {
	store := NewMockMessageStore()
	store.AddMessage(&Message{ID: "msg-1", Content: "Test"})

	watcher := NewMessageWatcher(store, time.Second)
	_ = watcher.poll()

	if watcher.SeenCount() != 1 {
		t.Errorf("expected 1 seen, got %d", watcher.SeenCount())
	}

	watcher.ClearSeen()

	if watcher.SeenCount() != 0 {
		t.Errorf("expected 0 seen after clear, got %d", watcher.SeenCount())
	}
}
