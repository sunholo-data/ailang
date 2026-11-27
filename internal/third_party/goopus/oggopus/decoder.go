package oggopus

import (
	"encoding/binary"
	"errors"
	"io"
)

// Decoder implements io.Reader and io.WriterTo, the go-flac pcm.Decoder shape.
var (
	_ io.Reader   = (*Decoder)(nil)
	_ io.WriterTo = (*Decoder)(nil)
)

// OutputSampleRate is the sample rate of all decoded PCM, in Hz. RFC 7845
// defines the pre-skip, every granule position, and the decode output at 48 kHz
// regardless of the informational input sample rate recorded in OpusHead, so
// Read and WriteTo always produce 48 kHz PCM. The constant is untyped so it
// compares directly with Info.OutputSampleRate (uint32) and with int sample
// rates. Label decoded PCM with this, never with Info.InputSampleRate.
const OutputSampleRate = 48000

// Info reports the stream parameters parsed from OpusHead.
type Info struct {
	Channels         int    // decoded channel count (mapping family 0: 1 or 2)
	InputSampleRate  uint32 // original source rate recorded in OpusHead (informational; 0 if unspecified); NOT the decode output rate
	OutputSampleRate uint32 // rate of the decoded PCM in Hz; always 48000 (OutputSampleRate)
	PreSkip          int    // samples to discard at the start, at 48 kHz
	OutputGain       int16  // Q7.8 gain in dB applied on decode
}

// Decoder reads an Ogg Opus stream from an io.Reader and yields interleaved
// little-endian int16 PCM at 48 kHz, implementing io.Reader and io.WriterTo. It
// is shaped like go-flac's pcm.Decoder.
//
// NewDecoder parses and validates the headers, and Info reports the stream
// parameters. The packet->PCM conversion goes through the codec seam (codec.go),
// which wraps the public opus.Decoder. Output is always 48 kHz, the rate RFC 7845
// defines the pre-skip and every granule position at, whatever OpusHead's
// informational input-sample-rate field says. A Decoder is not safe for
// concurrent use.
type Decoder struct {
	cr   *containerReader
	dec  frameDecoder // nil until the codec seam is wired
	info Info

	pcm         []byte // decoded, trimmed PCM awaiting delivery
	preSkipLeft int    // pre-skip samples per channel still to drop
	delivered   int64  // samples per channel already produced
	limit       int64  // total output samples per channel, or -1 until known
}

// NewDecoder reads and validates the OpusHead and OpusTags headers from r and
// returns a Decoder positioned at the first audio packet. Header parsing is
// real; the PCM output path is stubbed at the codec seam.
func NewDecoder(r io.Reader) (*Decoder, error) {
	d := &Decoder{}
	if err := d.init(r); err != nil {
		return nil, err
	}
	return d, nil
}

// Reset rebinds the Decoder to a new source r, re-parsing its OpusHead and
// OpusTags headers, so one Decoder can decode many independent streams without
// re-allocating. It mirrors Encoder.Reset and the sibling pcm.Decoder.Reset in
// go-wav, go-flac, and go-aac: the pooling path for a "many short clips" workload.
//
// The codec and container reader are cleared first, so a Reset that fails partway
// (a nil source, or headers that will not parse) leaves a Decoder that reports its
// uninitialized error on the next Read rather than one still bound to the previous
// stream. The decoded-PCM scratch buffer keeps its capacity for reuse.
func (d *Decoder) Reset(r io.Reader) error {
	d.cr = nil
	d.dec = nil
	// Clear info too, so a Reset that fails partway does not leave Info reporting
	// the previous stream's parameters (init only repopulates it on success). This
	// matches go-flac and go-aac, whose decoders zero their info up front.
	d.info = Info{}
	d.pcm = d.pcm[:0]
	return d.init(r)
}

// init parses and validates the container headers from r and populates the
// decoder's per-stream state. It backs both NewDecoder and Reset, so the two agree
// on header handling and on rejecting a nil source before any read. It does not
// touch d.pcm, so Reset can retain that buffer's capacity across streams.
func (d *Decoder) init(r io.Reader) error {
	// Reject a nil source up front; otherwise the first container read dereferences
	// a nil io.Reader and panics.
	if r == nil {
		return errNilReader
	}
	cr, err := newContainerReader(r)
	if err != nil {
		return err
	}
	dec, err := newFrameDecoder(cr.head)
	if err != nil {
		return err
	}
	d.cr = cr
	d.dec = dec
	d.preSkipLeft = int(cr.head.preSkip)
	d.delivered = 0
	d.limit = -1
	d.info = Info{
		Channels:         int(cr.head.channels),
		InputSampleRate:  cr.head.inputSampleRate,
		OutputSampleRate: OutputSampleRate,
		PreSkip:          int(cr.head.preSkip),
		OutputGain:       cr.head.outputGain,
	}
	return nil
}

// Info returns the stream parameters parsed from OpusHead.
func (d *Decoder) Info() Info { return d.info }

// Read fills p with interleaved little-endian int16 PCM at 48 kHz, applying the
// pre-skip drop, the end-trim, and the OpusHead output gain. It returns io.EOF
// when the stream is exhausted.
//
// The output rate is always 48000 (OutputSampleRate), whatever OpusHead's
// informational input rate says; label decoded PCM with Info().OutputSampleRate,
// never with Info().InputSampleRate.
func (d *Decoder) Read(p []byte) (int, error) {
	if d.dec == nil {
		return 0, errUninitialized
	}
	for len(d.pcm) == 0 {
		done, err := d.fill()
		if err != nil {
			return 0, err
		}
		if done {
			return 0, io.EOF
		}
	}
	n := copy(p, d.pcm)
	d.pcm = d.pcm[n:]
	return n, nil
}

// WriteTo streams the whole decoded output to w, implementing io.WriterTo so a
// caller can io.Copy a Decoder to a sink. The PCM written to w is interleaved
// little-endian int16 at 48 kHz (OutputSampleRate); see Read.
func (d *Decoder) WriteTo(w io.Writer) (int64, error) {
	if d.dec == nil {
		return 0, errUninitialized
	}
	var written int64
	for {
		if len(d.pcm) == 0 {
			done, err := d.fill()
			if err != nil {
				return written, err
			}
			if done {
				return written, nil
			}
		}
		n, err := w.Write(d.pcm)
		written += int64(n)
		d.pcm = d.pcm[n:]
		if err != nil {
			return written, err
		}
	}
}

// fill decodes the next packet into d.pcm, applying the pre-skip drop at the
// start and the granule end-trim at the end. It reports done when the stream is
// fully consumed.
func (d *Decoder) fill() (done bool, err error) {
	pkt, err := d.cr.nextPacket()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return true, nil
		}
		return false, err
	}
	samples, err := d.dec.decodeFrame(pkt)
	if err != nil {
		return false, err
	}
	if d.limit < 0 {
		if total, ok := d.cr.totalSamples(); ok {
			d.limit = total
		}
	}
	perChan := len(samples) / d.info.Channels

	// Drop leading pre-skip samples spread across the first packets.
	if d.preSkipLeft > 0 {
		drop := min(d.preSkipLeft, perChan)
		samples = samples[drop*d.info.Channels:]
		perChan -= drop
		d.preSkipLeft -= drop
	}
	// End-trim so the total delivered equals finalGranule - preSkip.
	if d.limit >= 0 && d.delivered+int64(perChan) > d.limit {
		keep := int(d.limit - d.delivered)
		if keep < 0 {
			keep = 0
		}
		samples = samples[:keep*d.info.Channels]
		perChan = keep
	}
	d.delivered += int64(perChan)

	d.pcm = d.pcm[:0]
	for _, s := range samples {
		d.pcm = binary.LittleEndian.AppendUint16(d.pcm, uint16(s))
	}
	return false, nil
}
