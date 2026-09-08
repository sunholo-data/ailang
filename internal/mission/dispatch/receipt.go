package dispatch

import (
	"encoding/json"
	"fmt"
	"github.com/sunholo-data/ailang/internal/fsyncdir"
	"os"
	"path/filepath"
	"time"
)

type Event struct {
	Version       int        `json:"version"`
	Time          time.Time  `json:"time"`
	Kind          string     `json:"kind"`
	Plan          *Plan      `json:"plan,omitempty"`
	Request       *Request   `json:"request,omitempty"`
	RequestDigest string     `json:"request_digest"`
	Attempt       *Attempt   `json:"attempt,omitempty"`
	Admission     *Admission `json:"admission,omitempty"`
	Report        *Report    `json:"report,omitempty"`
}

// Journal is an exclusive per-attempt receipt, not a coordinator task store.
// A crash leaves the original journal for reconciliation; it is never replaced.
type Journal struct{ f *os.File }

func OpenJournal(path string) (*Journal, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("create exclusive mission receipt: %w", err)
	}
	// Sync the directory entry as well as later file contents before launching.
	parent, err := os.Open(filepath.Dir(path))
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	syncErr := fsyncdirSync(parent)
	closeErr := parent.Close()
	if syncErr != nil {
		_ = f.Close()
		return nil, fmt.Errorf("sync receipt directory: %w", syncErr)
	}
	if closeErr != nil {
		_ = f.Close()
		return nil, closeErr
	}
	return &Journal{f: f}, nil
}
func (j *Journal) Record(e Event) error {
	e.Version = 1
	if e.Time.IsZero() {
		e.Time = time.Now().UTC()
	}
	if err := json.NewEncoder(j.f).Encode(e); err != nil {
		return fmt.Errorf("write mission receipt: %w", err)
	}
	if err := j.f.Sync(); err != nil {
		return fmt.Errorf("sync mission receipt: %w", err)
	}
	return nil
}
func (j *Journal) Close() error { return j.f.Close() }

// fsyncdirSync flushes an already-open directory handle where the platform supports it.
// Windows has no fsync-on-directory; see internal/fsyncdir.
func fsyncdirSync(dir *os.File) error { return fsyncdir.Sync(dir.Name()) }
