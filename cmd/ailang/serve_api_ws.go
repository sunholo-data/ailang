package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/sunholo-data/ailang/internal/apiserver"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/platform/streamcred"
	"github.com/sunholo-data/ailang/internal/runner"
)

// serve-api WebSocket and Stream session flags (M-SERVEAPI-WS-BRIDGE).
type serveAPIWSFlags struct {
	maxSessions   *int
	queueFrames   *int
	decisionLog   *bool
	maxDuration   *time.Duration
	idleTimeout   *time.Duration
	maxMessage    *string
	credentialRaw multiFlag
	passHeaders   multiFlag
}

func registerServeAPIWSFlags(fs *flag.FlagSet) *serveAPIWSFlags {
	f := &serveAPIWSFlags{
		maxSessions: fs.Int("ws-max-sessions", apiserver.DefaultWSMaxSessions, "Concurrent @route(\"WS\") sessions server-wide; the next one gets 503 before upgrade"),
		queueFrames: fs.Int("ws-queue-frames", apiserver.DefaultWSQueueFrames, "Inbound queue per WebSocket connection, in frames (backpressure blocks when full)"),
		decisionLog: fs.Bool("ws-decision-log", false, "Log one payload-free JSON line per bridged frame (call_id, seq, dir, kind, bytes, verdict, step_us)"),
		maxDuration: fs.Duration("stream-max-duration", 0, "Hard ceiling per Stream connection / bridge (default 5m)"),
		idleTimeout: fs.Duration("stream-idle-timeout", 0, "Idle timeout per Stream connection / bridge (default 60s)"),
		maxMessage:  fs.String("stream-max-message", "", "Cap on one Stream message, both directions and both legs of a bridge (e.g. 256KB, 4MB; default 1MB)"),
	}
	fs.Var(&f.passHeaders, "ws-pass-header", "Pass this request header to @route(\"WS\") handlers as req.headers (repeatable; lower-cased; credential headers are refused)")
	fs.Var(&f.credentialRaw, "stream-credential", "Bind a credential to one wss upstream host: HOST[:PORT]=gcp-key-file:PATH | gcp-metadata | bearer-file:PATH (repeatable)")
	return f
}

// config builds the WS session config; --ws-pass-header is validated against
// the credential headers, including the configured --api-key-header.
func (f *serveAPIWSFlags) config(apiKeyHeader string) (apiserver.WSConfig, error) {
	pass, err := apiserver.ValidatePassHeaders(f.passHeaders, apiKeyHeader)
	if err != nil {
		return apiserver.WSConfig{}, err
	}
	return apiserver.WSConfig{MaxSessions: *f.maxSessions, QueueFrames: *f.queueFrames, DecisionLog: *f.decisionLog, PassHeaders: pass}, nil
}

// apply sets the Stream session limits and the credential binding on the
// server's Stream context. Every flag here needs --caps Stream; setting one
// without it is an error, not a silently ignored flag.
func (f *serveAPIWSFlags) apply(effCtx *effects.EffContext) error {
	if *f.maxSessions < 1 || *f.queueFrames < 1 {
		return fmt.Errorf("--ws-max-sessions and --ws-queue-frames must be at least 1")
	}
	streamFlagSet := *f.maxDuration != 0 || *f.idleTimeout != 0 || *f.maxMessage != "" || len(f.credentialRaw) > 0
	if effCtx.Stream == nil {
		if streamFlagSet {
			return fmt.Errorf("--stream-max-duration, --stream-idle-timeout, --stream-max-message and --stream-credential need --caps Stream")
		}
		return nil
	}
	if *f.maxDuration < 0 || *f.idleTimeout < 0 {
		return fmt.Errorf("--stream-max-duration and --stream-idle-timeout must be positive")
	}
	if *f.maxDuration > 0 {
		effCtx.Stream.MaxDuration = *f.maxDuration
	}
	if *f.idleTimeout > 0 {
		effCtx.Stream.IdleTimeout = *f.idleTimeout
	}
	if *f.maxMessage != "" {
		n, err := runner.ParseStreamMaxMessage(*f.maxMessage)
		if err != nil {
			return err
		}
		effCtx.Stream.MaxMessageSize = n
	}
	if len(f.credentialRaw) == 0 {
		return nil
	}
	bindings := make([]streamcred.Binding, 0, len(f.credentialRaw))
	for _, spec := range f.credentialRaw {
		b, err := streamcred.Parse(spec)
		if err != nil {
			return err
		}
		bindings = append(bindings, b)
	}
	binder, err := streamcred.Binder(bindings)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := streamcred.Prefetch(ctx, bindings); err != nil {
		return err
	}
	effCtx.Stream.Credentials = binder
	for _, b := range bindings {
		log.Printf("Stream credential bound: wss://%s -> %s", b.Host, b.Source.Describe())
	}
	return nil
}

func printServeAPIWSHelp() {
	fmt.Println("  --ws-max-sessions N  Concurrent @route(\"WS\") sessions (default 4); more get 503")
	fmt.Println("  --ws-queue-frames N  Inbound queue per WebSocket connection, frames (default 64)")
	fmt.Println("  --ws-decision-log    Log one payload-free line per bridged frame")
	fmt.Println("  --ws-pass-header H   Give WS handlers this request header in req.headers (repeatable);")
	fmt.Println("                       Authorization, Cookie and the API-key header are refused")
	fmt.Println("  --stream-max-duration D  Ceiling per Stream connection / bridge (default 5m)")
	fmt.Println("  --stream-idle-timeout D  Idle timeout per Stream connection / bridge (default 60s)")
	fmt.Println("  --stream-max-message N   Cap on one Stream message, both directions (default 1MB).")
	fmt.Println("                       Lower it for public endpoints: worst-case memory is about")
	fmt.Println("                       ws-max-sessions x ws-queue-frames x N")
	fmt.Println("  --stream-credential HOST=SOURCE  Bind a credential to one wss upstream host;")
	fmt.Println("                       SOURCE: gcp-key-file:PATH | gcp-metadata | bearer-file:PATH (repeatable).")
	fmt.Println("                       The program never sees it; a program Authorization header to that host is refused")
}
