package loader

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// M-EXECUTOR-POLICY-HARDENING M3 (AC6) — the source snapshot.
//
// A policy-gated run admits a program (typecheck + row check) and then
// executes it, and both phases load modules through this package. Without a
// snapshot the two phases read the disk twice, and a file rewritten in
// between (the agent's own write tool, a concurrent process) is admitted as
// one program and executed as another. With the snapshot enabled, the FIRST
// read of every path is the only read: later reads return the recorded
// bytes, and the digest over the recorded graph is what gets banked.
//
// The snapshot also bounds what a run may load: a per-file ceiling (the
// policy's max_source_bytes) and an aggregate ceiling (max_module_graph_bytes)
// are checked BEFORE a file's contents are retained, so an oversized module
// graph is refused rather than allocated.

type sourceSnapshot struct {
	mu       sync.Mutex
	enabled  bool
	maxFile  int64
	maxTotal int64
	total    int64
	files    map[string][]byte
}

var snapshot sourceSnapshot

// ErrSourceTooLarge is wrapped by every size refusal so callers can name it.
var ErrSourceTooLarge = fmt.Errorf("E_SOURCE_TOO_LARGE")

// EnableSourceSnapshot freezes every source read for the rest of the process.
// maxFileBytes / maxTotalBytes of 0 mean unbounded. Enabling twice resets
// nothing: the recorded files are kept (a second enable only tightens caps
// that were unset).
func EnableSourceSnapshot(maxFileBytes, maxTotalBytes int64) {
	snapshot.mu.Lock()
	defer snapshot.mu.Unlock()
	snapshot.enabled = true
	if snapshot.files == nil {
		snapshot.files = map[string][]byte{}
	}
	if maxFileBytes > 0 {
		snapshot.maxFile = maxFileBytes
	}
	if maxTotalBytes > 0 {
		snapshot.maxTotal = maxTotalBytes
	}
}

// SourceSnapshotEnabled reports whether reads are being frozen.
func SourceSnapshotEnabled() bool {
	snapshot.mu.Lock()
	defer snapshot.mu.Unlock()
	return snapshot.enabled
}

// ReadSourceFile is the one way source bytes come off the disk: plain
// os.ReadFile when no snapshot is enabled, else the recorded bytes for the
// path (recording them on first read, under the caps).
func ReadSourceFile(path string) ([]byte, error) {
	snapshot.mu.Lock()
	enabled := snapshot.enabled
	snapshot.mu.Unlock()
	if !enabled {
		return os.ReadFile(path)
	}
	key := snapshotKey(path)
	snapshot.mu.Lock()
	if data, ok := snapshot.files[key]; ok {
		snapshot.mu.Unlock()
		return data, nil
	}
	maxFile, maxTotal, total := snapshot.maxFile, snapshot.maxTotal, snapshot.total
	snapshot.mu.Unlock()

	// Size the file BEFORE reading it so an oversized module is refused
	// without allocating its contents; the post-read length is still the
	// check that counts (a file can grow between stat and read).
	if st, err := os.Stat(path); err == nil {
		if maxFile > 0 && st.Size() > maxFile {
			return nil, fmt.Errorf("%w: %s is %d bytes, cap %d (max_source_bytes)", ErrSourceTooLarge, path, st.Size(), maxFile)
		}
		if maxTotal > 0 && total+st.Size() > maxTotal {
			return nil, fmt.Errorf("%w: loading %s would take the module graph past %d bytes (max_module_graph_bytes)", ErrSourceTooLarge, path, maxTotal)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	n := int64(len(data))
	snapshot.mu.Lock()
	defer snapshot.mu.Unlock()
	if prior, ok := snapshot.files[key]; ok {
		// Lost a race to another reader of the same path: theirs is the
		// snapshot, ours is discarded — one set of bytes per path.
		return prior, nil
	}
	if snapshot.maxFile > 0 && n > snapshot.maxFile {
		return nil, fmt.Errorf("%w: %s is %d bytes, cap %d (max_source_bytes)", ErrSourceTooLarge, path, n, snapshot.maxFile)
	}
	if snapshot.maxTotal > 0 && snapshot.total+n > snapshot.maxTotal {
		return nil, fmt.Errorf("%w: loading %s would take the module graph past %d bytes (max_module_graph_bytes)", ErrSourceTooLarge, path, snapshot.maxTotal)
	}
	snapshot.files[key] = data
	snapshot.total += n
	return data, nil
}

// SourceSnapshotDigest is the sha256 over the sorted (path, sha256(bytes))
// pairs of every recorded file, with the file count and total bytes. Empty
// digest when the snapshot is not enabled.
func SourceSnapshotDigest() (digest string, files int, bytes int64) {
	snapshot.mu.Lock()
	defer snapshot.mu.Unlock()
	if !snapshot.enabled {
		return "", 0, 0
	}
	keys := make([]string, 0, len(snapshot.files))
	for k := range snapshot.files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		sum := sha256.Sum256(snapshot.files[k])
		fmt.Fprintf(h, "%s\x00%x\n", k, sum)
	}
	return hex.EncodeToString(h.Sum(nil)), len(keys), snapshot.total
}

// snapshotKey normalises a path so the same file read under two spellings
// is one entry.
func snapshotKey(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		return real
	}
	return abs
}

// ResetSourceSnapshotForTest clears all state; tests only.
func ResetSourceSnapshotForTest() {
	snapshot.mu.Lock()
	defer snapshot.mu.Unlock()
	snapshot.enabled = false
	snapshot.maxFile, snapshot.maxTotal, snapshot.total = 0, 0, 0
	snapshot.files = nil
}
