package observatory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// Bounded, LOUD, fail-soft iteration spool (M-MISSION-COST-CHAINS, M2).
//
// When PostIteration cannot reach the observatory store, the mission iteration MUST
// NOT block or fail on telemetry. The post is instead appended to a JSONL spool that
// the next iteration flushes. The spool is strictly bounded and loud:
//
//   - hard cap on entry count (default 100) AND on file size (default 1 MiB);
//   - EVERY buffering event emits an explicit stderr warning (never silent);
//   - on overflow it drops the OLDEST entries with a one-line stderr notice;
//   - it never grows without bound.
//
// The quorum's round-1 objection was precisely an "unbounded silent fallback" — this
// component is the bounded+loud answer, and it is unit-testable in Go.

const (
	// DefaultSpoolMaxEntries caps the number of buffered iteration posts.
	DefaultSpoolMaxEntries = 100
	// DefaultSpoolMaxBytes caps the spool file size (1 MiB).
	DefaultSpoolMaxBytes = 1 << 20
)

// Spool is a bounded JSONL buffer of IterationPosts.
type Spool struct {
	Path       string
	MaxEntries int
	MaxBytes   int64
	// warn is where loud notices go (defaults to os.Stderr; overridable for tests).
	warn io.Writer
}

// NewSpool constructs a spool at path with the default caps.
func NewSpool(path string) *Spool {
	return &Spool{
		Path:       path,
		MaxEntries: DefaultSpoolMaxEntries,
		MaxBytes:   DefaultSpoolMaxBytes,
		warn:       os.Stderr,
	}
}

// SetWarnWriter overrides the loud-notice sink (tests capture it).
func (s *Spool) SetWarnWriter(w io.Writer) { s.warn = w }

func (s *Spool) warnf(format string, args ...interface{}) {
	w := s.warn
	if w == nil {
		w = os.Stderr
	}
	fmt.Fprintf(w, "chains-spool: "+format+"\n", args...)
}

// Append buffers one post, enforcing the entry- and size-caps by dropping the
// oldest entries first. It ALWAYS emits a stderr warning (the fallback is loud), and
// returns an error only if the spool file itself cannot be written — the caller
// treats even that as non-fatal (fail-soft).
func (s *Spool) Append(post *IterationPost) error {
	if post.SpooledAt.IsZero() {
		post.SpooledAt = time.Now().UTC()
	}

	s.warnf("BUFFERING iteration %q — observatory unreachable; will flush next iteration", post.Source)

	entries, _ := s.readAll() // best-effort; a corrupt/absent file starts fresh
	entries = append(entries, post)

	// Enforce entry cap (drop-oldest).
	if s.MaxEntries > 0 && len(entries) > s.MaxEntries {
		dropped := len(entries) - s.MaxEntries
		entries = entries[dropped:]
		s.warnf("OVERFLOW: entry cap %d exceeded — dropped %d oldest post(s)", s.MaxEntries, dropped)
	}

	// Enforce size cap (drop-oldest until under the limit).
	data := marshalEntries(entries)
	for s.MaxBytes > 0 && int64(len(data)) > s.MaxBytes && len(entries) > 1 {
		entries = entries[1:]
		s.warnf("OVERFLOW: size cap %d bytes exceeded — dropped 1 oldest post", s.MaxBytes)
		data = marshalEntries(entries)
	}

	if err := os.WriteFile(s.Path, data, 0o644); err != nil {
		s.warnf("FAILED to persist spool (%v) — this iteration's telemetry is lost, but the loop continues", err)
		return err
	}
	return nil
}

// Drain reads and CLEARS the spool, returning the buffered posts (oldest first) so
// the caller can retry posting them. The file is removed only after a successful
// read; on read error it returns the error and leaves the file intact.
func (s *Spool) Drain() ([]*IterationPost, error) {
	entries, err := s.readAll()
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}
	if err := os.Remove(s.Path); err != nil && !os.IsNotExist(err) {
		return entries, fmt.Errorf("spool drained but not cleared: %w", err)
	}
	s.warnf("FLUSHING %d buffered iteration post(s)", len(entries))
	return entries, nil
}

// Len returns the number of buffered entries (0 if the spool is absent).
func (s *Spool) Len() int {
	entries, _ := s.readAll()
	return len(entries)
}

// readAll parses the JSONL spool. Missing file => empty (no error). Malformed lines
// are skipped with a loud notice rather than failing the whole drain.
func (s *Spool) readAll() ([]*IterationPost, error) {
	f, err := os.Open(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []*IterationPost
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var p IterationPost
		if err := json.Unmarshal(line, &p); err != nil {
			s.warnf("skipping malformed spool line: %v", err)
			continue
		}
		entries = append(entries, &p)
	}
	return entries, sc.Err()
}

// marshalEntries renders posts as JSONL (one compact object per line).
func marshalEntries(entries []*IterationPost) []byte {
	var buf []byte
	for _, e := range entries {
		b, err := json.Marshal(e)
		if err != nil {
			continue
		}
		buf = append(buf, b...)
		buf = append(buf, '\n')
	}
	return buf
}
