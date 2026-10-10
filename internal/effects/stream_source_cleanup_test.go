package effects

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

type observedBlockedReader struct {
	reader  *io.PipeReader
	entered chan struct{}
}

func (r *observedBlockedReader) Read(data []byte) (int, error) {
	select {
	case <-r.entered:
	default:
		close(r.entered)
	}
	return r.reader.Read(data)
}

func TestCloseSources_ReportsPendingBorrowedReaderWithoutClosingIt(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	blocked := &observedBlockedReader{reader: reader, entered: make(chan struct{})}
	released := make(chan struct{})
	source := newOwnedStdinSource(blocked, "blocked", 0, func() { close(released) }).(*stdinSource)
	stream := NewStreamContext()
	id := stream.AcquireSource(source)
	select {
	case <-blocked.entered:
	case <-time.After(time.Second):
		t.Fatal("read never started")
	}
	start := time.Now()
	if pending := stream.CloseSources(); pending != 1 {
		t.Fatalf("pending readers=%d, want 1", pending)
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Fatal("source closure waited for borrowed reader")
	}
	if _, ok := stream.GetSource(id); ok {
		t.Fatal("source remains selectable after host teardown")
	}
	if !source.PendingRead() {
		t.Fatal("blocked reader incorrectly claimed complete")
	}
	select {
	case <-released:
		t.Fatal("CloseSources prematurely released input lease")
	default:
	}
	// The caller, not runtime teardown, closes the borrowed pipe. Successful
	// write shows runtime never closed it; the closed source then releases its lease.
	if _, err := writer.Write([]byte("unblock\n")); err != nil {
		t.Fatalf("borrowed reader was closed: %v", err)
	}
	select {
	case <-source.readerDone:
	case <-time.After(time.Second):
		t.Fatal("reader did not finish")
	}
	if source.PendingRead() {
		t.Fatal("finished reader reported pending")
	}
	select {
	case <-released:
	default:
		t.Fatal("completion published before lease release")
	}
}

func TestCloseSources_PreservesBorrowedConnection(t *testing.T) {
	conn := &StreamConnection{eventBuffer: make(chan streamEvent, 1)}
	source := NewConnSource(conn, "borrowed", 0).(*connSource)
	stream := NewStreamContext()
	sourceID := stream.AcquireSource(source)
	connID, err := stream.AcquireConnection(conn)
	if err != nil {
		t.Fatal(err)
	}
	if pending := stream.CloseSources(); pending != 0 {
		t.Fatal(pending)
	}
	if _, ok := stream.GetSource(sourceID); ok {
		t.Fatal("closed adapter still registered")
	}
	if live, ok := stream.GetConnection(connID); !ok || live != conn {
		t.Fatal("borrowed connection was released")
	}
	conn.eventBuffer <- streamEvent{kind: "message", text: "transport remains open"}
	if event := <-conn.eventBuffer; event.text != "transport remains open" {
		t.Fatal("transport changed")
	}
}

func TestStdinSource_PendingReadCompletesAfterRelease(t *testing.T) {
	released := make(chan struct{})
	source := newOwnedStdinSource(strings.NewReader(""), "eof", 0, func() { close(released) }).(*stdinSource)
	select {
	case <-source.readerDone:
	case <-time.After(time.Second):
		t.Fatal("EOF not observed")
	}
	if source.PendingRead() {
		t.Fatal("EOF reader pending")
	}
	select {
	case <-released:
	default:
		t.Fatal("lease release must precede readerDone")
	}
	stream := NewStreamContext()
	stream.AcquireSource(source)
	if pending := stream.CloseSources(); pending != 0 {
		t.Fatal(pending)
	}
}

type stalledOwnedSource struct {
	EventSource
	stopOnce  sync.Once
	entered   chan struct{}
	release   chan struct{}
	completed chan struct{}
}

func (s *stalledOwnedSource) RequestStop() error {
	s.stopOnce.Do(func() { close(s.entered); <-s.release; close(s.completed) })
	return nil
}
func (s *stalledOwnedSource) Close()                { _ = s.RequestStop() }
func (s *stalledOwnedSource) StdinClosing() bool    { return false }
func (s *stalledOwnedSource) Done() <-chan struct{} { return s.completed }
func (s *stalledOwnedSource) Join(ctx context.Context) error {
	select {
	case <-s.completed:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestCloseSourcesAfterWorkers_DoesNotWaitForAlreadyStalledStop(t *testing.T) {
	source := &stalledOwnedSource{EventSource: newMockSource("stalled", 0, 1), entered: make(chan struct{}), release: make(chan struct{}), completed: make(chan struct{})}
	stream := NewStreamContext()
	id := stream.AcquireSource(source)
	go source.Close()
	<-source.entered
	cleanup := make(chan int, 1)
	go func() { cleanup <- stream.CloseSourcesAfterWorkers() }()
	select {
	case pending := <-cleanup:
		if pending != 0 {
			t.Fatal(pending)
		}
	case <-time.After(500 * time.Millisecond):
		close(source.release)
		<-source.completed
		t.Fatal("post-supervisor source cleanup repeated a blocked stop")
	}
	close(source.release)
	<-source.completed
	if _, ok := stream.GetSource(id); ok {
		t.Fatal("stalled source still registered")
	}
}
