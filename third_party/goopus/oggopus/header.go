package oggopus

import (
	"encoding/binary"
	"fmt"
)

// RFC 7845 identification and comment header magic signatures.
const (
	opusHeadMagic = "OpusHead"
	opusTagsMagic = "OpusTags"
	// opusHeadFamily0Len is the exact OpusHead size for channel mapping family
	// 0 (no mapping table): magic (8) + version (1) + channels (1) + pre-skip
	// (2) + input sample rate (4) + output gain (2) + mapping family (1).
	opusHeadFamily0Len = 19
	// opusHeadVersion is the RFC 7845 identification header version go-opus emits.
	opusHeadVersion = 1
	// mappingFamily0 is the mono/stereo mapping family: no mapping table, 1 or 2
	// channels only.
	mappingFamily0 = 0
)

// opusHead is the parsed RFC 7845 identification header. go-opus emits channel
// mapping family 0 only; the parser additionally accepts family > 0 headers so
// it does not choke on third-party streams, retaining the mapping table for
// callers that inspect it.
type opusHead struct {
	version         byte
	channels        byte
	preSkip         uint16
	inputSampleRate uint32
	outputGain      int16
	mappingFamily   byte

	// Present only when mappingFamily != 0.
	streamCount    byte
	coupledCount   byte
	channelMapping []byte
}

// marshal serialises h as an OpusHead packet. v1 only emits family 0, so the
// output is always opusHeadFamily0Len bytes.
func (h opusHead) marshal() []byte {
	b := make([]byte, opusHeadFamily0Len)
	copy(b, opusHeadMagic)
	b[8] = h.version
	b[9] = h.channels
	binary.LittleEndian.PutUint16(b[10:12], h.preSkip)
	binary.LittleEndian.PutUint32(b[12:16], h.inputSampleRate)
	binary.LittleEndian.PutUint16(b[16:18], uint16(h.outputGain))
	b[18] = h.mappingFamily
	return b
}

// parseOpusHead decodes an OpusHead packet. It validates the magic, a supported
// major version, and channel/mapping consistency, and never reads out of bounds
// on a truncated or hostile packet.
func parseOpusHead(b []byte) (opusHead, error) {
	if len(b) < opusHeadFamily0Len {
		return opusHead{}, fmt.Errorf("oggopus: OpusHead too short (%d bytes)", len(b))
	}
	if string(b[0:8]) != opusHeadMagic {
		return opusHead{}, fmt.Errorf("oggopus: bad OpusHead magic %q", b[0:8])
	}
	h := opusHead{
		version:         b[8],
		channels:        b[9],
		preSkip:         binary.LittleEndian.Uint16(b[10:12]),
		inputSampleRate: binary.LittleEndian.Uint32(b[12:16]),
		outputGain:      int16(binary.LittleEndian.Uint16(b[16:18])),
		mappingFamily:   b[18],
	}
	// The major version lives in the top four bits; a nonzero major version is a
	// stream this decoder cannot interpret (RFC 7845 section 5.1).
	if h.version&0xf0 != 0 {
		return opusHead{}, fmt.Errorf("oggopus: unsupported OpusHead major version %d", h.version>>4)
	}
	if h.channels == 0 {
		return opusHead{}, fmt.Errorf("oggopus: OpusHead channel count is zero")
	}
	if h.mappingFamily == mappingFamily0 {
		if h.channels > 2 {
			return opusHead{}, fmt.Errorf("oggopus: mapping family 0 requires 1 or 2 channels, got %d", h.channels)
		}
		return h, nil
	}
	// Family > 0 carries a mapping table: stream count, coupled count, and one
	// mapping byte per channel.
	if len(b) < opusHeadFamily0Len+2+int(h.channels) {
		return opusHead{}, fmt.Errorf("oggopus: OpusHead mapping table truncated for family %d", h.mappingFamily)
	}
	h.streamCount = b[19]
	h.coupledCount = b[20]
	h.channelMapping = make([]byte, h.channels)
	copy(h.channelMapping, b[21:21+int(h.channels)])
	return h, nil
}

// opusTags is the parsed RFC 7845 comment header: a vendor string and an ordered
// list of "TAG=value" user comments.
type opusTags struct {
	vendor   string
	comments []string
}

// marshal serialises t as an OpusTags packet.
func (t opusTags) marshal() []byte {
	size := len(opusTagsMagic) + 4 + len(t.vendor) + 4
	for _, c := range t.comments {
		size += 4 + len(c)
	}
	b := make([]byte, 0, size)
	b = append(b, opusTagsMagic...)
	b = appendUint32(b, uint32(len(t.vendor)))
	b = append(b, t.vendor...)
	b = appendUint32(b, uint32(len(t.comments)))
	for _, c := range t.comments {
		b = appendUint32(b, uint32(len(c)))
		b = append(b, c...)
	}
	return b
}

// parseOpusTags decodes an OpusTags packet. Every length field is bounds-checked
// against the remaining input so a corrupt count cannot trigger an out-of-range
// read; trailing bytes after the declared comment list are ignored (RFC 7845
// permits padding, and unlike Vorbis there is no framing bit).
func parseOpusTags(b []byte) (opusTags, error) {
	const magicLen = 8
	if len(b) < magicLen+4 {
		return opusTags{}, fmt.Errorf("oggopus: OpusTags too short (%d bytes)", len(b))
	}
	if string(b[0:magicLen]) != opusTagsMagic {
		return opusTags{}, fmt.Errorf("oggopus: bad OpusTags magic %q", b[0:magicLen])
	}
	pos := magicLen

	vendorLen := int(binary.LittleEndian.Uint32(b[pos : pos+4]))
	pos += 4
	if vendorLen < 0 || pos+vendorLen > len(b) {
		return opusTags{}, fmt.Errorf("oggopus: OpusTags vendor length %d overruns packet", vendorLen)
	}
	t := opusTags{vendor: string(b[pos : pos+vendorLen])}
	pos += vendorLen

	if pos+4 > len(b) {
		return opusTags{}, fmt.Errorf("oggopus: OpusTags missing comment count")
	}
	count := int(binary.LittleEndian.Uint32(b[pos : pos+4]))
	pos += 4
	if count < 0 {
		return opusTags{}, fmt.Errorf("oggopus: OpusTags comment count %d invalid", count)
	}
	t.comments = make([]string, 0, min(count, 1024))
	for i := range count {
		if pos+4 > len(b) {
			return opusTags{}, fmt.Errorf("oggopus: OpusTags comment %d length truncated", i)
		}
		clen := int(binary.LittleEndian.Uint32(b[pos : pos+4]))
		pos += 4
		if clen < 0 || pos+clen > len(b) {
			return opusTags{}, fmt.Errorf("oggopus: OpusTags comment %d length %d overruns packet", i, clen)
		}
		t.comments = append(t.comments, string(b[pos:pos+clen]))
		pos += clen
	}
	return t, nil
}

// appendUint32 appends v to b in little-endian order.
func appendUint32(b []byte, v uint32) []byte {
	return append(b, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}
