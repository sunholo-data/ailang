package packet

import "strconv"

// Mode identifies the Opus coding mode selected by a TOC byte (RFC 6716 §3.1).
type Mode uint8

const (
	// ModeSILKOnly is a SILK-only (LP) frame, configs 0-11.
	ModeSILKOnly Mode = iota
	// ModeHybrid is a SILK+CELT hybrid frame, configs 12-15.
	ModeHybrid
	// ModeCELTOnly is a CELT-only (MDCT) frame, configs 16-31.
	ModeCELTOnly
)

// String returns a short human-readable name for the mode.
func (m Mode) String() string {
	switch m {
	case ModeSILKOnly:
		return "SILK"
	case ModeHybrid:
		return "Hybrid"
	case ModeCELTOnly:
		return "CELT"
	default:
		return "Mode(" + strconv.Itoa(int(m)) + ")"
	}
}

// Bandwidth identifies the audio bandwidth selected by a TOC byte (RFC 6716
// §3.1). The zero value is BandwidthNarrowband and the ordering is ascending,
// matching the OPUS_BANDWIDTH_* ordering in libopus.
type Bandwidth uint8

const (
	// BandwidthNarrowband is NB, 4 kHz passband.
	BandwidthNarrowband Bandwidth = iota
	// BandwidthMediumband is MB, 6 kHz passband.
	BandwidthMediumband
	// BandwidthWideband is WB, 8 kHz passband.
	BandwidthWideband
	// BandwidthSuperwideband is SWB, 12 kHz passband.
	BandwidthSuperwideband
	// BandwidthFullband is FB, 20 kHz passband.
	BandwidthFullband
)

// String returns the short RFC name for the bandwidth (NB, MB, WB, SWB, FB).
func (b Bandwidth) String() string {
	switch b {
	case BandwidthNarrowband:
		return "NB"
	case BandwidthMediumband:
		return "MB"
	case BandwidthWideband:
		return "WB"
	case BandwidthSuperwideband:
		return "SWB"
	case BandwidthFullband:
		return "FB"
	default:
		return "Bandwidth(" + strconv.Itoa(int(b)) + ")"
	}
}

// FrameDuration is the duration of a single Opus frame, expressed in
// microseconds. Opus permits exactly the six values below.
type FrameDuration int

const (
	// FrameDuration2500us is 2.5 ms (CELT only).
	FrameDuration2500us FrameDuration = 2500
	// FrameDuration5ms is 5 ms (CELT only).
	FrameDuration5ms FrameDuration = 5000
	// FrameDuration10ms is 10 ms.
	FrameDuration10ms FrameDuration = 10000
	// FrameDuration20ms is 20 ms.
	FrameDuration20ms FrameDuration = 20000
	// FrameDuration40ms is 40 ms (SILK only).
	FrameDuration40ms FrameDuration = 40000
	// FrameDuration60ms is 60 ms (SILK only).
	FrameDuration60ms FrameDuration = 60000
)

// Microseconds returns the frame duration in microseconds.
func (d FrameDuration) Microseconds() int { return int(d) }

// String returns the frame duration in milliseconds, e.g. "2.5ms" or "20ms".
func (d FrameDuration) String() string {
	whole := int(d) / 1000
	frac := (int(d) % 1000) / 100
	if frac == 0 {
		return strconv.Itoa(whole) + "ms"
	}
	return strconv.Itoa(whole) + "." + strconv.Itoa(frac) + "ms"
}

// TOC is a decoded table-of-contents byte, the first byte of every Opus packet.
// It carries the coding configuration (config number 0-31), the stereo flag,
// and the frame-count code. All accessors are pure functions of the raw byte.
type TOC struct {
	b uint8
}

// ParseTOC decodes a TOC byte. Parsing a TOC never fails: every one of the 256
// byte values is a valid TOC.
func ParseTOC(b byte) TOC { return TOC{b: b} }

// Byte returns the raw TOC byte.
func (t TOC) Byte() uint8 { return t.b }

// Config returns the 5-bit configuration number (0-31) from bits 3-7.
func (t TOC) Config() uint8 { return t.b >> 3 }

// Mode returns the coding mode selected by the configuration number.
func (t TOC) Mode() Mode {
	switch {
	case t.b&0x80 != 0:
		return ModeCELTOnly
	case t.b&0x60 == 0x60:
		return ModeHybrid
	default:
		return ModeSILKOnly
	}
}

// Bandwidth returns the audio bandwidth selected by the configuration number.
func (t TOC) Bandwidth() Bandwidth {
	switch {
	case t.b&0x80 != 0: // CELT-only, configs 16-31
		bw := BandwidthMediumband + Bandwidth((t.b>>5)&0x3)
		if bw == BandwidthMediumband {
			return BandwidthNarrowband
		}
		return bw
	case t.b&0x60 == 0x60: // hybrid, configs 12-15
		if t.b&0x10 != 0 {
			return BandwidthFullband
		}
		return BandwidthSuperwideband
	default: // SILK, configs 0-11
		return BandwidthNarrowband + Bandwidth((t.b>>5)&0x3)
	}
}

// FrameDuration returns the per-frame duration selected by the configuration
// number.
func (t TOC) FrameDuration() FrameDuration {
	switch samplesPerFrame(t.b, 48000) {
	case 120:
		return FrameDuration2500us
	case 240:
		return FrameDuration5ms
	case 480:
		return FrameDuration10ms
	case 960:
		return FrameDuration20ms
	case 1920:
		return FrameDuration40ms
	default: // 2880
		return FrameDuration60ms
	}
}

// Stereo reports whether the stereo flag (bit 2) is set.
func (t TOC) Stereo() bool { return t.b&0x4 != 0 }

// Channels returns the channel count encoded by the stereo flag: 2 if stereo,
// otherwise 1.
func (t TOC) Channels() int {
	if t.Stereo() {
		return 2
	}
	return 1
}

// FrameCountCode returns the 2-bit frame-count code c (bits 0-1): 0 for a single
// frame, 1 for two equal frames, 2 for two frames of unequal size, and 3 for an
// arbitrary frame count signalled by a following count byte.
func (t TOC) FrameCountCode() uint8 { return t.b & 0x3 }

// SamplesPerFrame returns the number of samples in a single frame at sample rate
// fs (in Hz). Multiply by the frame count to get the packet duration.
func (t TOC) SamplesPerFrame(fs int) int { return samplesPerFrame(t.b, fs) }

// samplesPerFrame returns the samples per frame at sample rate fs for the given
// TOC byte, mirroring opus_packet_get_samples_per_frame() in libopus.
func samplesPerFrame(toc byte, fs int) int {
	switch {
	case toc&0x80 != 0: // CELT-only
		shift := int((toc >> 3) & 0x3)
		return (fs << shift) / 400
	case toc&0x60 == 0x60: // hybrid
		if toc&0x08 != 0 {
			return fs / 50
		}
		return fs / 100
	default: // SILK
		shift := int((toc >> 3) & 0x3)
		if shift == 3 {
			return fs * 60 / 1000
		}
		return (fs << shift) / 100
	}
}
