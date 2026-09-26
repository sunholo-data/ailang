package oggopus

import "errors"

// Sentinel errors returned by the oggopus package, testable with errors.Is.
var (
	// ErrInvalidConfig reports that a Config is not usable: an unsupported sample
	// rate or channel count, or an out-of-range bitrate or complexity.
	ErrInvalidConfig = errors.New("oggopus: invalid config")
	// ErrClosed reports an operation on an Encoder or Decoder after Close.
	ErrClosed = errors.New("oggopus: use of closed stream")
)

// errUninitialized guards the zero value. NewEncoder, Reset and NewDecoder each
// either return a fully constructed stream or an error, so an Encoder or Decoder
// reached through the documented API always has its codec; this is what a
// zero-value oggopus.Encoder{} or a pooled Encoder whose Reset failed reports
// instead of panicking or, worse, appending to the previous stream's container.
//
// It replaces the construction-phase errCodecNotWired sentinel, which the PCM
// entry points returned until the codec was wired to the seam in codec.go. It is
// unexported because it reports a programming error rather than a condition a
// caller can act on. In-package tests assert against it directly.
var errUninitialized = errors.New("oggopus: encoder or decoder is not initialized")

// errNilWriter and errNilReader report a nil sink or source passed to the public
// constructors. They are plain errors, not exported sentinels: a nil io.Writer or
// io.Reader is a programming error, not a condition a caller branches on (matching
// go-flac's and go-aac's pcm packages). They replace the panic that the first
// header-page write (nil io.Writer) or the first container read (nil io.Reader)
// would otherwise raise, so a caller that passes nil gets a clean error instead.
var (
	errNilWriter = errors.New("oggopus: nil writer")
	errNilReader = errors.New("oggopus: nil reader")
)
