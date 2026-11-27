package opusenc

import "errors"

// Sentinel errors returned by the exported opusenc surface, mapping from the
// libopus OPUS_* codes the internal (int-returning) transliteration uses. The
// public opus package re-wraps these to its own sentinels. Mirrors
// internal/opusdec/errors.go.
var (
	// ErrBadArg maps OPUS_BAD_ARG.
	ErrBadArg = errors.New("opusenc: bad argument")
	// ErrBufferTooSmall maps OPUS_BUFFER_TOO_SMALL.
	ErrBufferTooSmall = errors.New("opusenc: buffer too small")
	// ErrInternal maps OPUS_INTERNAL_ERROR.
	ErrInternal = errors.New("opusenc: internal error")
	// ErrUnimplemented maps OPUS_UNIMPLEMENTED. This port returns it for the
	// OPUS_AUTO mode decision (:1473, which needs compute_stereo_width) and for the
	// SILK-only multiframe sub-split (40/60 ms sub-frames), neither of which is
	// ported. libopus itself SUPPORTS both; returning an error here is a deliberate
	// divergence, because the alternative is a silently wrong packet. (Frames longer
	// than 20 ms in CELT-only ARE coded, through encodeMultiframe.)
	ErrUnimplemented = errors.New("opusenc: unimplemented in this build")
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
	case opusUnimplemented:
		return ErrUnimplemented
	default: // opusInternalError and any unexpected code
		return ErrInternal
	}
}
