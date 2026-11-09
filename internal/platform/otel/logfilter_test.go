package otelplatform

import (
	"bytes"
	"log"
	"strings"
	"sync"
	"testing"
)

func TestDropLines_FiltersOnlyTheCloudTraceNotice(t *testing.T) {
	var buf bytes.Buffer
	l := log.New(&dropLines{w: &buf, drop: [][]byte{cloudTraceDeprecation}}, "", 0)
	l.Println("Google Cloud OpenTelemetry Trace exporter for Go is deprecated and will be archived after January 1st, 2027.")
	l.Println("keep me")
	if got := buf.String(); got != "keep me\n" {
		t.Fatalf("filtered output = %q, want only the unrelated line", got)
	}
}

func TestSilenceCloudTraceDeprecation_WrapsGlobalLoggerOnce(t *testing.T) {
	orig := log.Writer()
	defer log.SetOutput(orig)
	var buf bytes.Buffer
	log.SetOutput(&buf)
	installLogFilterOnce = sync.Once{}
	silenceCloudTraceDeprecation()
	silenceCloudTraceDeprecation() // idempotent: must not double-wrap
	log.Println("Google Cloud OpenTelemetry Trace exporter for Go is deprecated and will be archived")
	log.Println("other")
	if strings.Contains(buf.String(), "deprecated") || !strings.Contains(buf.String(), "other") {
		t.Fatalf("global log output = %q", buf.String())
	}
	if d, ok := log.Writer().(*dropLines); !ok || d.w != &buf {
		t.Fatalf("global writer should be one dropLines around the original, got %T", log.Writer())
	}
}
