# M-AUDIO-RELEASE: audio boundaries and trustworthy reports

**Status**: In progress; design approval and execution authorized by Mark on 2026-10-23
**Target**: v0.54.18 candidate; no tag/deploy in this sprint
**Baseline**: 731fa1b958bc8268889ede0ae90e6165cf1897fc; unrelated local work parked
**Estimated**: 7–10 engineering days, approximately 2800 implementation/test/docs LOC;
parallel execution reduces elapsed time but no speed claim is banked before measurement.
**AILANG prompt version loaded: v0.16.7** (`ailang prompt`, active version confirmed)

## Approved scope and decisions

User approved the revised WAV-read, WAV-write, formatFixed designs and additive
package-evidence addendum, then explicitly requested planning and execution.
All freeze items in those slices are approved by that instruction. The old
coverage status split and D-29/D-30 remain parked; resampling is deferred.
Design paths: ../v0_54_17/m-std-audio-wav-read.md,
../v0_54_17/m-std-string-format-fixed.md,
m-std-audio-wav-write-checked.md, ../m-contract-verification-coverage.md (audio addendum only).

## Registry audit and velocity

`ailang pkg search audio` found sunholo/audio@0.1.0 and sunholo/gemini_live@0.5.1.
Audio owns recipes/DSP and already depends on std/audio. Core cannot depend back
on that package for format primitives; use existing std/array storage and codecs.
`ailang pkg search format` found document/process packages, no reusable pure
fixed-decimal primitive. Reporting extends existing verifier/package structures.
M1/M2/M3 decisions: none (core boundary changes); M4: depend on existing std APIs,
not a new external dependency; M5: none (integration/evaluation).
Velocity script ran for seven days: recent history includes fleet fixes and release
v0.54.17, but no comparable audio milestone or usable recent LOC/day estimate.
Estimates therefore come from the designs and explicit test surfaces, not a
fabricated historical throughput. Inspect source before choosing helpers.

## Milestones

### M1: Checked WAV writing and metadata/packed PCM (~1450 LOC)

**Dependencies**: none
Owner: audio worker; std/audio.ail, native audio_wav/audio_pcm files and tests,
tests/stdlib/audio_wav_test.ail, audio codegen spec file if needed. Root owns
shared type golden, manifests and final docs. Keep all audio interfaces together
so shared validator rules cannot diverge between independent workers.

- [x] Canonical padded WAV, checked and legacy failure behavior match design.
- [x] WAV metadata validates chunk extents/format/frame fields and limits.
- [x] Packed s16le conversion preserves channels and explicit quantization.
- [x] Failing tests observed first, then focused tests pass with -count=20.

### M2: Bounded formatFixed and generated-Go guard (~300 LOC)

**Dependencies**: none
Owner: formatter worker; std/string.ail, string_format implementation/tests,
registry_codegen_string.go and focused codegen tests. No audio files.

- [x] Precision 0..1000, long digits, ties, signed zero and nonfinite output match strconv.
- [x] Negative/excessive precision fails ordinarily on evaluator, strict VM and generated Go.
- [x] Existing formatter outputs remain unchanged; focused deterministic tests pass.

### M3: Additive verification reason/model evidence (~600 LOC)

**Dependencies**: none
Owner: evidence worker; internal/smt, internal/pkg verification/quality structures,
cmd/ailang verify and ai-check projection, associated tests. No status/counter split.

- [x] Typed reason codes distinguish missing ensures, encoding, timeout/unknown and errors.
- [x] Package projection retains recursion/rejections/numeric model and module issues.
- [x] Old/new reports preserve counters, gate behavior and absent-evidence semantics.
- [x] Property-test skips remain separate from proof/runtime assertion evidence.

### M4: Engine integration, examples and release documentation (~350 LOC)

**Dependencies**: M1, M2, M3
Owner: root; shared builtin type golden, examples/runnable/audio_checked.ail and
string_format_fixed.ail, examples/manifest.json, audio reference documentation,
changelog fragments. Resolve adapter/codegen gaps with scoped worker follow-up.

- [x] Showcase examples check/run on interpreter and strict VM, and generated Go agrees.
- [x] Example contracts: include pure metadata/format invariants with requires/ensures where encodable; skip native byte/array solver claims with explicit reason.
- [x] Example effects: include named IO row on main; pure helpers remain effect-free.
- [x] Example inline tests: include format goldens and audio success/error predicates.
- [x] Input/result/memory benchmark evidence verifies packed path without boxed lists.
- [x] Docs and truthful release fragments match implemented APIs and compatibility changes.

### M5: Integrated gates and independent evaluation (~100 LOC)

**Dependencies**: M4
Root integrates, independent evaluator reviews diff/design/criteria without modifying implementation.

- [ ] make test-changed, lint, check-file-sizes, check-boundaries and changelog gates pass.
- [ ] Example manifest/verification and generated-Go harness pass for new features.
- [ ] Evaluator reports pass or concrete remaining failures; no self-awarded pass.
- [ ] Sprint JSON and plan record measured progress; approved feature docs reflect actual implementation.

## Schedule and execution

Days 1–4: independent M1/M2/M3 TDD workstreams, source-first checks, focused tests.
Days 5–7: M4 integrations and malformed/engine fixtures; buffer through day 10 for
adapter/compiler differences. These are estimates; execute now under the user's
standing instruction. No intermediate permission pauses for already-authorized work.
Workers share the current checkout: no branch switches, staging others' changes,
stashes or destructive Git operations. Root integrates shared files sequentially.
Independent evaluation follows completion, with at most three fix/review rounds.
No coordinator approval is acted on and no outbound message is sent by this sprint.

## Integration evidence and adjustments

The existing std/bytes codegen aliases were representation-incorrect and missing
several byte helpers; implementation now supplies owned []byte helpers, with
overflow-safe slice bounds. Concrete Array[float] uses an internal packed storage
tag in generated Go and checked converters at exported/record boundaries; generic
arrays and float lists retain their own identity. Four-argument function values
must dispatch through CallFunc just like direct audio calls. Independent review
found packed Set/export gaps and 4-arg dispatch before completion; durable runtime
and actual AILANG→Go tests cover them. Single-nullary enum formatting preserves
the sum delimiter, with a failing-first structural round-trip test.

Measured packed decode (Apple M4 Max, 10 MiB stereo source, 3 iterations): 4.499 ms,
2330.76 MB/s,41,943,208 B/op and 5 allocations; the owned float64 output alone is
41,943,040 bytes. This measures the packed native boundary, not downstream DSP or
resampler quality. Existing 41 audio tests pass on evaluator and strict VM; 24
malformed/valid fixtures are checked through actual CLI and generated Go.
The audio demo currently reports 5 passing tests and 1 generator skip (requires
random-input filtering); the skip is retained as a skip, not advertised as proof
or a pass. The formatter demo has 9 passing tests. Both integer example invariants
report verified under the explicitly labeled Real/Int model.
