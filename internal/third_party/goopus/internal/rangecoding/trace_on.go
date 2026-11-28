//go:build ectrace

package rangecoding

import "fmt"

// ectrace build: the symbol trace records one entry per coded symbol into a
// package-global log. The log is intentionally global (not a coder field) so the
// coder struct stays byte-identical to the default build and value-copy /
// snapshot semantics are unchanged; during an RDO trial the trial symbols simply
// appear in the log too, which is what a bisection wants to see.
//
// This machinery is single-threaded by design (a debugging aid, not production
// state). Call TraceReset before an encode/decode and TraceRecords afterward.

// TraceRecord is one logged range-coder symbol.
type TraceRecord struct {
	Index    int       // running symbol index (0-based)
	Kind     traceKind // which primitive produced it
	A        uint32    // fl / val / s / ret depending on Kind
	B        uint32    // fh / logp / ftb / bits depending on Kind
	C        uint32    // ft (0 when unused)
	TellFrac uint32    // ec_tell_frac() after the symbol
}

var traceLog []TraceRecord

func (_this *Encoder) trace(kind traceKind, a, b, c uint32) {
	traceLog = append(traceLog, TraceRecord{
		Index: len(traceLog), Kind: kind, A: a, B: b, C: c, TellFrac: _this.TellFrac(),
	})
}

func (_this *Decoder) trace(kind traceKind, a, b, c uint32) {
	traceLog = append(traceLog, TraceRecord{
		Index: len(traceLog), Kind: kind, A: a, B: b, C: c, TellFrac: _this.TellFrac(),
	})
}

// TraceReset clears the global trace log.
func TraceReset() { traceLog = traceLog[:0] }

// TraceRecords returns the accumulated trace records (aliases the global log; do
// not retain across a TraceReset).
func TraceRecords() []TraceRecord { return traceLog }

// String names a traceKind for human-readable trace dumps.
func (k traceKind) String() string {
	switch k {
	case traceEncode:
		return "ec_encode"
	case traceEncodeBin:
		return "ec_encode_bin"
	case traceEncBitLogp:
		return "ec_enc_bit_logp"
	case traceEncIcdf:
		return "ec_enc_icdf"
	case traceEncBits:
		return "ec_enc_bits"
	case traceDecode:
		return "ec_dec_update"
	case traceDecBitLogp:
		return "ec_dec_bit_logp"
	case traceDecIcdf:
		return "ec_dec_icdf"
	case traceDecBits:
		return "ec_dec_bits"
	default:
		return fmt.Sprintf("traceKind(%d)", uint8(k))
	}
}

// String renders a TraceRecord as one trace line.
func (r TraceRecord) String() string {
	return fmt.Sprintf("#%d %-16s a=%d b=%d c=%d tell_frac=%d",
		r.Index, r.Kind.String(), r.A, r.B, r.C, r.TellFrac)
}
