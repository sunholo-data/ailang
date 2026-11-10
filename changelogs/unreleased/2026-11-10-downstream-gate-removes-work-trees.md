### Changed — `make downstream-check` removes its working trees when the run ends (2026-11-10)

A release-tier run clones every downstream corpus and copies each arm into its output directory,
about 3.4 GB, and nothing removed them. Roughly twenty runs in one day left 80 GB in one session's
scratch directory and took the rig's disk to 96% full. The run now leaves only its four reports
(`report.md`, `report.json`, `rows.jsonl`, `pins.md`), on every verdict and on an early error, and
logs how much it freed. `KEEP_WORK=1` (`--keep-work`) keeps the trees for debugging. Read-only
directories are made writable first; symbolic links are removed, never followed.
