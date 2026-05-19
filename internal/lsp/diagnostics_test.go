package lsp

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// diagWaitTimeout returns the per-wait timeout for diagnostic-arrival
// assertions. Windows GitHub Actions runners have slower filesystem +
// LSP pipeline latency than Linux/macOS — the 5s budget that works
// locally times out in CI. Tripling on Windows preserves the test's
// intent without making non-Windows runs slower.
func diagWaitTimeout() time.Duration {
	if runtime.GOOS == "windows" {
		return 15 * time.Second
	}
	return 5 * time.Second
}

// diagSink captures publishDiagnostics notifications pushed from the server
// to the test client. drain returns the most recent params for the given
// URI, blocking up to timeout for one to arrive.
type diagSink struct {
	mu     sync.Mutex
	cond   *sync.Cond
	byURI  map[protocol.DocumentURI]*protocol.PublishDiagnosticsParams
	closed bool
}

func newDiagSink() *diagSink {
	s := &diagSink{byURI: map[protocol.DocumentURI]*protocol.PublishDiagnosticsParams{}}
	s.cond = sync.NewCond(&s.mu)
	return s
}

func (d *diagSink) handler(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	if req.Method() != "textDocument/publishDiagnostics" {
		return jsonrpc2.MethodNotFoundHandler(ctx, reply, req)
	}
	var params protocol.PublishDiagnosticsParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(ctx, nil, err)
	}
	d.mu.Lock()
	d.byURI[params.URI] = &params
	d.cond.Broadcast()
	d.mu.Unlock()
	return reply(ctx, nil, nil)
}

func (d *diagSink) wait(u protocol.DocumentURI, timeout time.Duration) (*protocol.PublishDiagnosticsParams, bool) {
	deadline := time.Now().Add(timeout)
	d.mu.Lock()
	defer d.mu.Unlock()
	for {
		if p, ok := d.byURI[u]; ok {
			return p, true
		}
		if d.closed {
			return nil, false
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, false
		}
		// sync.Cond doesn't support timeout; spawn a goroutine to wake us.
		timer := time.AfterFunc(remaining, func() {
			d.mu.Lock()
			d.cond.Broadcast()
			d.mu.Unlock()
		})
		d.cond.Wait()
		timer.Stop()
		if time.Now().After(deadline) {
			if p, ok := d.byURI[u]; ok {
				return p, true
			}
			return nil, false
		}
	}
}

// startTestServer spins up an in-memory LSP server connected to a test
// client. Returns the connected client conn and a sink that captures
// publishDiagnostics notifications. Caller is responsible for cancellation
// via the returned context.
func startTestServer(t *testing.T) (context.Context, jsonrpc2.Conn, *diagSink, func()) {
	t.Helper()
	// Server lifecycle must outlive the per-wait diagnostic timeout below
	// — otherwise the parent context expires while a wait is still in
	// flight. 3× diagWaitTimeout gives generous headroom on Windows.
	ctx, cancel := context.WithTimeout(context.Background(), 3*diagWaitTimeout())
	srvSide, cliSide := net.Pipe()
	srv := NewServer(nil)

	serverDone := make(chan error, 1)
	go func() { serverDone <- srv.Run(ctx, srvSide) }()

	sink := newDiagSink()
	cliStream := jsonrpc2.NewStream(cliSide)
	cliConn := jsonrpc2.NewConn(cliStream)
	cliConn.Go(ctx, sink.handler)

	// Drive the handshake so the server is `initialized` before tests send
	// document notifications.
	if _, err := cliConn.Call(ctx, "initialize", &protocol.InitializeParams{}, &protocol.InitializeResult{}); err != nil {
		cancel()
		t.Fatalf("initialize: %v", err)
	}
	if err := cliConn.Notify(ctx, "initialized", &protocol.InitializedParams{}); err != nil {
		cancel()
		t.Fatalf("initialized: %v", err)
	}

	cleanup := func() {
		_, _ = cliConn.Call(ctx, "shutdown", nil, nil)
		_ = cliConn.Notify(ctx, "exit", nil)
		select {
		case <-serverDone:
		case <-time.After(2 * time.Second):
			t.Log("server did not shut down within 2s of exit notification")
		}
		cancel()
	}
	return ctx, cliConn, sink, cleanup
}

// fileURI writes content to a temp .ail file and returns its LSP file URI.
func fileURI(t *testing.T, content string) (protocol.DocumentURI, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture.ail")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	u := uri.File(path)
	return protocol.DocumentURI(u), path
}

const wellTypedAilang = `module fixture

import std/io (println)

export func main() -> () ! {IO} {
  println("hello from M2 fixture")
}
`

const typeErrorAilang = `module fixture

-- Return-type annotation says int, body produces string. Verified to
-- trigger "type unification failed" via ailang check.
export pure func bad() -> int = "this string is not an int"
`

// TestDidOpenWellTyped: opening a well-typed .ail file produces a
// publishDiagnostics with an empty array (no errors).
func TestDidOpenWellTyped(t *testing.T) {
	t.Parallel()
	_, cli, sink, cleanup := startTestServer(t)
	defer cleanup()

	docURI, _ := fileURI(t, wellTypedAilang)
	if err := cli.Notify(context.Background(), "textDocument/didOpen", &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        docURI,
			LanguageID: "ailang",
			Version:    1,
			Text:       wellTypedAilang,
		},
	}); err != nil {
		t.Fatalf("didOpen: %v", err)
	}

	params, ok := sink.wait(docURI, diagWaitTimeout())
	if !ok {
		t.Fatal("no publishDiagnostics arrived for well-typed file within 5s")
	}
	if len(params.Diagnostics) != 0 {
		t.Errorf("expected empty diagnostics for well-typed file, got %d:\n%+v",
			len(params.Diagnostics), params.Diagnostics)
	}
}

// TestDidOpenTypeError: opening a file with a known type error produces a
// publishDiagnostics carrying at least one Error-severity diagnostic with
// source "ailang".
func TestDidOpenTypeError(t *testing.T) {
	t.Parallel()
	_, cli, sink, cleanup := startTestServer(t)
	defer cleanup()

	docURI, _ := fileURI(t, typeErrorAilang)
	if err := cli.Notify(context.Background(), "textDocument/didOpen", &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        docURI,
			LanguageID: "ailang",
			Version:    1,
			Text:       typeErrorAilang,
		},
	}); err != nil {
		t.Fatalf("didOpen: %v", err)
	}

	params, ok := sink.wait(docURI, diagWaitTimeout())
	if !ok {
		t.Fatal("no publishDiagnostics arrived for type-error file within 5s")
	}
	if len(params.Diagnostics) == 0 {
		t.Fatal("expected at least one diagnostic for type-error fixture, got 0")
	}
	d := params.Diagnostics[0]
	if d.Severity != protocol.DiagnosticSeverityError {
		t.Errorf("first diagnostic severity: got %v, want Error", d.Severity)
	}
	if d.Source != DiagnosticSource {
		t.Errorf("first diagnostic source: got %q, want %q", d.Source, DiagnosticSource)
	}
	if d.Message == "" {
		t.Error("first diagnostic message is empty")
	}
}

// TestDidSaveRepublishes: editing in-memory then saving re-runs the pipeline
// against the saved content, producing fresh diagnostics that supersede the
// open-time set.
func TestDidSaveRepublishes(t *testing.T) {
	t.Parallel()
	_, cli, sink, cleanup := startTestServer(t)
	defer cleanup()

	docURI, path := fileURI(t, wellTypedAilang)
	ctx := context.Background()

	// Open well-typed: should publish empty.
	if err := cli.Notify(ctx, "textDocument/didOpen", &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: docURI, LanguageID: "ailang", Version: 1, Text: wellTypedAilang},
	}); err != nil {
		t.Fatalf("didOpen: %v", err)
	}
	first, ok := sink.wait(docURI, diagWaitTimeout())
	if !ok {
		t.Fatal("no diagnostics on didOpen")
	}
	if len(first.Diagnostics) != 0 {
		t.Fatalf("baseline expected 0 diagnostics, got %d", len(first.Diagnostics))
	}

	// Simulate the user editing then saving: rewrite the file on disk to
	// the type-error text BEFORE sending didSave (the LSP `didSave`
	// notification is sent AFTER the editor has flushed to disk).
	if err := os.WriteFile(path, []byte(typeErrorAilang), 0o644); err != nil {
		t.Fatalf("rewrite fixture: %v", err)
	}
	if err := cli.Notify(ctx, "textDocument/didSave", &protocol.DidSaveTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		Text:         typeErrorAilang,
	}); err != nil {
		t.Fatalf("didSave: %v", err)
	}

	// Wait for the *new* publish — clear the old one first by deleting from sink.
	sink.mu.Lock()
	delete(sink.byURI, docURI)
	sink.mu.Unlock()

	second, ok := sink.wait(docURI, diagWaitTimeout())
	if !ok {
		t.Fatal("no diagnostics arrived after didSave")
	}
	if len(second.Diagnostics) == 0 {
		t.Fatal("didSave with type-error text expected ≥1 diagnostic, got 0")
	}
}
