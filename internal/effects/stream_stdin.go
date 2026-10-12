package effects

import (
	"bufio"
	"io"
	"strings"
	"sync"
)

// stdinSource implements EventSource for line-buffered reading from an io.Reader
// (typically os.Stdin).
//
// M-ASYNC-IO: Spawns a goroutine that reads lines and sends SourceText events.
// The source can be closed, which causes the goroutine to exit on the next scan.
type stdinSource struct {
	name       string
	priority   int
	ch         chan streamEvent
	done       chan struct{}
	readerDone chan struct{} // closed after reader exit and input lease release
	once       sync.Once
}

// NewStdinSource creates an EventSource that reads lines from the given reader.
// Typically called with os.Stdin. The goroutine reads until the reader is exhausted,
// the source is closed, or an error occurs.
func NewStdinSource(reader io.Reader, name string, priority int) EventSource {
	return newOwnedStdinSource(reader, name, priority, nil)
}

// Release ownership only when the reader goroutine actually stops; Close cannot
// cancel an arbitrary blocking reader, so releasing on Close would permit theft.
func newOwnedStdinSource(reader io.Reader, name string, priority int, release func(), restore ...func(string)) EventSource {
	src := &stdinSource{
		name:       name,
		priority:   priority,
		ch:         make(chan streamEvent, 100), // Stdin is slow relative to WebSocket
		done:       make(chan struct{}),
		readerDone: make(chan struct{}),
	}
	go func() {
		defer close(src.readerDone)
		if release != nil {
			defer release()
		}
		if shared, ok := reader.(*bufio.Reader); ok && len(restore) > 0 {
			src.readOwnedLoop(shared, restore[0])
		} else {
			src.readLoop(reader)
		}
	}()
	return src
}

func (ss *stdinSource) Name() string               { return ss.name }
func (ss *stdinSource) Priority() int              { return ss.priority }
func (ss *stdinSource) Events() <-chan streamEvent { return ss.ch }

// PendingRead reports an outstanding borrowed read or input lease. Close only
// signals; arbitrary io.Reader calls cannot be interrupted safely by this owner.
func (ss *stdinSource) PendingRead() bool {
	select {
	case <-ss.readerDone:
		return false
	default:
		return true
	}
}

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

// Owned input uses the persistent reader directly: Scanner would retain and
// discard a second private read-ahead buffer when a source closes.
func (ss *stdinSource) readOwnedLoop(reader *bufio.Reader, restore func(string)) {
	defer close(ss.ch)
	for {
		select {
		case <-ss.done:
			return
		default:
		}
		line, err := reader.ReadString('\n')
		if len(line) == 0 {
			return
		}
		event := streamEvent{kind: "source_text", sourceName: ss.name, text: strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")}
		select {
		case <-ss.done:
			restore(line)
			return
		default:
		}
		select {
		case ss.ch <- event:
		case <-ss.done:
			restore(line)
			return
		}
		if err != nil {
			return
		}
	}
}
