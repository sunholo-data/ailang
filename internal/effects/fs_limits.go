package effects

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

// FS read size cap (M-V1-MEMORY-FOOTPRINT M3, D-C).
//
// Net has had a 5 MB body cap since it existed; FS reads had none, and read
// the whole file twice ([]byte, then string). EffEnv.FSMaxBytes bounds every
// FS read: 0 (the CLI default) keeps reads unbounded, serve-api sets it to
// its upload cap. `--fs-max-bytes` / AILANG_FS_MAX_BYTES set it on the CLI.
//
// The guard is the POST-READ length, not the stat. The stat is an early exit
// for the common oversize case; a pseudo-file reports size 0, and a file can
// grow between stat and read. Returning a silently truncated string would
// trade host OOM for data corruption, which is the "no silent fallbacks"
// violation the design quorum named.

// errFSTooLarge is the typed error every capped read returns.
func errFSTooLarge(path string, size, capBytes int64) error {
	if size >= 0 {
		return fmt.Errorf("E_FS_FILE_TOO_LARGE: %s is %d bytes, cap %d (raise --fs-max-bytes or read it in parts)", path, size, capBytes)
	}
	return fmt.Errorf("E_FS_FILE_TOO_LARGE: %s exceeds the %d-byte cap (raise --fs-max-bytes or read it in parts)", path, capBytes)
}

// readCapped reads path under capBytes (<= 0 = unbounded, plain os.ReadFile).
func readCapped(path string, capBytes int64) ([]byte, error) {
	if capBytes <= 0 {
		return os.ReadFile(path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var statSize int64
	if st, err := f.Stat(); err == nil {
		statSize = st.Size()
		if statSize > capBytes {
			return nil, errFSTooLarge(path, statSize, capBytes)
		}
	}
	data, err := readCappedFrom(f, statSize, capBytes)
	if err != nil {
		if err == errOverCap {
			return nil, errFSTooLarge(path, -1, capBytes)
		}
		return nil, err
	}
	return data, nil
}

var errOverCap = fmt.Errorf("E_FS_FILE_TOO_LARGE")

// readCappedFrom reads at most capBytes from r into a buffer pre-sized from
// the stat hint (bounded by the cap), and rejects if a single byte more was
// available. hint may be 0 or wrong; only the read decides.
func readCappedFrom(r io.Reader, hint, capBytes int64) ([]byte, error) {
	size := hint
	if size > capBytes {
		size = capBytes
	}
	if size < 0 {
		size = 0
	}
	buf := bytes.NewBuffer(make([]byte, 0, size+1))
	n, err := buf.ReadFrom(io.LimitReader(r, capBytes+1))
	if err != nil {
		return nil, err
	}
	if n > capBytes {
		return nil, errOverCap
	}
	return buf.Bytes(), nil
}
