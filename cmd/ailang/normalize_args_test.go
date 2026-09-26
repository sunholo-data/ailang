package main

import (
	"flag"
	"io"
	"strings"
	"testing"
)

// M-COORDINATOR-EXECUTION-TRUST M5 (design doc V29).
//
// normalizeArgsForFlags refused to consume a flag value beginning with "-", so
// the value fell through into the positional list and every later token shifted.
// Measured in prod-shaped use on 2026-09-02:
//
//	ailang messages send diag-argparse "body" --title "--help is inconsistent" --from "diag-sender"
//
// delivered the message to inbox `diag-sender`, set from_agent to the `cli`
// default, set the title to the literal string "--from" — and printed
// "✓ Message sent". A misrouted message that reports success is the same defect
// class the whole design doc is about, one layer earlier.

func sendLikeFlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("send", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.String("title", "", "")
	fs.String("from", "", "")
	fs.String("type", "", "")
	fs.Bool("force", false, "")
	fs.Bool("github", false, "")
	return fs
}

// The exact reported repro.
func TestDashLeadingFlagValueIsNotAShift(t *testing.T) {
	fs := sendLikeFlagSet()
	args := []string{"diag-argparse", "body", "--title", "--help is inconsistent", "--from", "diag-sender"}

	out, err := normalizeArgsForFlags(args, fs)
	if err != nil {
		t.Fatalf("normalizeArgsForFlags: %v", err)
	}
	if err := fs.Parse(out); err != nil {
		t.Fatalf("parse: %v", err)
	}

	if got := fs.Lookup("title").Value.String(); got != "--help is inconsistent" {
		t.Errorf("title = %q, want the dash-leading value intact", got)
	}
	if got := fs.Lookup("from").Value.String(); got != "diag-sender" {
		t.Errorf("from = %q, want %q", got, "diag-sender")
	}
	// The inbox is the FIRST positional and must still be the addressed one.
	if got := fs.Arg(0); got != "diag-argparse" {
		t.Fatalf("inbox = %q — the message would be delivered to the wrong inbox", got)
	}
	if got := fs.Arg(1); got != "body" {
		t.Errorf("body = %q, want %q", got, "body")
	}
}

// A boolean flag must NOT swallow the next positional. Latent in the same
// function: --force is a Bool, so consuming the token after it ate the inbox
// whenever a bool flag was written before the positionals.
func TestBoolFlagDoesNotSwallowAPositional(t *testing.T) {
	fs := sendLikeFlagSet()
	out, err := normalizeArgsForFlags([]string{"--force", "my-inbox", "body"}, fs)
	if err != nil {
		t.Fatalf("normalizeArgsForFlags: %v", err)
	}
	if err := fs.Parse(out); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if fs.Lookup("force").Value.String() != "true" {
		t.Errorf("force should be set")
	}
	if got := fs.Arg(0); got != "my-inbox" {
		t.Fatalf("inbox = %q — a bool flag swallowed the inbox", got)
	}
}

// A flag that needs a value and has none must ERROR. Silently shifting is how
// the routing outcome became a side effect of a half-failed parse.
func TestMissingFlagValueIsAnErrorNotAShift(t *testing.T) {
	fs := sendLikeFlagSet()
	_, err := normalizeArgsForFlags([]string{"inbox", "body", "--title"}, fs)
	if err == nil {
		t.Fatal("a value flag with no value must be an error")
	}
	if !strings.Contains(err.Error(), "title") {
		t.Errorf("the error must name the offending flag, got %q", err)
	}
}

func TestEqualsFormAndUnknownFlagsSurvive(t *testing.T) {
	fs := sendLikeFlagSet()
	out, err := normalizeArgsForFlags([]string{"inbox", "--title=--dashy", "body"}, fs)
	if err != nil {
		t.Fatalf("normalizeArgsForFlags: %v", err)
	}
	if err := fs.Parse(out); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := fs.Lookup("title").Value.String(); got != "--dashy" {
		t.Errorf("title = %q, want %q", got, "--dashy")
	}
	if got := fs.Arg(0); got != "inbox" {
		t.Errorf("inbox = %q", got)
	}
}

// The ordinary case must stay byte-identical — this is what every existing
// invocation relies on.
func TestOrdinaryTrailingFlagsStillWork(t *testing.T) {
	fs := sendLikeFlagSet()
	out, err := normalizeArgsForFlags([]string{"inbox", "body", "--title", "hello", "--from", "me"}, fs)
	if err != nil {
		t.Fatalf("normalizeArgsForFlags: %v", err)
	}
	if err := fs.Parse(out); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if fs.Lookup("title").Value.String() != "hello" || fs.Lookup("from").Value.String() != "me" {
		t.Errorf("flags not parsed: title=%q from=%q", fs.Lookup("title").Value, fs.Lookup("from").Value)
	}
	if fs.Arg(0) != "inbox" || fs.Arg(1) != "body" {
		t.Errorf("positionals shifted: %v", fs.Args())
	}
}
