//go:build !js

package effects

import (
	"context"
	"testing"
	"time"
)

func TestProcessSource_EchoOutput(t *testing.T) {
	src, err := NewProcessSource(context.Background(), "echo", []string{"hello world"}, "test", 5, 1024)
	if err != nil {
		t.Fatalf("NewProcessSource: %v", err)
	}
	defer src.Close()

	select {
	case evt, ok := <-src.Events():
		if !ok {
			t.Fatal("channel closed before receiving event")
		}
		if evt.kind != "source_bytes" {
			t.Errorf("kind = %q, want source_bytes", evt.kind)
		}
		if evt.sourceName != "test" {
			t.Errorf("sourceName = %q, want test", evt.sourceName)
		}
		got := string(evt.data)
		if got != "hello world\n" {
			t.Errorf("data = %q, want %q", got, "hello world\n")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for event")
	}

	// Channel should close after echo exits (EOF)
	select {
	case _, ok := <-src.Events():
		if ok {
			t.Error("expected channel to close after EOF")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for channel close")
	}
}

func TestProcessSource_ChunkSize(t *testing.T) {
	// Use printf to output exactly 10 bytes without trailing newline
	src, err := NewProcessSource(context.Background(), "printf", []string{"0123456789"}, "chunk", 1, 4)
	if err != nil {
		t.Fatalf("NewProcessSource: %v", err)
	}
	defer src.Close()

	var chunks []string
	for evt := range src.Events() {
		chunks = append(chunks, string(evt.data))
	}

	// 10 bytes / 4 chunk = 2 full chunks + 1 partial (2 bytes)
	if len(chunks) != 3 {
		t.Fatalf("got %d chunks, want 3: %v", len(chunks), chunks)
	}
	if chunks[0] != "0123" {
		t.Errorf("chunk[0] = %q, want %q", chunks[0], "0123")
	}
	if chunks[1] != "4567" {
		t.Errorf("chunk[1] = %q, want %q", chunks[1], "4567")
	}
	if chunks[2] != "89" {
		t.Errorf("chunk[2] = %q, want %q", chunks[2], "89")
	}
}

func TestProcessSource_PartialFinalChunk(t *testing.T) {
	// 3 bytes output, chunkSize=1024 → one partial chunk
	src, err := NewProcessSource(context.Background(), "printf", []string{"abc"}, "partial", 1, 1024)
	if err != nil {
		t.Fatalf("NewProcessSource: %v", err)
	}
	defer src.Close()

	var events []streamEvent
	for evt := range src.Events() {
		events = append(events, evt)
	}

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if string(events[0].data) != "abc" {
		t.Errorf("data = %q, want %q", string(events[0].data), "abc")
	}
}

func TestProcessSource_CloseKillsProcess(t *testing.T) {
	// Start a long-running process
	src, err := NewProcessSource(context.Background(), "sleep", []string{"60"}, "sleeper", 1, 1024)
	if err != nil {
		t.Fatalf("NewProcessSource: %v", err)
	}

	// Close should kill the process
	src.Close()

	// Channel should close shortly after
	select {
	case _, ok := <-src.Events():
		if ok {
			// Might get an event, but eventually should close
			select {
			case <-src.Events():
			case <-time.After(6 * time.Second):
				t.Fatal("channel didn't close after Close()")
			}
		}
	case <-time.After(6 * time.Second):
		t.Fatal("channel didn't close after Close()")
	}
}

func TestProcessSource_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	src, err := NewProcessSource(ctx, "sleep", []string{"60"}, "ctx-test", 1, 1024)
	if err != nil {
		t.Fatalf("NewProcessSource: %v", err)
	}
	defer src.Close()

	// Cancel parent context
	cancel()

	// Channel should close as subprocess is killed
	select {
	case <-src.Events():
		// ok — either event or close
	case <-time.After(6 * time.Second):
		t.Fatal("channel didn't close after context cancellation")
	}
}

func TestProcessSource_CommandNotFound(t *testing.T) {
	_, err := NewProcessSource(context.Background(), "/nonexistent/binary", nil, "bad", 1, 1024)
	if err == nil {
		t.Fatal("expected error for nonexistent command, got nil")
	}
}

func TestProcessSource_InvalidChunkSize(t *testing.T) {
	_, err := NewProcessSource(context.Background(), "echo", []string{"test"}, "bad", 1, 0)
	if err == nil {
		t.Fatal("expected error for zero chunkSize")
	}

	_, err = NewProcessSource(context.Background(), "echo", []string{"test"}, "bad", 1, -1)
	if err == nil {
		t.Fatal("expected error for negative chunkSize")
	}
}

func TestProcessSource_EOFClosesCleanly(t *testing.T) {
	// Quick command that exits immediately
	src, err := NewProcessSource(context.Background(), "true", nil, "quick", 1, 1024)
	if err != nil {
		t.Fatalf("NewProcessSource: %v", err)
	}
	defer src.Close()

	// Channel should close after process exits
	select {
	case _, ok := <-src.Events():
		if ok {
			t.Log("got an event from 'true' — unexpected but not fatal")
		}
		// Channel closed or we got the only event
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for channel close")
	}
}

func TestProcessSource_NameAndPriority(t *testing.T) {
	src, err := NewProcessSource(context.Background(), "echo", []string{"x"}, "myproc", 42, 1024)
	if err != nil {
		t.Fatalf("NewProcessSource: %v", err)
	}
	defer src.Close()

	if src.Name() != "myproc" {
		t.Errorf("Name() = %q, want %q", src.Name(), "myproc")
	}
	if src.Priority() != 42 {
		t.Errorf("Priority() = %d, want 42", src.Priority())
	}
}
