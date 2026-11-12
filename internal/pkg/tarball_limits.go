package pkg

import (
	"fmt"
	"io"
)

// Package tarball limits — the one place they live.
//
// Both sides of the registry use them: the registry validator refuses a
// publish over any of them, and ExtractTarball (shared by the validator,
// `ailang install`, `ailang pkg docs` and the dependency resolver) refuses to
// extract past them. Sharing one set means anything the registry accepted,
// the client can install.
//
// Why bounded at all: a gzip stream of zeros compresses about 1000:1, so an
// unbounded extract lets a ~30 MB upload write tens of GB. On Cloud Run the
// validator's /tmp is memory-backed, so that kills the whole instance.
//
// Why these values: measured 2026-11-12 over all 450 published tarballs in
// gs://ailang-registry. The largest package, sunholo/ailang_parse@0.54.0, is
// 690 KB compressed, 2.4 MB extracted and 145 entries; the largest single
// entry is 434 KB (sunholo/mcp_files extapps_bundle.ail). The limits sit
// 13–46x above that, room for growth and bundled assets/, and low enough that
// one publish (compressed bytes in memory plus extracted bytes in /tmp) fits
// in the 128 MiB the validator keeps beside the compile check
// (checkRSSHeadroom in cmd/registry-validator/bounded.go).
//
// Exceeding one is a refusal naming the limit (*TarballLimitError), never a
// truncation. If a real package needs more, raise the value here and re-check
// the validator's memory headroom.
const (
	// MaxTarballBytes caps the compressed tarball (46x the largest, 690 KB).
	MaxTarballBytes = 32 << 20
	// MaxTarballEntryBytes caps one extracted file (19x the largest, 434 KB).
	MaxTarballEntryBytes = 8 << 20
	// MaxTarballTotalBytes caps the sum of extracted file sizes (13x the
	// largest package, 2.4 MB).
	MaxTarballTotalBytes = 32 << 20
	// MaxTarballEntries caps the entry count, of every type (7x the largest,
	// 145).
	MaxTarballEntries = 1000

	// maxTarballStreamBytes caps the decompressed gzip stream itself. The
	// size limits above are checked against each entry's declared size, but
	// archive/tar also reads bytes no entry declares — headers, padding and
	// PAX / GNU long-name records, which it consumes inside Next() — so the
	// stream needs its own bound. 4 KiB per entry covers a 512-byte header,
	// up to 511 bytes of padding and a long-name record.
	maxTarballStreamBytes = MaxTarballTotalBytes + MaxTarballEntries*4<<10
)

// TarballLimitError reports a tarball refused for exceeding one of the limits
// above. Callers can test for it with errors.As to answer 413 rather than 400.
type TarballLimitError struct {
	Limit    string // which limit, e.g. "per-entry size"
	Const    string // the Go constant that sets it, e.g. "pkg.MaxTarballEntryBytes"
	Max      int64
	Observed string // what was seen, e.g. "entry assets/big.bin declares 9437184 bytes"
}

func (e *TarballLimitError) Error() string {
	return fmt.Sprintf("tarball refused: %s exceeds the package %s limit of %s (%s). "+
		"Nothing was truncated; the whole package is refused. If a real package needs more, "+
		"raise %s in internal/pkg/tarball_limits.go",
		e.Observed, e.Limit, humanBytes(e.Max), e.Const, e.Const)
}

func humanBytes(n int64) string {
	if n >= 1<<20 && n%(1<<20) == 0 {
		return fmt.Sprintf("%d MiB", n>>20)
	}
	return fmt.Sprintf("%d", n)
}

// CheckTarballSize refuses a compressed tarball over MaxTarballBytes.
func CheckTarballSize(n int64) error {
	if n > MaxTarballBytes {
		return &TarballLimitError{
			Limit:    "compressed size",
			Const:    "pkg.MaxTarballBytes",
			Max:      MaxTarballBytes,
			Observed: fmt.Sprintf("the tarball is %d bytes", n),
		}
	}
	return nil
}

// cappedReader returns a *TarballLimitError once more than max bytes have
// been read from r, instead of the silent EOF io.LimitReader would give.
type cappedReader struct {
	r   io.Reader
	n   int64
	max int64
}

func (c *cappedReader) Read(p []byte) (int, error) {
	if c.n > c.max {
		return 0, c.limitErr()
	}
	// Read at most one byte past the cap: enough to know it was passed.
	if room := c.max + 1 - c.n; int64(len(p)) > room {
		p = p[:room]
	}
	n, err := c.r.Read(p)
	c.n += int64(n)
	if c.n > c.max {
		return n, c.limitErr()
	}
	return n, err
}

func (c *cappedReader) limitErr() error {
	return &TarballLimitError{
		Limit:    "decompressed stream size",
		Const:    "pkg.MaxTarballTotalBytes",
		Max:      c.max,
		Observed: fmt.Sprintf("the decompressed stream passed %d bytes", c.max),
	}
}
