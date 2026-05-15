// Package lsp implements an AILANG Language Server Protocol server targeting
// AI coding agents (and humans). See design_docs/planned/v0_21_0/m-ailang-lsp-for-ai.md
// for scope and capabilities.
package lsp

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
)

// ServerName / ServerVersion identify this server in LSP InitializeResult.
const (
	ServerName    = "ailang-lsp"
	ServerVersion = "0.1.0-mvp"
)

// Server is the AILANG LSP server. It satisfies protocol.Server by embedding
// unimplementedServer and overriding the lifecycle methods (Initialize,
// Initialized, Shutdown, Exit). Subsequent milestones override more methods
// as capabilities ship.
type Server struct {
	unimplementedServer

	logger *zap.Logger
	client protocol.Client // captured in Run; used to push diagnostics

	// docs holds the most recent text per open URI. Populated by DidOpen
	// (full text) and updated by DidChange. Used by DidSave to drive the
	// pipeline with the in-memory buffer rather than re-reading the file.
	docs sync.Map // map[protocol.DocumentURI]string

	// Lifecycle state guarded by atomics so test clients can race against
	// shutdown without us needing a mutex on the hot path.
	initialized   atomic.Bool
	shuttingDown  atomic.Bool
	exitRequested atomic.Bool
	exitCh        chan struct{}
}

// NewServer returns a Server ready to be wired to a stream via Run.
func NewServer(logger *zap.Logger) *Server {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Server{
		logger: logger,
		exitCh: make(chan struct{}),
	}
}

// Run wires the server to an io.ReadWriteCloser (typically stdio or a socket)
// and blocks until the connection closes or the LSP `exit` notification is
// received. The returned error is nil on a clean shutdown sequence
// (shutdown + exit), and non-nil if exit arrived without a prior shutdown.
func (s *Server) Run(ctx context.Context, rwc io.ReadWriteCloser) error {
	stream := jsonrpc2.NewStream(rwc)
	_, conn, client := protocol.NewServer(ctx, s, stream, s.logger)
	s.client = client

	select {
	case <-conn.Done():
	case <-s.exitCh:
		_ = conn.Close()
	case <-ctx.Done():
		_ = conn.Close()
	}

	if s.exitRequested.Load() && !s.shuttingDown.Load() {
		return fmt.Errorf("lsp: exit received before shutdown")
	}
	return nil
}

// Initialize implements LSP `initialize`. Returns the capabilities the
// AILANG LSP advertises in this milestone (M1: only TextDocumentSync; later
// milestones flip the rest as they ship).
func (s *Server) Initialize(_ context.Context, _ *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	if s.shuttingDown.Load() {
		return nil, jsonrpc2.NewError(jsonrpc2.InvalidRequest, "server is shutting down")
	}
	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			// Full text on every change, plus didOpen/didClose + didSave with text.
			// M2 only re-runs the pipeline on didSave; didChange just stores the buffer.
			TextDocumentSync: &protocol.TextDocumentSyncOptions{
				OpenClose: true,
				Change:    protocol.TextDocumentSyncKindFull,
				Save:      &protocol.SaveOptions{IncludeText: true},
			},
		},
		ServerInfo: &protocol.ServerInfo{
			Name:    ServerName,
			Version: ServerVersion,
		},
	}, nil
}

// Initialized implements LSP `initialized` (notification, no result).
func (s *Server) Initialized(_ context.Context, _ *protocol.InitializedParams) error {
	s.initialized.Store(true)
	s.logger.Info("ailang-lsp ready")
	return nil
}

// Shutdown implements LSP `shutdown`. After shutdown, only `exit` is allowed.
func (s *Server) Shutdown(_ context.Context) error {
	s.shuttingDown.Store(true)
	return nil
}

// Exit implements LSP `exit`. Triggers Run() to return.
func (s *Server) Exit(_ context.Context) error {
	s.exitRequested.Store(true)
	close(s.exitCh)
	return nil
}
