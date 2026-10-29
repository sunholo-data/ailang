# Phase 3a skill-resolution spike: findings (fleet iteration 34, 2026-10-13)

Plan: [`sprint-plan-phase3a-skill-resolution-spike.md`](../../planned/sprint-plan-phase3a-skill-resolution-spike.md).
Design: [M-HARNESS-MISSION-LOOP](../../planned/m-harness-mission-loop.md), Phase 3a, premises P1–P3.
Instrument: [`skill_resolution_spike.sh`](skill_resolution_spike.sh), built by the codex executor and
repaired by the controller (see "Instrument repairs" below). The controller ran it outside any sandbox.
pi was additionally probed by hand: [`pi-skill-invocation-probe.md`](pi-skill-invocation-probe.md).

**Rig**: claude 2.1.296 (`claude-haiku-5-5`), codex-cli 0.162.0 (`gpt-6.1-sol`), pi 0.85.1
(`ollama/kimi-k3:cloud`), macOS bash 3.2.57. The pin is the iteration worktree
`~/.ailang-driver-pin/fleet-i34` (a detached-style worktree of `origin/dev`, as every pinned mission
runs). "v1" is the main checkout `~/dev/sunholo-data/ailang`. "World" is a fresh
`/private/tmp/zz-fleet-spike-<runid>-world` directory with no `.claude/skills` in it or its ancestry.

**Method in one line.** Each marker skill carries per-run random 48-hex tokens: one in `SKILL.md`, one
in its relative `resources/probe.md`. The same skill name is planted in competing locations with
different tokens, so the returned pair names the location that won. Every group runs before-and-after
explicit-path positive controls for each planted location and a negative control (a name planted
nowhere). Only the final assistant answer is scored. A tool trace showing a directory search voids a
discovery verdict.

## Results (subject verdicts; full tables in `runs/<runid>/verdicts.tsv`)

### P1: claude (`claude -p`), runs `1791874306-…` (51/51 controls passed) and `1791873918-…` (repetition, valid arms only)

| Arm | Planted | cwd / flags | Result | Reading |
|---|---|---|---|---|
| C-project | P (pin `.claude/skills`) | pin | MATCH P/P (run 3918); INCONCLUSIVE (run 4306: the native `Skill` tool was used on the project marker, but haiku mis-copied one hex digit of a token) | project skills ARE discovered from cwd |
| C-user | U (`~/.claude/skills`) | World | MATCH U/U (both runs) | user skills discovered anywhere |
| C-collision | P + U, same name | pin | **U/U** (both runs) | **user-level wins a same-name collision** |
| C-v1 | P + U + V (main checkout) | v1 | **U/U** (both runs) | user-level beats both project copies |
| C-world | U + X, `--add-dir XP` | World | **U/U** | user-level beats an `--add-dir` copy |
| C-add-only | X only, `--add-dir XP` | World | MATCH X/X | `--add-dir <dir>` DOES load `<dir>/.claude/skills` |
| C-settings | P + U, `--setting-sources ''` | pin | ABSENT (controls fired) | empty setting sources disables BOTH user and project skill discovery |
| C-auth-base | — | World, `CLAUDE_CONFIG_DIR` unset | AUTH_OK, `apiKeySource: none` | subscription auth |
| C-auth-isolated | — | World, `CLAUDE_CONFIG_DIR=<fresh dir>` | **AUTH_OK**, `apiKeySource: none` | a per-fire config dir keeps subscription auth (**see caveat**) |
| C-synthetic | P + S (`$CLAUDE_CONFIG_DIR/skills`) | pin, `CLAUDE_CONFIG_DIR=<fresh>` | **S/S** | the config dir's `skills/` is the user level and wins over project |

**Caveat on auth.** The environment the spike inherited exports `CLAUDE_CODE_OAUTH_TOKEN` (names-only
listing in `runs/1791874306-…/metadata.txt`; values never recorded). The isolated-config auth success
is therefore a reading under the **token-in-env** condition, which is how the loops run today. A
keychain-only process (no `CLAUDE_CODE_OAUTH_TOKEN`) was **not** measured.

### P2: codex (`codex exec`), run `1791874447-…` (32/43 controls; every failure is named below)

| Arm | Planted | Result | Reading |
|---|---|---|---|
| K-claude | pin `.claude/skills` only | ABSENT (L+ controls MATCH, negatives ABSENT) | **codex does not discover `.claude/skills`** |
| K-project | pin `.agents/skills` | MATCH A/A | codex **discovers** project `.agents/skills` natively |
| K-legacy-user | `$CODEX_HOME/skills` (= `~/.codex-missions/skills` under the mission env) | MATCH H/H | discovers `$CODEX_HOME/skills` |
| K-agents-user | `~/.agents/skills` | subject B/B; verdict INSTRUMENT_FAILURE because the pre-control hit `Selected model is at capacity` | likely discovered; **one control lost to provider capacity** |
| K-collision | A + B + H, same name | **H/H** | `$CODEX_HOME/skills` wins over project and `~/.agents` |
| K-by-path | A + B + H, prompt names A's path | subject A/A; verdict INSTRUMENT_FAILURE: one post control lost to capacity, one rejected by the scorer (a newline-joined `cat` it does not parse) | an explicit path read reaches the pin copy despite the collision |
| K-home | `CODEX_HOME=<fresh dir>` | all calls `401 Unauthorized` | **a per-fire CODEX_HOME cannot authenticate without `auth.json`**, which the plan forbade copying |
| K+/K- (baseline) | — | `Selected model is at capacity` | provider capacity, not a reading; the same controls passed in every other group |

So P2's answer: codex **does** discover skills (it does not merely follow a by-path instruction), from
project `.agents/skills` and `$CODEX_HOME/skills`, with `$CODEX_HOME` winning a collision. It never
looks at `.claude/skills`.

### P3: pi, run `1791874311-…` (44/45 controls) plus the hand probe

| Arm | Result | Reading |
|---|---|---|
| I-project, I-collision, I-world, I-no-skills (discovery prompt) | ABSENT throughout (controls fired) | pi in `--print` mode discovers **no** skills: not pin `.agents/skills`, not `~/.agents/skills`, not `$PI_CODING_AGENT_DIR/skills` |
| I+ (`--skill <abs>/SKILL.md` + discovery prompt) | ABSENT → INSTRUMENT_FAILURE | `--skill` does not put the skill into a catalog the model sees |
| I-explicit | model found the file by `ls`/`find`/`grep` → INVALID_DISCOVERY | same reading |
| hand probe rows 6–9 | `/skill:<name>` with `--skill <file or dir>` returns the token; without `--skill` it is "not found" | **`--skill <abs path>` loads the skill; the controller must invoke it as `/skill:<name>`** (or be told the path) |

## The two named cases

- **World** (no project `.claude/skills`): a claude controller gets user-level skills. `--add-dir <pin>`
  adds the pin's `.claude/skills` (C-add-only X/X), but a same-name user-level copy still wins
  (C-world U/U). Retiring the user-level mission-skill symlinks without supplying `--add-dir` or a
  per-fire config dir would leave World with none.
- **v1** (cwd = a checkout whose tree differs from the pin): the user-level copy wins over BOTH the v1
  and the pin project copies (C-v1 U/U, two runs). With user-level retired, a claude controller would
  read the **cwd's** project copy. The charter's Repo Profile states that v1 now runs in a pin worktree
  (`pin-root.sh _set_pin_workdir`), which supersedes the design doc's V8. This spike did not re-measure
  v1's driver cwd.

## Per-provider pinnability (what Phase 3b can build on)

| Provider | Pin-rooted skill source available? | Mechanism the measurements support |
|---|---|---|
| claude | **yes** | (a) a per-fire `CLAUDE_CONFIG_DIR` whose `skills/` points at the pin (S won; auth held under the env token); or (b) remove the user-level mission-skill symlinks so the cwd=pin project copy is the only one (C-project), plus `--add-dir <pin>` for World |
| codex | **yes, conditionally** | pin `.agents/skills` is discovered, but only while `$CODEX_HOME/skills` holds no same-name copy (H wins). The `.agents` mirror is the tracked, already-diverged copy (#544), so it must be kept in parity. A per-fire `CODEX_HOME` is **not** available without copying credentials |
| pi | **yes, explicit only** | `--skill <abs pin path>` per skill, plus a prompt that invokes `/skill:<name>` or names the path. No catalog discovery |

## Instrument repairs made by the controller (all recorded; generator ≠ judge applies to them)

1. Real-config claude arms ran with `CLAUDE_CONFIG_DIR=$HOME/.claude` set explicitly, which is not the
   driver's condition: claude then looked for `~/.claude/.claude.json`. They now run with the
   variable **unset**.
2. The scorer's `json.loads` rejected answers wrapped in one markdown code fence. It now strips one fence.
3. `K:H` pointed at `~/.codex/skills`; the mission env sets `CODEX_HOME=~/.codex-missions`. It now
   points at `$CODEX_HOME/skills`.
4. The synthetic pi agent dir lacked `models.json`, so no model could resolve. It now symlinks (never
   copies) `models.json` and `auth.json`.
5. `--add-dir` is variadic and swallowed the prompt. Extra arguments now precede a fixed option.
6. A negative control that asked the `Skill` tool for its own unplanted name was scored as a "leak".
   That is now allowed for the run's `-n<k>` names only.
7. Auth scoring read `apiKeySource` from a hook event, not the `init` event, and demanded exactly 1
   output token (the API reports 3). It now uses the init event and allows ≤5.
8. A shared parent (`/private/tmp`) changed under a concurrent writer (World iteration 256's
   `iter256_merge_body.txt`). A delta that does not name this run's id is now `CONCURRENT_CHANGE`; one
   that does is still `SAFETY_FAILURE`.
9. pi 0.85.1 emits a trailing `agent_settled` event, while the scorer required `agent_end` to be last.
   It now looks for the last `agent_end`.
10. Codex reads `CLAUDE.md` first because the repo's `AGENTS.md` tells it to. Reads of `CLAUDE.md` and
    `AGENTS.md` in cwd are no longer counted as skill search.
11. The fence (`sandbox-exec`, writes denied) stops codex starting (`failed to initialize in-process
    app-server client: Operation not permitted`, run `1791874261-…`). Codex therefore ran with
    `--fence off`; claude and pi ran fenced. Under the fence, claude's SessionStart hooks fail with
    EPERM and the session proceeds.

12. Judge N2: after repair 8, a REMOVED pre-existing entry in a non-world parent was classed
    `CONCURRENT_CHANGE`. It is now `SAFETY_FAILURE` unless the removed entry is another spike run's
    `zz-fleet-spike-*` marker. Drill, run by the controller: an injected foreign deletion in
    `~/.claude/skills` gives `SAFETY_FAILURE`, rc 1; an injected foreign addition gives
    `CONCURRENT_CHANGE`, rc 0. The judge had measured the deletion case as `CONCURRENT_CHANGE`/`safety=OK`
    before this fix. No live run is affected, because none of their deltas removed a non-spike entry.

Aborted runs (`1791873811`, `1791874255`, `1791874263`, `1791874309`) were stopped on these defects
and are not cited. Their raw streams, with those of every live run, are kept **outside the repo** at
`~/.ailang/state/fleet-i34-spike-raw/` on the rig. The raw claude stream-json init events carry the
user's full skill listing, including personal addresses, so only distilled per-call artifacts are
tracked here: verdict, final answer, provenance, timing, rc and argv.

**Safety.** Every live run's audits are `OK` or `CONCURRENT_CHANGE`. No audit delta names **that
run's own** id; deltas in runs 4306, 4311 and 4447 name the overlapping sibling runs' markers
(judge N1). After all runs, the controller checked the real parents (`~/.claude/skills`,
`~/.agents/skills`, `~/.codex-missions/skills`, the main checkout's `.claude/skills`, `/private/tmp`)
for `zz-fleet-spike-*` and found **none**.

## Independent judge (sonnet, Agent tool, own worktree): PASS 88/100, 0 blocking

Report: [judge-report.md](judge-report.md). The judge re-derived the load-bearing verdicts from
`tokens.tsv` and the raw streams, and **replicated the three decision-relevant claims with its own
minimal method** (not this script):
- **C1**: claude collision → user-level token, 3 of 3; the negative control gave "Unknown skill".
- **C2**: codex → `.claude/skills`-only gave ABSENT, `.agents/skills`-only returned its token, and the
  negative gave ABSENT.
- **C3**: pi `--skill <dir>` plus `/skill:<name>` → token; without `--skill`, no token.

**Scope (judge N3, adopted).** Each collision precedence is a small-n reading on one model per CLI.
claude "user wins": haiku, 2 spike runs plus 3 judge calls. codex "`$CODEX_HOME` wins": 1 arm. They
are observations under the stated conditions, not universal precedence rules. K-by-path is an
INSTRUMENT_FAILURE verdict: its subject read A/A, but a control was lost.
