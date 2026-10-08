### Added — Gate 0 reads the driver's own crash notices (ailang#1160)

- `scripts/mission_gate0_self_notices.sh`: a no-authority read of the bookkeeping issue (current and previous) for the driver's own `⚠️ Mission iteration **FAILED to complete** (rc=…` notices, which the directive allowlist drops by design. Ported from ailang-world; exit 0 = none, 1 = a fire died, 2 = instrument floor.
- One signature literal (bash) is passed to the classifier; `--control auto` resolves a per-repo control; a drift guard reds if the driver's prefix and the script's `SIG_TEXT` diverge.
- Gate 0 step 6a (both skill copies) runs it; wired into `make test-launchd-drivers`.
