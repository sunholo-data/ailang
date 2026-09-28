package riggate

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Entry is one ledger line: one request the gateway saw, whatever happened to it.
// The ledger is the instrument the 2026-09-27 audit lacked — "who used the GPU,
// for how long, and what was refused" — which then took ~30 commands against a
// 190 MB ollama log. It is also one half of the bypass detector (design doc,
// Threat model): ollama's own request count minus this ledger's is traffic that
// reached ollama without passing the gate.
type Entry struct {
	TS         string `json:"ts"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	Model      string `json:"model,omitempty"`
	Decision   string `json:"decision"`
	Status     int    `json:"status"`
	DurationMS int64  `json:"duration_ms"`
	Holder     string `json:"holder,omitempty"`
	TokenOK    bool   `json:"token_ok"`
	Cancelled  bool   `json:"cancelled"`
	Reason     string `json:"reason,omitempty"`
}

// Ledger appends JSONL entries. Writes are serialized; a failed write is
// reported once to stderr and never blocks or fails the request it describes.
type Ledger struct {
	mu     sync.Mutex
	f      *os.File
	warned bool
}

// OpenLedger opens path for append, creating it 0640.
func OpenLedger(path string) (*Ledger, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return nil, fmt.Errorf("riggate: opening ledger %s: %w", path, err)
	}
	return &Ledger{f: f}, nil
}

// Write appends one entry.
func (l *Ledger) Write(e Entry) {
	b, err := json.Marshal(e)
	if err != nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, err := l.f.Write(append(b, '\n')); err != nil && !l.warned {
		l.warned = true
		fmt.Fprintf(os.Stderr, "riggate: ledger write failed (further failures silent): %v\n", err)
	}
}

// Close closes the ledger file.
func (l *Ledger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.f.Close()
}
