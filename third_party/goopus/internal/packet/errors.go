package packet

import "errors"

// Sentinel errors returned by the packet layer. They mirror the libopus error
// codes so the public opus package can map them one-to-one:
//
//	OPUS_BAD_ARG          -> ErrBadArg
//	OPUS_INVALID_PACKET   -> ErrInvalidPacket
//	OPUS_BUFFER_TOO_SMALL -> ErrBufferTooSmall
//	OPUS_INTERNAL_ERROR   -> ErrInternal
//
// The parser never panics on hostile input; malformed packets return
// ErrInvalidPacket (fuzz invariant).
var (
	// ErrBadArg reports an invalid argument, such as a nil or negative-length
	// buffer where a packet was expected.
	ErrBadArg = errors.New("opus/packet: bad argument")
	// ErrInvalidPacket reports that the byte stream is not a well-formed Opus
	// packet: a truncated header, an inconsistent frame length, or a frame
	// count that exceeds the 120 ms limit.
	ErrInvalidPacket = errors.New("opus/packet: invalid packet")
	// ErrBufferTooSmall reports that a destination buffer cannot hold the
	// requested repacketized output.
	ErrBufferTooSmall = errors.New("opus/packet: buffer too small")
	// ErrInternal reports an internal invariant violation.
	ErrInternal = errors.New("opus/packet: internal error")
)
