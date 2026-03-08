package effects

import (
	"sync"
)

// EventSource represents a readable event source for the selectEvents multiplexer.
// Any type that can produce streamEvent values through a Go channel implements this.
//
// M-ASYNC-IO: This is the core abstraction enabling multi-source event multiplexing.
// Sources are consumed by selectEventsLoop which dispatches events to a handler.
type EventSource interface {
	// Name returns a human-readable identifier for this source (e.g. "stdin", "ws:echo.example.com").
	Name() string

	// Priority returns the dispatch priority (higher = checked first in multiplexer).
	// When multiple sources have events ready, the highest priority wins.
	Priority() int

	// Events returns the read-only channel delivering events from this source.
	Events() <-chan streamEvent

	// Close signals the source to stop producing events and clean up resources.
	// Must be safe to call multiple times.
	Close()
}

// connSource adapts a StreamConnection to the EventSource interface.
// It wraps the existing eventBuffer channel without modifying StreamConnection.
type connSource struct {
	conn     *StreamConnection
	name     string
	priority int
	closed   bool
	mu       sync.Mutex
}

// NewConnSource creates an EventSource from an existing StreamConnection.
// The source reads from the connection's eventBuffer channel.
func NewConnSource(conn *StreamConnection, name string, priority int) EventSource {
	return &connSource{
		conn:     conn,
		name:     name,
		priority: priority,
	}
}

func (cs *connSource) Name() string               { return cs.name }
func (cs *connSource) Priority() int              { return cs.priority }
func (cs *connSource) Events() <-chan streamEvent { return cs.conn.eventBuffer }

func (cs *connSource) Close() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if cs.closed {
		return
	}
	cs.closed = true
	// Don't close the connection itself — disconnect() handles that.
	// We just mark ourselves as closed so the mux knows to skip us.
}

// StreamContext source management — stores EventSource handles alongside connections.

// AcquireSource registers a new event source and returns its ID.
func (sc *StreamContext) AcquireSource(source EventSource) int {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.sources == nil {
		sc.sources = make(map[int]EventSource)
	}

	id := sc.nextSourceID
	sc.nextSourceID++
	sc.sources[id] = source
	return id
}

// GetSource retrieves an event source by ID.
func (sc *StreamContext) GetSource(id int) (EventSource, bool) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	src, ok := sc.sources[id]
	return src, ok
}

// ReleaseSource removes a source from tracking.
func (sc *StreamContext) ReleaseSource(id int) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	delete(sc.sources, id)
}
