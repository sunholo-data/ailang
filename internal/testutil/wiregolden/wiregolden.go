// Package wiregolden records what an HTTP client puts on the wire and pins it
// against golden files, so a client can be rewritten and proven to send the
// same bytes and to decode the same replies.
//
// It exists for the Ollama client (M-V1 public-surface hygiene, §7.1): the
// goldens under internal/ai/ollama/testdata/wire, internal/builtins/testdata
// and internal/messaging/testdata were captured while those packages still
// spoke through github.com/ollama/ollama/api, and the local wire client must
// reproduce them byte for byte. Regenerate only on purpose:
//
//	go test ./internal/ai/ollama/ -run Parity -update-wire-goldens
//
// A regenerated golden is a wire change and needs its own justification.
package wiregolden

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
)

var update = flag.Bool("update-wire-goldens", false, "rewrite wire golden files instead of comparing")

// Reply is one canned server response.
type Reply struct {
	Status int      // 0 means 200
	Lines  []string // written one per line ("\n"-terminated), flushed after each
	Body   string   // written verbatim when Lines is nil
	// Stall keeps the response open after the body until the client goes away,
	// for exercising client deadlines.
	Stall bool
}

// Exchange is one request as the server saw it.
type Exchange struct {
	Method string            `json:"method"`
	Path   string            `json:"path"`
	Query  string            `json:"query,omitempty"`
	Header map[string]string `json:"header"`
	Body   string            `json:"body"`
}

// Outcome is what one scenario produced: the requests sent and either the
// caller-visible result or the error.
type Outcome struct {
	Requests  []Exchange `json:"requests"`
	Result    any        `json:"result,omitempty"`
	Error     string     `json:"error,omitempty"`
	ErrorCode string     `json:"error_code,omitempty"`
}

// Server is an httptest server that answers with Replies in order and
// records every request.
type Server struct {
	*httptest.Server
	mu      sync.Mutex
	replies []Reply
	got     []Exchange
}

// NewServer starts a recording server; it is closed when t ends.
func NewServer(t *testing.T, replies ...Reply) *Server {
	t.Helper()
	s := &Server{replies: replies}
	s.Server = httptest.NewServer(http.HandlerFunc(s.serve))
	t.Cleanup(s.Close)
	return s
}

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	s.mu.Lock()
	n := len(s.got)
	s.got = append(s.got, Exchange{
		Method: r.Method,
		Path:   r.URL.Path,
		Query:  r.URL.RawQuery,
		Header: normalizeHeader(r.Header),
		Body:   string(body),
	})
	var rep Reply
	ok := n < len(s.replies)
	if ok {
		rep = s.replies[n]
	}
	s.mu.Unlock()
	if !ok {
		http.Error(w, "wiregolden: unexpected request", http.StatusTeapot)
		return
	}
	status := rep.Status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	flusher, _ := w.(http.Flusher)
	if rep.Lines != nil {
		for _, l := range rep.Lines {
			_, _ = io.WriteString(w, l+"\n")
			if flusher != nil {
				flusher.Flush()
			}
		}
	} else {
		_, _ = io.WriteString(w, rep.Body)
	}
	if rep.Stall {
		if flusher != nil {
			flusher.Flush()
		}
		<-r.Context().Done()
	}
}

// Exchanges returns the requests recorded so far.
func (s *Server) Exchanges() []Exchange {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Exchange(nil), s.got...)
}

// normalizeHeader flattens headers and replaces the runtime-specific parts of
// the User-Agent so a golden captured on one machine holds on every other.
func normalizeHeader(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k, v := range h {
		out[k] = strings.Join(v, ", ")
	}
	if ua, ok := out["User-Agent"]; ok {
		ua = strings.ReplaceAll(ua, "("+runtime.GOARCH+" "+runtime.GOOS+")", "(GOARCH GOOS)")
		ua = strings.ReplaceAll(ua, "Go/"+runtime.Version(), "Go/GOVERSION")
		out["User-Agent"] = ua
	}
	return out
}

// Check compares got with the golden file at path (relative to the test's
// package directory), or rewrites it under -update-wire-goldens.
func Check(t *testing.T, path string, got map[string]Outcome) {
	t.Helper()
	data, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatalf("wiregolden: marshal: %v", err)
	}
	data = append(data, '\n')
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wiregolden: wrote %s", path)
		return
	}
	want, err := os.ReadFile(path) //nolint:gosec // test fixture path
	if err != nil {
		t.Fatalf("wiregolden: read %s: %v (run with -update-wire-goldens to create it)", path, err)
	}
	if bytes.Equal(want, data) {
		return
	}
	var wantMap map[string]json.RawMessage
	if err := json.Unmarshal(want, &wantMap); err != nil {
		t.Fatalf("wiregolden: %s is not a scenario map: %v", path, err)
	}
	gotMap := map[string]json.RawMessage{}
	_ = json.Unmarshal(data, &gotMap)
	names := map[string]bool{}
	for k := range wantMap {
		names[k] = true
	}
	for k := range gotMap {
		names[k] = true
	}
	sorted := make([]string, 0, len(names))
	for k := range names {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	for _, k := range sorted {
		w, g := wantMap[k], gotMap[k]
		if !bytes.Equal(compact(w), compact(g)) {
			t.Errorf("wiregolden: %s scenario %q differs from the golden\n--- golden\n%s\n--- got\n%s", path, k, indent(w), indent(g))
		}
	}
	if !t.Failed() {
		t.Errorf("wiregolden: %s differs only in formatting; rerun with -update-wire-goldens", path)
	}
}

func compact(b []byte) []byte {
	var buf bytes.Buffer
	if err := json.Compact(&buf, b); err != nil {
		return b
	}
	return buf.Bytes()
}

func indent(b []byte) string {
	if b == nil {
		return "<missing>"
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, b, "", "  "); err != nil {
		return string(b)
	}
	return buf.String()
}
