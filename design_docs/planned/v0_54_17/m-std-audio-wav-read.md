# M-STD-AUDIO-WAV-READ: validated WAV metadata and packed PCM conversion

**Status**: Approved — implementation and integration in progress (Mark, 2026-10-23)
**Target**: v0.54.18 candidate (next patch after std/VERSION v0.54.17)
**Priority**: P1
**Estimated**: 3–5 engineering days including malformed-input tests and engine parity
**Dependencies**: existing std/array; coordinate validation with M-STD-AUDIO-WAV-WRITE-CHECKED
**Revision**: 2026-10-23, replaces the proposal in pending PR #1931 at the same path

## Problem statement and provenance

[Issue #1930](https://github.com/sunholo-data/ailang/issues/1930) requested WAV
metadata and PCM16 decoding for sunholo/audio. Task `task-68aabd73` produced
[PR #1931](https://github.com/sunholo-data/ailang/pull/1931), still awaiting
approval. Its list-returning decoder would retain a large boxed sample store.
Today's follow-up `inbox_1792742861419_98f58254` instead requests checked
conversion through the existing packed `Array[float]` representation, validated
data extents, explicit channel layout, and agreement on the deferred resampler.
This revision continues that design rather than creating a competing reader.

The reporter measured roughly 10 seconds to decode and measure 19 files / 10 MB
through its list implementation; that timing is reported evidence, not a local
benchmark. Local source inspection and a live check confirm packed arrays and
f32/f64 codecs already exist. Domain effects, synthesis recipes and measurements
remain owned by `sunholo/audio`; core supplies reusable format boundaries.

## Goals and non-goals

Return validated metadata, decode and encode interleaved s16le without a boxed
list per byte or sample, and preserve every channel. Invalid inputs return `Err`
in ordinary execution with contract verification disabled. No syntax, effect,
device access, streaming, RF64, compressed WAV decoding, or new buffer type.

## High-impact decisions

| Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|
| Packed Array[float] is the conversion boundary | Determines memory and downstream API | human approval | design | high |
| Phase 1 accepts RIFF/WAVE integer PCM, 8/16/24/32 container bits; conversion is s16le only | Avoids silently reinterpreting float/extensible formats | human approval | design | high |
| Exactly one fmt and one data chunk; duplicates rejected | Prevents ambiguous metadata and payload selection | human approval | design | med |
| Decode uses /32768; encode policy is explicit | Defines endpoint and round-trip behavior | human approval | design | high |
| Fixed-point 24k↔16k resampler remains separate optional M3 | Prevents a reader release from promising unmeasured DSP quality | human approval | design | med |

### Design freeze

- [x] Approve packed API, format subset, duplicate-chunk and trailing-byte policy.
- [x] Approve quantization and limits below together with the writer design.
- [x] Review replacement locally; user approved this revised scope. External PR #1931 remains untouched.

## Solution design

The following are proposed signatures, not callable exports in v0.54.17:

```text
wavInfo(wav: bytes) -> Result[WavInfo, string]
pcm16ToFloats(pcm: bytes, offset: int, frames: int, channels: int)
  -> Result[Array[float], string]
floatsToPcm16(samples: Array[float], channels: int, policy: Pcm16Quantization)
  -> Result[bytes, string]
Pcm16Quantization = RejectOutOfRange | ClipOutOfRange
```

`WavInfo` contains `encoding` (a WavEncoding ADT, initially IntegerPcm), `rate`,
`channels`, `layout` (a PcmLayout ADT, initially Interleaved), `containerBits`,
`validBits`, `blockAlign`, `byteRate`, `dataOffset`, `dataBytes`, and `frames`.
For accepted format tag 1, validBits equals containerBits. Channel count and
interleaving do not imply speaker identities: no speaker-mask information is
invented. Format tag 3 (IEEE float) and 65534 (extensible) return explicit
unsupported-format errors in this slice. Supporting them requires a later design
with float sanitation and channel-mask/subformat validation.

### RIFF traversal

1. Require the 12-byte RIFF/WAVE prefix. Compute `riffEnd = 8 + riffSize` in
   checked wide arithmetic; require `12 <= riffEnd <= byteLength`.
2. Walk only that extent. Every chunk needs an 8-byte header, a payload entirely
   inside riffEnd, and its required odd-byte pad inside riffEnd. Unknown chunks
   are skipped; their pad byte need not be zero. Require exact termination at
   riffEnd. Ignore bytes after riffEnd; never let them supply a missing pad or
   satisfy a truncated chunk.
3. Accept fmt and data in either order, but reject duplicates. Require fmt size
   at least 16 and an accepted format tag. A 16-byte PCM fmt is valid; if a PCM
   extension is present require at least 18 bytes and `18 + cbSize <= fmtSize`.
4. Validate supported bits, positive rate/channels, header widths and products
   using the same format validator as the writer. Require stored blockAlign and
   byteRate to match the computed values. Require dataBytes % blockAlign == 0.
5. Stop with Err after 65,536 chunks. Empty data is valid; absent data is not.
   Return the declared data extent excluding its pad and any later chunks.

Keep the RIFF walk in pure AILANG over byteAt when its bounded-fixture benchmark
passes. It must inspect headers rather than materialize payload lists. Share
format validation through an internal native helper if the writer needs checked
wide arithmetic; do not maintain divergent acceptance rules in the two paths.
Native bulk conversion is justified by the requested packed result and existing
array kernels; additional native parsing is contingent on measured need.

### Conversion and quantization

Decode reads exactly frames * channels * 2 bytes starting at offset. Validate
nonnegative offset/frames, channels in 1..65535, offset % 2 == 0, and bounds via
subtraction/division before multiplication or slicing. Decoding a raw segment
does not imply metadata validation: a WAV caller must use wavInfo's dataOffset
and frames, with containerBits == 16. Every sample is retained in file order;
Array length is frames * channels. Return an owned packed array via the existing
array representation, never bytes.toInts or a temporary boxed sample list.

The exact decode maps -32768 to -1, 0 to 0, and 32767 to 32767/32768. Encoding
requires positive channels and arrayLength % channels == 0. Always reject NaN
and infinities; never turn them into silence. RejectOutOfRange accepts finite
[-1, 1]; ClipOutOfRange clamps finite values to that interval. For each value,
round x * 32768 to nearest, ties to even, then saturate to [-32768,32767]. Thus
+1 maps to 32767 and every decoded s16 value round-trips exactly. The package's
legacy *32767 rounding remains a distinct compatibility API in sunholo/audio;
changing it is not part of this core change.

Cap decoded/encoded PCM extents at 100 MiB, consistent with audio's codec input
limit; cap conversion output at the corresponding 52,428,800 samples. Enforce
these bounds before allocation, including host-int representability checks.
This still permits a substantial packed allocation; it is not a whole-process
memory guarantee. Record input/output bytes and peak allocations in benchmarks.

### Resampling ownership and dependency

The [M-STD-AUDIO design](../../implemented/v0_44_0/m-std-audio.md#m3-optional-resample)
already specifies optional fixed-point s16le 24000↔16000 M3. That remains the
proposed core direction, not a newly delivered export. Before M3 is scheduled,
revise that existing section to freeze output-frame rounding, startup/end
extension, group-delay compensation, flush length, coefficient precision,
saturation, passband ripple, and stopband attenuation with measurable thresholds.
Use DC, impulse, passband tone and above-output-Nyquist tone fixtures, plus
amd64/arm64 byte goldens and benchmark comparisons against a package reference.
The package may ship a reference implementation independently. 44.1k/48k pairs
and streaming state require evidence and a separate approval; they do not block
the WAV/PCM release slice. No quality or speed claim is accepted from a function
name or a linear-interpolation demonstration alone.

## Implementation phases and files

1. Metadata/type definitions and shared validation, about 1–2 days:
   `std/audio.ail`, new `internal/builtins/audio_pcm.go`, writer helper interface.
2. Packed conversion, about 1–2 days: same builtin module, registry metadata,
   builtin type golden, VM adaptation and Go codegen specification as required.
   Inspect Result/Array handling before claiming automatic engine parity.
3. Malformed corpus, measurements and docs, about 1 day:
   `tests/stdlib/audio_wav_test.ail`, new Go conversion tests, docs/reference audio
   page, package migration example and an implementation-only changelog fragment.

The implementer may choose helper names and fixture organization. These are design
estimates, not an approved sprint or delivery promise. New builtins must follow
the builtin-developer skill when implementation is authorized.

## Conflict surface and acceptance criteria

The export set, Result payloads, builtin registry, array marshalling and generated
Go must agree. No parser or type-inference extension is required. Existing
`encode` and its deterministic Ogg bytes, `durationMs`, and array f32/f64 codecs
remain regression fixtures. Deliberate changes to legacy WAV writing are confined
to the linked writer design. No existing PCM decode export is being retyped.

- [ ] Mono and stereo fixtures preserve channel/rate identity and sample order.
- [ ] Canonical headers, extra/unknown chunks, odd padding, fmt-after-data, empty
      data, and trailing bytes follow the exact policy above.
- [ ] Missing/duplicate chunks, short fmt, inconsistent byteRate/blockAlign,
      unsupported float/extensible tags, partial frames and overflowing extents
      return field-specific Err without contract flags.
- [ ] Boundary samples and tie cases round-trip; both policies reject nonfinite
      values. Conversion cannot read a WAV's padding as another sample.
- [ ] Interpreter, strict VM and generated Go agree on valid output and invalid
      input behavior; packed representation and allocations are measured.
- [ ] Existing audio/array regression tests, required lint/build/boundary checks,
      documentation and release notes pass before marking implemented.

## Verification log (2026-10-23)

Source HEAD `731fa1b958bc8268889ede0ae90e6165cf1897fc`; installed binary
v0.54.17-dirty / 8f58117. Source and binary identities are deliberately separate.

| Claim | Evidence | Result |
|---|---|---|
| Current std/audio surface | Read std/audio.ail and implemented M-STD-AUDIO | Writer helpers and encode; optional M3 remains planned |
| Packed arrays/codecs already exist | Read std/array.ail, internal/eval/value.go and array_float.go; live scratch check/run of decodeF32LE | Ok(#[1.0]) on interpreter and strict VM |
| Existing proposal and ownership | gh pr diff 1931; canonical messages/task records | Design only; task pending approval |
| Writer validation gaps affect reader fixtures | Checked scratch AILANG, then both execution engines | Odd length 45, RIFF size 37; channel 65536 stored as zero; partial s16 accepted |
| Duplicate search | Scaffold SimHash/neural attempts; direct WAV/PCM source/doc search | Neural attempt used fallback-simhash (not neural evidence); existing read design revised, existing M3 reused |
| Proposed format rules | Microsoft RIFF reference and field validator design | Format specification, not an implementation claim |

## Axiom compliance

| Axiom | Score | Reason |
|---|---|---|
| A1 Determinism | +1 | Fixed decode/quantization rules |
| A2 Replayability | +1 | Metadata preserves input interpretation |
| A3 Effect legibility | 0 | Pure byte operations |
| A4 Explicit authority | 0 | No capability change |
| A5 Bounded verification | +1 | Checked lengths and finite format cases |
| A6 Safe concurrency | 0 | Immutable owned results |
| A7 Machines first | +1 | Typed metadata and errors |
| A8 Minimal syntax | 0 | Existing ADTs, arrays and Result |
| A9 Cost visibility | +1 | Packed allocation and benchmark evidence |
| A10 Composability | +1 | std/array and package DSP share a boundary |
| A11 Structured failure | +1 | Checked Result APIs |
| A12 System boundary | +1 | Core formats, package domain behavior |

Net +8; no hard violations on A1/A3/A4/A7. Risks are ambiguous format policy,
allocation size, and engine adapter drift; the explicit subset, limits, and
cross-engine acceptance tests address them.

## Related documents and references

- [Checked WAV writing](../v0_54_18/m-std-audio-wav-write-checked.md): malformed
  output and legacy compatibility; shares validation, does not duplicate parsing.
- [Fixed formatting](m-std-string-format-fixed.md): separate #1930 reporting API.
- [RIFF specification](https://learn.microsoft.com/en-us/windows/win32/xaudio2/resource-interchange-file-format--riff-).
- [Design axioms](../../../docs/docs/references/axioms.mdx).

**Created**: 2026-10-22 (PR #1931). **Revised**: 2026-10-23, attended audio triage.

## Implementation integration notes (2026-10-23)

Generated Go needed the existing bytes primitive to map to []byte consistently
in AST/signature, inferred-type and ADT-field mappings; treating it as an opaque
*Bytes compiled but panicked on Result payloads. Tests captured that failure.
The compiled PCM result uses an internal FloatArrayVal ([]float64) storage tag
for packed ownership and canonical array Show. This is a runtime representation
of the existing Array[float], not a new language or public buffer type. Generic
array reads, updates and exported returns must preserve that representation.

The public single-nullary enums use leading `|` syntax. The formatter previously
dropped that delimiter and reparsed an enum as an alias. The incidental fix
preserves it only for a single constructor with no fields, with structural
round-trip tests for enums, payload constructors, multi-constructor sums and
aliases. No parser or type-inference semantics change.

Expanded conflict surface: codegen bytes/array signatures and ADT payload
marshalling share the generator with lists, tuples, named records and ordinary
ArrayVal literals. Regression anchors are TestGenerateArrayPreservesRuntimeIdentity,
TestConvertersAcceptArrayVal, TestFormatFixedGeneratedGo, the formatter generated
round-trip cases, and the real interpreter/VM/generated-Go audio fixture corpus.
Intentional changes are correct bytes slice mapping, packed array preservation
and enum delimiter retention; list/tuple rendering and user-defined pointers
must remain unchanged.
