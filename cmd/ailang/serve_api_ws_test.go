package main

import (
	"flag"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
)

func applyWSFlags(t *testing.T, withStream bool, args ...string) (*effects.EffContext, error) {
	t.Helper()
	fs := flag.NewFlagSet("serve-api", flag.ContinueOnError)
	f := registerServeAPIWSFlags(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	ctx := effects.NewEffContext(nil)
	if withStream {
		ctx.Stream = effects.NewStreamContext()
	}
	return ctx, f.apply(ctx)
}

func TestServeAPIWSFlags_StreamMaxMessage(t *testing.T) {
	ctx, err := applyWSFlags(t, true, "--stream-max-message", "4MB")
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Stream.MaxMessageSize != 4<<20 {
		t.Errorf("MaxMessageSize = %d, want 4MB", ctx.Stream.MaxMessageSize)
	}
	if ctx, _ = applyWSFlags(t, true); ctx.Stream.MaxMessageSize != 1<<20 {
		t.Errorf("default MaxMessageSize = %d, want 1MB", ctx.Stream.MaxMessageSize)
	}
	if _, err := applyWSFlags(t, true, "--stream-max-message", "0"); err == nil {
		t.Error("--stream-max-message 0 must be refused")
	}
	_, err = applyWSFlags(t, false, "--stream-max-message", "4MB")
	if err == nil || !strings.Contains(err.Error(), "need --caps Stream") {
		t.Errorf("without --caps Stream: err = %v", err)
	}
}
