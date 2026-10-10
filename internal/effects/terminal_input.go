package effects

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// State follows one execution across budget scopes; clones receive fresh state.
type terminalInputState struct {
	mu      sync.Mutex
	reader  *bufio.Reader
	active  *terminalSession
	reading bool
	lineEOF bool
}

func (ctx *EffContext) inputState() *terminalInputState {
	if ctx.terminalInput == nil {
		ctx.terminalInput = &terminalInputState{reader: ctx.stdinReader}
	}
	return ctx.terminalInput
}
func (ctx *EffContext) inputEndpoint() io.Reader {
	if ctx.IOReader != nil {
		return ctx.IOReader
	}
	return os.Stdin
}

// All descriptor aliases for a terminal share a device key, not an fd number.
var terminalOwners = struct {
	sync.Mutex
	devices  map[string]bool
	buffered map[string]map[*terminalInputState]io.Reader
}{devices: map[string]bool{}, buffered: map[string]map[*terminalInputState]io.Reader{}}

func reserveTerminalDevice(reader io.Reader) (func(), error) {
	endpoint, ok := reader.(interface{ Fd() uintptr })
	if !ok {
		return func() {}, nil
	}
	key, err := terminalDeviceKey(int(endpoint.Fd()))
	if err != nil {
		return nil, err
	}
	terminalOwners.Lock()
	defer terminalOwners.Unlock()
	if terminalOwners.devices[key] {
		return nil, fmt.Errorf("terminal input already owned")
	}
	terminalOwners.devices[key] = true
	var once sync.Once
	return func() {
		once.Do(func() { terminalOwners.Lock(); delete(terminalOwners.devices, key); terminalOwners.Unlock() })
	}, nil
}
func (ctx *EffContext) beginInputRead() (func(), error) {
	state := ctx.inputState()
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.active != nil || state.reading {
		return nil, fmt.Errorf("terminal input is busy")
	}
	release, err := reserveTerminalDevice(ctx.inputEndpoint())
	if err != nil {
		return nil, err
	}
	state.reading = true
	return func() {
		state.mu.Lock()
		state.reading = false
		buffered := 0
		if state.reader != nil {
			buffered = state.reader.Buffered()
		}
		if endpoint, ok := ctx.inputEndpoint().(interface{ Fd() uintptr }); ok {
			if key, keyErr := terminalDeviceKey(int(endpoint.Fd())); keyErr == nil {
				terminalOwners.Lock()
				if buffered > 0 {
					if terminalOwners.buffered[key] == nil {
						terminalOwners.buffered[key] = map[*terminalInputState]io.Reader{}
					}
					terminalOwners.buffered[key][state] = ctx.inputEndpoint()
				} else {
					delete(terminalOwners.buffered[key], state)
					if len(terminalOwners.buffered[key]) == 0 {
						delete(terminalOwners.buffered, key)
					}
				}
				terminalOwners.Unlock()
			}
		}
		state.mu.Unlock()
		release()
	}, nil
}

// Buffered bytes belong to their line reader even after the physical read ends.
// Native mode must not jump ahead of another context's already-buffered data.
func terminalBufferedInput(reader io.Reader) (bool, error) {
	endpoint, ok := reader.(interface{ Fd() uintptr })
	if !ok {
		return false, nil
	}
	key, err := terminalDeviceKey(int(endpoint.Fd()))
	if err != nil {
		return false, err
	}
	terminalOwners.Lock()
	defer terminalOwners.Unlock()
	for state, source := range terminalOwners.buffered[key] {
		owner, ok := source.(interface{ Fd() uintptr })
		if !ok {
			delete(terminalOwners.buffered[key], state)
			continue
		}
		current, queryErr := terminalDeviceKey(int(owner.Fd()))
		if queryErr != nil || current != key {
			delete(terminalOwners.buffered[key], state)
		}
	}
	if len(terminalOwners.buffered[key]) == 0 {
		delete(terminalOwners.buffered, key)
		return false, nil
	}
	return true, nil
}

// Called only by the owner of the input reservation, before releasing it. Put
// an un-emitted line back ahead of all existing read-ahead, exposing every byte
// in the replacement buffer so native activation continues to reject it.
func (ctx *EffContext) prependIOInput(line string) {
	if len(line) == 0 {
		return
	}
	state := ctx.inputState()
	state.mu.Lock()
	defer state.mu.Unlock()
	size := len(line) + state.reader.Buffered()
	restored := bufio.NewReaderSize(io.MultiReader(strings.NewReader(line), state.reader), size)
	_, _ = restored.Peek(size) // prefix plus already-buffered bytes; never blocks on new input
	state.reader = restored
	ctx.stdinReader = restored
}
