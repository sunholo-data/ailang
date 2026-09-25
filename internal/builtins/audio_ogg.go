package builtins

// Deterministic Ogg Opus (RFC 7845 over RFC 3533) writer for std/audio.encode.
//
// Why our own container rather than go-opus's oggopus.Encoder: upstream draws
// the Ogg logical-stream serial from rand.Uint32, so two encodes of the same
// PCM differ in the serial and in every page CRC. std/audio.encode is a pure
// function, so the serial here is a hash of the input. Everything else follows
// the upstream writer's accounting: a pre-skip of Encoder.PreSkip() samples at
// 48 kHz, a zero-padded final frame, extra silent frames until the coded audio
// covers pre-skip + source length, and an end-trimmed final granule.

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"

	"github.com/sunholo-data/ailang/third_party/goopus/opus"
)

const (
	oggOpusFrameMs     = 20
	oggOpusRate48k     = 48000
	oggOpusMaxPacket   = 1276 * 6 // libopus's hard ceiling on one Encode call
	oggPageMaxSegments = 255
	oggPageMaxPackets  = 50 // 50 x 20 ms = at most 1 s of audio per page
	oggOpusVendor      = "ailang std/audio (go-opus v1.1.0)"

	oggFlagBOS = 0x02
	oggFlagEOS = 0x04
)

// encodeOggOpus encodes interleaved s16le PCM to a complete Ogg Opus stream.
// The caller has validated rate, channels, bitrate and the PCM length.
func encodeOggOpus(pcm []byte, rate, channels, bitrate int) ([]byte, error) {
	enc, err := opus.NewEncoder(opus.EncoderConfig{SampleRate: rate, Channels: channels, Bitrate: bitrate})
	if err != nil {
		return nil, err
	}
	preSkip := int64(enc.PreSkip())
	frameLen := rate * oggOpusFrameMs / 1000 // samples per channel per frame
	frame48k := int64(frameLen * oggOpusRate48k / rate)
	bytesPerSample := 2 * channels
	srcSamples := int64(len(pcm) / bytesPerSample)
	src48k := srcSamples * oggOpusRate48k / int64(rate)

	ow := &oggWriter{serial: oggSerial(pcm, rate, channels, bitrate)}
	ow.writePacket(oggOpusHead(channels, uint16(preSkip), rate), 0, true)
	ow.writePacket(oggOpusTags(), 0, true)
	ow.flush(false) // OpusTags gets its own page; audio starts on a fresh one

	frame := make([]int16, frameLen*channels)
	buf := make([]byte, oggOpusMaxPacket)
	var coded48k int64
	var packets [][]byte
	encodeFrame := func() error {
		n, err := enc.Encode(frame, buf)
		if err != nil {
			return err
		}
		packets = append(packets, append([]byte(nil), buf[:n]...))
		coded48k += frame48k
		return nil
	}

	frameBytes := frameLen * bytesPerSample
	for off := 0; off < len(pcm); off += frameBytes {
		end := min(off+frameBytes, len(pcm))
		clear(frame)
		for i := 0; off+2*i < end; i++ {
			frame[i] = int16(binary.LittleEndian.Uint16(pcm[off+2*i:]))
		}
		if err := encodeFrame(); err != nil {
			return nil, err
		}
	}
	// RFC 7845 §4.5: the end-of-stream granule may not claim samples that were
	// never coded, so pad with silence until coded audio covers preSkip+src48k.
	// An empty input still gets one (fully trimmed) audio packet, since
	// ffmpeg-based readers refuse a header-only stream.
	clear(frame)
	for len(packets) == 0 || coded48k < preSkip+src48k {
		if err := encodeFrame(); err != nil {
			return nil, err
		}
	}

	var granule int64
	for i, p := range packets {
		granule += frame48k
		g := granule
		if i == len(packets)-1 {
			g = preSkip + src48k // end trim: decoders drop everything past this
		}
		ow.writePacket(p, g, false)
	}
	ow.flush(true)
	return ow.out, nil
}

// oggSerial derives the logical-stream serial from the whole input, so equal
// inputs give equal bytes and different clips still get distinct serials if
// someone concatenates them (the reason RFC 3533 wants serials to differ).
func oggSerial(pcm []byte, rate, channels, bitrate int) uint32 {
	h := fnv.New32a()
	var hdr [12]byte
	binary.LittleEndian.PutUint32(hdr[0:], uint32(rate))
	binary.LittleEndian.PutUint32(hdr[4:], uint32(channels))
	binary.LittleEndian.PutUint32(hdr[8:], uint32(bitrate))
	h.Write(hdr[:])
	h.Write(pcm)
	return h.Sum32()
}

// oggOpusHead is the RFC 7845 §5.1 identification header, mapping family 0.
func oggOpusHead(channels int, preSkip uint16, inputRate int) []byte {
	b := make([]byte, 19)
	copy(b, "OpusHead")
	b[8] = 1 // version
	b[9] = byte(channels)
	binary.LittleEndian.PutUint16(b[10:], preSkip)
	binary.LittleEndian.PutUint32(b[12:], uint32(inputRate))
	// b[16:18] output gain 0, b[18] mapping family 0
	return b
}

// oggOpusTags is the RFC 7845 §5.2 comment header with no user comments.
func oggOpusTags() []byte {
	b := make([]byte, 0, 16+len(oggOpusVendor))
	b = append(b, "OpusTags"...)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(oggOpusVendor)))
	b = append(b, oggOpusVendor...)
	return binary.LittleEndian.AppendUint32(b, 0)
}

// oggWriter packs whole packets into pages. Opus packets are at most 7,656
// bytes (31 lacing values), so a packet never has to span two pages.
type oggWriter struct {
	out     []byte
	serial  uint32
	seq     uint32
	segs    []byte
	body    []byte
	granule int64
	packets int
}

func (w *oggWriter) writePacket(p []byte, granule int64, ownPage bool) {
	lacing := len(p)/255 + 1
	if w.packets > 0 && (ownPage || len(w.segs)+lacing > oggPageMaxSegments || w.packets >= oggPageMaxPackets) {
		w.flush(false)
	}
	for n := len(p); ; n -= 255 {
		if n < 255 {
			w.segs = append(w.segs, byte(n))
			break
		}
		w.segs = append(w.segs, 255)
	}
	w.body = append(w.body, p...)
	w.granule = granule
	w.packets++
}

func (w *oggWriter) flush(eos bool) {
	if w.packets == 0 {
		return
	}
	var flags byte
	if w.seq == 0 {
		flags |= oggFlagBOS
	}
	if eos {
		flags |= oggFlagEOS
	}
	start := len(w.out)
	w.out = append(w.out, "OggS"...)
	w.out = append(w.out, 0, flags) // stream structure version 0
	w.out = binary.LittleEndian.AppendUint64(w.out, uint64(w.granule))
	w.out = binary.LittleEndian.AppendUint32(w.out, w.serial)
	w.out = binary.LittleEndian.AppendUint32(w.out, w.seq)
	w.out = append(w.out, 0, 0, 0, 0) // CRC, filled below
	w.out = append(w.out, byte(len(w.segs)))
	w.out = append(w.out, w.segs...)
	w.out = append(w.out, w.body...)
	binary.LittleEndian.PutUint32(w.out[start+22:], oggCRC(w.out[start:]))
	w.seq++
	w.segs, w.body, w.packets = w.segs[:0], w.body[:0], 0
}

// oggCRCTable is the RFC 3533 CRC-32: polynomial 0x04C11DB7, no reflection,
// zero initial value and no final XOR (so not hash/crc32's IEEE variant).
var oggCRCTable = func() (t [256]uint32) {
	for i := range t {
		r := uint32(i) << 24
		for range 8 {
			if r&0x80000000 != 0 {
				r = r<<1 ^ 0x04C11DB7
			} else {
				r <<= 1
			}
		}
		t[i] = r
	}
	return t
}()

func oggCRC(b []byte) uint32 {
	var crc uint32
	for _, x := range b {
		crc = crc<<8 ^ oggCRCTable[byte(crc>>24)^x]
	}
	return crc
}

// validateAudioEncode checks every argument and names the offending field.
// Nothing is clamped: an out-of-range value is the caller's bug to see.
func validateAudioEncode(pcmLen, rate, channels, bitrate int) error {
	switch rate {
	case 8000, 12000, 16000, 24000, 48000:
	default:
		return fmt.Errorf("rate %d Hz is not supported (want 8000, 12000, 16000, 24000 or 48000; resample first)", rate)
	}
	if channels != 1 && channels != 2 {
		return fmt.Errorf("channels %d is not supported (want 1 or 2)", channels)
	}
	if bitrate < audioMinBitrate || bitrate > audioMaxBitrate {
		return fmt.Errorf("bitrate %d is out of range (want %d..%d bits/s)", bitrate, audioMinBitrate, audioMaxBitrate)
	}
	if pcmLen > audioMaxPCMBytes {
		return fmt.Errorf("pcm length %d bytes exceeds the %d byte limit", pcmLen, audioMaxPCMBytes)
	}
	if pcmLen%(2*channels) != 0 {
		return fmt.Errorf("pcm length %d bytes is not a whole number of 16-bit %d-channel samples", pcmLen, channels)
	}
	return nil
}
