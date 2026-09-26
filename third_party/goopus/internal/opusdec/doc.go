// Package opusdec is a Go port of libopus src/opus_decoder.c (v1.6.1): the
// top-level Opus decoder that unifies the CELT and SILK decoders behind the
// mode-transition and redundancy state machine (RFC 6716, the "hard 20%"). It
// preserves C names and provenance references to the pinned libopus source and
// is held bit-exact against it by the refc differential gate; bit-exactness of
// the output is the sole correctness contract.
//
// The frozen build config is FIXED_POINT + DISABLE_FLOAT_API with no
// QEXT/RES24/DRED/DEEP_PLC/OSCE, so opus_res is opus_int16: the API output is
// int16 throughout and RES2INT16 / INT16TORES are the identity. The float API,
// soft-clip, DRED and deep-PLC branches of opus_decoder.c compile away and are
// omitted here.
//
// opusDecodeFrame is the transition machine: it routes each frame to CELT-only,
// SILK-only or hybrid decode, decodes the 5 ms CELT redundancy frames at
// SILK<->CELT and hybrid boundaries with their smooth crossovers, runs the mode
// transition crossfades, and drives the opus-level PLC (data==NULL) and FEC/LBRR
// (decode_fec) paths. opusDecodeNative is the packet parse + multi-frame loop +
// FEC entry. The public opus package wraps this internal core.
package opusdec
