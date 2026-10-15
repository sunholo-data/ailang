### Added — Triage mission design and charter (Level 0)

- `design_docs/planned/m-triage-mission.md`: a new mission loop that verifies coordinator approval cards, agent PRs and bot-filed issues on `sunholo-data/ailang`. It documents what approving and merging actually trigger (dispatch before merge, the CLI authority gate is advisory, remote reject is final, every merge redeploys dev) and gates each level against those consequences (G1–G9). Reports go to Discord through the `user` inbox and to Daneel through `daneel-reports`.
- `design_docs/triage-mission.md`: the charter, starting at Level 0 (observe and shadow-ledger only). Awaiting ratification (D-TRIAGE-1).
