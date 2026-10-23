# M-STD-AUDIO-WAV-WRITE-CHECKED: valid RIFF output and ordinary input guards

**Status**: Approved — implementation and integration in progress (Mark, 2026-10-23)
**Target**: v0.54.18 candidate
**Priority**: P1 (format correctness)
**Estimated**: 2–3 engineering days including compatibility and boundary fixtures
**Dependencies**: existing M-STD-AUDIO; share format validation with the revised reader
**Created**: 2026-10-23

## Problem and systemic audit

Audio report `inbox_1792742860257_d0cd3f0f` identifies three independent ways to
produce malformed WAV output. Reproduced locally with a checked scratch AILANG
program on both the interpreter and strict bytecode VM:

| Input | Observed | Required |
|---|---|---|
| One 8-bit mono byte at 8 kHz | File length 45, RIFF size 37 | Length 46, RIFF size 38, data size 1, final pad |
| Header with 65536 channels | Channel field becomes zero | Refuse unrepresentable channels |
| One byte of 16-bit mono | Accepted, length 45 | Refuse incomplete frame |

`std/audio.ail` serializes low bytes through le16/le32 and multiplies byteRate
before checking its field width. Its requires clauses are not sufficient ordinary
runtime validation: `--verify-contracts` is opt-in. This is a shared format/size
validation problem, not just a final-byte append. Audit covers wavHeader,
wavFromPcm, durationMs, and the existing native Opus encoder's validation.

The native encoder already validates its own format and whole-frame boundary
before encoding (`validateAudioEncode`); its codec-specific accepted rates and
channels remain separate. The writer and reader should share PCM field rules,
not accidentally inherit Opus-only restrictions.

## Goals

Emit spec-conforming canonical integer-PCM WAV, reject invalid inputs before
allocation in ordinary execution, and provide Result-returning APIs without
breaking the signatures of legacy callers. Preserve the existing mono PCM16 and
stereo PCM16 header goldens for accepted lengths.

## High-impact decisions

| Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|
| Add checked APIs; retain legacy signatures with loud errors on invalid inputs | Existing callers expect bytes | human approval | design | high |
| Integer PCM 8/16/24/32 only; never relabel float bytes | Determines format promise | human approval | design | high |
| Full frames required for headers and buffers | Some existing tests deliberately construct partial headers | human approval | design | med |
| Cap materialized WAV PCM at 100 MiB; header-only API keeps RIFF format limit | Prevents multi-GB output allocation | human approval | design | med |
| Share overflow-safe validation with WAV reading | Avoids inconsistent acceptance | agent | implementation | med |

### Design freeze

- [x] Approve legacy invalid-input failures and supported format subset.
- [x] Approve materialization cap and RIFF padding policy.
- [x] Approve header-only complete-frame requirement and fixture migration.

## Solution design

Proposed new exports (specifications, not current callable APIs):

```text
wavHeaderChecked(dataLen, rate, channels, bitsPerSample)
  -> Result[bytes, string]
wavFromPcmChecked(pcm, rate, channels, bitsPerSample)
  -> Result[bytes, string]
```

### One format/size validator

Validate signs/ranges before conversion to unsigned values. Use wide checked
arithmetic and only narrow to serialized widths after proving bounds:

- channels in 1..65535; bits in {8,16,24,32}; rate in 1..4294967295.
- bytesPerSample = bits/8; require channels <= 65535/bytesPerSample before
  forming blockAlign. Require rate <= 4294967295/blockAlign before byteRate.
- dataLen >= 0 and dataLen % blockAlign == 0. This is a count of bytes, not
  samples or frames. Empty PCM is valid.
- pad = dataLen % 2. Require dataLen <= 4294967295 - 36 - pad, then form
  riffSize = 36 + dataLen + pad. The outer size includes the pad; the data
  chunk size is exactly dataLen. No modulo truncation of high bits.
- For a materialized writer, enforce dataLen <= 100*1024*1024 and output length
  representable by host int before allocation. Header-only output is 44 bytes
  and does not allocate a conceptual PCM buffer. Both paths check RIFF limits.

Use an internal Go helper for checked-width validation and canonical header
construction if ordinary legacy hard failures cannot be expressed through the
existing pure std wrapper. Proposed native adapters for legacy and checked calls
must invoke the same helper; no duplicated acceptance predicates. Header output
is built with fixed little-endian field writes. Any pure AILANG composition
should operate over header bytes and PCM directly, not bytes.toInts on payloads.
The builtin-developer skill is required when implementing new registrations.

`wavHeaderChecked` returns the 44-byte prefix, including the RIFF size of the
future padded file. Its documentation must say that a caller writing the payload
manually must append the odd-byte pad. `wavFromPcmChecked` appends PCM followed by
one zero byte iff dataLen is odd. It never changes PCM samples. Errors name the
operation and invalid field; external callers must not parse prose to recover a
numeric value. Result has string errors for consistency with current audio.encode;
a future typed-error migration is separate work.

### Legacy compatibility

`wavHeader` and `wavFromPcm` retain their existing parameter and bytes-returning
signatures. They use identical validation and loud builtin runtime errors for
invalid input, including when contract flags are absent. A documented requires
clause may mirror the rule but must not be the enforcement mechanism.

Intentional changes: odd-length valid PCM gains its required pad and corrected
RIFF size; partial frames, unsupported bit depths, field truncation/overflow and
materialized PCM above 100 MiB fail. Accepted even-length PCM16 headers remain
byte-for-byte equal to the existing goldens. This is a bug/safety correction with
an explicit compatibility boundary, not a claim that all prior calls keep working.

`durationMs` is audited but not silently changed in this slice. Its current
flooring behavior for partial sample lengths (including the existing `(1,0)`
test) remains. Its multiplication-overflow and invalid-input policy needs a
separate decision if exposed as a checked duration API; do not widen this sprint
by changing that established helper. Tests must ensure writer validation does
not accidentally route durationMs through a new whole-frame restriction.

## Files, implementation phases and acceptance criteria

1. Shared validator and checked writer, roughly 1 day:
   `std/audio.ail`, new `internal/builtins/audio_wav.go`, native unit tests and
   builtin type golden/metadata. Reuse codec limit constant where appropriate.
2. Legacy adapters and cross-engine verification, roughly 1 day: registry/VM/
   codegen specifications if required; `tests/stdlib/audio_wav_test.ail` and
   `internal/builtins/audio_test.go` regression coverage.
3. Docs, release notes and package migration evidence, roughly half to one day:
   `docs/docs/reference/std-audio.md`, new checked writer example, changelog fragment.

- [ ] Golden odd u8 output has RIFF size 38, data size 1, total length 46 and
      zero pad; accepted even PCM16 header goldens are unchanged.
- [ ] Empty, mono/stereo, and 8/16/24/32 fixtures follow the stated rules.
- [ ] Negative lengths, zero/overflowed fields, huge products, partial frames
      and over-cap buffers fail before allocation with contracts disabled.
- [ ] Header-only max-size boundary tests use integers, not multi-GB allocations.
- [ ] A regression deliberately removes the pad or width check and is caught by
      a fixture with a known expected result, independent of our new reader.
- [ ] Interpreter, strict VM and generated Go agree on bytes and error outcomes.
- [ ] Existing Opus deterministic golden and duration tests pass; documentation,
      tests, formatting, lint and required boundary gates pass before release.

The implementer may choose fixture layout and internal helper names. These phase
estimates are not an approved sprint.

## Conflict surface

Existing WAV wrappers, requires annotations, native registry and engine error
adapters are affected. No new syntax or type-inference rule. Three existing
regression anchors were inspected: `tests/stdlib/audio_wav_test.ail`,
`examples/runnable/std_audio_brief.ail`, and `TestAudioEncode_DeterministicAndGolden`
in `internal/builtins/audio_test.go`.

The existing inline header tests include `(2418713,false)` and `(101,false)` to
compare malformed headers against a golden; they do not test rejection. Move
those invalid cases to explicit checked-Err tests instead of preserving an
invalid call in a boolean comparison. Keep valid goldens and durationMs's
partial-length fixture. Do not adjust an independent golden by round-tripping
through the same implementation under test.

## Verification log (2026-10-23)

| Claim | Evidence | Result |
|---|---|---|
| Missing odd pad, truncation and incomplete frames | Live checked scratch audit fixture, interpreter and strict VM | `(45,Some(37))`, `(Some(0),Some(0))`, `45` |
| Width checks absent in writer body | Read std/audio.ail le16/le32, wavHeader/wavFromPcm | Low-byte serialization and unchecked product |
| Existing tests' actual behavior | Read tests/stdlib/audio_wav_test.ail | Header equality tests, valid PCM survival, floor duration |
| Contract checks are opt-in | ailang run --help | --verify-contracts enables runtime validation |
| Existing encoder has validation and cap | Read internal/builtins/audio.go and audio_test.go | Codec path validates before encoding; cap is 100 MiB |
| RIFF pad/size rule | Microsoft RIFF reference, read 2026-10-23 | Chunk padding is outside data chunkSize and inside containing RIFF extent |
| Coverage/duplicate search | Scaffold + direct WAV source/doc search | Existing implemented M-STD-AUDIO and revised read proposal reused; neural search reported fallback-simhash, not usable neural scores |

Source HEAD `731fa1b958bc8268889ede0ae90e6165cf1897fc`; installed binary
`v0.54.17-dirty / 8f58117`. Findings above are measured; proposed APIs are not
implemented. The original report's package reader/NaN issues remain package-owned.

The checked reproduction used the current stdlib functions below (all calls
type-checked and ran on the interpreter and strict VM). Save it as a temporary
module and use `ailang check <file>`, then
`ailang run --stdlib-path std --caps IO <file>` and the same run with
`--strict-bytecode`:

```ailang
module ailang_audio_audit
import std/io (println)
import std/audio (wavFromPcm, wavHeader)
import std/bytes (fromInts, length, byteAt)
export func main() -> () ! {IO} {
  let odd = wavFromPcm(fromInts([128]), 8000, 1, 8);
  println(show((length(odd), byteAt(odd, 4))));
  let wide = wavHeader(0, 8000, 65536, 8);
  println(show((byteAt(wide, 22), byteAt(wide, 23))));
  println(show(length(wavFromPcm(fromInts([0]), 8000, 1, 16))))
}
```

## Release integration and request tracking

Next-patch label v0.54.18 is a candidate, not a release reservation or version bump.
The following is the review/handoff order, not an authorization to execute a sprint:

| Work | Evidence / request | Release disposition |
|---|---|---|
| Correct PUB016 explanation | inbox_1792742862200_201d0d83 | Small wording fix and fragment prepared in this session; no counter change |
| WAV writer safety | inbox_1792742860257_d0cd3f0f | This design; approval, sprint, evaluation required |
| WAV metadata/packed PCM | inbox_1792742861419_98f58254; #1930 | Revised existing read design; reconcile pending PR #1931 before approval |
| formatFixed | #1930 and roadmap follow-up | Revised existing formatter design; independent of writer/reader |
| Verification reason/model evidence | inbox_1792742862200_201d0d83 | Additive package-reporting slice in existing coverage design |
| Resampler M3 | roadmap follow-up; implemented M-STD-AUDIO's planned M3 | Deferred; quality/length/delay semantics must be frozen first |
| Package reader/NaN/generator fixes | package roadmap report | sunholo/audio ownership; no core claim of resolution |

The small wording fix may ship independently. Prefer writer correctness and
honest verification evidence before advertising an expanded audio surface; the
formatter can land independently. Do not mark WAV/PCM features Added/Fixed in a
release changelog until implemented and evaluated. Before tagging, run the
release-manager checks, fold fragments, synchronize version constants, run full
required CI, then follow test/prod release verification. No tag, deployment,
message acknowledgement or existing coordinator approval is part of this prep.

Preparation validation on 2026-10-23: the existing
`go test ./internal/pkg -run 'Test(BuildQualityReport|QualityReport)' -count=1`
passed for the PUB016 wording change (12.452s). `make check-changelog` passed;
new/revised local Markdown links and fences were checked. Full release tests,
implementation tests for proposed APIs, design quorum and release CI were not
run. No version constants changed; unrelated dirty files remain parked. The two
v0_54_17-path designs deliberately retain PR #1931's paths while targeting the
next patch; a later PR reconciliation must not reintroduce the older proposals.

## Axiom compliance

| Axiom | Score | Reason |
|---|---|---|
| A1 Determinism | +1 | Canonical bytes and padding |
| A2 Replayability | +1 | Stable header goldens |
| A3 Effect legibility | 0 | Pure byte functions |
| A4 Explicit authority | 0 | No new capabilities |
| A5 Bounded verification | +1 | Checked widths and allocation limits |
| A6 Safe concurrency | 0 | Immutable output |
| A7 Machines first | +1 | Checked success/failure boundary |
| A8 Minimal syntax | 0 | Existing Result and builtin shapes |
| A9 Cost visibility | +1 | Materialization cap stated |
| A10 Composability | +1 | Shared reader/writer validation |
| A11 Structured failure | +1 | New Result APIs |
| A12 System boundary | 0 | Existing core format ownership |

Net +7; no A1/A3/A4/A7 hard violations. Compatibility and native error-adapter
drift are the main risks; explicit legacy rules and engine tests are mandatory.

## Related documents and references

- [M-STD-AUDIO](../../implemented/v0_44_0/m-std-audio.md): original writer/codec,
  not a duplicate of this hardening request.
- [Revised WAV reader](../v0_54_17/m-std-audio-wav-read.md).
- [Revised fixed formatter](../v0_54_17/m-std-string-format-fixed.md).
- [Coverage design and audio reporting addendum](../m-contract-verification-coverage.md#audio-follow-up-2026-10-23-additive-package-evidence).
- [RIFF format](https://learn.microsoft.com/en-us/windows/win32/xaudio2/resource-interchange-file-format--riff-).
- [Design axioms](../../../docs/docs/references/axioms.mdx).
