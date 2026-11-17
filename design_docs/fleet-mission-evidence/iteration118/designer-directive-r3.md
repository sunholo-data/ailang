MISSION-ROLE: designer

You are the DESIGNER for fleet mission iteration 118 (unattended; no human is present). You are running
in the git worktree `/Users/voightkampff/.ailang-driver-pin/.fleet-i118-rulings` (branch
`fleet/i118-rulings-ledger`, based on origin/dev). Work ONLY in that directory.

## Your single deliverable

Rewrite `design_docs/planned/m-fleet-rulings-ledger.md` as revision R3 of the fleet rulings-ledger design.
Edit that one file only. Do NOT commit, push, open PRs, send messages, or touch any other file.
The controller commits your output.

## What the human ruled (this is the spec — quote it in the doc's header)

Decision D-FLEET-25, answered by Mark (attended, 2026-11-16): **"A (as recommended): a design doc (quorum,
plan, judge) for one fleet-wide rulings ledger every charter links to and cannot override, plus an
`ailang mission doctor` check reporting charter/skill text that contradicts a RESOLVED fleet-wide ruling;
the fleet also corrects the World fork claim in world.toml and the doctor note."**
The World-fork text correction is being landed SEPARATELY this iteration (draft PR #2082, refreshed); your
doc must not re-plan it — mention it only as a sibling dependency already handled.
Full ledger row: `/tmp/fleet_i118/d-fleet-25-row.md`.

## Why a rewrite, not a patch

The current file (R2, 698 lines, authored by GLM 5.3 at iteration 93) was BLOCKED by the design quorum
TWICE (all present reviewers rejecting both rounds), and an independent judge returned
NEEDS-HUMAN-REVIEW. Read all of them first:
- `/tmp/fleet_i118/prior-quorum.md` — rounds 1 and 2, every reviewer's strongest objection and findings.
- `/tmp/fleet_i118/prior-design-judge.md` — the judge's blockers B1–B4.

Diagnosis (controller's, verify it yourself before relying on it): most objections land on machinery the
ruling never asked for — an "attended-provenance binding" that tries to make the doctor ENFORCE that only
a human activated a claim (approved-blob snapshots, provenance-helper verdicts, claimset extraction
across a shared fence). The repo's own standing rule (CLAUDE.md principle 4, and
`scripts/mission_answer.sh`'s header "KNOWN WEAKNESS") is that the attended-identity guard is a
CONVENTION, not an enforcement: on this rig the fleet bot and Mark's attended sessions commit with the same
identity, so no git-level check can distinguish them. A design that pretends otherwise gets rejected for
it (judge B1). So:

**Design target: the SMALLEST mechanism that satisfies the ruling, sprint-sized (≤ 2 days build at 2x
estimate), doc ≤ ~350 lines.** Concretely the controller expects something like (you decide; justify
any departure):
1. One file, e.g. `design_docs/fleet-rulings.md`, holding fleet-wide rulings: id, date, ruling text,
   status, and a small machine-readable detector per ruling (e.g. forbidden literals with a scope). Every
   mission charter carries one link line to it and a sentence that it cannot be overridden by charter text.
2. An `ailang mission doctor` check that READS the ledger and REPORTS (does not enforce, block, or
   mutate anything) charter/skill text matching a RESOLVED ruling's forbidden literal in its scope.
   It is diagnostics. Decide and justify severity (warn vs drift/exit code), and make an absent ledger
   or a registry without one NOT a permanent red (judge B4).
3. Authority stated honestly as convention-level: rulings and detectors change only via reviewed PRs;
   git history is the audit; the doctor never resolves, activates, or edits anything. No provenance
   binding, no approved-blob snapshots — or, if you keep any of it, show why the ruling requires it.
4. Supersession between rulings (judge B3): a simple explicit rule (e.g. a `supersedes:` field; a
   superseded ruling's detectors are inactive).
5. Seed content that does NOT self-fail (quorum R1 objection on `require-text "gpt-6-astra" scope=skills`):
   prefer forbid-literal detectors only; if you keep a require verb, define it so it cannot fire on
   every file of a scope. Every seed detector needs a Verification-Log row showing its CURRENT match count
   on the target corpus (command + output) — zero matches today for a forbid detector is the expected
   state, proved with a known-positive control.
6. Parser robustness (quorum R2 gemini objection): a malformed line must be attributable or fail loudly
   for the whole ledger — no silent ignore, no cross-ruling denial without a named finding.

## Mandatory doc structure

Follow the repo's design-doc template (look at 2–3 recent docs in `design_docs/planned/` for shape: header
fields Status/Target/Priority/Estimated/Dependencies, Axiom Compliance, Problem, Design, Implementation
plan with files, Tests, Success criteria). Keep a `**Planner-Lane**:` header line (value `codex-ok`).
Required sections beyond the template:
- **Quorum Revision Log**: one row per prior objection (R1 ×3 reviewers, R2 ×3, judge B1–B4), naming the
  SURFACE it lands on, and how R3 answers it (fixed / removed-with-scope / rebutted-with-measurement).
  An objection about machinery you deleted is answered "removed — out of the ruling's scope", with the
  ruling's words.
- **Verification Log**: EVERY sentence claiming "the codebase currently does X" needs a row with the exact
  command and its observed output. An empty/negative result is a CLAIM, not a fact: pair it with a
  known-positive control in the same command. Cite file:line you actually read. Do not cite artifacts you
  did not open.
- **Files touched**: the planner will route this through the fleet's write-scope guard; list every path
  (they must sit in: `internal/mission/**`, `cmd/ailang/mission*`, `missions/**`, `design_docs/**`,
  `.claude/skills/mission-*/**`, `.agents/skills/mission-*/**`, `scripts/mission_*`,
  `scripts/test_mission_*`, `changelogs/**`, `docs/**`). Anything else is out of scope and must not appear.
- **Non-goals**: explicitly list what was removed vs R2 and why.

## Useful anchors (verify — they are claims)
`internal/mission/doctor.go` (doctor rows; `row.Fork` ~:281), `cmd/ailang/mission_cmd.go`,
`internal/mission/registry.go`, `missions/*.toml`, `design_docs/*-mission.md` (charters),
`scripts/mission_decisions.sh` (ledger block parsing), `scripts/mission_answer.sh` (header, KNOWN WEAKNESS),
`.claude/skills/mission-control/` and its `.agents` mirror.

## Operating rules
- Read-only exploration is free (`rg`, `grep`, `git log`, `sed -n`, `go doc`). You may run
  `go test ./internal/mission/ -run <X>` if it helps verify a claim. Do NOT run anything that takes the rig
  GPU lock, neural docs search, eval suites, `make test`, or network writes.
- No background processes. If you start anything long, poll its output file inside a bounded
  `date +%s` loop (≤5 min). A Monitor is an event stream, not a wait; nothing will re-invoke you, so never
  end your turn announcing that you will wait.
- Finish within ~40 minutes. Your final message: a 10-line summary — line count of the doc, the mechanism
  in two sentences, how each of the 10 prior objections was disposed (one word each), and any claim you
  could not verify.
