package effects

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// Signals belong to the process: restore every active native device before any
// host termination hook runs, even when embedded contexts use separate TTYs.
var activeTerminalSessions = struct {
	sync.Mutex
	sessions map[*terminalSession]struct{}
}{sessions: make(map[*terminalSession]struct{})}

func registerTerminalSession(session *terminalSession) {
	activeTerminalSessions.Lock()
	activeTerminalSessions.sessions[session] = struct{}{}
	activeTerminalSessions.Unlock()
}
func unregisterTerminalSession(session *terminalSession) {
	activeTerminalSessions.Lock()
	delete(activeTerminalSessions.sessions, session)
	activeTerminalSessions.Unlock()
}
func restoreActiveTerminals() error {
	activeTerminalSessions.Lock()
	sessions := make([]*terminalSession, 0, len(activeTerminalSessions.sessions))
	for session := range activeTerminalSessions.sessions {
		sessions = append(sessions, session)
	}
	activeTerminalSessions.Unlock()
	var failures []error
	for _, session := range sessions {
		if err := session.close(); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

// HandleWorkerSignal restores native terminals before bounded owned-worker
// shutdown. It never waits for the application evaluator to return.
func (ctx *EffContext) HandleWorkerSignal(code int) {
	if err := restoreActiveTerminals(); err != nil {
		fmt.Fprintf(os.Stderr, "terminal cleanup: %v\n", err)
	}
	if err := ctx.CloseWorkers(); err != nil {
		fmt.Fprintf(os.Stderr, "worker cleanup: %v\n", err)
	}
	if ctx.TerminalSignalExit != nil {
		ctx.TerminalSignalExit(code)
	}
}

// InstallWorkerSignalHandler is an explicit process-host operation. Embedders
// do not call it implicitly. The CLI uses one subscription for native terminal
// and non-TTY executions, so a blocked input/AI call cannot bypass shutdown.
func (ctx *EffContext) InstallWorkerSignalHandler(exit func(int)) func() {
	if exit == nil {
		return func() {}
	}
	ctx.TerminalSignalManaged = true
	ctx.TerminalSignalExit = exit
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		var received os.Signal
		select {
		case received = <-signals:
		case <-stop:
			select {
			case received = <-signals:
			default:
				return
			}
		}
		code := 130
		if received == syscall.SIGTERM {
			code = 143
		}
		ctx.HandleWorkerSignal(code)
	}()
	var once sync.Once
	return func() { once.Do(func() { signal.Stop(signals); close(stop); <-done }) }
}
