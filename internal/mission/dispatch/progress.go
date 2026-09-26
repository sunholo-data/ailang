package dispatch

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

const maxTrackedCalls = 256
const maxProgressEvents = 2048

// Progress contains diagnostics only. Tool inputs, outputs and model text are
// never retained. Repetition is exact byte identity, not an assessment of quality.
type Progress struct {
	ToolCalls         int       `json:"tool_calls"`
	CompletedTools    int       `json:"completed_tools"`
	RepeatedCalls     int       `json:"repeated_calls"`
	LastCompletedTool string    `json:"last_completed_tool,omitempty"`
	LastProgressTime  time.Time `json:"last_progress_time"`
	TrackingTruncated bool      `json:"tracking_truncated,omitempty"`
	JournalTruncated  bool      `json:"journal_truncated,omitempty"`
}

type progressObserver struct {
	executor.NoOpEventHandler
	mu       sync.Mutex
	progress Progress
	calls    map[[32]byte]int
	record   func(Event) error
	digest   string
	events   int
	err      error
}

func newProgressObserver(digest string, record func(Event) error) *progressObserver {
	return &progressObserver{calls: make(map[[32]byte]int), record: record, digest: digest}
}
func (p *progressObserver) OnToolUse(name, input string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.progress.ToolCalls++
	h := sha256.New()
	h.Write([]byte(name))
	h.Write([]byte{0})
	h.Write([]byte(input))
	var key [32]byte
	copy(key[:], h.Sum(nil))
	if p.calls[key] > 0 {
		p.progress.RepeatedCalls++
		p.calls[key]++
	} else if len(p.calls) < maxTrackedCalls {
		p.calls[key] = 1
	} else {
		p.progress.TrackingTruncated = true
	}
	p.emit()
}
func (p *progressObserver) OnToolResult(name, _ string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.progress.CompletedTools++
	p.progress.LastCompletedTool = safeToolName(name)
	p.progress.LastProgressTime = time.Now().UTC()
	p.emit()
}
func safeToolName(name string) string {
	if len(name) > 80 {
		sum := sha256.Sum256([]byte(name))
		return "sha256:" + hex.EncodeToString(sum[:])
	}
	for _, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-' || c == '.' || c == ':') {
			sum := sha256.Sum256([]byte(name))
			return "sha256:" + hex.EncodeToString(sum[:])
		}
	}
	return name
}
func (p *progressObserver) emit() {
	if p.events >= maxProgressEvents {
		p.progress.JournalTruncated = true
		return
	}
	if p.err != nil {
		return
	}
	p.events++
	if p.events == maxProgressEvents {
		p.progress.JournalTruncated = true
	}
	snapshot := p.progress
	p.err = p.record(Event{Kind: "progress", RequestDigest: p.digest, Report: &Report{Version: 1, RequestDigest: p.digest, Status: "executing", Progress: &snapshot}})
}
func (p *progressObserver) snapshot() Progress { p.mu.Lock(); defer p.mu.Unlock(); return p.progress }
func (p *progressObserver) failure() error     { p.mu.Lock(); defer p.mu.Unlock(); return p.err }
