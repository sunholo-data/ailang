package oggopus

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/rand/v2"
	"sync"
)

// Encoder implements io.WriteCloser, byte-for-byte the go-flac pcm.Encoder shape.
var _ io.WriteCloser = (*Encoder)(nil)

// sampleRate48k is the fixed Opus coding/granule rate, the same rate the
// decoder's exported OutputSampleRate names for consumers. Tying the two
// together keeps the internal coding rate and the public output rate from
// drifting apart.
const sampleRate48k = OutputSampleRate

// Encoder streams interleaved little-endian int16 PCM (as []byte) to an Ogg
// Opus stream on an io.Writer, implementing io.WriteCloser. It is shaped exactly
// like go-flac's pcm.Encoder: a flat Config, NewEncoder(w, cfg), Reset(w, cfg)
// for pooling, and a one-shot EncodeInterleaved.
//
// Duration is carried in per-page granule positions written inline as the
// stream progresses, so the Encoder is correct on a plain io.Writer with no
// seeking and no length known up front. This is the structural fix for the
// go-flac non-seekable zero-duration bug and requires no io.WriteSeeker (see
// docs/api-design.md "Non-seekable sinks are correct by construction").
//
// The PCM<->packet conversion goes through the codec seam (codec.go), which
// wraps the public opus.Encoder. An Encoder is not safe for concurrent use.
type Encoder struct {
	w   io.Writer
	cfg Config

	enc frameEncoder     // nil only on a zero value or after a failed reset
	cw  *containerWriter // created once the codec pre-skip is known

	frameLen   int     // input samples per channel per frame (SampleRate * effective FrameDurationMS / 1000)
	frameBytes int     // bytes in one full frame (frameLen * channels * 2)
	frame      []int16 // reusable scratch for one deinterleaved frame
	carry      []byte  // buffered PCM bytes not yet a full frame (bounded to one frame)

	srcSamples int64 // real source samples per channel consumed (input rate)
	coded48k   int64 // 48 kHz samples per channel actually coded into packets
	closed     bool
}

// NewEncoder validates cfg, builds the codec, and returns an Encoder writing to
// w, having already emitted the OpusHead and OpusTags header pages. It does not
// require an io.WriteSeeker. A config error (bad sample rate, channel count,
// bitrate or complexity) returns ErrInvalidConfig, immediately, before a byte is
// written.
func NewEncoder(w io.Writer, cfg Config) (*Encoder, error) { //nolint:gocritic // Config passed by value to match the go-flac-aligned public API (docs/api-design.md)
	e := &Encoder{}
	if err := e.reset(w, &cfg); err != nil {
		return nil, err
	}
	return e, nil
}

// Reset rebinds the Encoder to a new sink w and reconfigures it with cfg so one
// Encoder can encode many independent streams without re-allocating, the pooling
// path for the BirdNET-Go "many short clips" workload. It re-validates cfg,
// discards buffered input, resets per-stream state, and re-emits the OpusHead and
// OpusTags header pages to w.
func (e *Encoder) Reset(w io.Writer, cfg Config) error { //nolint:gocritic // Config passed by value to match the go-flac-aligned public API (docs/api-design.md)
	return e.reset(w, &cfg)
}

// reset takes cfg by pointer so the flat, slice-carrying Config is not copied
// on every pooled Reset; the public entry points still take it by value to
// match the go-flac-aligned API.
//
// The codec and container writer are cleared FIRST, so a reset that fails partway
// leaves an Encoder that reports errUninitialized rather than one still holding
// the previous stream's container: a pooled Encoder whose Reset was rejected must
// not be able to append to the stream it was last used for.
func (e *Encoder) reset(w io.Writer, cfg *Config) error {
	e.enc = nil
	e.cw = nil
	e.carry = e.carry[:0]
	e.srcSamples = 0
	e.coded48k = 0
	e.closed = false
	if err := cfg.validate(); err != nil {
		return err
	}
	// Reject a nil sink after validation, so NewEncoder and EncodeInterleaved (which
	// validates before it ever calls reset) both report a bad config first and agree
	// on precedence. Without this, writing the header pages to a nil io.Writer panics.
	if w == nil {
		return errNilWriter
	}
	e.w = w
	e.cfg = *cfg
	e.frameLen = cfg.SampleRate * cfg.frameDurationMS() / 1000
	e.frameBytes = e.frameLen * cfg.Channels * 2
	if cap(e.frame) >= e.frameLen*cfg.Channels {
		e.frame = e.frame[:0]
	} else {
		e.frame = make([]int16, 0, e.frameLen*cfg.Channels)
	}

	enc, err := newFrameEncoder(e.cfg)
	if err != nil {
		return err
	}
	e.enc = enc
	return e.initStream()
}

// initStream builds the container writer and emits the header pages. It runs only
// after the codec exists, because OpusHead's pre-skip IS the encoder lookahead
// (scaled to 48 kHz by the seam; see opusFrameEncoder.lookahead).
func (e *Encoder) initStream() error {
	head := opusHead{
		version:         opusHeadVersion,
		channels:        byte(e.cfg.Channels),
		preSkip:         uint16(e.enc.lookahead()),
		inputSampleRate: uint32(e.cfg.SampleRate),
		outputGain:      0,
		mappingFamily:   mappingFamily0,
	}
	tags := opusTags{vendor: e.cfg.vendorString(), comments: e.cfg.Comments}
	cw, err := newContainerWriter(e.w, randomSerial(), head, tags)
	if err != nil {
		return err
	}
	e.cw = cw
	e.frame = e.frame[:e.frameLen*e.cfg.Channels]
	return nil
}

// Write consumes interleaved little-endian int16 PCM. Bytes that do not yet
// complete a frame are buffered until the next Write or Close, exactly like
// go-flac's block buffering; the carry stays bounded to a single frame. It
// returns the number of bytes consumed from p (io.Writer contract).
func (e *Encoder) Write(p []byte) (int, error) {
	if e.closed {
		return 0, ErrClosed
	}
	if e.enc == nil {
		return 0, errUninitialized
	}
	return e.write(p)
}

func (e *Encoder) write(p []byte) (int, error) {
	total := len(p)
	// Top off the carry to a full frame from the front of p.
	if len(e.carry) > 0 {
		need := e.frameBytes - len(e.carry)
		take := min(need, len(p))
		e.carry = append(e.carry, p[:take]...)
		p = p[take:]
		if len(e.carry) == e.frameBytes {
			if err := e.encodeFrameBytes(e.carry); err != nil {
				return total - len(p), err
			}
			e.carry = e.carry[:0]
		}
	}
	// Encode whole frames straight from p without copying into the carry.
	for len(p) >= e.frameBytes {
		if err := e.encodeFrameBytes(p[:e.frameBytes]); err != nil {
			return total - len(p), err
		}
		p = p[e.frameBytes:]
	}
	// Hold the sub-frame remainder for next time.
	e.carry = append(e.carry, p...)
	return total, nil
}

// encodeFrameBytes loads one full frame of interleaved PCM bytes into the scratch
// frame and encodes it. chunk must hold exactly frameBytes.
func (e *Encoder) encodeFrameBytes(chunk []byte) error {
	n := e.frameLen * e.cfg.Channels
	for i := range n {
		e.frame[i] = int16(binary.LittleEndian.Uint16(chunk[2*i : 2*i+2]))
	}
	return e.encodeFrame(e.frameLen)
}

// encodeFrame encodes whatever is in the scratch frame through the codec seam and
// queues the packet. realSamples is the number of genuine (non-padding) source
// samples per channel the frame carries: it drives the granule end-trim, and it
// is deliberately separate from the frame's CODED duration, which is always a
// full frame and is what coded48k accumulates. Keeping the two apart is what lets
// Close know how much real audio the stream carries and how much of it the coded
// frames actually cover.
func (e *Encoder) encodeFrame(realSamples int) error {
	pkt, samples48k, err := e.enc.encodeFrame(e.frame[:e.frameLen*e.cfg.Channels])
	if err != nil {
		return err
	}
	e.srcSamples += int64(realSamples)
	e.coded48k += int64(samples48k)
	return e.cw.writePacket(pkt, samples48k)
}

// encodeSilenceFrame codes one all-zero frame. It contributes coded samples but
// no source samples, which is exactly what the RFC 7845 end padding is: audio the
// decoder must have in order to reconstruct the real samples that precede it, and
// which the granule then trims away.
func (e *Encoder) encodeSilenceFrame() error {
	clear(e.frame[:e.frameLen*e.cfg.Channels])
	return e.encodeFrame(0)
}

// Close encodes the final partial frame (zero-padded), emits whatever additional
// silent frames RFC 7845 requires behind it, writes the last page with an
// end-trimmed granule position, and flushes. The stream duration is then exact.
// Close is idempotent. It errors if buffered trailing bytes are not a whole
// number of samples.
func (e *Encoder) Close() error {
	if e.closed {
		return nil
	}
	if e.enc == nil {
		return errUninitialized
	}
	if len(e.carry) > 0 {
		bytesPerSample := e.cfg.Channels * 2
		if len(e.carry)%bytesPerSample != 0 {
			return fmt.Errorf("oggopus: %d trailing bytes are not a whole number of samples", len(e.carry))
		}
		// Load the partial frame into the scratch buffer and zero-fill the tail
		// rather than allocating a padded copy.
		realSamples := len(e.carry) / bytesPerSample
		have := len(e.carry) / 2
		for i := range have {
			e.frame[i] = int16(binary.LittleEndian.Uint16(e.carry[2*i : 2*i+2]))
		}
		clear(e.frame[have : e.frameLen*e.cfg.Channels])
		if err := e.encodeFrame(realSamples); err != nil {
			return err
		}
		e.carry = e.carry[:0]
	}

	// Convert real source samples per channel from the input rate to 48 kHz for
	// the granule end-trim. Every valid rate divides 48000 exactly.
	src48k := e.srcSamples * sampleRate48k / int64(e.cfg.SampleRate)

	// RFC 7845 section 7: "encode at least (length + delay_samples + extra_samples)
	// samples, and set the granule position of the last page to (length +
	// delay_samples + extra_samples)". The granule this stream is about to claim is
	// preSkip + src48k, and section 4.5 makes an end-of-stream granule that exceeds
	// the samples actually coded INVALID. Let F be the frame length in 48 kHz
	// samples (960 at the default 20 ms, up to 5760 at 120 ms). Coding only
	// ceil(src48k/F) frames leaves coded48k - src48k anywhere in [0, F-1], so
	// whenever that gap is smaller than the pre-skip the stream would claim samples
	// it never coded: it is why, at 20 ms, a 960-sample input used to claim granule
	// 1272 over 960 coded samples.
	//
	// Emit silent frames until the coded audio covers the granule. The gap is under
	// F and one frame adds F, so this runs at most once; the loop states the
	// invariant instead of relying on that arithmetic. F is at least 960 (every
	// allowed duration is a multiple of 20 ms) and the pre-skip is 312, so a single
	// silence frame always closes the gap regardless of frame duration. The samples
	// are pure padding: the decoder needs them to reconstruct the tail of the real
	// audio, and the end-trimmed granule then discards them.
	//
	// A stream with no audio at all still needs one audio page. RFC 7845 does not
	// forbid a header-only stream, but libavformat will not open one: ffmpeg exits
	// with "Error opening input: End of file", so a caller who hands us an empty clip
	// gets a file no ffmpeg-based consumer can read. That is reachable in the pooled
	// short-clip workload this API is meant for. The reference encoder does not
	// produce one either: libopusenc's ope_encoder_drain always codes at least one
	// frame. Emit a single silence frame; the granule is then preSkip + 0 = preSkip,
	// which end-trims the whole frame away, so the stream still decodes to exactly
	// zero samples and ffmpeg accepts it.
	if e.cw.audioCount == 0 {
		if err := e.encodeSilenceFrame(); err != nil {
			return err
		}
	}
	for e.coded48k < e.cw.preSkip+src48k {
		if err := e.encodeSilenceFrame(); err != nil {
			return err
		}
	}

	e.closed = true
	e.enc.close()
	return e.cw.close(src48k)
}

// encoderPool backs EncodeInterleaved so repeated same-shape calls are
// allocation-light, mirroring go-flac pcm.EncodeInterleaved.
var encoderPool = sync.Pool{New: func() any { return &Encoder{} }}

// EncodeInterleaved encodes a complete interleaved little-endian int16 PCM
// buffer to an Ogg Opus stream on w in a single call, mirroring go-flac
// pcm.EncodeInterleaved. It draws an Encoder from an internal sync.Pool, so
// repeated same-shape calls are allocation-light, and it is safe for concurrent
// use. pcm must hold a whole number of samples for cfg.
func EncodeInterleaved(w io.Writer, cfg Config, pcm []byte) error { //nolint:gocritic // Config passed by value to match the go-flac-aligned public API (docs/api-design.md)
	if err := cfg.validate(); err != nil {
		return err
	}
	bytesPerSample := cfg.Channels * 2
	if len(pcm)%bytesPerSample != 0 {
		return fmt.Errorf("%w: pcm length %d is not a whole number of %d-channel samples", ErrInvalidConfig, len(pcm), cfg.Channels)
	}
	e, _ := encoderPool.Get().(*Encoder)
	defer encoderPool.Put(e)
	if err := e.reset(w, &cfg); err != nil {
		return err
	}
	if _, err := e.Write(pcm); err != nil {
		return err
	}
	return e.Close()
}

// randomSerial returns a random Ogg logical bitstream serial number. A fresh
// serial per stream keeps distinct streams distinguishable if concatenated.
func randomSerial() uint32 {
	return rand.Uint32()
}
