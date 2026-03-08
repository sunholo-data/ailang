package effects

import (
	"bufio"
	"io"
	"sync"
)

// stdinSource implements EventSource for line-buffered reading from an io.Reader
// (typically os.Stdin).
//
// M-ASYNC-IO: Spawns a goroutine that reads lines and sends SourceText events.
// The source can be closed, which causes the goroutine to exit on the next scan.
type stdinSource struct {
	name     string
	priority int
	ch       chan streamEvent
	done     chan struct{}
	once     sync.Once
}

// NewStdinSource creates an EventSource that reads lines from the given reader.
// Typically called with os.Stdin. The goroutine reads until the reader is exhausted,
// the source is closed, or an error occurs.
func NewStdinSource(reader io.Reader, name string, priority int) EventSource {
	src := &stdinSource{
		name:     name,
		priority: priority,
		ch:       make(chan streamEvent, 100), // Stdin is slow relative to WebSocket
		done:     make(chan struct{}),
	}
	go src.readLoop(reader)
	return src
}

func (ss *stdinSource) Name() string               { return ss.name }
func (ss *stdinSource) Priority() int              { return ss.priority }
func (ss *stdinSource) Events() <-chan streamEvent { return ss.ch }

func (ss *stdinSource) Close() {
	ss.once.Do(func() {
		close(ss.done)
	})
}

func (ss *stdinSource) readLoop(reader io.Reader) {
	defer close(ss.ch)
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		select {
		case <-ss.done:
			return
		default:
		}

		evt := streamEvent{
			kind:       "source_text",
			sourceName: ss.name,
			text:       scanner.Text(),
		}

		select {
		case ss.ch <- evt:
		case <-ss.done:
			return
		}
	}
	// Scanner exhausted (EOF or error) — goroutine exits, channel closes via defer
}
