package motoko

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
)

// Local ollama turns stream by default (2026-10-16).
//
// Buffered, a whole generation arrives as one response: motoko's session log
// gains nothing while it runs, so the idle watchdog cannot tell a long turn
// from a hung one. Measured on the rig: 9 of 39 v0.54.9 qwen3.8 runs were
// killed at 10 minutes mid-generation — every session ended on
// provider_call_prepared, rig-gate logged ollama answering 200 and the call
// running 618 s before we cancelled it. Streamed, motoko journals each delta
// and the client's own windows (idle between bytes, time to first byte, hard
// deadline) catch a wedged stream.
//
// The flag only reaches the runtime through motoko's child-env allowlist
// (motoko_agent fix/forward-ollama-stream-env); a motoko without it ignores
// the value and stays buffered, which is why the watchdog below also defers to
// the buffered call's own bound.

// isOllamaModel reports whether motoko will call the model through the AILANG
// runtime's ollama client.
func isOllamaModel(model string) bool { return strings.HasPrefix(model, "ollama/") }

// ollamaStreamEnv returns the env that opts an ollama run into streaming, or
// nothing when an operator already chose (an explicit AILANG_OLLAMA_V1_STREAM
// wins either way) or when streaming would refuse the configured timeout.
// Streaming rejects a non-positive AILANG_OLLAMA_HTTP_TIMEOUT_SEC at client
// construction, so flipping the flag under one would turn a working buffered
// run into a startup failure; that case keeps the buffered path and says so.
func ollamaStreamEnv(model string, warn io.Writer) []string {
	if !isOllamaModel(model) {
		return nil
	}
	if config.RawSet(config.EnvOllamaV1Stream) {
		return nil
	}
	if v := strings.TrimSpace(config.OllamaHTTPTimeoutSec()); v != "" {
		if n, err := strconv.Atoi(v); err != nil || n <= 0 {
			fmt.Fprintf(warn, "[motoko] %s=%q is not a positive deadline, which streaming refuses; this run stays on the buffered ollama path\n",
				config.EnvOllamaHTTPTimeoutSec, v)
			return nil
		}
	}
	return []string{config.EnvOllamaV1Stream + "=1"}
}

// ollamaCallBound is how long one in-flight ollama call may legitimately run
// before the client's own deadline fires: AILANG_OLLAMA_HTTP_TIMEOUT_SEC, or
// the client's default for the path (300 s buffered, 3600 s streaming). Zero
// means the client has no deadline (buffered with a non-positive value). The
// env is the CHILD's, so it reflects what this run will actually use.
func ollamaCallBound(childEnv []string) time.Duration {
	streaming, raw := false, ""
	for _, kv := range childEnv {
		switch {
		case kv == config.EnvOllamaV1Stream+"=1":
			streaming = true
		case strings.HasPrefix(kv, config.EnvOllamaHTTPTimeoutSec+"="):
			raw = strings.TrimSpace(strings.TrimPrefix(kv, config.EnvOllamaHTTPTimeoutSec+"="))
		}
	}
	def := 300
	if streaming {
		def = 3600
	}
	if raw == "" {
		return time.Duration(def) * time.Second
	}
	n, err := strconv.Atoi(raw)
	switch {
	case err != nil:
		return time.Duration(def) * time.Second
	case n <= 0:
		return 0
	}
	return time.Duration(n) * time.Second
}

// inFlightEvents are the session events after which a provider call is open:
// the request went out and its response has not finished. While one is the
// latest event, silence in the log is the model working, not motoko hanging.
var inFlightEvents = map[string]bool{
	"provider_call_prepared": true,
	"thinking_stream_start":  true,
	"thinking_delta":         true,
	"reasoning_delta":        true,
}

// sessionTailBytes bounds the read: the last event is all that matters, and a
// single event line is far smaller than this.
const sessionTailBytes = 16 << 10

// lastEventType returns the "type" of the final complete line of a session
// JSONL, "" when it cannot be read.
func lastEventType(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return ""
	}
	off := fi.Size() - sessionTailBytes
	if off < 0 {
		off = 0
	}
	buf := make([]byte, fi.Size()-off)
	if _, err := f.ReadAt(buf, off); err != nil && err != io.EOF {
		return ""
	}
	lines := bytes.Split(bytes.TrimRight(buf, "\n"), []byte("\n"))
	if len(lines) == 0 {
		return ""
	}
	var ev struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(lines[len(lines)-1], &ev) != nil {
		return ""
	}
	return ev.Type
}

// sessionCallInFlight reports whether the run's latest session event leaves a
// provider call open.
func sessionCallInFlight(workspace, sessionID, repo string) func() bool {
	return func() bool {
		p, err := findSessionJSONL(workspace, sessionID, repo)
		if err != nil {
			return false
		}
		return inFlightEvents[lastEventType(p)]
	}
}
