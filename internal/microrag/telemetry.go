package microrag

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/sunholo-data/ailang/internal/telemetry"
)

// tracer is the OTEL tracer for μRAG. When no TracerProvider is configured
// this is the no-op tracer (~2ns overhead per Start), so it is always safe to call.
var tracer = telemetry.Tracer("ailang.microrag")

// startContextSpan opens the span covering one Engine.Context() call and
// returns a finisher that records the result envelope as span attributes.
//
// Why span-per-call rather than per-engine counters: spans flow through the
// existing OTLP exporter (no new metric pipeline), and each call already has
// rich correlation state (file_path, namespace, dedup outcome).
func startContextSpan(ctx context.Context, req Request) (context.Context, func(*ContextResult)) {
	ctx, span := tracer.Start(ctx, "microrag.context",
		trace.WithAttributes(
			attribute.String("microrag.tool", req.ToolName),
			attribute.String("microrag.file_path", req.FilePath),
		),
	)
	return ctx, func(res *ContextResult) {
		defer span.End()
		if res == nil {
			return
		}
		span.SetAttributes(
			attribute.String("microrag.state", res.State),
			attribute.String("microrag.reason", res.Reason),
		)
		if res.Injection != nil {
			span.SetAttributes(
				attribute.String("microrag.namespace", res.Injection.Namespace),
				attribute.Int("microrag.tokens", res.Injection.Tokens),
				attribute.Float64("microrag.score", res.Injection.Score),
			)
		}
	}
}

// startUserPromptSpan opens the span covering one Engine.UserPrompt() call.
// Records prompt length (not the prompt itself — PII concern) and the
// outcome envelope as span attributes.
func startUserPromptSpan(ctx context.Context, req UserPromptRequest) (context.Context, func(*UserPromptResult)) {
	ctx, span := tracer.Start(ctx, "microrag.user_prompt",
		trace.WithAttributes(
			attribute.Int("microrag.prompt_len", len(req.Prompt)),
			attribute.Int("microrag.namespace_count", len(req.Namespaces)),
		),
	)
	return ctx, func(res *UserPromptResult) {
		defer span.End()
		if res == nil {
			return
		}
		span.SetAttributes(
			attribute.String("microrag.state", res.State),
			attribute.String("microrag.reason", res.Reason),
		)
		if res.Injection != nil {
			span.SetAttributes(
				attribute.String("microrag.namespace", res.Injection.Namespace),
				attribute.Int("microrag.tokens", res.Injection.Tokens),
				attribute.Float64("microrag.score", res.Injection.Score),
			)
		}
	}
}

// startLintSpan opens the span around one LintBuiltins() call.
func startLintSpan(ctx context.Context, sourceLen int) (context.Context, func(*LintResult)) {
	ctx, span := tracer.Start(ctx, "microrag.lint",
		trace.WithAttributes(
			attribute.Int("microrag.source_bytes", sourceLen),
		),
	)
	return ctx, func(res *LintResult) {
		defer span.End()
		if res == nil {
			return
		}
		span.SetAttributes(
			attribute.Int("microrag.nudge_count", len(res.Nudges)),
		)
	}
}
