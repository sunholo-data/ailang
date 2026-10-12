// Package runlife is the stdlib-only private lifetime protocol for an owned
// invocation of the same runtime executable. It never transports PID authority.
package runlife

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// MetadataEnv names private inherited descriptor metadata, not a user setting.
const MetadataEnv = "AILANG_INTERNAL_RUN_LIFETIME"
const envKey = MetadataEnv
const MaxFrame = 1024
const ShutdownTimeout = 2 * time.Second

type frame struct {
	Version     int    `json:"version"`
	Kind        string `json:"kind"`
	RemainingNS int64  `json:"remaining_ns,omitempty"`
	Error       string `json:"error,omitempty"`
}

func readFrame(r io.Reader) (frame, error) {
	var f frame
	// A new bounded reader is safe: each protocol direction has one frame at a
	// time except ready+receipt, whose persistent reader is passed by Child.
	b := bufio.NewReaderSize(r, MaxFrame+1)
	if existing, ok := r.(*bufio.Reader); ok {
		b = existing
	}
	line, err := b.ReadSlice('\n')
	if err != nil {
		return f, fmt.Errorf("lifetime frame: %w", err)
	}
	if len(line) > MaxFrame {
		return f, fmt.Errorf("lifetime frame exceeds %d bytes", MaxFrame)
	}
	d := json.NewDecoder(bufio.NewReaderSize(bytes.NewReader(line), MaxFrame+1))
	d.DisallowUnknownFields()
	if err = d.Decode(&f); err != nil {
		return f, fmt.Errorf("lifetime frame: %w", err)
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return f, fmt.Errorf("trailing lifetime frame data")
	}
	if f.Version != 1 || (f.Kind != "ready" && f.Kind != "stop" && f.Kind != "receipt") {
		return f, fmt.Errorf("unsupported lifetime frame")
	}
	if f.RemainingNS < 0 || f.RemainingNS > int64(ShutdownTimeout) {
		return f, fmt.Errorf("invalid lifetime duration")
	}
	if f.Kind != "stop" && f.RemainingNS != 0 || f.Kind != "receipt" && f.Error != "" {
		return f, fmt.Errorf("invalid lifetime state fields")
	}
	return f, nil
}

func writeFrame(w io.Writer, f frame) error {
	f.Version = 1
	// Diagnostic details are intentionally bounded and contain no credentials.
	if len(f.Error) > 512 {
		f.Error = f.Error[:512]
	}
	b, err := json.Marshal(f)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if len(b) > MaxFrame {
		return fmt.Errorf("lifetime frame exceeds limit")
	}
	_, err = w.Write(b)
	return err
}
