package oggopus

import "io"

// streamWriter accumulates packets into Ogg pages and writes them to an
// io.Writer. It holds exactly one page in progress (a one-page lookahead) and
// commits it only when the next page begins or when close is called, so the
// final page can always be flagged end-of-stream without seeking. This is the
// structural reason oggopus never needs an io.WriteSeeker: every field a page
// carries (including its granule position) is known before the page is written.
type streamWriter struct {
	w      io.Writer
	serial uint32
	seq    uint32

	// Pending page in progress.
	seg      []byte // segment (lacing) table
	body     []byte // concatenated segment payloads
	gran     int64  // granule to stamp; granuleNone until a packet completes here
	isCont   bool   // pending page begins with the tail of a prior packet
	havePage bool   // a pending page has been started

	audioStarted bool
	closed       bool
}

// newStreamWriter returns a streamWriter bound to w with the given serial
// number. No bytes are written until the first packet is placed.
func newStreamWriter(w io.Writer, serial uint32) *streamWriter {
	return &streamWriter{w: w, serial: serial, gran: granuleNone}
}

// writeHeaders places the OpusHead and OpusTags packets, each forced onto its
// own page so OpusHead is alone on the first (BOS) page as RFC 7845 requires.
// The OpusTags page is left pending; it commits when the first audio packet or
// close arrives.
func (sw *streamWriter) writeHeaders(head opusHead, tags opusTags) error {
	if err := sw.writePacket(head.marshal(), 0, true); err != nil {
		return err
	}
	return sw.writePacket(tags.marshal(), 0, true)
}

// writeAudioPacket places one audio packet. granuleAfter is the cumulative
// 48 kHz sample count (including pre-skip) through this packet, or the
// end-trimmed value for the final packet. The first audio packet starts a fresh
// page, committing the pending OpusTags page.
func (sw *streamWriter) writeAudioPacket(data []byte, granuleAfter int64) error {
	forceNewPage := !sw.audioStarted
	sw.audioStarted = true
	return sw.writePacket(data, granuleAfter, forceNewPage)
}

// close commits the final pending page with the end-of-stream flag. It is
// idempotent.
func (sw *streamWriter) close() error {
	if sw.closed {
		return nil
	}
	sw.closed = true
	return sw.commit(true)
}

// writePacket appends one packet, splitting it across pages when the segment
// table fills. When forceNewPage is set, any pending page is committed first so
// the packet begins a fresh page.
func (sw *streamWriter) writePacket(data []byte, granuleAfter int64, forceNewPage bool) error {
	if forceNewPage && sw.havePage {
		if err := sw.commit(false); err != nil {
			return err
		}
	}
	segs := lacing(len(data))
	off := 0
	for i, s := range segs {
		if sw.havePage && len(sw.seg) == maxPageSegments {
			// The pending page is full and more of this packet follows; commit
			// it and continue the packet on the next page.
			if err := sw.commit(false); err != nil {
				return err
			}
		}
		sw.havePage = true
		sw.seg = append(sw.seg, s)
		sw.body = append(sw.body, data[off:off+int(s)]...)
		off += int(s)
		if i == len(segs)-1 {
			// The packet completes on the pending page: stamp its granule.
			sw.gran = granuleAfter
		}
	}
	return nil
}

// commit writes the pending page with the given eos flag and resets the page
// buffers for reuse. The BOS flag is derived from the sequence number and the
// CONTINUED flag from whether the previous page ended mid-packet, so no caller
// has to track them.
func (sw *streamWriter) commit(eos bool) error {
	if !sw.havePage {
		return nil
	}
	var flags headerType
	if sw.seq == 0 {
		flags |= flagBOS
	}
	if sw.isCont {
		flags |= flagContinued
	}
	if eos {
		flags |= flagEOS
	}
	if err := writePage(sw.w, flags, sw.gran, sw.serial, sw.seq, sw.seg, sw.body); err != nil {
		return err
	}
	// A page whose last lacing value is 255 ends mid-packet, so the next page's
	// first packet is a continuation.
	sw.isCont = sw.seg[len(sw.seg)-1] == maxSegmentLen
	sw.seq++
	sw.seg = sw.seg[:0]
	sw.body = sw.body[:0]
	sw.gran = granuleNone
	sw.havePage = false
	return nil
}

// containerWriter is the codec-independent packet-level Ogg Opus writer. Given
// opus packets and their 48 kHz durations plus the stream parameters, it emits a
// valid stream and keeps the granule accounting correct on a plain io.Writer.
// It holds one packet back so the last page can be end-trimmed at close.
type containerWriter struct {
	sw      *streamWriter
	preSkip int64

	granule    int64 // cumulative 48 kHz samples through all flushed packets
	heldPacket []byte
	heldDur    int64
	hasHeld    bool
	audioCount int
}

// newContainerWriter writes the OpusHead and OpusTags header pages to w and
// returns a writer ready to accept audio packets. serial is the logical
// bitstream serial number.
func newContainerWriter(w io.Writer, serial uint32, head opusHead, tags opusTags) (*containerWriter, error) {
	sw := newStreamWriter(w, serial)
	if err := sw.writeHeaders(head, tags); err != nil {
		return nil, err
	}
	return &containerWriter{sw: sw, preSkip: int64(head.preSkip)}, nil
}

// writePacket queues one opus packet whose decoded duration is samples48k (at
// 48 kHz). The packet bytes are copied, so the caller may reuse the slice. The
// packet is held back one step so close can end-trim the final page.
func (cw *containerWriter) writePacket(pkt []byte, samples48k int) error {
	if cw.hasHeld {
		cw.granule += cw.heldDur
		if err := cw.sw.writeAudioPacket(cw.heldPacket, cw.granule); err != nil {
			return err
		}
	}
	cw.heldPacket = append(cw.heldPacket[:0], pkt...)
	cw.heldDur = int64(samples48k)
	cw.hasHeld = true
	cw.audioCount++
	return nil
}

// close flushes the final held packet with an end-trimmed granule position so
// that finalGranule - preSkip equals sourceSamples (the exact source sample
// count per channel at 48 kHz), then commits the last page with the
// end-of-stream flag. close is idempotent.
func (cw *containerWriter) close(sourceSamples int64) error {
	if cw.hasHeld {
		finalGranule := cw.preSkip + sourceSamples
		// Guard monotonicity: the final granule must not regress below the
		// previous page's cumulative count. This never triggers for consistent
		// inputs; it defends against a caller passing a sourceSamples smaller
		// than the already-committed audio.
		if finalGranule < cw.granule {
			finalGranule = cw.granule
		}
		if err := cw.sw.writeAudioPacket(cw.heldPacket, finalGranule); err != nil {
			return err
		}
		cw.hasHeld = false
	}
	return cw.sw.close()
}
