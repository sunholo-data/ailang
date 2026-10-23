# M-AUDIO-RELEASE retrospective — 2026-10-23

The attended session traced the original audio request to a completed design task
waiting for approval, and found three later requests parked in the human-triage
inbox. Mark approved the revised designs and explicitly requested sprint planning
and execution. Implementation stayed in the current checkout; unrelated local
benchmarks and planning work were preserved.

## Execution and evidence

Three independent workstreams owned audio, formatting, and verification evidence.
Root owned shared registration, codegen integration, examples, documentation and
release fragments. Independent review used a separate agent, with actual compiled
Go reproductions rather than treating successful compilation as runtime parity.

The core/native work was straightforward once binary extent and quantization
policies were explicit. Integration consumed more work than the initial estimate:
missing byte codegen helpers, bytes misclassified as a user-defined pointer,
packed-array update/export adaptation, and four-argument function values all
needed concrete execution tests. Review also corrected the mixed-contract human
summary. A formatter round-trip test caught loss of the delimiter distinguishing
a single-constructor enum from a type alias.

Native pure boundaries were tested repeatedly. The audio suite has 41 passing
cases on the interpreter and strict VM, with zero named-body fallback. A separate
24-fixture malformed/valid corpus checks interpreter, strict VM and actual
compiled Go acceptance. Generated-array tests cover immutable updates, literal
conversion, exported returns, typed ADT/record fields and first-class calls.

The packed 10 MiB decode benchmark on Apple M4 Max measured 4.499 ms, 2330.76 MB/s,
41,943,208 B/op and five allocations. Output storage itself is 41,943,040 bytes.
This demonstrates the packed boundary's allocation shape; it is not a measurement
of DSP pipeline speed or resampling quality. No additional native DSP kernels or
resampler were introduced.

## Lessons applied

- Run generated programs: a successful Go build missed interface assertions and
  function-value dispatch failures.
- Treat packed storage as a representation of the existing array API; preserve
  updates, exported signatures, record fields and canonical display together.
- Verify shared-name resolution using local aliases such as decode, rather than
  only non-colliding demo names.
- Preserve absent verification evidence as unavailable, and keep test skips and
  module-budget sentinels separate from solver proofs.
- Respect skipped property generators: the small audio demo currently records a
  requires-filter skip, not an extra passing property.

## Final gate record

The integrated milestone checkpoint and runnable-example/manifest gate passed.
Final clean-checkout full-suite and independent evaluation results are recorded
in the sprint JSON and evaluation report; no release tag or deployment is part of
this sprint. The pending external design PR #1931 and coordinator approval were
left untouched, so they must not later reintroduce the superseded proposal.
