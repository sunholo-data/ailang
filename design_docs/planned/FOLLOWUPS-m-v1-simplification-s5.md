# Follow-ups from M-V1-SIMPLIFY-S5 (Phase 3, CLI surface)

**Written**: 2026-09-18, sprint close-out, dev @ `9659396bd`
**Sprint**: [m-v1-simplification-s5-sprint-plan.md](m-v1-simplification-s5-sprint-plan.md) · **Program**: [m-v1-simplification-program.md](m-v1-simplification-program.md)

Phase 3 is closed (`commands_top_level` 89 → 17, gate ≤ 20). These are the items the six
milestones deliberately did **not** do, each with its evidence and a named home. They existed
only in merge-commit bodies until now, which is not somewhere a fresh session looks.

Every item below was **re-verified against `dev` on 2026-09-18** — several follow-ups reported
by agents mid-sprint had already been fixed by a later milestone and are not listed.

---

## Blocking other work

### F1. The caller sweep — ~1,800 sites, gated on a release
Rename `ailang messages` → `ailang ops messages` etc. across launchd drivers, `make/*.mk`,
`tools/`, `.claude/skills/**`, `.github/workflows` and docs. Counts (git grep, tracked files):
`messages` 593, `coordinator` 332, `chains` 216, `eval-suite` 182, `trace` 66, `observatory` 63.

**Gate**: aliases must ship in a tagged release **and** one attended iteration of each mission
loop must run on it. v0.40.0 was tagged mid-sprint, before the aliases landed, so the earliest
is the next release. Verify with `make test-launchd-drivers` **on a quiet box** (see F12).

**This unblocks**: F2, and the deletion release that lets F6 and F7 move.

### F2. Phase 3b — the `ailang` / `ailang-ops` split
Ruled out for v1.0.0: precondition 2 (F1) cannot be met before the cut, so **v1.0.0 ships one
binary** and the split is v1.1's first item. That is D1 applying itself, not a new decision.
Five build paths must ship both binaries — see the program doc's Phase 3b.

---

## Defects (small, verified, no blocker)

### F3. The shipped binary accepts a golden-file test flag
`internal/parser/testutil.go:18` is a **non-test file** that does
`var update = flag.Bool("update", false, "update golden files")` on `flag.CommandLine`, and
`internal/parser` is in `go list -deps ./cmd/ailang`. **Verified 2026-09-18: `ailang -update
--version` is accepted by the shipped binary.** Nothing reads it in `main`, so it is surface
pollution rather than a behaviour bug — but it is why M6's CLI-reference generator had to name
its flag `-update-cli-reference` (registering `-update` panicked with `flag redefined`).
Fix: move the flag into a `_test.go` file or guard it behind a build tag.

### F4. `commands_platform.go` `budget` still exits 0 bare
M3 normalised `trace`/`observatory`/`dashboard` and M4 normalised `models`/`workspaces`/
`storage` to exit 1 with no subcommand, matching `chains`/`pkg`/`daemon`/`dev`/`ops`.
`budget` was **deliberately excluded**: bare `ailang budget` defaults to `status` and really
runs it, so it is not the no-subcommand shape. Decide whether that default is wanted; if not,
it joins the others.

### F5. 21 commands answer `--help` on **stderr**
Via `flag.ExitOnError` (exit 0 on `ErrHelp`, usage to stderr): `run check iface select-best
export-training eval eval-analyze eval-paired eval-censored-pairs eval-suite eval-publish debug
lsp replay exec design-review design-quorum verify generate-extension-registry ai-check
policy-check`. They pass `help_exit0_rate` (which checks the exit code) but a consumer piping
**stdout** gets nothing. Normalising moves ~21 snapshot records, so it wants its own pass.

---

## Convergence debt (the next flag pass)

### F6. Two hand-parsed `--format` sites, invisible to any FlagSet audit
- `cmd/ailang/eval_tools.go:326` — `strings.HasPrefix(arg, "--format=")`, five renderings
  (`markdown|html|docusaurus|json|csv`). `--json` would alias one, exactly as M5 did for `test`.
- `cmd/ailang/doctor.go:638` — `arg == "--format=json" || arg == "-format=json"`. A boolean-shaped
  switch with one accepted value; collapses to `--json` in ~3 lines.

Both **require the `=` form** and neither accepts `--json`. Not defects — both work. They are
named in the generated `docs/docs/reference/cli.md` so the page does not lie about them.
Deferred from M6 because its acceptance bar was that the CLI surface does not move.

### F7. `flag_names_distinct` cannot move until the deletion release
Measured 384 → 384 across M5. **Unmeetable by construction**: D1 keeps every superseded spelling
registered for one release, so a convergence release adds a name and removes none. It moves only
when F1 lets the old spellings be deleted. `output_format_spellings` (4, gate 4) is the ratchet
that tracks this instead, and falls to 1 at that point.

**Do not "fix" this by excluding `aliasStringFlag` names** — measured, that is a no-op (six other
sites register `"model"`) and it is inverted: it would drop the canonical `model` and keep the
superseded `models`.

### F8. Deprecation warnings on superseded flag spellings — deliberately NOT emitted
`ailang test --format json` is taught by **fifteen shipped, frozen** teaching prompts that
`check-prompt-freeze` forbids editing, and `pkg_quality.go:171` execs it internally. Warning
there is the `ailang fmt` defect exactly: telling every eval agent its correct, prompt-taught
invocation is wrong. `--models` has ~180 fleet call sites, so warning is a de-facto demand for
F1. **Reversible — the machinery is one small function away, and the natural time is after the
caller sweep.** Owner's call.

---

## Dead code exposed by the removals (do NOT delete on a linter's word)

### F9. `internal/observatory/seed.go`
`SeedDatabase`, `DefaultSeedConfig`, `MinimalSeedConfig`, `StressSeedConfig` lost their last
non-test caller when `observatory seed` went. `internal/` was outside M3's file set.

### F10. `AccessControlCache`'s five writer methods
`AddUserByEmail`, `AddUserToWorkspace`, `RemoveUserFromWorkspace`, `UpdateUserRole`,
`ListWorkspaceMembers` in `internal/server/auth/firestore.go` have **zero non-test callers** —
the stranded half of the access-control feature, whose CLI half is load-bearing (see the M4
merge body: `ailang access-control` is the sole writer of a key `AuthMiddleware` reads).
Untangling spans `internal/server/auth` and was nobody's file set.

### F11. `storage.NewSQLiteBackends` / `NewGCPBackends`
Lost their last non-test callers with the storage-migration removal. Exported `internal/storage`
API with tests at `backend_test.go:99,167,189,199`, thin wrappers over `NewBackendsForSelection`.
Left in place deliberately.

**For all three**: the coding standard is that "the linter says unused" is never sufficient.
`git log -S` first, and read [[F10]]'s lesson — a zero reference count cannot see coupling
through a data key.

---

## Documentation and tooling

### F12. `make test-launchd-drivers` needs a quiet box
Its `bound derivation responds to a slowed stimulus` arm (`tools/eval/test_motoko_connection_probe.sh:1022`)
measures a fork rate against a 10s cap and **dies under concurrent compilation** — CPU contention
is indistinguishable from the signal it looks for. Measured: 59/59 arms pass quiet; fails at that
arm under load. It sits second-to-last, so when it dies the four bash-3.2 syntax checks after it
never run. Not a flake to fix blindly; either accept the constraint or make the arm load-aware.

### F13. `cli-doc-maintainer` resources still teach hand-written help
`resources/best_practices.md` and `resources/help_template.md` describe hand-maintaining
`help.go`, which M1 **generates**. `SKILL.md` labels them pre-S5 history; they should be deleted.
(Its three scripts were retired in M6 — they were not stale but actively wrong.)

### F14. `ui/dist` and `internal/server/dist` carry stale CLI hints
Two copies of one bundle (`assets/index-Bg_C3mji.js`) still contain 1 `ailang dashboard` and
2 `ailang trace` literals. M3 fixed the **sources** (`CliCommandHint.tsx`,
`ExecHierarchyPopover.tsx`, `ChatHistory.tsx`) and deliberately did not rebuild the bundles —
they are built artifacts and belong to whoever owns the UI build. Harmless (the folded spellings
still resolve), but they are why a naive `grep -r` over the repo root inflates every CLI
reference count by ~2.5x. **Use `git grep`.**

---

## Not follow-ups — recorded so they are not "fixed" by mistake

- **Changelog entries and `design_docs/implemented/**` that cite removed commands are correct.**
  They are historical records of removals; rewriting them falsifies history. M4 and M6 both
  applied this judgement.
- **`ailang trace hierarchy`, `dashboard health`, `dashboard stats`, `eval-chains`, `daemon`
  are ALIVE** (verified exit 0). The Phase 3 audit listed them as dead; M3's fold kept far more
  than it removed. Do not "fix" docs that cite them.
- **`ailang disk`** appears only in a design doc *proposing* it. A design doc for unbuilt work
  is not a stale citation.
