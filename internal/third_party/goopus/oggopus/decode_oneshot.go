package oggopus

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

// DefaultMaxDecodedBytes is the ceiling DecodeInterleaved applies to its output.
// An Ogg Opus stream's decoded length is not proportional to its size: a two-byte
// packet of zero-length DTX frames decodes to 120 ms of silence, so a small
// crafted stream can decode to far more than its bytes suggest. The one-shot
// helpers hold the whole decode in memory, so they stop at this ceiling rather
// than let a tiny input drive an unbounded allocation. It is 1 GiB, about 101
// minutes of 48 kHz stereo int16.
//
// A caller decoding a stream it did not produce, or one legitimately longer than
// this, should use NewDecoder and stream the audio instead, which bounds memory
// to a single reusable buffer regardless of length.
const DefaultMaxDecodedBytes = 1 << 30

// ErrDecodeLimit reports that a one-shot decode was stopped because its output
// would exceed the byte ceiling in effect. Test for it with errors.Is. See
// DefaultMaxDecodedBytes for why the limit exists.
var ErrDecodeLimit = errors.New("oggopus: decoded size limit exceeded")

// DecodeInterleaved reads an entire Ogg Opus stream from r and returns the decoded
// interleaved little-endian int16 PCM at 48 kHz together with the stream info. It
// is the one-shot mirror of EncodeInterleaved, and of the sibling one-shot
// decoders in the go-audio family (go-wav's pcm.DecodeInterleaved, go-flac's,
// go-m4a's opusm4a).
//
// It stops at DefaultMaxDecodedBytes and returns a wrapped ErrDecodeLimit if the
// output would exceed it. For a different ceiling use DecodeInterleavedLimit; for
// a stream of unknown or unbounded length use NewDecoder, which streams the audio
// in memory proportional to a single buffer.
func DecodeInterleaved(r io.Reader) ([]byte, Info, error) {
	return DecodeInterleavedLimit(r, DefaultMaxDecodedBytes)
}

// DecodeInterleavedLimit is DecodeInterleaved with a caller-chosen ceiling.
// maxBytes is the largest decoded output it will return; a decode that would
// exceed it stops and returns a wrapped ErrDecodeLimit. A maxBytes of zero or
// less removes the ceiling, which is only safe for a stream the caller produced
// or has otherwise bounded.
func DecodeInterleavedLimit(r io.Reader, maxBytes int) ([]byte, Info, error) {
	d, err := NewDecoder(r)
	if err != nil {
		return nil, Info{}, err
	}
	info := d.Info()

	var buf bytes.Buffer
	cw := &cappedWriter{buf: &buf, max: maxBytes}
	if _, err := d.WriteTo(cw); err != nil {
		return nil, info, err
	}
	return buf.Bytes(), info, nil
}

// cappedWriter accumulates into buf and refuses a write that would carry the
// total past max. It writes nothing on the failing call, so buf holds only whole
// frames worth of samples up to the point the limit was hit. A max of zero or
// less is unbounded.
type cappedWriter struct {
	buf *bytes.Buffer
	n   int
	max int
}

func (c *cappedWriter) Write(p []byte) (int, error) {
	if c.max > 0 && c.n > c.max-len(p) {
		return 0, fmt.Errorf(
			"oggopus: DecodeInterleaved: %w: output would exceed %d bytes",
			ErrDecodeLimit, c.max)
	}
	n, err := c.buf.Write(p)
	c.n += n
	return n, err
}
