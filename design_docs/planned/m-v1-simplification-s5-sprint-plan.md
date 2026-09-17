# Sprint Plan: M-V1-SIMPLIFY-S5 — CLI dispatch table, groups, aliases (Phase 3)

**Design doc**: [m-v1-simplification-program.md](m-v1-simplification-program.md) Phase 3 (items 1–7) and Design Freeze D1/D7
**Handover**: [HANDOVER-m-v1-simplification-phase3.md](HANDOVER-m-v1-simplification-phase3.md)
**Sprint ID**: M-V1-SIMPLIFY-S5
**Created**: 2026-09-17, from dev @ fa5d79221
**Status**: Planned

## Summary

Phase 3 owns exactly one failing gate: `commands_top_level` **89 → ≤ 20**. Everything else in the
gate table is either closed (six gates) or Phase 4's. The work is a dispatch table plus grouping,
and the constraint that makes it non-trivial is D1: **every current top-level spelling keeps
working**, because the fleet calls them from launchd drivers, make targets, skills and workflows.

Re-measured at HEAD with `git grep` over **tracked** files in `Makefile make/ tools/ scripts/
.github/ .claude/skills/ .claude/rules/ docs/docs/ internal/ cmd/`:

| command | refs | command | refs | command | refs |
|---|---|---|---|---|---|
| `messages` | 593 | `chains` | 216 | `trace` | 66 |
| `coordinator` | 332 | `eval-suite` | 182 | `observatory` | 63 |
| `eval-matrix` | 47 | `eval-report` | 45 | `eval-chains` | 37 |
| `dashboard` | 32 | `eval-sweet-spot` | 15 | `budget` / `storage` | 13 / 13 |
| `eval-trend` | 9 | `access-control` / `axioms` | 8 / 8 | `watch` / `policy-check` | 3 / 3 |

**Measure with `git grep`, not `grep -r`.** A plain `grep -r` from the repo root walks
`.claude/worktrees/pkg-quality-ladder-s1/`, whose checked-in `ui/dist/assets/index-*.js` bundle is
minified JavaScript containing `ailang dashboard`/`ailang trace` string literals. That inflated the
first cut of this table by roughly 2.5x (`messages` read 1,463 instead of 593). The conclusion is
unchanged and D1 still binds; the numbers above are the corrected ones.

**Duration:** 5 days
**Risk Level:** High for M1 (every command's entry point moves at once) and M3 (~5,000 LOC deleted);
Low for the rest. M1's risk is bought down by a byte-diff harness, not by care.

## Sequencing — this is the part that is not negotiable

M1 touches `main.go` and `help.go`; M2 rewrites the table M1 creates. Both are single-agent,
sequential, merged to `dev` before the next starts. Only after M2 is on origin do M3–M6 run in
parallel, and their file sets are disjoint by construction:

```
M1 (alone) ──▶ M2 (alone) ──┬──▶ M3  cmd/ailang/{trace,observatory,dashboard,chains}*.go, eval_chains.go
                            ├──▶ M4  the other D7 command files
                            ├──▶ M5  flag families (eval_*.go flag defs + shared flag helper)
                            └──▶ M6  docs/ + tools/simplicity_metrics.sh  (lands last)
```

M5 and M3/M4 can collide on `eval_*.go`: M5 owns **only** flag *registration* lines and the shared
helper; M3/M4 own whole-file deletions. Any agent that needs a file it does not own stops and
reports — it does not reach across.

## Measurement harness (build it in M1, every later milestone reuses it)

`tools/cli_surface_snapshot.sh` — for every command and group, run `<bin> <cmd> --help`,
`<bin> <cmd>` with no args, and `<bin> <cmd> --json` where accepted; record exit code + stdout +
stderr into one file. Build `bin/ailang-before` from the **unedited** tree with identical ldflags
(a version stamp produced 196 false diffs once — pass the same `-X` values), snapshot, then diff
against the rebuilt binary. Every intentional difference is listed in the milestone report; zero
unintentional ones is the acceptance bar.

## Milestones

### M1 — Dispatch table (~600 LOC, alone, blocks M2–M6)

`cmd/ailang/commands.go`: `[]Command{Name, Aliases, Group, Hidden, Summary, Run}` with `Run` taking
the arg tail and returning an error; `main.go`'s 79-case / 89-label switch becomes a lookup. Help is
generated from the table: `--help` honoured at **every** level (today 8 groups reject it), an unknown
command prints **one line plus a did-you-mean**, not 16 KB of `printHelp()`. `isLanguageCommand()`
(`main.go:~660`, S1 M6) becomes the `Group` attribute — language commands must still run **no**
`observatory.CheckHealth` and **no** `checkStaleBinary`, and a test must prove it by counting probes,
not by reading the source.

- [ ] `cmd/ailang/commands.go` is the only place a command name appears; `main.go` has no `case` labels for commands
- [ ] `cmd/ailang/help.go`'s 505 hand-written lines are generated from the table
- [ ] `tools/cli_surface_snapshot.sh` diff vs `bin/ailang-before`: every difference intentional and listed
- [ ] `ailang <group> --help` exits 0 for all groups; `ailang daemon` with no args prints help and does NOT start the daemon
- [ ] unknown command: one line + suggestion, exit 1, nothing on stdout
- [ ] negative control: a test that fails against pre-M1 code (help-exit-0 for the 8 rejecting groups)
- [ ] language commands open no observatory DB and run no git probe — proven by a counting fake, with the test failing if the probe is re-added
- [ ] `make test-core` green; `go test ./cmd/ailang/... -count=1` green

### M2 — Groups and aliases, D1 (~400 LOC, alone, after M1 is on origin)

Visible top level ≤ 20: `run check fmt test repl iface verify prompt docs examples lsp init serve
version help pkg eval`. Hidden `dev`: `disasm compile replay export-training builtins doctor axioms
select-best ast-edit dump-iface sandbox-check policy-check devtools-prompt agent-prompt editor pi
micro-rag cache prompt-freeze`. Hidden `ops`: `messages coordinator mission chains models exec daemon
server budget workspaces design-review design-quorum mcp`. `eval` becomes a group with the 15
`eval-*` names as aliases. `check --verify --format agent` absorbs `ai-check`, alias kept.

- [ ] every name in today's switch — including `msg`, `brain`, `microrag`, `urag`, `serve`, `serve-api`, `internal-dump-iface`, and the seven bare `pkg` verbs (`add lock tree install search publish unpublish`) — resolves to the same behaviour it has today
- [ ] a table-driven test enumerates every pre-S5 spelling from a **checked-in fixture list** and asserts non-"unknown command"; the fixture is generated from `bin/ailang-before`, not hand-typed
- [ ] `ailang --help` fits one screen: ≤ 20 visible entries, hidden groups discoverable via `ailang dev --help` / `ailang ops --help`
- [ ] `ai_check_exit_test.go` passes unchanged (its exit contract and the DP7 gate depend on it)
- [ ] `make test-launchd-drivers` green — **run it on a quiet box, and capture make's own exit code**, not a pipeline's (see the trap below)
- [ ] snapshot diff: alias paths byte-identical to `ailang-before` on stdout; help text is the only intended change

### M3 — Fold `trace` + `observatory` + `dashboard` + `eval-chains` into `chains` (−~5,000 LOC)

`chains` is what the fleet uses (510 refs vs 86 for `dashboard`). Fold the useful subcommands in;
delete the rest under D7 with `git log -S` evidence in each commit body. **Before deleting anything**,
confirm the web dashboard reaches these code paths through `server` HTTP and not through the CLI —
if it does not, the milestone stops and reports.


**The precondition is already checked — here is the answer, so do not re-derive it.** `internal/server`
contains **no** `exec.Command` to the `ailang` binary (the only subprocess calls are `osascript`,
`zenity`, `kdialog`, `powershell`, `ps` and `pgrep`, in `handlers_util.go` and `monitor.go`). The web
dashboard reaches chain, span and observatory data through the HTTP handlers in
`internal/server/handlers_chains.go`, `handlers_statistics.go` and
`handlers_controlplane_heatmap.go`, which read the stores as a library. **Folding and deleting the
CLI paths does not break the dashboard.**

**But the UI prints these commands as copy-paste hints**, and deleting them would make the dashboard
display commands the binary rejects — the same defect class Phase 3 item 6 fixes in the docs. These
files are part of M3's ownership and must be updated in the same PR:
`ui/src/features/controlplane/components/CliCommandHint.tsx`,
`.../ExecHierarchy/ExecHierarchyPopover.tsx` (5 hint strings: `ailang dashboard spans|tools`,
`ailang trace view`), `.../ExecHierarchy/ChatHistory.tsx:1027,1032` (`ailang trace view`).
`ChainExplorer.tsx` and `StageDetail.tsx` already emit `ailang chains …` and need no change.
Do **not** rebuild `ui/dist` or `internal/server/dist` — those bundles are built artifacts; note the
drift in the report and leave the rebuild to whoever owns the UI build.

- [ ] evidence recorded first: for each deleted subcommand, last-commit date + reference count + the `git log -S` line, in the commit body (coding standard: never delete on "unused" alone)
- [ ] web dashboard verified to call `server` HTTP, with the handler cited, before any deletion
- [ ] `trace`, `observatory`, `dashboard`, `eval-chains` survive as aliases into `chains` for one release
- [ ] `ailang chains --help` lists the folded subcommands; the eight zero-reference `observatory` subcommands are gone
- [ ] `go test ./cmd/ailang/... ./internal/observatory/... -count=1` green; chains tests (`chains_list_options`, `chains_live`, `chains_read_backend`, `chains_stats_cvs`) unchanged and passing

### M4 — D7 removals, the non-chains set

**Re-derive the evidence; the audit's dates are stale.** D7 (program doc line 114) is a
*deleted vs hidden* decision per command, and Phase 3 item 2 already assigns `axioms` and
`policy-check` to the hidden `dev` group — so those two are **hidden, not deleted**, despite the
flat "D7 removals" phrasing in the handover.

Measured 2026-09-17 (`git grep` over tracked files; exec = `Makefile make/ tools/ scripts/ .github/`):

| command | file | last **substantive** commit | LOC | exec | skills | docs |
|---|---|---|---|---|---|---|
| `access-control` | `access_control.go` | **2026-02-09** | 302 | 0 | 0 | 0 |
| `eval-trend` | `eval_trend.go` | 2026-06-12 | 285 | 0 | 0 | 0 |
| `axioms` | `axioms.go` | 2026-03-26 | 306 | 0 | 2 | 0 |
| `budget` | `budget.go` | 2026-09-08 | 608 | 0 | 0 | 0 |
| `storage` | `storage.go` | 2026-09-15 (S4 plane switch) | 296 | 0 | 0 | 2 |
| `policy-check` | `policy_check.go` | **2026-09-16 — LIVE** | 126 | 0 | 0 | 0 |
| `eval-censored-pairs` · `eval-sweet-spot` · `watch` | no dedicated file | — | — | 0 | 1 | 7 |

Two corrections that matter:

- **`access-control`'s "2026-04-21"** in the audit is commit `de4aa7ba8`, the module-path rename — a
  mechanical sweep, not a change to the command. Its 2026-09-15 touch is S4's env-var migration, also
  mechanical. The real last substantive commit is **2026-02-09**. Every date in the audit taken after
  S1–S4 is suspect this way: `git log` the file and skip the sweeps.
- **`policy-check` must NOT be deleted.** Commit `6236deb24` (2026-09-16) extracted `admitProgram`
  from it to share with the new `ailang run --policy`, and `policyCheckOutput`'s JSON field names are
  **part of the runner contract** (M-AGENT-SAFE-RUNNER; renaming them needs a message-schema bump).
  It goes to the hidden `dev` group per Phase 3 item 2. `axioms` likewise — hidden, not deleted.

- [ ] every removal commit body carries: last-commit date, live-caller count, `git log -S` output
- [ ] any command whose count is non-zero in `tools/`, `make/`, `.claude/skills/**`, `.github/` is **not** removed this sprint — it is reported for the caller sweep instead
- [ ] docs citing removed commands updated in the same commit (`check-referenced-paths` green)
- [ ] `make lint` green — no orphaned helpers left behind (and none deleted on "unused" alone)

### M5 — Four flag families (~200 LOC)

`--json` (58 existing sites) replaces `--format json`, `--pretty`, `--stream-json`; `--model`
(`--models` alias on `eval suite`); `--output`/`-o`; `--dry-run` meaning **"default acts"**. Old
spellings keep working for one release and warn once. The other six families are Future Work — do
not touch time windows, timeouts, IDs, namespaces, `--version`'s six meanings, or snake_case leaks.

- [ ] one shared registration helper; `flag_names_distinct` in the metrics snapshot drops
- [ ] each superseded spelling accepted with a single deprecation line on **stderr**, never stdout (JSON consumers)
- [ ] `--dry-run` audited at every site: any site where the default was already "don't act" is listed in the report, not silently flipped
- [ ] snapshot diff shows no stdout change for any existing invocation

### M6 — Generated CLI reference + metrics rewire (lands last)

- [ ] `docs/docs/reference/cli.md` generated from the table by a make target, with a CI diff gate; `docs/sidebars.js` updated; the 11-line orphan `docs/docs/guides/cli.md` redirected or deleted
- [ ] docs citing commands the binary rejects fixed: `eval-chains`, `daemon` (7 cites), `disk`
- [ ] `tools/simplicity_metrics.sh` `commands_top_level` reads the dispatch table (not `grep 'case "'` on main.go) and its `how` string updated; **counts visible top-level entries only**
- [ ] new metric `help_exit0_rate` = share of commands+groups where `--help` exits 0, gate 100
- [ ] `make simplicity-audit` shows `commands_top_level` ≤ 20 and no other gate regressed (exit 2 = stop and name it in the changelog)

## Close-out (orchestrator, in the main checkout)

`go build ./internal/... ./cmd/ailang/... ./tools/...` → `go vet` → `make lint` →
`go test ./internal/... ./cmd/ailang/... -count=1` (~5 min; `internal/lsp` deadline and `internal/pkg`
500 ms cancellation flake under load — re-run alone before believing a failure) → the gates
`check-boundaries check-architecture-closure check-file-sizes check-changelog check-referenced-paths
check-skills check-context-docs check-golden-drift fmt-check check-tmpfile-hygiene
check-home-isolation check-prompt-freeze verify-examples test-imports` → `make test-launchd-drivers`
→ `make simplicity-audit` → `scripts/gen_architecture_closure.sh` → sprint JSON `passes`/`notes` →
commit → push → memory.

## Explicitly NOT in this sprint

- The **caller sweep** (renaming `ailang messages` → `ailang ops messages` in drivers, make, skills,
  workflows). Separate PR, after aliases ship in a tagged release, gated on `make test-launchd-drivers`
  and one attended iteration of each mission loop.
- **Phase 3b** (the `ailang` / `ailang-ops` split). Five preconditions in the program doc; the first
  one this sprint can satisfy is "aliases shipped in a tagged release".
- Phase 4's four gates (`packages_without_doc`, `skill_trees`, `instruction_surface_bytes`,
  `tracked_files`) and the six other flag families.

## Two traps measured in THIS sprint, before a line of code was written

**`make test-launchd-drivers` fails under concurrent compilation, and the failure looks like a real
regression.** Measured 2026-09-17: run while a worktree agent was building the tree (load avg 5.85),
the suite died at `bound derivation responds to a slowed stimulus: helper rc=199`
(`tools/eval/test_motoko_connection_probe.sh:1022`). `rc=199` is `run_bounded` hitting its 10s cap.
Re-run alone on the same box: **59/59 arms pass, exit 0**, `fork rate 590/s drift=none`. The arm
measures whether a deliberately slowed stimulus yields a measurably lower fork rate, so CPU
contention is indistinguishable from the thing it measures. The suite is otherwise load-robust by
design (a sibling arm is documented as "0 failures in 8 local runs, quiet and under 8x CPU
contention") — this one arm is not. **Never run this target while an agent is compiling**, and do
not chase it as a regression before running the quiet control. Note also that it sits second-to-last
in the target, so when it dies the four bash-3.2 `-n` syntax checks after it never run at all.

**A piped `make` reports the pipe's exit code, and the harness believes it.** `make <gate> 2>&1 |
tail -25` reported success while `make: *** Error 1` sat in the scrolled output; the background-task
notification said "exit code 0" because that was `tail`'s. Same family as the `grep -q`/`pipefail`
trap below, but it bites the *gates*, which is worse — a gate that reports green when it failed is
the one thing a gate may not do. Run gate commands unpiped, or capture `$?` from `make` itself
before anything else touches the stream.

## Traps carried forward (each cost real time — see the handover)

ugrep shadows `grep` interactively, BSD grep in scripts (`grep -q` on a pipe under `pipefail` reads
FALSE); another agent's `git commit` in the shared checkout sweeps your staged files (stage+commit in
ONE command); `tools/launchd/install_*.sh` re-renders plists and drops hand-edits; test failure
messages that print header values print credentials; a retired env var in a long-lived shell breaks
language commands if the plane resolves eagerly; `.golangci.yml` exclusions can be silently ignored
(prove a rule with a canary); a gate that misses by a definitional amount is moved ONCE, with reasons.
