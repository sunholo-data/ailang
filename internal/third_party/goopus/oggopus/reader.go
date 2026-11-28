package oggopus

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

// pageInfo records the structural facts of one parsed page. It exists so tests
// (and future diagnostics) can assert granule progression and paging without
// re-decoding, the container-structure check the go-flac seekability regression
// argues for.
type pageInfo struct {
	granule   int64
	flags     headerType
	sequence  uint32
	packets   int  // packets that completed on this page
	continued bool // page begins with the tail of a prior packet
}

// containerReader parses an Ogg Opus stream from an io.Reader: it reassembles
// packets across page boundaries, exposes the OpusHead and OpusTags, and tracks
// the granule accounting needed for pre-skip and end-trim. It is
// codec-independent; turning packets into PCM is the decoder's job.
type containerReader struct {
	br *bufio.Reader

	serial    uint32
	serialSet bool
	head      opusHead
	tags      opusTags

	partial []byte   // bytes of a packet still spanning pages
	queue   [][]byte // completed packets from the current page
	qi      int      // index of the next packet to serve from queue

	lastGranule  int64 // granule of the most recent page that completed a packet
	finalGranule int64 // granule of the EOS page, or -1 if not yet seen
	sawEOS       bool

	pages []pageInfo
}

// newContainerReader reads and validates the OpusHead and OpusTags header
// packets, leaving the reader positioned at the first audio packet.
func newContainerReader(r io.Reader) (*containerReader, error) {
	cr := &containerReader{br: bufio.NewReader(r), finalGranule: -1}
	hp, err := cr.nextRawPacket()
	if err != nil {
		return nil, headerErr("OpusHead", err)
	}
	head, err := parseOpusHead(hp)
	if err != nil {
		return nil, err
	}
	tp, err := cr.nextRawPacket()
	if err != nil {
		return nil, headerErr("OpusTags", err)
	}
	tags, err := parseOpusTags(tp)
	if err != nil {
		return nil, err
	}
	cr.head = head
	cr.tags = tags
	return cr, nil
}

// nextPacket returns the next audio packet, or io.EOF when the stream ends.
func (cr *containerReader) nextPacket() ([]byte, error) {
	return cr.nextRawPacket()
}

// totalSamples reports the playable sample count per channel at 48 kHz derived
// from the granule accounting (final granule minus pre-skip). The boolean is
// false until the EOS page has been read, i.e. until nextPacket has returned
// io.EOF on a stream that carried an end-of-stream page.
func (cr *containerReader) totalSamples() (int64, bool) {
	if cr.finalGranule < 0 {
		return 0, false
	}
	return cr.finalGranule - int64(cr.head.preSkip), true
}

// nextRawPacket returns the next reassembled packet from the logical stream,
// reading pages as needed. It serves the OpusHead and OpusTags packets too;
// newContainerReader consumes those before returning.
func (cr *containerReader) nextRawPacket() ([]byte, error) {
	for {
		if cr.qi < len(cr.queue) {
			p := cr.queue[cr.qi]
			cr.qi++
			return p, nil
		}
		if cr.sawEOS {
			return nil, io.EOF
		}
		pg, err := readPage(cr.br)
		if err != nil {
			return nil, err // io.EOF here means a clean end with no EOS page
		}
		if err := cr.processPage(pg); err != nil {
			return nil, err
		}
	}
}

// processPage reassembles the packets on one page into the queue and updates the
// granule and end-of-stream accounting.
func (cr *containerReader) processPage(pg *page) error {
	if !cr.serialSet {
		cr.serial = pg.serial
		cr.serialSet = true
		if pg.flags&flagBOS == 0 {
			return errors.New("oggopus: first page missing BOS flag")
		}
	} else if pg.serial != cr.serial {
		return fmt.Errorf("oggopus: unexpected serial %#08x (chained or multiplexed streams are unsupported)", pg.serial)
	}

	// If this page does not continue a packet but one was left dangling, the
	// previous packet was truncated; drop it rather than fuse across a gap.
	if pg.flags&flagContinued == 0 && len(cr.partial) > 0 {
		cr.partial = cr.partial[:0]
	}

	cr.queue = cr.queue[:0]
	cr.qi = 0
	off := 0
	for _, l := range pg.segTable {
		cr.partial = append(cr.partial, pg.body[off:off+int(l)]...)
		off += int(l)
		if l < maxSegmentLen {
			pkt := make([]byte, len(cr.partial))
			copy(pkt, cr.partial)
			cr.queue = append(cr.queue, pkt)
			cr.partial = cr.partial[:0]
		}
	}

	if pg.granule != granuleNone {
		cr.lastGranule = pg.granule
	}
	if pg.flags&flagEOS != 0 {
		cr.sawEOS = true
		cr.finalGranule = cr.lastGranule
	}
	cr.pages = append(cr.pages, pageInfo{
		granule:   pg.granule,
		flags:     pg.flags,
		sequence:  pg.sequence,
		packets:   len(cr.queue),
		continued: pg.flags&flagContinued != 0,
	})
	return nil
}

// headerErr wraps an error from reading a header packet, mapping a premature EOF
// to a clearer "truncated" message.
func headerErr(which string, err error) error {
	if errors.Is(err, io.EOF) {
		return fmt.Errorf("oggopus: stream ended before %s", which)
	}
	return fmt.Errorf("oggopus: reading %s: %w", which, err)
}
