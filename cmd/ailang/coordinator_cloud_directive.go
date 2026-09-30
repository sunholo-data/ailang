package main

import (
	"context"
	"fmt"

	"github.com/sunholo-data/ailang/internal/config"
	fsstore "github.com/sunholo-data/ailang/internal/storage/firestore"
)

// directiveReader is the read half of fsstore.TaskDirectiveStore — the seam
// resolveJobDirective is tested through.
type directiveReader interface {
	GetDirective(ctx context.Context, taskID string) (string, error)
}

// resolveJobDirective returns the directive this job must run.
//
// The dispatcher stores it in Firestore (task_directives/<task id>) and says so
// with AILANG_DIRECTIVE_SOURCE=firestore; Cloud Run's 32,768-byte env cap made
// the old inline AILANG_DIRECTIVE undeliverable for a 43 KB request. Once the
// source is named, a missing or unreadable document fails the job — it never
// falls back to the inline copy, which the dispatcher drops above 16 KB, so a
// fallback would run a large request as an empty one.
//
// With the source unset, the coordinator predates the hand-off and the inline
// value is the only copy there is.
func resolveJobDirective(ctx context.Context, source, taskID string, open func(context.Context) (directiveReader, error)) (string, error) {
	switch source {
	case "":
		return config.Directive(), nil
	case config.DirectiveSourceFirestore:
		r, err := open(ctx)
		if err != nil {
			return "", fmt.Errorf("directive: %w", err)
		}
		return r.GetDirective(ctx, taskID)
	default:
		return "", fmt.Errorf("directive: unknown %s %q", config.EnvDirectiveSource, source)
	}
}

// openDirectiveStore opens the task_directives store in projectID.
func openDirectiveStore(projectID string) func(context.Context) (directiveReader, error) {
	return func(ctx context.Context) (directiveReader, error) {
		c, err := fsstore.NewClientForProject(ctx, projectID)
		if err != nil {
			return nil, err
		}
		return fsstore.NewTaskDirectiveStore(c), nil
	}
}
