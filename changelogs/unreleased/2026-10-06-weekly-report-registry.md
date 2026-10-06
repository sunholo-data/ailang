### Fixed — mission weekly report knew only v1, world and motoko (2026-10-06)

`tools/mission-weekly-report.py --mission stapledon` exited with "unknown mission" because its mission table was hard-coded. The stapledon loop's Gate-5 rotation has skipped its weekly summary since iteration 14.

- The table is now read from `missions/*.toml`, so every registered mission is reported, including docs, fleet and stapledon.
- The report also reads stapledon's log header form (`## DATE: iteration N, …`).
- Commits are counted on `origin/HEAD`, because stapledons-godot's default branch is `main`.
- docs and fleet share the ailang repo with no commit-subject convention, so their commit and line counts print as "—" rather than a guessed number.
- The totals row is labelled `all`, because `fleet` is also the name of a mission.
