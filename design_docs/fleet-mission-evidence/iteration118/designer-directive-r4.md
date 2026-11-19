MISSION-ROLE: designer

You are the DESIGNER for fleet mission iteration 118, revision pass (unattended; no human present). Same
worktree as before: `/Users/voightkampff/.ailang-driver-pin/.fleet-i118-rulings` (branch
`fleet/i118-rulings-ledger`). Edit ONLY `design_docs/planned/m-fleet-rulings-ledger.md`. Do not commit,
push, message, or touch other files.

Your R3 went to the design quorum (round 3; Claude benched as author vendor). **BLOCKED, all three reject —
but none disputes the design direction.** Every objection is a verification-completeness defect with a
concrete fix. Full text: `/tmp/fleet_i118/quorum-r3-full.md`. Read it. This is your ONE permitted revision
(R4); there will be one re-quorum and no further revision this iteration, so close every point.

**The pattern across all three (name it in your Quorum Revision Log):** line-number citations that were
never read, and a stated bound that the design does not enforce. A doc pitched as "measured, not assumed"
was rejected for citing unread lines. Before you submit, re-verify EVERY `file:line` citation in the doc by
actually printing the line (`sed -n 'Np' file`) and fix any that are wrong — not just the ones named below.

## The controller has already measured the named premises — use these, re-check them, cite them as V-rows

1. **`tools/launchd/mission-lane-check.sh` (gemini's objection).** Lines 49–50:
   `if _doc=$(ailang mission doctor "$NAME" 2>&1); then row config ok "$(printf '%s' "$_doc" | grep -v '^⚠\|make quick-install' | tail -1)"`
   `else row config dead "$(... | tail -1)"; fi`.
   So lane-check keys on the EXIT CODE and displays only the LAST line (`tail -1`). Notes printed before the
   summary line, with exit 0, yield `config ok` + the summary text. Add a V-row with this; ideally also run a
   stub (a fake `ailang` on PATH that prints a Note line then `1 mission(s): no drift`, exit 0) through the
   exact pipeline expression and record the output. Also check whether any `cmd/ailang` test asserts doctor
   output text (`grep -rn 'no drift' cmd/ailang/*_test.go internal/mission/*_test.go tools/launchd/`) and
   record the result with a positive control.
2. **`internal/mission/doctor.go` (glm and kimi).** `:136` is `const sharedDriverRepo = "sunholo-data/ailang"`
   — it is NOT a charter-location branch. `doctor.go` contains NO charter/`m.Doc` path logic at all
   (`grep -n 'm.Doc' internal/mission/doctor.go` → nothing; positive control: `grep -n 'm.Doc\|Doc ' internal/mission/registry.go`).
   So the `charters` scope's location rule (in-repo `root/m.Doc` vs cross-repo `m.Workdir/m.Doc`) is NEW
   design you must specify, not cite. Note `missions/*.toml`: v1/fleet/docs/motoko/triage have
   `repo = "sunholo-data/ailang"` with `doc = design_docs/<name>-mission.md`; world/stapledon are other repos.
   `:154` is the START of `func DoctorWith(reg *Registry, p Paths, lc LaunchCtl) *Report` (the function runs
   to ~`:297`, 144 lines, ending with `sort.Slice(rep.Rows…); return rep`). `root` is NOT a parameter:
   it is the per-mission unexported field `m.root`, read via `(*Mission).Root()` (`registry.go:90`), set by
   `Load` (`registry.go:251`). Specify exactly where `CheckRulings` is called, how it gets root (e.g. from the
   first mission with non-empty `Root()`; what if missions disagree; what on an empty registry — must be a
   named, non-silent outcome), and update "~5 lines".
   Callers: `cmd/ailang/mission_cmd.go:251` and `doctor_test.go:307,332,356,370,394`.
3. **`gate-0-preflight.md` (glm).** Both exist:
   `.claude/skills/mission-control/resources/gate-0-preflight.md` and
   `.agents/skills/mission-control/resources/gate-0-preflight.md`. Add a V-row (and whether they are
   byte-identical today: `cmp`).
4. **PR #2396 (glm, kimi).** `gh pr view 2396 --json state,isDraft,headRefOid` →
   `{"headRefOid":"c34c409c3881c02a597941b102eb546934b79a33","isDraft":false,"state":"OPEN"}` (it may merge
   while you work; re-run and record what you see). It is the refresh of draft #2082 onto dev. Make the
   Dependencies line and every #2082 reference consistent with that (the sibling is now #2396; #2082 is
   superseded).
5. **The `≤ 64 detector lines` cap (kimi, glm).** It is asserted in the A5 axiom row but enforced nowhere.
   Take kimi's Option A: make it real (grammar text, a `rulings-malformed` condition "more than 64 detector
   lines", and a test row with a 65-line fixture whose mutant is "cap not enforced"). Keep the A5 row
   consistent with what is enforced.
6. **Live-run gate (glm's catch).** Add a success criterion: before the doc moves to `implemented/`, one
   recorded live `ailang mission doctor` run whose ruling-finding counts match the doc's predictions (or the
   doc is corrected to what was measured).

## Rules
- Quorum Revision Log: add a "Round 3" block with one row per objection: reviewer, surface, fix, V-row.
- Every new codebase claim gets a Verification-Log row with command + observed output; negative results
  need a known-positive control in the same command.
- Keep the design itself: report-only Notes, forbid-only detectors, supersession, convention-level authority.
  Do not grow scope.
- No background processes; nothing will re-invoke you, so never end your turn announcing a wait. Finish in
  ~25 minutes. Final message: ≤10 lines — line count, each round-3 objection's disposition, any citation you
  found wrong beyond those listed.
