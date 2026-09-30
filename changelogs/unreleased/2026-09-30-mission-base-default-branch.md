### Fixed — mission-base.sh hard-coded origin/dev; now derives the base from origin/HEAD (2026-09-30)

- Stapledon's repo (`main`, no `origin/dev`) failed every Gate-1 `mission-base.sh record gate1` with `cannot resolve origin/dev` (5 slots lost).
- The base ref is now `MISSION_BASE_REF` if set, else the target of `refs/remotes/origin/HEAD`; it is resolved once per `snap`/`record`/`drift` invocation. `last` reads only the state file and needs no ref.
- When neither exists it fails loudly (rc 1, no state row written) and names both remedies: `export MISSION_BASE_REF=origin/<default-branch>` or `git remote set-head origin --auto`.
- There is no silent fallback to `origin/dev`, `main` or `master`; `drift` now also reports an unresolvable ref on stderr.
