### Fixed — Mission idle pre-check woke on record-only evaluation commits (2026-11-26)

Stapledon's per-iteration judge verdicts advanced the mission idle fingerprint's
head and woke otherwise unchanged fires. The shared helper now skips
`.ailang/state/evaluations/eval_MISSION-ITER-*_record.json` when selecting the
fingerprint head. Ordinary sprint verdicts still wake the mission.
The fleet ticket signature is idle-fingerprint:record-eval-json-moves-head.
