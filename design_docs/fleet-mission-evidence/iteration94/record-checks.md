# Record verification

Scope: documentation and captured observation evidence only; no implementation or runtime change.

- Base canonical-heading check: `go test ./internal/mission/ -run Canonical -count=1`, rc 0, darwin/arm64, current source. These selected tests do not shell out to a compiler binary.
- `make check-no-personal-email check-context-docs check-referenced-paths check-changelog`, rc 0 on the base plus role reports. 104 referenced paths enumerated/checked. Final record checks will be re-run after its edits.
- Source of check selection: `.github/workflows/ci.yml` checks job, plus Gate 4 canonical-heading contract. Full remote CI remains required for the eventual record merge.
- No ticket resolved; no driver/skill/Go feature code, host config or live surface edited. Fleet done-gate driver dry-runs and launchd suites therefore N/A to a fix, not claimed passed.
- Last-20 class measurement is from full canonical log/archive headlines, sorted by iteration; the generated index truncates long headlines and strips class suffixes. Reading its tail or counting its absent suffixes is not a valid harness-share instrument. With an ADMIN iteration 94, iterations 75–94 contain 18 HARNESS and 2 ADMIN rows.
- Last three actual landings 91, 87, 86 each reduced an outstanding clause-2 ticket; no three-landing DRIFT alarm. This iteration changes no goal clause.

Record rotation proof: charter410 + new4 - archived6 =408 lines; three dated STATUS stamps, exact old91 block present in archive (control90 present), CURRENT GOAL/queue/footer and decision ledger byte-identical to base. Both regenerated indexes point into this record worktree; orphan log row60 retained.

Controller correction: a misplaced `rotate-log --stream status --keep 3` rotated the archive stream, not the charter. Only this controller's resulting archive changes were restored from the verified base; its new duplicate archive-old file was moved to a per-iteration /tmp backup. The charter then used an exact bounded old91-block move and both-end assertions. Final diff has the intended six-line archive addition, no mass rotation or loss. The initial count also included the structural STATUS heading; the final invariant counts dated stamps only.

Final local gates returned success on darwin/arm64: current-source canonical selected tests; all four documentation guards (104 referenced paths); decision ledger42rows valid; routing-model scan and diff whitespace check clean. Read-audit controller recheck covered all152/357/178 lines,24 successful ranges, no excerpts and acked=true. Artifact paths are not ignored; known ignored-path control fired.
