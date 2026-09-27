package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/riggate"
	"github.com/sunholo-data/ailang/internal/riglock"
)

// rigGateCommand runs the rig GPU admission gateway (M-RIG-GPU-ADMISSION-GATEWAY):
// a loopback proxy in front of ollama that admits long GPU work only from the
// holder of the rig lock's lease. See internal/riggate and the design doc.
func rigGateCommand(args []string) error {
	fs := flag.NewFlagSet("rig-gate", flag.ExitOnError)
	listen := fs.String("listen", "127.0.0.1:11434", "Address clients use (ollama's usual address)")
	upstream := fs.String("upstream", "http://127.0.0.1:11435", "ollama behind the gate (private port)")
	ledger := fs.String("ledger", filepath.Join(config.RigSharedDir(), "rig-gate.jsonl"), "JSONL ledger, one line per request")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: ailang rig-gate [--listen ADDR] [--upstream URL] [--ledger PATH]")
		fmt.Fprintln(os.Stderr, "\nAdmits long GPU work (chat/generate) only with the live rig-lock lease token")
		fmt.Fprintln(os.Stderr, "(Authorization: Bearer or X-Rig-Lease) while the lock is held; refuses others")
		fmt.Fprintln(os.Stderr, "with 423 at once. Short calls (embed, tags) always pass.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	up, err := url.Parse(*upstream)
	if err != nil || up.Host == "" {
		return fmt.Errorf("rig-gate: bad --upstream %q", *upstream)
	}
	if sameAddr(*listen, up.Host) {
		return fmt.Errorf("rig-gate: --listen %s and --upstream %s are the same address; move ollama to its private port first (OLLAMA_HOST)", *listen, up.Host)
	}
	if !isLoopback(*listen) {
		return fmt.Errorf("rig-gate: --listen %s is not a loopback address; the rig GPU is not served off-box", *listen)
	}

	pol, err := riggate.LoadPolicy()
	if err != nil {
		return err // a gate that cannot evaluate its rule must not start
	}
	led, err := riggate.OpenLedger(*ledger)
	if err != nil {
		return err
	}
	defer func() { _ = led.Close() }()

	srv := &http.Server{
		Addr:              *listen,
		Handler:           riggate.NewGate(up, pol, riglock.CurrentLease, led),
		ReadHeaderTimeout: 10 * time.Second,
		// No write timeout: a long generation streams for many minutes.
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()
	fmt.Fprintf(os.Stderr, "rig-gate: %s -> %s, ledger %s\n", *listen, up.Host, *ledger)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("rig-gate: %w", err)
	}
	return nil
}

func sameAddr(a, b string) bool {
	ha, pa, err1 := net.SplitHostPort(a)
	hb, pb, err2 := net.SplitHostPort(b)
	if err1 != nil || err2 != nil {
		return a == b
	}
	norm := func(h string) string {
		if h == "localhost" || h == "" {
			return "127.0.0.1"
		}
		return h
	}
	return pa == pb && norm(ha) == norm(hb)
}

func isLoopback(addr string) bool {
	h, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if h == "localhost" {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}
