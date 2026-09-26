package opusdec

import "errors"

// Sentinel errors returned by the exported opusdec surface, mapping from the
// libopus OPUS_* codes the internal (int-returning) transliteration uses. The
// public opus package re-wraps these to its own sentinels.
var (
	// ErrBadArg maps OPUS_BAD_ARG.
	ErrBadArg = errors.New("opusdec: bad argument")
	// ErrBufferTooSmall maps OPUS_BUFFER_TOO_SMALL.
	ErrBufferTooSmall = errors.New("opusdec: buffer too small")
	// ErrInvalidPacket maps OPUS_INVALID_PACKET.
	ErrInvalidPacket = errors.New("opusdec: invalid packet")
	// ErrInternal maps OPUS_INTERNAL_ERROR.
	ErrInternal = errors.New("opusdec: internal error")
)

// codeErr maps an internal negative Opus error code to its sentinel error. A
// non-negative code (including opusOK) is not an error and maps to ErrInternal
// defensively, since codeErr is only called on the error path.
func codeErr(code int) error {
	switch code {
	case opusBadArg:
		return ErrBadArg
	case opusBufferTooSmall:
		return ErrBufferTooSmall
	case opusInvalidPacket:
		return ErrInvalidPacket
	default: // opusInternalError and any unexpected code
		return ErrInternal
	}
}
