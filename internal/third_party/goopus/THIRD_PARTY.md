# Third-party provenance

go-opus is a native Go port of the Opus audio codec. Unlike a clean-room
reimplementation, the codec logic here is a deliberate transliteration of
libopus: the decoder is specified by RFC 6716, but the encoder is not specified
by any RFC, so libopus is effectively the specification and the only sane route
is a faithful port of it. go-opus is therefore a derivative work of libopus and
is licensed under the same 3-clause BSD license (see LICENSE).

## libopus (BSD 3-clause, transliteration reference)

- Upstream: https://gitlab.xiph.org/xiph/opus (GitHub mirror: xiph/opus).
- Transliteration reference pinned at tag **v1.6.1**. Doc line numbers in docs/
  cite commit 3da9f7a6 (v1.6.1+50); expect a few lines of drift against the tag.
- Every transliterated Go file keeps the upstream libopus BSD 3-clause copyright
  header. The frozen build configuration ported is FIXED_POINT +
  DISABLE_FLOAT_API, with no QEXT/RES24/dnn/CUSTOM_MODES.
- Opus is an IETF codec with royalty-free patent grants (see LICENSE for the
  IPR links). A reimplementation/port is clean with respect to those grants.
- The pinned libopus sources live under internal/reftest as a git submodule and
  are built only by the cgo differential-test harness (build tag `refc`); they
  are never compiled into the published pure-Go module.

## SIMD

- github.com/tphakala/simd (MIT, same author) is a runtime dependency. It backs
  the two hot pitch kernels (celt_inner_prod and xcorr_kernel) through
  i16.DotProduct and i16.XCorr, the radix-3/radix-5 FFT butterflies
  (kiss_fft.go) through the cint fixed-point complex ops (cint.Mul / Add / Sub /
  MulByScalar), the haar1 transform through the i32 kernels (i32.ScaleQ31 for the
  in-place Q31 scale, i32.Butterfly for the wide-stride butterfly), and the
  band-gain requant on both paths (normaliseBands / denormaliseBands) through
  i32.GainQ31, mirroring the go-flac scalar-reference-plus-SIMD structure. The
  library is pure Go plus its own per-arch assembly (SMLAL/SMLAL2 on arm64,
  PMADDWD/AVX2 on amd64) and has no cgo; it pulls golang.org/x/sys (indirect,
  also cgo-free) for CPU-feature detection. Each kernel keeps a scalar reference
  (internal/celt/pitch_ref.go for pitch, internal/celt/haar1_ref.go and
  internal/celt/bandgain_ref.go for the i32 kernels, the frozen *Scalar
  butterflies for the FFT) that the library results are asserted bit-exact
  against (internal/celt/pitch_simd_test.go, internal/celt/haar1_simd_test.go,
  internal/celt/fft_cint_oracle_test.go).

## Test corpus and conformance vectors

- RFC 6716 / RFC 8251 official Opus test vectors: fetched and sha256-verified by
  scripts/fetch-vectors.sh, used for testing only, never committed (see
  .gitignore and testdata/vectors.sha256). RFC 8251 is the archive that matters;
  it only revised the .dec reference outputs.
- Local corpus (bird recordings, speech, music, sine sweeps, silence, noise) is
  used by differential tests and is not committed.

## Ruleguard DSL

- github.com/quasilyte/go-ruleguard/dsl (BSD 3-clause): build-time-only
  dependency backing the custom gocritic ruleguard matchers in rules/. Anchored
  by tools/tools.go; never compiled into the module.
