package ai

import (
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// RecordSpanError annotates a provider span with one failure in the one
// telemetry shape: error.code / error.message / error.retryable from the
// classified error, error.category for trace filtering ("timeout" or
// "api_error", the two categories a provider call can be), http.status_code
// when a response was received, and the span status set to the code.
// Replaces the per-client recordStepError/recordSpanError copies and their
// asAIError unwrappers (M-V1-SIMPLIFY-S3 M4). Nil is a no-op.
//
// internal/telemetry cannot be imported here (telemetry → effects → ai), so
// the 200-rune truncation is local.
func RecordSpanError(span trace.Span, err error) {
	if err == nil || span == nil {
		return
	}
	e := ClassifyError(err)
	category := "api_error"
	if e.Code == CodeTimeout {
		category = "timeout"
	}
	attrs := []attribute.KeyValue{
		attribute.String("error.code", e.Code),
		attribute.String("error.message", truncateRunes(e.Message, 200)),
		attribute.Bool("error.retryable", e.Retryable),
		attribute.String("error.category", category),
	}
	var perr *ProviderError
	if errors.As(err, &perr) && perr.StatusCode > 0 {
		attrs = append(attrs, attribute.Int("http.status_code", perr.StatusCode))
	}
	span.SetAttributes(attrs...)
	span.RecordError(err)
	span.SetStatus(codes.Error, e.Code)
}

// truncateRunes keeps the first max runes of s, appending "..." when cut.
func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-3]) + "..."
}
