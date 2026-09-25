package silk

// This file hand-writes the SILK table struct types that mirror libopus's
// codebook structs (silk/structs.h) for the frozen FIXED_POINT,
// DISABLE_FLOAT_API decoder build. The data itself (the two NLSF codebooks and
// every leaf table) is machine-generated into tables_gen.go by cmd/gentables;
// see docs/hard-parts.md section 6. Regenerate with `go generate ./internal/silk/`.
//
// Type mapping (silk/typedef.h): opus_int8 -> int8, opus_uint8 -> byte,
// opus_int16 -> int16, opus_int32 -> int32.

//go:generate go run github.com/sunholo-data/ailang/third_party/goopus/cmd/gentables -target silk -silk-out tables_gen.go

// silkNLSFCBStruct mirrors silk/structs.h silk_NLSF_CB_struct: the NLSF
// (normalized line spectral frequency) codebook the decoder uses to reconstruct
// the short-term LPC filter. Field order and names follow the C struct so the
// port stays diffable; the four scalar fields are const opus_int16 upstream. The
// pointer members become slices into the generated leaf tables.
type silkNLSFCBStruct struct {
	nVectors           int16
	order              int16
	quantStepSizeQ16   int16
	invQuantStepSizeQ6 int16
	cb1NLSFQ8          []byte  // CB1_NLSF_Q8:  nVectors*order first-stage codebook, Q8
	cb1WghtQ9          []int16 // CB1_Wght_Q9:  nVectors*order first-stage weights, Q9
	cb1ICDF            []byte  // CB1_iCDF:     first-stage inverse CDF (2 tables)
	predQ8             []byte  // pred_Q8:      second-stage backward-prediction weights, Q8
	ecSel              []byte  // ec_sel:       per-coefficient entropy-table selectors
	ecICDF             []byte  // ec_iCDF:      second-stage residual inverse CDFs
	ecRatesQ5          []byte  // ec_Rates_Q5:  second-stage residual rate table, Q5
	deltaMinQ15        []int16 // deltaMin_Q15: order+1 minimum NLSF spacing, Q15
}
