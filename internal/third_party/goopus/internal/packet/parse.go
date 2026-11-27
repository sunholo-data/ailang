package packet

// MaxFrames is the maximum number of frames an Opus packet can carry. A code 3
// packet at the shortest frame duration (2.5 ms) can hold up to 48 frames,
// which is the 120 ms ceiling (RFC 6716 §3.2.5).
const MaxFrames = 48

// Packet is the decoded structure of an Opus packet: its TOC byte, the byte
// slice of each frame, and any trailing padding. Frames and Padding alias the
// input buffer (no copy); they are capped with a three-index slice so appending
// to them cannot clobber neighbouring frames.
type Packet struct {
	// TOC is the decoded table-of-contents byte.
	TOC TOC
	// Frames holds one sub-slice per compressed frame, in order. A frame may be
	// empty (zero length) for DTX.
	Frames [][]byte
	// Padding holds the trailing padding bytes, or nil when there is none. The
	// bytes carry no audio and are ignored by the decoder.
	Padding []byte
	// Consumed is the number of input bytes this packet occupies, including
	// padding. For a normal packet this equals len(data); for a self-delimited
	// packet it may be smaller, and the remaining bytes belong to the next
	// packet in the stream.
	Consumed int
}

// Duration returns the packet duration in samples at sample rate fs, i.e. the
// number of frames times the samples per frame.
func (p *Packet) Duration(fs int) int {
	return len(p.Frames) * p.TOC.SamplesPerFrame(fs)
}

// Parse decodes a standard (non-self-delimited) Opus packet. It returns
// ErrInvalidPacket for any malformed input and never panics.
func Parse(data []byte) (*Packet, error) {
	return parse(data, false)
}

// ParseSelfDelimited decodes a self-delimited Opus packet, the framing used
// inside code 3 sub-packets, multistream packets, and the repacketizer. A
// self-delimited packet carries an explicit length for its last frame, so
// Consumed reports how far into data the packet extends and the caller can
// continue parsing the next packet from data[Consumed:].
func ParseSelfDelimited(data []byte) (*Packet, error) {
	return parse(data, true)
}

// ParseInto decodes a standard (non-self-delimited) Opus packet into
// caller-provided storage, allocating nothing: dst and frames are owned by the
// caller. On success dst.Frames is set to frames[:count], with each entry
// aliasing data (no copy), and dst.TOC / dst.Padding / dst.Consumed are filled.
// It is the zero-allocation form of Parse for the steady-state decode path (the
// multi-frame loop in internal/opusdec). frames must stay live for as long as the
// caller reads dst.Frames. On error dst is left in an unspecified partial state
// and must not be read. It returns ErrInvalidPacket for any malformed input and
// never panics.
func ParseInto(data []byte, dst *Packet, frames *[MaxFrames][]byte) error {
	p := parser{data: data}
	if err := p.run(false); err != nil {
		return err
	}
	dst.Frames = frames[:p.count]
	// Clear the tail so entries beyond this packet's count do not keep aliasing
	// an earlier packet's data. The frames array outlives the call (it lives on
	// the decoder), and without this a 2-frame packet after a 48-frame packet
	// would pin the old packet's backing buffer through the 46 stale entries.
	clear(frames[p.count:])
	return p.fillPacket(data, dst)
}

// parser transliterates opus_packet_parse_impl() from libopus src/opus.c. It
// carries the running parse state so each frame-count code can be handled by a
// small method instead of one deeply nested function.
type parser struct {
	data      []byte
	pos       int            // read cursor into data
	remaining int            // logically available bytes (may exclude padding)
	sizes     [MaxFrames]int // decoded frame lengths
	count     int            // number of frames
	cbr       bool           // constant bitrate (all frames equal length)
	lastSize  int            // length of the last (implicit) frame
	pad       int            // total padding data bytes
}

func parse(data []byte, selfDelimited bool) (*Packet, error) {
	p := parser{data: data}
	if err := p.run(selfDelimited); err != nil {
		return nil, err
	}
	pkt := &Packet{Frames: make([][]byte, p.count)}
	if err := p.fillPacket(data, pkt); err != nil {
		return nil, err
	}
	return pkt, nil
}

// fillPacket slices the decoded frames out of data into dst.Frames (which the
// caller has already sized to p.count) and sets dst.TOC / dst.Padding /
// dst.Consumed. It aliases data (no copy) and validates every frame length and the
// padding against the buffer bounds, returning ErrInvalidPacket on any overflow.
// Both the allocating Parse and the zero-allocation ParseInto funnel through here.
func (p *parser) fillPacket(data []byte, dst *Packet) error {
	pos := p.pos
	for i := range p.count {
		sz := p.sizes[i]
		if sz < 0 || pos+sz > len(data) {
			return ErrInvalidPacket
		}
		dst.Frames[i] = data[pos : pos+sz : pos+sz]
		pos += sz
	}

	var padding []byte
	if p.pad > 0 {
		if pos+p.pad > len(data) {
			return ErrInvalidPacket
		}
		padding = data[pos : pos+p.pad : pos+p.pad]
	}

	dst.TOC = TOC{b: data[0]}
	dst.Padding = padding
	dst.Consumed = pos + p.pad
	return nil
}

// run decodes the header and frame lengths, leaving p.pos at the first frame,
// p.count set, and p.sizes[0:count] populated.
func (p *parser) run(selfDelimited bool) error {
	if len(p.data) == 0 {
		return ErrInvalidPacket
	}
	toc := p.data[0]
	p.pos = 1
	p.remaining = len(p.data) - 1
	p.lastSize = p.remaining

	var err error
	switch toc & 0x3 {
	case 0: // one frame
		p.count = 1
	case 1: // two equal (CBR) frames
		err = p.parseCode1(selfDelimited)
	case 2: // two frames, first length signalled (VBR)
		err = p.parseCode2()
	default: // code 3: arbitrary frame count
		err = p.parseCode3(selfDelimited)
	}
	if err != nil {
		return err
	}

	if selfDelimited {
		return p.finishSelfDelimited()
	}
	return p.finishNormal()
}

func (p *parser) parseCode1(selfDelimited bool) error {
	p.count = 2
	p.cbr = true
	if selfDelimited {
		return nil
	}
	if p.remaining&0x1 != 0 {
		return ErrInvalidPacket
	}
	p.lastSize = p.remaining / 2
	p.sizes[0] = p.lastSize
	return nil
}

func (p *parser) parseCode2() error {
	p.count = 2
	nbytes, size := parseSize(p.data[p.pos:], p.remaining)
	p.remaining -= nbytes
	if size < 0 || size > p.remaining {
		return ErrInvalidPacket
	}
	p.sizes[0] = size
	p.pos += nbytes
	p.lastSize = p.remaining - size
	return nil
}

func (p *parser) parseCode3(selfDelimited bool) error {
	if p.remaining < 1 {
		return ErrInvalidPacket
	}
	ch := p.data[p.pos]
	p.pos++
	p.count = int(ch & 0x3F)
	framesize := samplesPerFrame(p.data[0], 48000)
	if p.count <= 0 || framesize*p.count > 5760 {
		return ErrInvalidPacket
	}
	p.remaining--

	if ch&0x40 != 0 { // padding flag
		if err := p.parseCode3Padding(); err != nil {
			return err
		}
	}
	if p.remaining < 0 {
		return ErrInvalidPacket
	}

	p.cbr = ch&0x80 == 0 // VBR flag is bit 7
	switch {
	case !p.cbr:
		return p.parseCode3VBR()
	case !selfDelimited:
		return p.parseCode3CBR()
	default:
		return nil // self-delimited CBR: sizes filled in finishSelfDelimited
	}
}

// parseCode3Padding consumes the run of padding-length bytes: a byte of 255
// means "254 padding bytes, continue", any smaller value ends the run.
func (p *parser) parseCode3Padding() error {
	for {
		if p.remaining <= 0 {
			return ErrInvalidPacket
		}
		pv := int(p.data[p.pos])
		p.pos++
		p.remaining--
		tmp := pv
		if pv == 255 {
			tmp = 254
		}
		p.remaining -= tmp
		p.pad += tmp
		if pv != 255 {
			return nil
		}
	}
}

func (p *parser) parseCode3VBR() error {
	p.lastSize = p.remaining
	for i := range p.count - 1 {
		nbytes, size := parseSize(p.data[p.pos:], p.remaining)
		p.remaining -= nbytes
		if size < 0 || size > p.remaining {
			return ErrInvalidPacket
		}
		p.sizes[i] = size
		p.pos += nbytes
		p.lastSize -= nbytes + size
	}
	if p.lastSize < 0 {
		return ErrInvalidPacket
	}
	return nil
}

func (p *parser) parseCode3CBR() error {
	p.lastSize = p.remaining / p.count
	if p.lastSize*p.count != p.remaining {
		return ErrInvalidPacket
	}
	for i := range p.count - 1 {
		p.sizes[i] = p.lastSize
	}
	return nil
}

// finishSelfDelimited reads the explicit length of the last frame that
// self-delimited framing appends after the header.
func (p *parser) finishSelfDelimited() error {
	nbytes, size := parseSize(p.data[p.pos:], p.remaining)
	p.remaining -= nbytes
	p.sizes[p.count-1] = size
	if size < 0 || size > p.remaining {
		return ErrInvalidPacket
	}
	p.pos += nbytes
	if p.cbr {
		if size*p.count > p.remaining {
			return ErrInvalidPacket
		}
		for i := range p.count - 1 {
			p.sizes[i] = size
		}
		return nil
	}
	if nbytes+size > p.lastSize {
		return ErrInvalidPacket
	}
	return nil
}

// finishNormal assigns the implicit last-frame length for non-self-delimited
// packets, rejecting a frame larger than the 1275-byte maximum.
func (p *parser) finishNormal() error {
	if p.lastSize > 1275 {
		return ErrInvalidPacket
	}
	p.sizes[p.count-1] = p.lastSize
	return nil
}

// CountFrames returns the number of frames in a packet without decoding the
// frame lengths, mirroring opus_packet_get_nb_frames() in libopus. It returns
// ErrBadArg for an empty packet and ErrInvalidPacket for a truncated code 3
// header.
func CountFrames(data []byte) (int, error) {
	if len(data) < 1 {
		return 0, ErrBadArg
	}
	switch data[0] & 0x3 {
	case 0:
		return 1, nil
	case 3:
		if len(data) < 2 {
			return 0, ErrInvalidPacket
		}
		return int(data[1] & 0x3F), nil
	default:
		return 2, nil
	}
}

// Samples returns the packet duration in samples per channel at sample rate fs,
// mirroring opus_packet_get_nb_samples() in libopus. It returns ErrInvalidPacket
// when the duration would exceed 120 ms.
func Samples(data []byte, fs int) (int, error) {
	count, err := CountFrames(data)
	if err != nil {
		return 0, err
	}
	samples := count * samplesPerFrame(data[0], fs)
	if samples*25 > fs*3 { // more than 120 ms
		return 0, ErrInvalidPacket
	}
	return samples, nil
}
