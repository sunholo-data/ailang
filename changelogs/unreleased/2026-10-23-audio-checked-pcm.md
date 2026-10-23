### Added — checked WAV metadata and packed PCM16 conversion (2026-10-23)

`std/audio` adds `wavInfo`, `wavHeaderChecked`, `wavFromPcmChecked`,
`pcm16ToFloats` and `floatsToPcm16`. Metadata validates RIFF/chunk extents,
padding, format fields and whole frames; integer PCM 8/16/24/32 is supported,
while float/extensible encodings return explicit errors. Packed s16le conversion
preserves interleaved channels, uses /32768 decoding and explicit nearest-even
quantization with reject-or-clip policies; nonfinite samples always fail.
Materialized PCM extents are capped at 100 MiB. Resampling remains deferred.

### Fixed — WAV padding and ordinary writer validation (2026-10-23)

Odd PCM data now receives its RIFF-required pad, with correct outer size and an
unchanged unpadded data-chunk size. Legacy WAV wrappers keep their signatures
but reject partial frames, unsupported bit depths, field overflow and oversized
buffers without needing contract flags. Valid even-length PCM16 headers remain
byte-identical. Generated Go supports the same checked audio boundaries and
byte operations; bytes use native slice storage and decoded arrays stay packed.

### Fixed — formatter preserves single-constructor enum identity (2026-10-23)

Formatting a nullary single-constructor sum such as the audio encoding/layout
types retains its required leading pipe. Previously it would reparse as a type
alias. Structural round-trip tests cover enums and neighboring type forms.
