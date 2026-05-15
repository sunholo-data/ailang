package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/sunholo-data/ailang/internal/lsp"
	"go.uber.org/zap"
)

// lspCommand starts the AILANG LSP server. The default transport is stdio
// (the Language Server Protocol's standard for editor/agent integration).
func lspCommand(args []string) error {
	fs := flag.NewFlagSet("lsp", flag.ExitOnError)
	stdio := fs.Bool("stdio", true, "Communicate over stdin/stdout (LSP standard)")
	verbose := fs.Bool("verbose", false, "Log to stderr (off by default — many LSP clients capture stderr too)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if !*stdio {
		return fmt.Errorf("only --stdio transport is supported in this milestone")
	}

	var logger *zap.Logger
	if *verbose {
		var err error
		logger, err = zap.NewDevelopment()
		if err != nil {
			return fmt.Errorf("init logger: %w", err)
		}
	} else {
		logger = zap.NewNop()
	}
	defer func() { _ = logger.Sync() }()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	srv := lsp.NewServer(logger)
	rwc := stdioReadWriteCloser{in: os.Stdin, out: os.Stdout}
	return srv.Run(ctx, rwc)
}

// stdioReadWriteCloser bundles stdin and stdout into a single
// io.ReadWriteCloser so the LSP server can treat the agent the same way it
// treats any other transport.
type stdioReadWriteCloser struct {
	in  io.Reader
	out io.Writer
}

func (s stdioReadWriteCloser) Read(p []byte) (int, error)  { return s.in.Read(p) }
func (s stdioReadWriteCloser) Write(p []byte) (int, error) { return s.out.Write(p) }
func (s stdioReadWriteCloser) Close() error                { return nil }
