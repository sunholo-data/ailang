package otelplatform

import (
	"bytes"
	"io"
	"log"
	"sync"
)

// cloudTraceDeprecation is the line the Cloud Trace exporter library prints, through the
// global log package, the first time its constructor runs (exporter/trace v1.37.0,
// cloudtrace.go logDeprecated). It reached stderr on EVERY ailang command run with
// GOOGLE_CLOUD_PROJECT set — and the eval harness sets it for agent children — so the
// first line of every tool result in every agent eval was this notice (#2151). The
// migration to OTLP is tracked separately; this only keeps the notice out of user output.
var cloudTraceDeprecation = []byte("Google Cloud OpenTelemetry Trace exporter for Go is deprecated")

var installLogFilterOnce sync.Once

// dropLines forwards every write except a log line containing one of drop.
// The log package hands the writer exactly one formatted line per call.
type dropLines struct {
	w    io.Writer
	drop [][]byte
}

func (d *dropLines) Write(p []byte) (int, error) {
	for _, s := range d.drop {
		if bytes.Contains(p, s) {
			return len(p), nil
		}
	}
	return d.w.Write(p)
}

// silenceCloudTraceDeprecation wraps the global logger's current output once.
func silenceCloudTraceDeprecation() {
	installLogFilterOnce.Do(func() {
		log.SetOutput(&dropLines{w: log.Writer(), drop: [][]byte{cloudTraceDeprecation}})
	})
}
