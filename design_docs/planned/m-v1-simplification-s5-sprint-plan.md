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

Re-measured at HEAD (`grep -rhoE "ailang <cmd>( |\"|$)"` over `Makefile make tools .claude .github
scripts docs/docs internal cmd`, excluding design_docs/changelogs/state):

| command | live-caller refs | command | live-caller refs |
|---|---|---|---|
| `messages` | 1,463 | `observatory` | 143 |
| `coordinator` | 769 | `eval-report` | 133 |
| `chains` | 510 | `eval-matrix` | 123 |
| `eval-suite` | 427 | `dashboard` | 86 |
| `trace` | 179 | `eval-chains` | 74 |

These are higher than the counts in the Phase 3 audit (which excluded `.claude/skills`); the
conclusion is the same and stronger. **Aliases ship before any caller is renamed, and the caller
sweep is not in this sprint.**

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
- [ ] `make test-launchd-drivers` green (bash 3.2: pin-root + routing + notices + hook stdout)
- [ ] snapshot diff: alias paths byte-identical to `ailang-before` on stdout; help text is the only intended change

### M3 — Fold `trace` + `observatory` + `dashboard` + `eval-chains` into `chains` (−~5,000 LOC)

`chains` is what the fleet uses (510 refs vs 86 for `dashboard`). Fold the useful subcommands in;
delete the rest under D7 with `git log -S` evidence in each commit body. **Before deleting anything**,
confirm the web dashboard reaches these code paths through `server` HTTP and not through the CLI —
if it does not, the milestone stops and reports.

- [ ] evidence recorded first: for each deleted subcommand, last-commit date + reference count + the `git log -S` line, in the commit body (coding standard: never delete on "unused" alone)
- [ ] web dashboard verified to call `server` HTTP, with the handler cited, before any deletion
- [ ] `trace`, `observatory`, `dashboard`, `eval-chains` survive as aliases into `chains` for one release
- [ ] `ailang chains --help` lists the folded subcommands; the eight zero-reference `observatory` subcommands are gone
- [ ] `go test ./cmd/ailang/... ./internal/observatory/... -count=1` green; chains tests (`chains_list_options`, `chains_live`, `chains_read_backend`, `chains_stats_cvs`) unchanged and passing

### M4 — D7 removals, the non-chains set

`access-control` (0 refs in code, 19 in docs/skills, last commit 2026-04-21), `eval-trend`,
`eval-censored-pairs`, `eval-sweet-spot`, `watch`, `axioms`, `policy-check`, `budget` CLI,
`models source|publish`, `storage` (a one-time migration). Each its own commit with dates and counts.

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

## Traps carried forward (each cost real time — see the handover)

ugrep shadows `grep` interactively, BSD grep in scripts (`grep -q` on a pipe under `pipefail` reads
FALSE); another agent's `git commit` in the shared checkout sweeps your staged files (stage+commit in
ONE command); `tools/launchd/install_*.sh` re-renders plists and drops hand-edits; test failure
messages that print header values print credentials; a retired env var in a long-lived shell breaks
language commands if the plane resolves eagerly; `.golangci.yml` exclusions can be silently ignored
(prove a rule with a canary); a gate that misses by a definitional amount is moved ONCE, with reasons.
