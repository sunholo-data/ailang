package opus

import (
	"errors"
	"fmt"

	"github.com/sunholo-data/ailang/third_party/goopus/internal/opusdec"
)

// Decoder decodes Opus packets to interleaved 16-bit PCM. It is the thin,
// idiomatic public wrapper over internal/opusdec, which holds the verbatim
// transliteration of libopus opus_decode_native / opus_decode_frame (the
// CELT/SILK/hybrid mode-transition machine). All three coding modes decode:
// CELT-only, SILK-only and hybrid, including the mode transitions and the CELT
// redundancy frames at SILK<->CELT boundaries.
//
// A Decoder is stateful: the underlying SILK and CELT decoders carry overlap-add,
// energy prediction, post-filter, LTP, resampler and transition memory across
// packets, so packets from one stream must be fed to one Decoder in order. It is
// not safe for concurrent use; use one Decoder per goroutine.
type Decoder struct {
	sampleRate int
	channels   int
	dec        *opusdec.OpusDecoder
}

// NewDecoder returns a decoder producing sampleRate-Hz, channels-channel PCM.
// sampleRate must be one of 8000, 12000, 16000, 24000, or 48000; channels must
// be 1 or 2. Anything else returns ErrBadArg. The RFC 6716 conformance vectors
// decode at 48000 Hz.
func NewDecoder(sampleRate, channels int) (*Decoder, error) {
	switch sampleRate {
	case 8000, 12000, 16000, 24000, 48000:
	default:
		return nil, fmt.Errorf("%w: sample rate %d (want 8000, 12000, 16000, 24000, or 48000)", ErrBadArg, sampleRate)
	}
	if channels < 1 || channels > 2 {
		return nil, fmt.Errorf("%w: channels %d (want 1 or 2)", ErrBadArg, channels)
	}

	dec, err := opusdec.NewDecoder(int32(sampleRate), channels)
	if err != nil {
		return nil, mapDecErr(err)
	}
	return &Decoder{
		sampleRate: sampleRate,
		channels:   channels,
		dec:        dec,
	}, nil
}

// Decode decodes one Opus packet into interleaved PCM and returns the number of
// samples produced per channel. pcm must hold at least that many samples times
// the channel count; a too-small buffer returns ErrBufferTooSmall.
//
// CELT-only, SILK-only and hybrid packets all decode, including mode transitions
// and redundancy. A nil or empty packet requests packet-loss concealment (PLC)
// for the frame duration implied by len(pcm), which must be a multiple of 2.5 ms
// per channel. A malformed packet returns ErrInvalidPacket; the decoder never
// panics on hostile input. Like opus_decode, this does not drop the codec
// pre-skip: dropping it is the container (oggopus) decoder's job.
func (d *Decoder) Decode(pkt []byte, pcm []int16) (int, error) {
	return d.decode(pkt, pcm, 0)
}

// DecodeFEC reconstructs a lost frame from the in-band FEC (LBRR) data carried by
// the NEXT received packet, mirroring opus_decode with decode_fec=1. pkt is that
// next packet; pcm receives the recovered frame and its length per channel must
// be a multiple of 2.5 ms. If pkt carries no FEC for the lost duration, the
// decoder falls back to PLC. Returns the per-channel sample count.
func (d *Decoder) DecodeFEC(pkt []byte, pcm []int16) (int, error) {
	return d.decode(pkt, pcm, 1)
}

// decode is the shared body of Decode / DecodeFEC. frameSize is the caller's
// per-channel buffer capacity, exactly the frame_size opus_decode passes down.
func (d *Decoder) decode(pkt []byte, pcm []int16, decodeFec int) (int, error) {
	frameSize := len(pcm) / d.channels
	n, err := d.dec.Decode(pkt, pcm, frameSize, decodeFec)
	if err != nil {
		return 0, mapDecErr(err)
	}
	return n, nil
}

// FinalRange returns the range-coder state after the last decoded frame
// (OPUS_GET_FINAL_RANGE). For a conformant decode it equals the encoder's final
// range recorded in the test bitstream, the strong per-packet bit-exactness
// check across all modes, including the mode-transition vectors.
func (d *Decoder) FinalRange() uint32 { return d.dec.FinalRange() }

// LastPacketDuration returns the number of samples per channel produced by the
// most recent successful Decode call.
func (d *Decoder) LastPacketDuration() int { return d.dec.LastPacketDuration() }

// Reset clears cross-packet decoder state (SILK and CELT overlap-add, energy
// prediction, post-filter, LTP, resampler and transition memory) while keeping
// the configured sample rate and channel count, mirroring
// opus_decoder_ctl(OPUS_RESET_STATE).
func (d *Decoder) Reset() {
	d.dec.ResetState()
}

// mapDecErr translates an internal opusdec error into the public sentinel set.
func mapDecErr(err error) error {
	switch {
	case errors.Is(err, opusdec.ErrBadArg):
		return fmt.Errorf("%w: %w", ErrBadArg, err)
	case errors.Is(err, opusdec.ErrBufferTooSmall):
		return fmt.Errorf("%w: %w", ErrBufferTooSmall, err)
	case errors.Is(err, opusdec.ErrInvalidPacket):
		return fmt.Errorf("%w: %w", ErrInvalidPacket, err)
	default:
		return fmt.Errorf("%w: %w", ErrInternal, err)
	}
}
