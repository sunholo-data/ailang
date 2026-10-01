package otelplatform

import (
	"context"
	"errors"
	"testing"
	"time"

	cloudtrace "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// A failing Cloud Trace constructor must produce an ERROR, not a segfault.
//
// cloudtrace.New is declared (*Exporter, error) and returns `nil, err` on every
// failure path. Assigning that straight into the sdktrace.SpanExporter interface
// yields a non-nil interface holding a nil *Exporter, so `exporter != nil`
// passes and the dispose path calls (*Exporter).Shutdown, which dereferences
// e.traceExporter and kills the process.
//
// Measured 2026-09-30: with no Application Default Credentials reachable — a
// test's temp HOME, or simply an offline box — `ailang repl --help` exited 2
// with a SIGSEGV instead of printing help, and three cmd/ailang tests failed on
// it. This test reproduces the exact shape: the typed nil the real dependency
// returns, paired with the error it returns alongside it.
func TestCloudExporter_TypedNilFromConstructorErrorsInsteadOfPanicking(t *testing.T) {
	for _, tc := range []struct {
		name string
		give func(string) (sdktrace.SpanExporter, error)
	}{
		{
			// Exactly what cloudtrace.New does on a credential failure.
			name: "typed nil *cloudtrace.Exporter with an error",
			give: func(string) (sdktrace.SpanExporter, error) {
				var typedNil *cloudtrace.Exporter
				return typedNil, errors.New("stackdriver: no project found with application default credentials")
			},
		},
		{
			// The same trap with no error to ride along on: the constructor
			// contract is violated and that must still be reported, not crash.
			name: "typed nil with no error",
			give: func(string) (sdktrace.SpanExporter, error) {
				var typedNil *cloudtrace.Exporter
				return typedNil, nil
			},
		},
		{
			name: "honest untyped nil with an error",
			give: func(string) (sdktrace.SpanExporter, error) {
				return nil, errors.New("boom")
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			exporter, err := cloudExporter(context.Background(), initConfig{
				cloudProject:     "some-project",
				cloudTimeout:     2 * time.Second,
				newCloudExporter: tc.give,
			})
			if err == nil {
				t.Fatal("want an error from a constructor that produced no usable exporter, got nil")
			}
			if !isNilExporter(exporter) {
				t.Errorf("want no exporter alongside the error, got %#v", exporter)
			}
		})
	}
}

// initTelemetry must survive the same failure and say so, rather than dying
// while disposing an exporter that was never built.
func TestInitTelemetry_SurvivesATypedNilCloudExporter(t *testing.T) {
	isolateExporters(t)
	_, _, err := initTelemetry(context.Background(), "typed-nil-guard", initConfig{
		cloudProject: "some-project",
		cloudTimeout: 2 * time.Second,
		newCloudExporter: func(string) (sdktrace.SpanExporter, error) {
			var typedNil *cloudtrace.Exporter
			return typedNil, errors.New("stackdriver: no credentials")
		},
	})
	if err == nil {
		t.Fatal("want initTelemetry to report the Cloud Trace failure, got nil")
	}
}

func TestIsNilExporter(t *testing.T) {
	var typedNil *cloudtrace.Exporter
	for _, tc := range []struct {
		name string
		give sdktrace.SpanExporter
		want bool
	}{
		{"untyped nil", nil, true},
		{"typed nil pointer", typedNil, true},
		{"a real exporter", &nonNilExporter{}, false},
	} {
		if got := isNilExporter(tc.give); got != tc.want {
			t.Errorf("%s: isNilExporter = %v, want %v", tc.name, got, tc.want)
		}
	}
}

type nonNilExporter struct{}

func (*nonNilExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error { return nil }
func (*nonNilExporter) Shutdown(context.Context) error                             { return nil }
