package rangecoding

// Symbol-trace mode (docs/hard-parts.md 7, plan test-strategy 4).
//
// Each range-coder primitive calls _this.trace(...) at its exit. In the default
// build the trace method is an empty, inlined no-op (trace_noop.go), so it costs
// nothing. Building with `-tags ectrace` swaps in the real implementation
// (trace_on.go), which appends one TraceRecord per symbol (index, kind, the
// (fl,fh,ft)/logp/etc. arguments, and tell_frac after the symbol). Combined with
// docs/celt-bitstream.md this turns a "packet byte N differs" report into a named
// symbol/stage for phase-4 bisection.
//
// traceKind and its constants are compiled in every build so the primitives can
// name their symbol; only the recording machinery is behind the build tag.

// traceKind identifies which range-coder primitive produced a trace record.
type traceKind uint8

const (
	traceEncode     traceKind = iota // ec_encode(fl, fh, ft)
	traceEncodeBin                   // ec_encode_bin(fl, fh, bits)
	traceEncBitLogp                  // ec_enc_bit_logp(val, logp)
	traceEncIcdf                     // ec_enc_icdf(s, _, ftb)
	traceEncBits                     // ec_enc_bits(fl, bits)
	traceDecode                      // ec_dec_update(fl, fh, ft)
	traceDecBitLogp                  // ec_dec_bit_logp(ret, logp)
	traceDecIcdf                     // ec_dec_icdf(s, ftb)
	traceDecBits                     // ec_dec_bits(ret, bits)
)
