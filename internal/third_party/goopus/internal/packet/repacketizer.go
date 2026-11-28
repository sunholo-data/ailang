package packet

// Repacketizer merges the frames of several equally-configured Opus packets, or
// splits one packet, into new packets. It transliterates the OpusRepacketizer
// API from libopus src/repacketizer.c (init, cat, out_range, out,
// get_nb_frames). Extension/DRED padding metadata handled by the C code is out
// of scope here; padding is treated as opaque filler.
//
// Typical use: init, Cat one or more input packets whose TOC bytes agree in the
// mode/bandwidth/duration/stereo fields (the low two frame-count bits may
// differ), then Out into a caller-provided buffer.
type Repacketizer struct {
	toc       byte
	nbFrames  int
	frames    [MaxFrames][]byte
	framesize int // samples per frame at 8 kHz, used for the 120 ms cap
}

// NewRepacketizer returns an initialised, empty Repacketizer.
func NewRepacketizer() *Repacketizer { return &Repacketizer{} }

// Init resets the repacketizer to empty so it can be reused. It mirrors
// opus_repacketizer_init().
func (rp *Repacketizer) Init() { rp.nbFrames = 0 }

// GetNbFrames returns the number of frames accumulated so far.
func (rp *Repacketizer) GetNbFrames() int { return rp.nbFrames }

// Cat appends the frames of data to the repacketizer. The first packet fixes
// the TOC; every subsequent packet must share the mode/bandwidth/duration/
// stereo configuration (TOC byte with the low two bits masked off). It returns
// ErrInvalidPacket on a configuration mismatch, a malformed packet, or when the
// combined duration would exceed 120 ms. Mirrors opus_repacketizer_cat().
func (rp *Repacketizer) Cat(data []byte) error {
	return rp.catImpl(data, false)
}

func (rp *Repacketizer) catImpl(data []byte, selfDelimited bool) error {
	if len(data) < 1 {
		return ErrInvalidPacket
	}
	if rp.nbFrames == 0 {
		rp.toc = data[0]
		rp.framesize = samplesPerFrame(data[0], 8000)
	} else if rp.toc&0xFC != data[0]&0xFC {
		return ErrInvalidPacket
	}

	// Parse the frame boundaries directly into rp.frames, allocation-free. The
	// general parse() allocates a fresh Packet.Frames slice per call; Cat is on the
	// encoder's 0-allocs/op multiframe path (opusenc.encodeMultiframe calls it once
	// per sub-frame), so it parses in place instead. This single run also subsumes
	// the separate CountFrames pass the previous body did before parsing.
	p := parser{data: data}
	if err := p.run(selfDelimited); err != nil {
		return err
	}
	if p.count < 1 {
		return ErrInvalidPacket
	}
	// Enforce the 120 ms (960 samples at 8 kHz) maximum across all frames.
	if (p.count+rp.nbFrames)*rp.framesize > 960 {
		return ErrInvalidPacket
	}
	if rp.nbFrames+p.count > MaxFrames {
		return ErrInvalidPacket
	}
	// pkt is a stack-local view; its Frames alias rp.frames[nbFrames:], so fillPacket
	// writes the sub-frame boundaries straight into rp's storage with no allocation.
	var pkt Packet
	pkt.Frames = rp.frames[rp.nbFrames : rp.nbFrames+p.count]
	if err := p.fillPacket(data, &pkt); err != nil {
		return err
	}
	rp.nbFrames += p.count
	return nil
}

// Out writes all accumulated frames as a single packet into data and returns the
// packet length, or ErrBufferTooSmall if data is too short. Mirrors
// opus_repacketizer_out().
func (rp *Repacketizer) Out(data []byte) (int, error) {
	return rp.outRangeImpl(0, rp.nbFrames, data, false, false)
}

// OutRange writes frames [begin, end) as a single packet into data and returns
// the packet length. begin and end are frame indices in [0, GetNbFrames()].
// Mirrors opus_repacketizer_out_range().
func (rp *Repacketizer) OutRange(begin, end int, data []byte) (int, error) {
	return rp.outRangeImpl(begin, end, data, false, false)
}

// Pad grows the packet in data[:length] in place to exactly newLen bytes, adding
// code-3 padding, and reports whether it succeeded. It transliterates
// opus_packet_pad() / opus_packet_pad_impl() (src/repacketizer.c:335-369) with no
// extensions, and data must have room for newLen bytes.
//
// The short-circuit at repacketizer.c:343 is load-bearing, not an optimization:
// when length == newLen it returns OPUS_OK having written NOTHING, so a CBR
// packet that already fills its budget is left byte-for-byte alone rather than
// being re-framed. The Opus encoder calls this on every CBR frame
// (opus_encoder.c:2646, apply_padding = !use_vbr), and it only does real work
// when cbr_bytes > 1276 forced max_data_bytes to be clamped below the requested
// size.
//
// Like the C, the payload is copied out before the repacketizer writes back into
// data, so the in-place move to the end of the buffer is safe.
func Pad(data []byte, length, newLen int) error {
	if length < 1 {
		return ErrBadArg
	}
	if length == newLen {
		return nil // repacketizer.c:343-344
	}
	if length > newLen {
		return ErrBadArg // repacketizer.c:345-346
	}
	if newLen > len(data) || length > len(data) {
		return ErrBufferTooSmall
	}
	cp := make([]byte, length)
	copy(cp, data[:length])

	var rp Repacketizer
	rp.Init()
	if err := rp.Cat(cp); err != nil {
		return err
	}
	n, err := rp.outRangeImpl(0, rp.nbFrames, data[:newLen], false, true)
	if err != nil {
		return err
	}
	if n <= 0 { // C: `if (ret > 0) return OPUS_OK; else return ret;`
		return ErrInternal
	}
	return nil
}

// outRangeImpl transliterates opus_repacketizer_out_range_impl() from libopus,
// without the extension handling. It selects frame-count code 0/1/2/3, writes
// the header and frame lengths, copies the frame payloads, and optionally pads
// to len(data).
func (rp *Repacketizer) outRangeImpl(begin, end int, data []byte, selfDelimited, pad bool) (int, error) {
	if begin < 0 || begin >= end || end > rp.nbFrames {
		return 0, ErrBadArg
	}
	frames := rp.frames[begin:end]
	count := len(frames)
	maxlen := len(data)

	ptr, totSize, err := rp.writeHeader(data, frames, selfDelimited, pad)
	if err != nil {
		return 0, err
	}

	// Self-delimited framing carries the last frame length explicitly.
	if selfDelimited {
		ptr += encodeSize(len(frames[count-1]), data[ptr:])
	}

	for _, f := range frames {
		copy(data[ptr:], f)
		ptr += len(f)
	}

	if pad { // fill any remaining space with zero padding bytes
		for ptr < maxlen {
			data[ptr] = 0
			ptr++
		}
	}
	return totSize, nil
}

// OutRangePadded assembles frames [begin, end) into data with an EXPLICIT maxlen and
// the encoder's pad flag, mirroring opus_repacketizer_out_range_impl (called from
// opus_encoder.c:1831 to assemble a multiframe packet). The exported OutRange / Out
// derive maxlen from len(data) and never pad; the Opus encoder needs both a maxlen
// that may be shorter than the caller buffer (repacketize_len) and the pad flag.
//
// maxlen must be in [0, len(data)]. Under pad the assembled packet is zero-filled to
// maxlen (as a code-3 packet) and the return value is maxlen; otherwise the return is
// the natural assembled length. selfDelimited framing is not used by the encoder.
func (rp *Repacketizer) OutRangePadded(begin, end int, data []byte, maxlen int, pad bool) (int, error) {
	if maxlen < 0 || maxlen > len(data) {
		return 0, ErrBadArg
	}
	return rp.outRangeImpl(begin, end, data[:maxlen], false, pad)
}

// writeHeader writes the TOC byte plus any count byte, padding descriptor, and
// frame-length fields, returning the write cursor positioned at the first frame
// payload and the running total size. It first tries code 0/1/2 for one or two
// frames, then promotes to code 3 when there are more than two frames or when
// padding must be added.
func (rp *Repacketizer) writeHeader(data []byte, frames [][]byte, selfDelimited, pad bool) (ptr, totSize int, err error) {
	count := len(frames)
	maxlen := len(data)
	totSize = selfDelimHeaderSize(frames, selfDelimited)

	switch count {
	case 1:
		// Code 0: a single frame.
		totSize += len(frames[0]) + 1
		if totSize > maxlen {
			return 0, 0, ErrBufferTooSmall
		}
		data[0] = rp.toc & 0xFC
		ptr = 1
	case 2:
		ptr, totSize, err = rp.writeTwoFrames(data, frames, totSize)
		if err != nil {
			return 0, 0, err
		}
	}

	if count > 2 || (pad && totSize < maxlen) {
		return rp.writeCode3(data, frames, selfDelimited, pad)
	}
	return ptr, totSize, nil
}

// writeTwoFrames writes a code 1 (two equal frames) or code 2 (two unequal
// frames) header.
func (rp *Repacketizer) writeTwoFrames(data []byte, frames [][]byte, totSize int) (ptr, newTot int, err error) {
	maxlen := len(data)
	if len(frames[1]) == len(frames[0]) {
		// Code 1: two frames of equal length.
		totSize += 2*len(frames[0]) + 1
		if totSize > maxlen {
			return 0, 0, ErrBufferTooSmall
		}
		data[0] = (rp.toc & 0xFC) | 0x1
		return 1, totSize, nil
	}
	// Code 2: two frames, the first length signalled explicitly.
	totSize += len(frames[0]) + len(frames[1]) + 2
	if len(frames[0]) >= 252 {
		totSize++
	}
	if totSize > maxlen {
		return 0, 0, ErrBufferTooSmall
	}
	data[0] = (rp.toc & 0xFC) | 0x2
	ptr = 1 + encodeSize(len(frames[0]), data[1:])
	return ptr, totSize, nil
}

// writeCode3 writes a code 3 header (count byte, optional padding descriptor,
// and VBR frame-length fields), returning the cursor at the first frame payload.
func (rp *Repacketizer) writeCode3(data []byte, frames [][]byte, selfDelimited, pad bool) (ptr, totSize int, err error) {
	count := len(frames)
	maxlen := len(data)
	totSize = selfDelimHeaderSize(frames, selfDelimited)

	vbr := false
	for i := 1; i < count; i++ {
		if len(frames[i]) != len(frames[0]) {
			vbr = true
			break
		}
	}

	if vbr {
		totSize += 2
		for i := range count - 1 {
			totSize += 1 + len(frames[i])
			if len(frames[i]) >= 252 {
				totSize++
			}
		}
		totSize += len(frames[count-1])
		if totSize > maxlen {
			return 0, 0, ErrBufferTooSmall
		}
		data[0] = (rp.toc & 0xFC) | 0x3
		data[1] = byte(count) | 0x80
	} else {
		totSize += count*len(frames[0]) + 2
		if totSize > maxlen {
			return 0, 0, ErrBufferTooSmall
		}
		data[0] = (rp.toc & 0xFC) | 0x3
		data[1] = byte(count)
	}
	ptr = 2

	padAmount := 0
	if pad {
		padAmount = maxlen - totSize
	}
	if padAmount != 0 {
		ptr, err = writePadding(data, ptr, totSize, padAmount)
		if err != nil {
			return 0, 0, err
		}
		totSize += padAmount
	}

	if vbr { // frame-length fields follow the padding descriptor
		for i := range count - 1 {
			ptr += encodeSize(len(frames[i]), data[ptr:])
		}
	}
	return ptr, totSize, nil
}

// writePadding sets the padding flag in the count byte and writes the padding
// length descriptor: a run of 0xFF bytes followed by a final remainder byte.
func writePadding(data []byte, ptr, totSize, padAmount int) (int, error) {
	maxlen := len(data)
	data[1] |= 0x40
	nb255s := (padAmount - 1) / 255
	if totSize+nb255s+1 > maxlen {
		return 0, ErrBufferTooSmall
	}
	for range nb255s {
		data[ptr] = 255
		ptr++
	}
	data[ptr] = byte(padAmount - 255*nb255s - 1)
	ptr++
	return ptr, nil
}

// selfDelimHeaderSize returns the number of extra header bytes self-delimited
// framing needs for the explicit last-frame length: 1, or 2 when that frame is
// at least 252 bytes.
func selfDelimHeaderSize(frames [][]byte, selfDelimited bool) int {
	if !selfDelimited {
		return 0
	}
	if len(frames[len(frames)-1]) >= 252 {
		return 2
	}
	return 1
}
