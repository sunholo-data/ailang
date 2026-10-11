# Memory probe calibration

The previous floor did 500 interpreter iterations versus 200,000 in the measured arm. The dev macOS matrix reported +35/+39 MB on two attempts; the previous release had also seen this test fail. Local pinned old/new binaries passed, including CPU-limited probes; the precise CI allocator cause remains unproven.

The calibrated floor constructs the same 200,000 messages through identical recursion, logging once per inner batch (400 calls), versus every iteration (200,000 calls). The original one-batch/500-call raw floor remains a diagnostic. Every arm must exit successfully, print `done`, and emit no `tick`. GOGC=100 and the <=30 MB bound remain unchanged. Measurements fail immediately; there is no retry or skip.

The worker normal run passed at 0 MB; a genuine collect-then-flush control failed at +61 MB. The independent evaluator rebuilt its own control: normal passed at 0 MB, control failed at +53 MB. Both controls failed specifically at the RSS assertion. Source hashes were frozen throughout the independent proof.

For reproduction, make temporary copies of `internal/runner/run.go` and `cmd/ailang/main_test.go`; apply the banked control diffs only to those copies. Set the copied test helper’s binary path to the isolated binary. Create Go `-overlay` JSON mappings from each original absolute source path to its temporary copy. Build `./cmd/ailang` with the runtime overlay, then run the exact focused test with the test-helper overlay and `AILANG_MEMPROBE_CONTROL=1`. These overrides are temporary test controls; no runtime hook or override is committed. `proof.json` records commands and original hashes; `independent-review.json` records the separately repeated proof.

This repairs approved M4 validation; runtime semantics and release acceptance scope do not change. Renewed platform CI is required before publication.
