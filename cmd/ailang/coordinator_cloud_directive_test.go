package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
)

type stubDirectives struct {
	text string
	err  error
}

func (s stubDirectives) GetDirective(context.Context, string) (string, error) { return s.text, s.err }

func openStub(s stubDirectives) func(context.Context) (directiveReader, error) {
	return func(context.Context) (directiveReader, error) { return s, nil }
}

func TestResolveJobDirective(t *testing.T) {
	ctx := context.Background()
	big := strings.Repeat("x", 43*1024)

	// Source named: the stored directive, in full, even with an inline copy set.
	t.Setenv(config.EnvDirective, "stale inline copy")
	got, err := resolveJobDirective(ctx, config.DirectiveSourceFirestore, "task-1", openStub(stubDirectives{text: big}))
	if err != nil || got != big {
		t.Fatalf("firestore source: got %d bytes, err %v; want the stored %d bytes", len(got), err, len(big))
	}

	// Source named but the document is missing: an error, NOT the inline copy.
	_, err = resolveJobDirective(ctx, config.DirectiveSourceFirestore, "task-1", openStub(stubDirectives{err: errors.New("no directive stored")}))
	if err == nil {
		t.Fatal("a missing stored directive must fail the job, not fall back to the inline copy")
	}

	// The store cannot be opened: an error.
	_, err = resolveJobDirective(ctx, config.DirectiveSourceFirestore, "task-1", func(context.Context) (directiveReader, error) {
		return nil, errors.New("no credentials")
	})
	if err == nil {
		t.Fatal("an unopenable store must fail the job")
	}

	// Source unset (a coordinator older than the hand-off): the inline value.
	got, err = resolveJobDirective(ctx, "", "task-1", nil)
	if err != nil || got != "stale inline copy" {
		t.Fatalf("unset source: got %q, err %v; want the inline value", got, err)
	}

	// An unknown source is a contract break, not a guess.
	if _, err := resolveJobDirective(ctx, "gcs", "task-1", nil); err == nil {
		t.Fatal("unknown source must be an error")
	}
}
