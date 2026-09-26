package oggopus

import (
	"fmt"

	"github.com/sunholo-data/ailang/third_party/goopus/opus"
)

// libVersion is the go-opus version reported in the default OpusTags vendor
// string. It DERIVES from opus.Version rather than restating it: the two used to be
// independent "0.1.0-dev" literals, which is a version string that can go stale in
// exactly one of the two places a consumer reads it, and nothing in a packet or a
// container byte would say so (the vendor string is free-form text, so no gate can
// tell a stale one from a correct one). Now the container cannot claim a version the
// codec does not. TestVendorStringDerivesFromOpusVersion is the guard on the
// FORMAT ("go-opus <version>"); the equality itself is enforced by the compiler.
const libVersion = opus.Version

// maxComplexity is the top of the Opus complexity range, and defaultComplexity is
// what Config.Complexity's zero value selects. The zero value MUST mean the
// default rather than an explicit complexity 0, so the useful range is 1..10 and
// 0 is the "unset" marker; the mapping itself is opus.EncoderConfig's, which owns
// it for both packages. defaultComplexity is duplicated here only to name the
// number in Config's documentation and in the validation error, and
// TestComplexityZeroMeansDefault pins the two against each other by comparing the
// packets they produce, so they cannot drift apart silently.
const (
	maxComplexity     = 10
	defaultComplexity = 10
)

// validSampleRates enumerates the sample rates Opus accepts. Anything else is
// rejected rather than clamped (unlike go-flac, which accepts a wide range).
var validSampleRates = [...]int{8000, 12000, 16000, 24000, 48000}

// defaultFrameDurationMS is the frame duration Config.FrameDurationMS's zero
// value selects, and validFrameDurationsMS enumerates the durations the encoder
// accepts. The set is restricted to multiples of 20 ms: every allowed
// rate/duration pair yields an integer samples-per-frame (rates are multiples
// of 4000, durations of 20, so SampleRate*ms/1000 is exact), and each frame
// stays at least 960 samples at 48 kHz, which the Close end-padding relies on
// (see encoder.go). The shorter 2.5, 5 and 10 ms Opus durations are deliberately
// omitted: 2.5 ms is fractional and does not fit an integer-ms field, and the
// sub-20 ms durations only add container overhead for a file encoder.
const defaultFrameDurationMS = 20

var validFrameDurationsMS = [...]int{20, 40, 60, 80, 100, 120}

// Config configures an oggopus Encoder. It is a flat struct mirroring go-flac's
// pcm.Config convention: no embedding, and every field's zero value is
// documented so a literal reads cleanly. See docs/api-design.md.
type Config struct {
	SampleRate int // 8000, 12000, 16000, 24000, or 48000; required (no zero default)
	Channels   int // 1 or 2; required (no zero default)

	Bitrate        int  // bits per second; zero selects automatic
	CBR            bool // zero value (false) means VBR
	ConstrainedVBR bool // meaningful only when CBR is false
	Complexity     int  // 1..10; zero selects the library default (10)

	// FrameDurationMS is the Opus frame duration in milliseconds: 20, 40, 60,
	// 80, 100, or 120. The zero value selects the default (20). A duration above
	// 20 ms is carried as one longer (multiframe) Opus packet per Ogg packet,
	// which lowers the per-packet container overhead at the cost of a coarser
	// end-of-stream trim. 20 ms is the opusenc default and the best
	// quality/overhead balance for a general file encoder.
	FrameDurationMS int

	// DTX requests discontinuous transmission. When enabled, long runs of exact
	// digital silence are coded as 1-byte TOC-only packets past a ~200 ms onset,
	// shrinking silent stretches; the granulepos and duration stay correct across
	// the gaps, so a decoder reconstructs full-length audio. The zero value
	// (false) is off. See opus.EncoderConfig.DTX.
	DTX bool

	// Vendor overrides the OpusTags vendor string; zero value uses
	// "go-opus <version>".
	Vendor string
	// Comments are OpusTags user comments in "TAG=value" order; nil emits tags
	// with only the vendor string.
	Comments []string
}

// validate checks the fields that have a hard domain. It leaves the tuning
// fields (bitrate/complexity zero-defaulting, CBR/VBR, DTX) for the codec, which
// owns their semantics. The receiver is a pointer to avoid copying the flat
// Config; the public entry points still take it by value (go-flac-aligned API).
func (c *Config) validate() error {
	if !validSampleRate(c.SampleRate) {
		return fmt.Errorf("%w: sample rate %d (want 8000, 12000, 16000, 24000, or 48000)", ErrInvalidConfig, c.SampleRate)
	}
	if c.Channels != 1 && c.Channels != 2 {
		return fmt.Errorf("%w: channels %d (want 1 or 2)", ErrInvalidConfig, c.Channels)
	}
	if c.Bitrate < 0 {
		return fmt.Errorf("%w: negative bitrate %d", ErrInvalidConfig, c.Bitrate)
	}
	if c.Complexity < 0 || c.Complexity > maxComplexity {
		return fmt.Errorf("%w: complexity %d (want 1..%d, or 0 for the default %d)",
			ErrInvalidConfig, c.Complexity, maxComplexity, defaultComplexity)
	}
	if c.FrameDurationMS != 0 && !validFrameDuration(c.FrameDurationMS) {
		return fmt.Errorf("%w: frame duration %d ms (want 20, 40, 60, 80, 100, or 120, or 0 for the default %d)",
			ErrInvalidConfig, c.FrameDurationMS, defaultFrameDurationMS)
	}
	return nil
}

// frameDurationMS returns the effective frame duration in milliseconds, mapping
// the zero value to the default so the encoder reads it in one place.
func (c *Config) frameDurationMS() int {
	if c.FrameDurationMS == 0 {
		return defaultFrameDurationMS
	}
	return c.FrameDurationMS
}

// vendorString returns the vendor string to write into OpusTags.
func (c *Config) vendorString() string {
	if c.Vendor != "" {
		return c.Vendor
	}
	return "go-opus " + libVersion
}

func validSampleRate(r int) bool {
	for _, v := range validSampleRates {
		if r == v {
			return true
		}
	}
	return false
}

func validFrameDuration(ms int) bool {
	for _, v := range validFrameDurationsMS {
		if ms == v {
			return true
		}
	}
	return false
}
