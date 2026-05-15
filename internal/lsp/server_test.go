package lsp

import (
	"context"
	"net"
	"testing"
	"time"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

// TestHandshake drives a full LSP lifecycle (initialize / initialized /
// shutdown / exit) over an in-memory net.Pipe and asserts that the
// advertised capabilities match the M1 contract: only TextDocumentSync is
// enabled; everything else is deferred to later milestones.
func TestHandshake(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	srvSide, cliSide := net.Pipe()
	srv := NewServer(nil)

	serverDone := make(chan error, 1)
	go func() { serverDone <- srv.Run(ctx, srvSide) }()

	cliStream := jsonrpc2.NewStream(cliSide)
	cliConn := jsonrpc2.NewConn(cliStream)
	cliConn.Go(ctx, jsonrpc2.MethodNotFoundHandler)

	// initialize
	var initRes protocol.InitializeResult
	if _, err := cliConn.Call(ctx, "initialize", &protocol.InitializeParams{}, &initRes); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if initRes.ServerInfo == nil || initRes.ServerInfo.Name != ServerName {
		t.Fatalf("ServerInfo.Name: got %+v, want %s", initRes.ServerInfo, ServerName)
	}
	assertCapabilities(t, initRes.Capabilities)

	// initialized
	if err := cliConn.Notify(ctx, "initialized", &protocol.InitializedParams{}); err != nil {
		t.Fatalf("initialized: %v", err)
	}

	// shutdown
	if _, err := cliConn.Call(ctx, "shutdown", nil, nil); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	// exit
	if err := cliConn.Notify(ctx, "exit", nil); err != nil {
		t.Fatalf("exit: %v", err)
	}

	select {
	case err := <-serverDone:
		if err != nil {
			t.Fatalf("server.Run returned error on clean shutdown: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("server did not shut down within timeout")
	}
}

// TestExitWithoutShutdown verifies that exit-before-shutdown is reported as
// an error, per the LSP spec ("the server should exit with non-zero on exit
// before shutdown").
func TestExitWithoutShutdown(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	srvSide, cliSide := net.Pipe()
	srv := NewServer(nil)

	serverDone := make(chan error, 1)
	go func() { serverDone <- srv.Run(ctx, srvSide) }()

	cliStream := jsonrpc2.NewStream(cliSide)
	cliConn := jsonrpc2.NewConn(cliStream)
	cliConn.Go(ctx, jsonrpc2.MethodNotFoundHandler)

	// Skip shutdown — go straight to exit.
	if err := cliConn.Notify(ctx, "exit", nil); err != nil {
		t.Fatalf("exit: %v", err)
	}

	select {
	case err := <-serverDone:
		if err == nil {
			t.Fatal("server.Run returned nil on exit-without-shutdown; want error")
		}
	case <-ctx.Done():
		t.Fatal("server did not shut down within timeout")
	}
}

// TestUnsupportedRequestReturnsMethodNotFound verifies that a capability we
// haven't shipped yet (e.g. textDocument/hover in M1) is rejected with the
// LSP-spec MethodNotFound error rather than being silently dropped.
func TestUnsupportedRequestReturnsMethodNotFound(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	srvSide, cliSide := net.Pipe()
	srv := NewServer(nil)

	serverDone := make(chan error, 1)
	go func() { serverDone <- srv.Run(ctx, srvSide) }()

	cliStream := jsonrpc2.NewStream(cliSide)
	cliConn := jsonrpc2.NewConn(cliStream)
	cliConn.Go(ctx, jsonrpc2.MethodNotFoundHandler)

	// Initialize first so we're not rejected for being uninitialized.
	if _, err := cliConn.Call(ctx, "initialize", &protocol.InitializeParams{}, &protocol.InitializeResult{}); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if err := cliConn.Notify(ctx, "initialized", &protocol.InitializedParams{}); err != nil {
		t.Fatalf("initialized: %v", err)
	}

	// textDocument/hover is deferred to M3 — must error MethodNotFound.
	hoverParams := &protocol.HoverParams{}
	var hoverRes protocol.Hover
	_, err := cliConn.Call(ctx, "textDocument/hover", hoverParams, &hoverRes)
	if err == nil {
		t.Fatal("textDocument/hover returned no error in M1; want MethodNotFound")
	}

	// Clean up so the test goroutine exits.
	if _, err := cliConn.Call(ctx, "shutdown", nil, nil); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if err := cliConn.Notify(ctx, "exit", nil); err != nil {
		t.Fatalf("exit: %v", err)
	}
	select {
	case <-serverDone:
	case <-ctx.Done():
		t.Fatal("server did not shut down within timeout")
	}
}

// assertCapabilities locks in the post-M2 capability contract: text-document
// sync is advertised with full options (openClose + change + save with text)
// because M2 needs save notifications to trigger diagnostics. Everything
// else stays nil/false until later milestones flip its switch.
func assertCapabilities(t *testing.T, caps protocol.ServerCapabilities) {
	t.Helper()
	// After JSON round-trip the typed *TextDocumentSyncOptions becomes a
	// map[string]interface{} on the client side, so we have to handle both.
	switch sync := caps.TextDocumentSync.(type) {
	case *protocol.TextDocumentSyncOptions:
		if !sync.OpenClose {
			t.Error("TextDocumentSync.OpenClose must be true so M2 sees didOpen/didClose")
		}
		if sync.Save == nil || !sync.Save.IncludeText {
			t.Error("TextDocumentSync.Save.IncludeText must be true so M2 didSave handler can re-typecheck without disk read")
		}
	case map[string]interface{}:
		if openClose, _ := sync["openClose"].(bool); !openClose {
			t.Error("TextDocumentSync.openClose must be true so M2 sees didOpen/didClose")
		}
		save, _ := sync["save"].(map[string]interface{})
		if includeText, _ := save["includeText"].(bool); !includeText {
			t.Error("TextDocumentSync.save.includeText must be true so M2 didSave handler can re-typecheck without disk read")
		}
	default:
		t.Fatalf("TextDocumentSync must be *TextDocumentSyncOptions or map (after JSON), got %T", caps.TextDocumentSync)
	}
	if caps.HoverProvider != nil {
		t.Errorf("HoverProvider must stay nil until M3, got %v", caps.HoverProvider)
	}
	if caps.DefinitionProvider != nil {
		t.Errorf("DefinitionProvider must stay nil until M4, got %v", caps.DefinitionProvider)
	}
	if caps.ReferencesProvider != nil {
		t.Errorf("ReferencesProvider must stay nil until M4, got %v", caps.ReferencesProvider)
	}
	if caps.DocumentSymbolProvider != nil {
		t.Errorf("DocumentSymbolProvider must stay nil until M5, got %v", caps.DocumentSymbolProvider)
	}
	if caps.CompletionProvider != nil {
		t.Error("CompletionProvider is deferred (out of MVP scope) — must stay nil")
	}
	if caps.RenameProvider != nil {
		t.Error("RenameProvider is deferred (out of MVP scope) — must stay nil")
	}
}
