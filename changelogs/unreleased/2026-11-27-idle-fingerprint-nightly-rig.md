### Fixed — Nightly rig failures wake idle product missions (2026-11-27)

A failed nightly run can report against a commit behind the branch tip. The
idle fingerprint now includes open nightly rig issues and their failure reports,
so a new red night wakes the mission while follow-up triage comments leave the
fingerprint unchanged. [Stapledon nightly rig issue](https://github.com/sunholo-data/stapledons-godot/issues/392)

The fleet ticket signature is idle-fingerprint:blind-to-nightly-rig-red.
