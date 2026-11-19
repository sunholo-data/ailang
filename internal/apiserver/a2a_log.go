package apiserver

// One JSON line per /a2a/ request (event "a2a_request"): the JSON-RPC method,
// what came of it, how long it took, and for message/stream how many
// heartbeats were sent. Neither the request text nor the answer is logged,
// and no identity (the Bearer gate logs that). Cloud Logging parses the line
// as jsonPayload. It answers what a live caller such as Gemini Enterprise
// actually calls and how long it holds a stream (M-EXECUTOR-GE-A2A).

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

// a2aLogOut receives the lines. A var for tests.
var (
	a2aLogOut io.Writer = os.Stderr
	a2aLogMu  sync.Mutex
)

// a2aOutcome is what a method handler reports: result is the final task state
// (completed, failed) or why there was none (invalid_params, client_gone, ...).
type a2aOutcome struct {
	result     string
	heartbeats int
}

// a2aMaxLoggedMethod bounds a caller-chosen method name in the log.
const a2aMaxLoggedMethod = 64

func logA2ARequest(method string, out a2aOutcome, d time.Duration) {
	line := map[string]any{
		"event":       "a2a_request",
		"method":      clipText(method, a2aMaxLoggedMethod),
		"outcome":     out.result,
		"duration_ms": d.Milliseconds(),
	}
	if method == "message/stream" {
		line["heartbeats"] = out.heartbeats
	}
	b, err := json.Marshal(line)
	if err != nil {
		return
	}
	a2aLogMu.Lock()
	defer a2aLogMu.Unlock()
	_, _ = a2aLogOut.Write(append(b, '\n'))
}

// taskState reads a finished task's status.state.
func taskState(task map[string]any) string {
	if st, ok := task["status"].(map[string]any); ok {
		if s, ok := st["state"].(string); ok {
			return s
		}
	}
	return "unknown"
}
