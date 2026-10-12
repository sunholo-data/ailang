package effects

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/sunholo-data/ailang/internal/eval"
)

func init() {
	RegisterOp("IO", "terminalInfo", TerminalInfo)
	RegisterOp("IO", "withTerminal", TerminalWith)
	RegisterOp("IO", "terminalReadEvent", TerminalReadEvent)
}

type terminalDevice interface {
	read([]byte) (int, error)
	poll(int) (bool, error)
	size() (int, int, error)
	restore() error
}
type terminalSession struct {
	mu                sync.Mutex
	id                int
	device            terminalDevice
	output            io.Writer
	ctx               *EffContext
	state             *terminalInputState
	release           func()
	decoder           terminalDecoder
	columns, rows     int
	eof, closed       bool
	alternate, hidden bool
	cleanupErr        error
	stop              chan struct{}
	signals           chan os.Signal
	signalDone        chan struct{}
}

var terminalSessionID atomic.Int64

func terminalErr(name, message string) eval.Value {
	return terminalTag("Err", terminalTag(name, &eval.StringValue{Value: message}))
}
func terminalOk(value eval.Value) eval.Value { return terminalTag("Ok", value) }
func terminalSize(columns, rows int) eval.Value {
	return &eval.RecordValue{Fields: map[string]eval.Value{"columns": &eval.IntValue{Value: columns}, "rows": &eval.IntValue{Value: rows}}}
}
func terminalFD(endpoint interface{}) (int, bool) {
	value, ok := endpoint.(interface{ Fd() uintptr })
	if !ok {
		return 0, false
	}
	return int(value.Fd()), true
}

// TerminalInfo queries the configured endpoints without changing terminal mode.
func TerminalInfo(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("terminalInfo: expected 0 arguments")
	}
	if !terminalSupported() {
		return terminalErr("Unsupported", "native terminals require Linux or macOS"), nil
	}
	in, hasIn := terminalFD(ctx.inputEndpoint())
	out, hasOut := terminalFD(ctx.GetIOWriter())
	inputTTY := hasIn && terminalIsTTY(in)
	outputTTY := hasOut && terminalIsTTY(out)
	size := terminalTag("None")
	if outputTTY {
		columns, rows, err := terminalQuerySize(out)
		if err != nil {
			return terminalErr("QueryFailure", err.Error()), nil
		}
		if columns <= 0 || rows <= 0 {
			return terminalErr("QueryFailure", "terminal reported nonpositive dimensions"), nil
		}
		size = terminalTag("Some", terminalSize(columns, rows))
	}
	return terminalOk(&eval.RecordValue{Fields: map[string]eval.Value{
		"input_tty": &eval.BoolValue{Value: inputTTY}, "output_tty": &eval.BoolValue{Value: outputTTY},
		"raw_supported": &eval.BoolValue{Value: inputTTY && outputTTY && ctx.TerminalSignalExit != nil}, "size": size,
	}}), nil
}

// TerminalWith owns acquisition and restoration, including panic/exit unwind.
func TerminalWith(ctx *EffContext, args []eval.Value) (result eval.Value, err error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("withTerminal: expected options and callback")
	}
	options, ok := args[0].(*eval.RecordValue)
	if !ok {
		return nil, fmt.Errorf("withTerminal: expected options record")
	}
	alternate, altOK := options.Fields["alternate_screen"].(*eval.BoolValue)
	hidden, hideOK := options.Fields["hide_cursor"].(*eval.BoolValue)
	if !altOK || !hideOK {
		return nil, fmt.Errorf("withTerminal: expected boolean alternate_screen/hide_cursor")
	}
	if !terminalSupported() {
		return terminalErr("Unsupported", "native terminals require Linux or macOS"), nil
	}
	if ctx.TerminalSignalExit == nil {
		return terminalErr("Unsupported", "host must provide terminal signal termination integration"), nil
	}
	if ctx.FnCaller == nil {
		return terminalErr("Unsupported", "host has no callback dispatcher"), nil
	}
	in, hasIn := terminalFD(ctx.inputEndpoint())
	out, hasOut := terminalFD(ctx.GetIOWriter())
	if !hasIn || !terminalIsTTY(in) {
		return terminalErr("NotTTY", "configured input is not a terminal"), nil
	}
	if !hasOut || !terminalIsTTY(out) {
		return terminalErr("NotTTY", "configured output is not a terminal"), nil
	}
	state := ctx.inputState()
	state.mu.Lock()
	if state.active != nil || state.reading {
		state.mu.Unlock()
		return terminalErr("Busy", "input reader already active"), nil
	}
	if state.reader != nil && state.reader.Buffered() > 0 {
		state.mu.Unlock()
		return terminalErr("Busy", "line reader has unread buffered input"), nil
	}
	release, acquireErr := reserveTerminalDevice(ctx.inputEndpoint())
	if acquireErr != nil {
		state.mu.Unlock()
		return terminalErr("Busy", acquireErr.Error()), nil
	}
	buffered, bufferErr := terminalBufferedInput(ctx.inputEndpoint())
	if bufferErr != nil || buffered {
		release()
		state.mu.Unlock()
		if bufferErr != nil {
			return terminalErr("InputFailure", bufferErr.Error()), nil
		}
		return terminalErr("Busy", "another line reader has unread buffered input"), nil
	}
	var signals chan os.Signal
	if !ctx.TerminalSignalManaged {
		signals = make(chan os.Signal, 2)
		signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	}
	device, acquireErr := terminalActivate(in, out)
	if acquireErr != nil {
		signal.Stop(signals)
		release()
		state.mu.Unlock()
		return terminalErr("InputFailure", acquireErr.Error()), nil
	}
	session := &terminalSession{id: int(terminalSessionID.Add(1)), device: device, ctx: ctx, state: state, release: release, output: ctx.GetIOWriter(), alternate: alternate.Value, hidden: hidden.Value, stop: make(chan struct{}), signals: signals}
	state.active = session
	registerTerminalSession(session)
	state.mu.Unlock()
	return session.run(args[1])
}

func (session *terminalSession) run(body eval.Value) (result eval.Value, err error) {
	ctx := session.ctx
	device := session.device
	var acquireErr error

	defer func() {
		cleanupErr := session.close()
		if session.signalDone != nil {
			<-session.signalDone
		} else {
			session.terminatePendingSignal()
		}

		if cleanupErr != nil {
			if err != nil {
				err = errors.Join(err, fmt.Errorf("terminal cleanup: %w", cleanupErr))
			} else if tagged, ok := result.(*eval.TaggedValue); ok && tagged.CtorName == "Err" {
				fmt.Fprintf(os.Stderr, "terminal cleanup: %v\n", cleanupErr)
			} else if result != nil {
				result = terminalErr("CleanupFailure", cleanupErr.Error())
			} else {
				fmt.Fprintf(os.Stderr, "terminal cleanup: %v\n", cleanupErr)
			}
		}
	}()
	session.columns, session.rows, acquireErr = device.size()
	if acquireErr != nil || session.columns <= 0 || session.rows <= 0 {
		if acquireErr == nil {
			acquireErr = fmt.Errorf("terminal reported nonpositive dimensions")
		}
		return terminalErr("QueryFailure", acquireErr.Error()), nil
	}
	sequences := ""
	if session.alternate {
		sequences += "\x1b[?1049h"
	}
	if session.hidden {
		sequences += "\x1b[?25l"
	}
	if _, acquireErr = io.WriteString(session.output, sequences); acquireErr == nil {
		acquireErr = ctx.FlushIO()
	}
	if acquireErr != nil {
		return terminalErr("InputFailure", acquireErr.Error()), nil
	}
	if session.signals != nil {
		session.watchSignals()
	}
	value, callErr := ctx.FnCaller(body, terminalTag("TerminalSession", &eval.IntValue{Value: session.id}))
	if callErr != nil {
		return nil, callErr
	}
	return terminalOk(value), nil
}

func (s *terminalSession) watchSignals() {
	s.signalDone = make(chan struct{})
	go func() {
		defer close(s.signalDone)
		select {
		case sig := <-s.signals:
			s.terminateSignal(sig)
		case <-s.stop:
			// close called signal.Stop before closing stop. Stop guarantees no further
			// delivery; drain a signal already diverted into our queue before returning.
			s.terminatePendingSignal()
		}
	}()
}
func (s *terminalSession) terminatePendingSignal() {
	if s.signals == nil {
		return
	}
	select {
	case sig := <-s.signals:
		s.terminateSignal(sig)
	default:
	}
}
func (s *terminalSession) terminateSignal(sig os.Signal) {

	code := 130
	if sig == syscall.SIGTERM {
		code = 143
	}
	s.ctx.HandleWorkerSignal(code)
}

func (s *terminalSession) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return s.cleanupErr
	}
	s.closed = true
	if s.signals != nil {
		signal.Stop(s.signals)
	}
	close(s.stop)
	sequences := ""
	if s.hidden {
		sequences += "\x1b[?25h"
	}
	if s.alternate {
		sequences += "\x1b[?1049l"
	}
	restoreErr := s.device.restore()
	_, writeErr := io.WriteString(s.output, sequences)
	flushErr := s.ctx.FlushIO()
	s.cleanupErr = errors.Join(writeErr, flushErr, restoreErr)
	s.state.mu.Lock()
	if s.state.active == s {
		s.state.active = nil
	}
	s.state.mu.Unlock()
	s.release()
	unregisterTerminalSession(s)
	return s.cleanupErr
}

// TerminalReadEvent validates the context lease before touching its descriptor.
func TerminalReadEvent(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("terminalReadEvent: expected session and timeout")
	}
	timeout, ok := args[1].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("terminalReadEvent: expected integer timeout")
	}
	if timeout.Value < -1 || timeout.Value > 60000 {
		return terminalTag("Err", terminalTag("InvalidTimeout", timeout)), nil
	}
	handle, ok := args[0].(*eval.TaggedValue)
	if !ok || handle.CtorName != "TerminalSession" || len(handle.Fields) != 1 {
		return terminalErr("InvalidSession", "invalid session handle"), nil
	}
	id, ok := handle.Fields[0].(*eval.IntValue)
	if !ok {
		return terminalErr("InvalidSession", "invalid session handle"), nil
	}
	state := ctx.inputState()
	state.mu.Lock()
	session := state.active
	state.mu.Unlock()
	if session == nil || session.id != id.Value {
		return terminalErr("InvalidSession", "session is stale or belongs to another context"), nil
	}
	deadline := time.Now().Add(time.Duration(timeout.Value) * time.Millisecond)
	for {
		session.mu.Lock()
		if session.closed {
			session.mu.Unlock()
			return terminalErr("InvalidSession", "session is closed"), nil
		}
		event, readErr := session.nextEvent(deadline, int(timeout.Value))
		session.mu.Unlock()
		if readErr != nil {
			var query *terminalQueryError
			if errors.As(readErr, &query) {
				return terminalErr("QueryFailure", query.Error()), nil
			}
			return terminalErr("InputFailure", readErr.Error()), nil
		}
		if event != nil {
			return terminalOk(event), nil
		}
	}
}
func (s *terminalSession) nextEvent(deadline time.Time, timeout int) (eval.Value, error) {
	now := time.Now()
	if ev, err := s.decoder.next(now, s.eof); ev != nil || err != nil {
		return ev, err
	}
	if s.eof {
		return terminalTag("EndOfInput"), nil
	}
	columns, rows, err := s.device.size()
	if err != nil {
		return nil, &terminalQueryError{err}
	}
	if columns <= 0 || rows <= 0 {
		return nil, &terminalQueryError{fmt.Errorf("terminal reported nonpositive dimensions")}
	}
	if columns != s.columns || rows != s.rows {
		s.columns = columns
		s.rows = rows
		return terminalTag("Resize", terminalSize(columns, rows)), nil
	}
	wait := 20
	if timeout >= 0 {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			wait = 0
		} else if remaining < 20*time.Millisecond {
			wait = int((remaining + time.Millisecond - 1) / time.Millisecond)
		}
	}
	if !s.decoder.escapeAt.IsZero() {
		remaining := terminalEscapeDelay - now.Sub(s.decoder.escapeAt)
		if remaining < time.Duration(wait)*time.Millisecond {
			wait = int((remaining + time.Millisecond - 1) / time.Millisecond)
			if wait < 0 {
				wait = 0
			}
		}
	}
	ready, err := s.device.poll(wait)
	if err != nil {
		return nil, err
	}
	if ready {
		data := make([]byte, 256)
		n, err := s.device.read(data)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n == 0 || err == io.EOF {
			s.eof = true
		}
		if err = s.decoder.feed(data[:n], time.Now()); err != nil {
			return nil, err
		}
		return s.decoder.next(time.Now(), s.eof)
	}
	if timeout >= 0 && !time.Now().Before(deadline) {
		return terminalTag("Idle"), nil
	}
	return nil, nil
}

type terminalQueryError struct{ error }
