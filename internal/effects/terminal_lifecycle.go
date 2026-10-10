package effects

import (
	"errors"
	"sync"
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
