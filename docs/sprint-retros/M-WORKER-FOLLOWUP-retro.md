# M-WORKER-FOLLOWUP retrospective

Released [v0.54.5](https://github.com/sunholo-data/ailang/releases/tag/v0.54.5) from 296231d. M1–M4 passed. Historical independent implementation evaluation 87/100; final independent AC10/AC11 delivery review passed; all final delivery evidence is banked with the implemented design.

## Planned and actual

The estimate was 5 engineering days and 2100 changed LOC, using 420 LOC/day as an unmeasured capacity assumption. Observed calendar time from sprint creation 2026-10-11T18:21:20Z to this delivery record is 4.55 hours; it includes parallel work and CI/cloud waits. Actual engineering days, LOC/day and parallel speedup were not measured.

| Milestone | Estimated changed LOC | First integrated completion | Outcome |
|---|---:|---|---|
| M1 lifecycle/lifeline |650|18:55Z|Passed, subsequent ownership corrections in M4|
| M2 acknowledged cancellation |570|19:13Z|Passed, subsequent stdout/receipt corrections in M4|
| M3 lazy AI preparation |500|19:06Z|Passed, including named-stream immutable routing|
| M4 integration/release |380|Final delivery 2026-10-11|Passed; PRs 1765/1766 and full release verification|

Tagged-source diff against v0.54.4: 114 files, +6690/-180 (6870 changed LOC). Runtime Go: +1662/-143; Go tests: +2465/-12; AILANG examples/fixtures: +78/-12; documentation/metadata: +842/-13; validation evidence: +1643. Shared fixes prevent reliable per-milestone LOC attribution. Later release records are documentation-only and outside that immutable-tag diff.

## Execution and friction

Two implementation lanes ran in isolated worktrees: lifecycle M1→M2 and AI M3. The primary integrated and released; a separate evaluator reviewed implementation and subsequent corrections. Maximum implementation-agent parallelism was 2. No source merge conflict required dropping another lane’s work. Root unrelated changes stayed intact.

CI exposed stale generated architecture inventory, then genuine async stdout/cleanup-receipt races. Canonical inventory regeneration and deterministic failing controls addressed them. The stdout fixes passed 100 repetitions, 10 race repetitions and an independent focused race review; all final source gates passed.

The existing RSS test compared 500 interpreter iterations with 200,000, and two dev macOS attempts failed at +35/+39 MB. Local old/new binaries and CPU-limited probes passed; the precise CI allocator cause remains unproven. The calibrated floor matches 200,000 message constructions and varies 400 versus 200,000 log calls, preserving GOGC=100, 30 MB, completion/output assertions and original diagnostics. Real collect-then-flush controls fail at +61/+53 MB. Final macOS PR/dev checks pass. Identical Linux 299 MB readings do not demonstrate retention sensitivity there; the pre-existing measurement-floor caveat is recorded. No threshold relaxation, skip or further blind retry was used.

## Delivery and recommendations

The official Darwin ARM64 artifact passed checksum and cryptographic SLSA verification; 40 assets and 9 attested subjects match. Managed marker/original consumer cancellation, two unfinished async policy workers, 19 AI CLI/bin cases and 38 lifecycle cases pass with zero live providers. Async EOF remains natural completion; new EOF source delivery, strict native AI expansion, remote inference/billing cancellation and Windows process-tree guarantees remain outside scope.

Brain refresh indexed 68/396/255 while retaining active teaching prompt 16.7. TEST validates 16 images and rolls 3 existing services/15 jobs; PROD copies those images and rolls 4 existing services/17 jobs through the current shared library. Optional unprovisioned TEST credit runtime and ntfy are explicitly skipped. Public MCP 0.54.5 and release notice are verified.

Use equal-work baselines plus deliberate regression controls for future RSS checks. Treat generated inventory and all platform CI as release gates, preserve receipt-to-reader ownership through late cancellation, and record actual shared deployment counts rather than assuming older skill counts.

[Complete evidence](../../design_docs/implemented/v0_54_5/validation/release/README.md).
