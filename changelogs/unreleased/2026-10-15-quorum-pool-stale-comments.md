### Changed — design-quorum pool references updated to gemini-4-argon

Comments in `tools/launchd/mission-control.sh` and `models.yml` and the Quorum row in the mission
model-fleet guide still named `gemini-3-1-pro` (and older OpenAI/Z-AI seats) as reviewers. They now
point at `defaultQuorumPool` in `cmd/ailang/design_quorum.go`, where Argon has held the Google seat
since 2026-10-13. No behaviour change.
