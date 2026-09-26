package opus

import (
	"errors"

	"github.com/sunholo-data/ailang/third_party/goopus/internal/packet"
)

// Version is the go-opus release string, mirroring flac.Version in go-flac.
const Version = "1.0.0"

// Sentinel errors returned by the opus package, testable with errors.Is. They
// map from the libopus OPUS_* error codes (see docs/api-design.md): OPUS_BAD_ARG
// -> ErrBadArg, OPUS_BUFFER_TOO_SMALL -> ErrBufferTooSmall, OPUS_INVALID_PACKET
// -> ErrInvalidPacket, OPUS_INTERNAL_ERROR/OPUS_ALLOC_FAIL -> ErrInternal.
var (
	// ErrBadArg reports an invalid argument (bad sample rate, channel count, or
	// decode parameter).
	ErrBadArg = errors.New("opus: bad argument")
	// ErrBufferTooSmall reports that a caller-provided buffer cannot hold the
	// result.
	ErrBufferTooSmall = errors.New("opus: buffer too small")
	// ErrInvalidPacket reports a malformed Opus packet. The decoder never panics
	// on hostile input; it returns this instead.
	ErrInvalidPacket = errors.New("opus: invalid packet")
	// ErrInternal reports an internal decoder error that should not occur for
	// well-formed input.
	ErrInternal = errors.New("opus: internal error")
)

// Mode, Bandwidth, and FrameDuration re-export the packet-inspection types from
// the internal packet layer so callers of the public API can name them. They are
// type aliases, so values flow through with no conversion.
type (
	// Mode is the Opus coding mode selected by a TOC byte.
	Mode = packet.Mode
	// Bandwidth is the audio bandwidth selected by a TOC byte.
	Bandwidth = packet.Bandwidth
	// FrameDuration is the per-frame duration selected by a TOC byte.
	FrameDuration = packet.FrameDuration
)

// Coding modes, re-exported from the internal packet layer.
const (
	// ModeSILKOnly is a SILK-only (LP) frame.
	ModeSILKOnly = packet.ModeSILKOnly
	// ModeHybrid is a SILK+CELT hybrid frame.
	ModeHybrid = packet.ModeHybrid
	// ModeCELTOnly is a CELT-only (MDCT) frame.
	ModeCELTOnly = packet.ModeCELTOnly
)

// Audio bandwidths, re-exported from the internal packet layer.
const (
	// BandwidthNarrowband is NB, 4 kHz passband.
	BandwidthNarrowband = packet.BandwidthNarrowband
	// BandwidthMediumband is MB, 6 kHz passband.
	BandwidthMediumband = packet.BandwidthMediumband
	// BandwidthWideband is WB, 8 kHz passband.
	BandwidthWideband = packet.BandwidthWideband
	// BandwidthSuperwideband is SWB, 12 kHz passband.
	BandwidthSuperwideband = packet.BandwidthSuperwideband
	// BandwidthFullband is FB, 20 kHz passband.
	BandwidthFullband = packet.BandwidthFullband
)

// Frame durations, re-exported from the internal packet layer.
const (
	// FrameDuration2500us is 2.5 ms (CELT only).
	FrameDuration2500us = packet.FrameDuration2500us
	// FrameDuration5ms is 5 ms (CELT only).
	FrameDuration5ms = packet.FrameDuration5ms
	// FrameDuration10ms is 10 ms.
	FrameDuration10ms = packet.FrameDuration10ms
	// FrameDuration20ms is 20 ms.
	FrameDuration20ms = packet.FrameDuration20ms
	// FrameDuration40ms is 40 ms (SILK only).
	FrameDuration40ms = packet.FrameDuration40ms
	// FrameDuration60ms is 60 ms (SILK only).
	FrameDuration60ms = packet.FrameDuration60ms
)

// ParseTOC decodes the table-of-contents byte (the first byte of every Opus
// packet) into its coding mode, audio bandwidth, per-frame duration, and channel
// count (1 or 2). Parsing a TOC never fails: every byte value is a valid TOC.
func ParseTOC(b byte) (mode Mode, bandwidth Bandwidth, duration FrameDuration, channels int) {
	t := packet.ParseTOC(b)
	return t.Mode(), t.Bandwidth(), t.FrameDuration(), t.Channels()
}

// PacketFrames returns the number of frames in an Opus packet, mirroring
// opus_packet_get_nb_frames. It returns ErrInvalidPacket for a malformed packet.
func PacketFrames(pkt []byte) (int, error) {
	n, err := packet.CountFrames(pkt)
	if err != nil {
		return 0, mapPacketErr(err)
	}
	return n, nil
}

// PacketDuration returns the total duration of an Opus packet in samples per
// channel at 48 kHz, mirroring opus_packet_get_nb_samples. It returns
// ErrInvalidPacket for a malformed packet or a duration exceeding 120 ms.
func PacketDuration(pkt []byte) (int, error) {
	n, err := packet.Samples(pkt, 48000)
	if err != nil {
		return 0, mapPacketErr(err)
	}
	return n, nil
}

// mapPacketErr translates an internal packet error into the public sentinel set.
func mapPacketErr(err error) error {
	switch {
	case errors.Is(err, packet.ErrBadArg):
		return ErrBadArg
	case errors.Is(err, packet.ErrInvalidPacket):
		return ErrInvalidPacket
	default:
		return err
	}
}
