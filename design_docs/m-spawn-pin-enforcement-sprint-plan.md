# Sprint Plan — M-SPAWN-PIN-ENFORCEMENT

**Design doc**: [`design_docs/m-spawn-pin-enforcement.md`](m-spawn-pin-enforcement.md)
**Worktree**: `/Users/voightkampff/.ailang-driver-pin/.wt-v1-iter324-spawn-pin`
**Branch**: `sprint/iter324-spawn-pin-enforcement` (HEAD `bf7f0fb4f`, based on `origin/dev` `08ab6ba7c`)
**Planned**: 2026-09-03 (V1 mission-control iteration 324, planner role)
**Risk**: medium — no compiler/runtime code, but it edits a shared skill, `.claude/settings.json`
and the fleet driver; a wrong hook branch denies every spawn on four missions.
**Reported at #1012** (do not convert to an auto-close keyword).

---

## 0. How to read this plan

Every acceptance criterion below gives (a) the **exact command**, (b) the **expected output**, and
(c) the **observed result on the pristine tree at `08ab6ba7c`**, measured by the planner on
2026-09-03. Rule 3e: a gate that is already red at base measures the repo, not the change — so a
criterion marked `RED at base` is a criterion this sprint must turn green, and a criterion marked
`GREEN at base` is a regression guard that must **stay** green.

**Velocity**: this loop lands one milestone-group per iteration. There are no day numbers in this
plan on purpose. Each milestone is sized for ONE bounded executor run of ≤30 minutes on the
flat-rate DeepSeek-V4-Flash lane driven by pi. Directives are prescriptive (exact file, exact
strings, exact arms) because that lane executes prescriptive work and fails judgment-heavy work.

---

## 1. Baseline measurements (pristine tree, `08ab6ba7c`, 2026-09-03)

| # | Command | Observed | Verdict |
|---|---|---|---|
| B1 | `bash tools/launchd/test_mission_routing.sh` | `==== 36 passed, 0 failed ====`, rc=0 | **GREEN at base** — must stay green |
| B2 | `PATH=/usr/sbin:$PATH make test-launchd-drivers` | ends `launchd drivers: tests + bash 3.2 syntax OK`, rc=0 (the probe self-test needs `lsof` from `/usr/sbin`) | **GREEN at base** — must stay green |
| B3 | `ls tools/launchd/resolve-role-spawn.sh` | `No such file or directory` | **RED at base** (M1 creates it) |
| B4 | `ls tools/launchd/spawn-pin-hook.sh` | `No such file or directory` | **RED at base** (M2 creates it) |
| B5 | `ls tools/launchd/test_spawn_pin_hook.sh` | `No such file or directory` | **RED at base** (M2 creates it) |
| B6 | `grep -c 'test_spawn_pin_hook.sh' make/test.mk` | `0` | **RED at base** (M2 wires it) |
| B7 | `grep -c 'spawn-pin-hook' .claude/settings.json` | `0` | **RED at base** (M2 wires it) |
| B8 | `grep -c 'MISSION_CONTROL_ACTIVE' tools/launchd/mission-control.sh` | `0` | **RED at base** (M3 adds it) |
| B9 | `grep -n 'MISSION_PLANNER_ALLOWLIST' tools/launchd/mission-env/mission-docs.env \| grep -c '\|scripts/\*'` | `0`; **known-positive control** `... \| grep -c '\|docs/\*'` → `1` | **RED at base** (M3 widens it) |
| B10 | `grep -c 'MISSION-ROLE' .claude/skills/mission-control/SKILL.md` | `0` | **RED at base** (M4 adds it) |
| B11 | arm-12 behaviour, both directions: a doc declaring `scripts/verify_examples.go` under the **current** docs allowlist → `opus fail-closed:path-not-in-codex-allowlist`; under `…\|scripts/*` → `codex:gpt-5.6-luna declared:codex-ok` | exactly as stated (measured with `derive-planner-lane.sh` and a temp fixture) | **RED at base** in the widened direction; the unwidened direction is the control and is already correct |
| B12 | `command -v jq` | `/usr/bin/jq`, `jq-1.7.1-apple` | **GREEN at base** — jq is present on the rig |

### 1.1 Facts the planner verified that CHANGE the design doc

These are loop successes — the design doc was wrong or under-specified in four places. **The plan
below follows the corrected fact, not the doc.**

1. **`make test-launchd-drivers` does NOT pick up a new sibling test script automatically.**
   The target names each suite explicitly (`make/test.mk:52-58`). Only the bash-3.2 *syntax* sweep
   (`for f in tools/launchd/*.sh`) is a glob. So `tools/launchd/test_spawn_pin_hook.sh` **must be
   wired** into `make/test.mk`, and that wiring itself needs a guard arm (M2, arm W). The design
   doc's "invoked by `make test-launchd-drivers`" read as automatic; it is not.
   CI job **confirmed**: `.github/workflows/ci.yml:559-579`, job name `launchd drivers (bash 3.2)`,
   `runs-on: macos-latest`, step `run: make test-launchd-drivers`, with a guard step that fails the
   job if `/bin/bash` is no longer 3.x.

2. **Layer 3's exports must NOT go "beside the existing role exports".** The design doc points at
   lines 544/569/586/630/637/793/821 (all **verified correct** by the planner) and says to put
   `MISSION_<ROLE>_RESOLVED/PATH` there. That would export the **pre-degradation** values. The
   driver's codex lane-probe loop rewrites `MISSION_<ROLE>_MODEL` in place at
   `mission-control.sh:722` (`printf -v "$var" '%s' "$fb"; export "$var"`) and the pi loop again at
   `:770` and `:779`; the one-shot override rewrites it at `:904`; `LANE DEGRADED` is logged at
   `:943`. The only correct insertion point is **immediately before the `roles:` log line at
   `:1003`**, which is where the driver itself reports the bytes it verified.

3. **The design doc's line-904 gloss is wrong.** It calls `:904` "per-fire degradation rewrites
   `MISSION_EXECUTOR_MODEL`". Line 904 is the **one-shot executor override** consumed from
   `~/.ailang/state/mission-executor-model-once`. Per-fire lane degradation happens at `:722` /
   `:770` / `:779`. Harmless to the design, but the executor must not use `:904` as an anchor.

4. **The docs mission's planner allowlist is already widened well past "infra-only".** It is
   `tools/*|.claude/skills/mission-control/SKILL.md|.claude/skills/design-doc-creator/*|docs/*|examples/*|README.md|CHANGELOG.md|.claude/skills/docs-sync/scripts/*`.
   `scripts/*` is genuinely absent (B9/B11 prove it both directions), so the widening is still
   needed — but it is an **append of one alternative**, not a rewrite, and the entry
   `.claude/skills/docs-sync/scripts/*` must NOT be mistaken for it.

5. **Under-specified in the doc, pinned here:** the resolver's re-route emission had no target.
   The doc says "emit the re-route instead" without saying to what. This plan makes it deterministic:
   the head of `MISSION_EVALUATOR_FALLBACK`, using the driver's own chain semantics. If that chain is
   empty the resolver refuses rather than inventing a model.

---

## 2. Hard constraints for the executor (read before touching anything)

- **bash 3.2.57** everywhere in `tools/launchd/`. NO `declare -A`, NO `${v,,}` (use
  `tr 'a-z' 'A-Z'`), NO `mapfile`, NO `${var^^}`. `printf -v`, `${!var}` indirect expansion and
  `set -f` ARE available and are already used in `mission-control.sh` — copy those idioms.
- **`jq` MAY be used** (B12: present on the rig; present on `macos-latest` runners; existing
  precedent in `tools/launchd/lib/pin-root.sh:230`, `tools/launchd/nightly-lang-eval.sh`,
  `scripts/hooks/format_ail.sh:16`). But the hook MUST carry a **fail-closed** branch: if
  `command -v jq` fails **and** `MISSION_CONTROL_ACTIVE=1`, the hook emits a **deny** with reason
  token `fail-closed:jq-missing`. A missing parser is never an allow while a mission is running.
- **Every hook branch exits 0 with explicit JSON on stdout.** A non-zero exit or malformed JSON is
  an ALLOW in Claude Code semantics — i.e. a silently disabled safeguard. `set -e` is FORBIDDEN in
  the hook. Use `set -u` only.
- **Do NOT run any git write command.** No `git add`, `commit`, `stash`, `checkout`, `branch`,
  `push`, `reset`, `clean`. The controller commits per milestone from `.snap/M<k>/` snapshots.
- **Snapshot after each milestone, cumulatively**: after finishing M<k>, copy **every file you
  created or modified so far in this sprint** (not just this milestone's) into `.snap/M<k>/`,
  preserving repo-relative paths, e.g.
  `mkdir -p .snap/M2/tools/launchd && cp tools/launchd/spawn-pin-hook.sh .snap/M2/tools/launchd/`.
- **Do NOT edit `.claude/skills/mission-control/SKILL.md` outside the paragraphs quoted in M4.**
  It is 2819 lines of shared fleet policy; an incidental reflow there is a fleet-wide change.
- **Do NOT touch `scripts/hooks/coordinator_hook.sh`.** It is the existing advisory PreToolUse hook
  (25 lines; POSTs the payload to the coordinator daemon and always exits 0). The new hook runs
  **alongside** it — measured in the design's M5/M7: both fire and the deny wins.
- **Do NOT edit `~/.config/ailang/mission-docs.env`.** That file is outside the repo and outside
  the worktree; deploying it is the controller's job. Edit only the versioned copy
  `tools/launchd/mission-env/mission-docs.env`.
- Run every command from inside the worktree. Never `cd` into
  `/Users/voightkampff/dev/sunholo-data/ailang`.

---

## 3. Frozen interface contracts (copy these verbatim — do not improvise)

### 3.1 Resolver CLI — `tools/launchd/resolve-role-spawn.sh <role> [<design-doc-path>]`

Always exits **0**. Emits exactly **one line** on stdout, in the
`derive-planner-lane.sh` convention `<value…> <reason-token>`:

| Emission | When |
|---|---|
| `recipe <provider:model> declared:provider-pin` | `MISSION_<ROLE>_MODEL` contains `:` |
| `agent-tool <alias> declared:alias-pin` | `MISSION_<ROLE>_MODEL` is a bare alias, no collision |
| `reroute <alias> generator-equals-judge` | role=`evaluator`, its alias equals the executor's resolved model, and `MISSION_EVALUATOR_FALLBACK` has a head — `<alias>` is that head |
| `refuse fail-closed:evaluator-collision-no-fallback` | same collision, but `MISSION_EVALUATOR_FALLBACK` is unset/empty |
| `refuse fail-closed:<role>-model-missing` | `MISSION_<ROLE>_MODEL` unset or empty |
| `refuse fail-closed:role-unknown` | `$1` not in `{designer,planner,executor,evaluator}` |
| `refuse fail-closed:role-missing` | `$1` absent |
| `refuse fail-closed:derive-script-missing` | role=`planner` and `tools/launchd/derive-planner-lane.sh` is not executable |

**Planner role is special and CONSUMES `derive-planner-lane.sh` verbatim** (design's Conflict
Surface; M2 counts 0/0/0/5). It does not re-derive. Take that script's single output line
`<lane> <reason-token>` and map it:

- output begins `opus ` → emit `agent-tool opus <its reason-token verbatim>`
- output begins `<provider>:<model> ` → emit `recipe <provider>:<model> <its reason-token verbatim>`

The reason token is **copied through unchanged** so the Gate-4 evidence row keeps naming the real
cause (`fail-closed:path-not-in-codex-allowlist`, `anthropic-fallback:*`, `declared:codex-ok`, …).

**Resolved executor model** (used by the evaluator collision check) is read as
`${MISSION_EXECUTOR_RESOLVED:-${MISSION_EXECUTOR_MODEL:-}}` — `_RESOLVED` when M3 has landed,
the raw model otherwise. **Chain head** = the substring of `MISSION_EVALUATOR_FALLBACK` before the
first comma (mirror `_chain_head` in `mission-control.sh`).

### 3.2 Hook stdin schema — `tools/launchd/spawn-pin-hook.sh`

Claude Code writes the PreToolUse payload to **stdin** (measured live, design M1/M7). The hook reads
**only** these five fields and ignores everything else:

```json
{
  "hook_event_name": "PreToolUse",
  "tool_name": "Agent",
  "tool_input": {
    "model": "sonnet",
    "subagent_type": "general-purpose",
    "prompt": "MISSION-ROLE: executor\nInvoke the sprint-executor skill for ...",
    "description": "Execute sprint M1"
  }
}
```

jq accessors: `.tool_name`, `.tool_input.model // ""`, `.tool_input.subagent_type // ""`,
`.tool_input.prompt // ""`, `.tool_input.description // ""`.

### 3.3 Hook decision order (implement in exactly this order)

1. `PAYLOAD=$(cat)`.
2. **Marker gate.** If `"${MISSION_CONTROL_ACTIVE:-}" != "1"` → **allow**, reason token
   `passthrough:marker-absent`. (Status quo for attended sessions and other repos is untouched.)
3. **Parser gate.** `command -v jq >/dev/null 2>&1` fails → **deny**,
   token `fail-closed:jq-missing`.
4. Parse the five fields. If `tool_name` is neither `Agent` nor `Task` → **allow**,
   token `passthrough:not-a-spawn`.
5. **Role token.** Match `^MISSION-ROLE:[[:space:]]*([A-Za-z-]+)[[:space:]]*$` against the FIRST
   line of `.tool_input.prompt`; if no match there, match the same pattern against the WHOLE of
   `.tool_input.description`. Lower-case the captured value with `tr 'A-Z' 'a-z'`.
   The hook reads **nothing else** — it never infers a role from prose (design M8 measured a false
   positive doing exactly that).
6. **No token found**: if `subagent_type == "Explore"` → **allow**, token `explore-readonly`.
   Otherwise → **deny**, token `fail-closed:role-missing`.
7. **Token found but not in `{designer,planner,executor,evaluator}`** → **deny**, token
   `fail-closed:role-unknown`. (Token presence beats the Explore exception: Explore only excuses
   ABSENCE.)
8. `PIN=$(eval echo \"\${MISSION_$(echo "$role" | tr 'a-z' 'A-Z')_MODEL:-}\")` — or the safer
   bash-3.2 indirect form `_v="MISSION_${ROLE_UC}_MODEL"; PIN="${!_v:-}"`. Empty → **deny**, token
   `fail-closed:<role>-model-missing`.
9. `PIN` contains `:` → **deny**, token `deny:provider-pin`. This does **not** look at
   `tool_input.model` at all, so retrying with a different alias is denied identically.
10. Bare alias, role `evaluator`, and `PIN` equals `${MISSION_EXECUTOR_RESOLVED:-${MISSION_EXECUTOR_MODEL:-}}`
    → **deny**, token `deny:generator-equals-judge`.
11. Otherwise → **allow**, token `allow:alias-pin`.

### 3.4 Hook output JSON (exact)

Deny:
```json
{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"<REASON>"}}
```
Allow:
```json
{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow","permissionDecisionReason":"<REASON>"}}
```
One line on stdout, `exit 0`, in **every** branch. Build it with `printf '%s'` on a
jq-constructed string (`jq -nc --arg r "$REASON" '…'`) so a reason containing a quote cannot
produce malformed JSON — malformed JSON is an allow.

### 3.5 Reason strings (verbatim — the executor must not reword these)

| Token | `permissionDecisionReason` |
|---|---|
| `deny:provider-pin` | `<role> is pinned to <pin>; Agent-tool alias spawn refused — use the cross-provider recipe (resolve-role-spawn.sh <role>)` |
| `fail-closed:role-missing` | `fail-closed:role-missing — MISSION_CONTROL_ACTIVE=1 and this Agent/Task call carries no role token. Add "MISSION-ROLE: <designer\|planner\|executor\|evaluator>" as the FIRST line of the prompt, or use subagent_type: Explore for a read-only reality-check.` |
| `fail-closed:role-unknown` | `fail-closed:role-unknown — role token '<value>' is not one of designer, planner, executor, evaluator` |
| `fail-closed:<role>-model-missing` | `fail-closed:<role>-model-missing — MISSION_<ROLE>_MODEL is unset; no default is applied` |
| `deny:generator-equals-judge` | `evaluator alias '<alias>' equals the executor's resolved model; generator!=judge — re-route the evaluator (resolve-role-spawn.sh evaluator)` |
| `fail-closed:jq-missing` | `fail-closed:jq-missing — the spawn-pin hook cannot parse the PreToolUse payload without jq; refusing to allow an unchecked mission spawn` |
| `passthrough:marker-absent` | `MISSION_CONTROL_ACTIVE is not 1 — spawn-pin enforcement inactive` |
| `passthrough:not-a-spawn` | `tool_name '<name>' is not Agent/Task` |
| `explore-readonly` | `subagent_type Explore — read-only agent, no role token required` |
| `allow:alias-pin` | `<role> alias pin '<pin>' permits the Agent-tool spawn path` |

Tests assert the **token** as a substring of the reason, so the prose may be checked loosely but
must contain the token literally.

### 3.6 Spawn-hook log

Path: `${HOME}/.ailang/state/mission-${MISSION_NAME:-v1}-spawn-hook.log`. Create the directory with
`mkdir -p` first. Append **one tab-separated line per decision**, and never let a log failure change
the verdict (`… >> "$LOG" 2>/dev/null || true`):

```
<ISO8601-UTC>\t<role>\t<pin>\t<tool_input.model>\t<subagent_type>\t<verdict>\t<reason-token>
```

- timestamp: `date -u +%Y-%m-%dT%H:%M:%SZ`
- empty/unknown fields are written as a single `-`
- `<verdict>` is literally `allow` or `deny`
- example:
  `2026-09-03T17:04:11Z	executor	codex:gpt-5.6-luna	sonnet	general-purpose	deny	deny:provider-pin`

Tests override the log path via `HOME=$tmpdir` so no arm writes to the real state directory.

---

## 4. Milestones

### M1 — Resolver `tools/launchd/resolve-role-spawn.sh` + its arms

**Files**: `tools/launchd/resolve-role-spawn.sh` (new, ~110 LOC),
`tools/launchd/test_mission_routing.sh` (extend, ~+60 LOC).

**Do**
1. Write the resolver to the contract in §3.1. Mirror `derive-planner-lane.sh`'s shape: an `emit()`
   helper that `printf`s one line and `exit 0`s; `set -f` wherever a glob could be matched; no
   `set -e`.
2. Extend `test_mission_routing.sh` with arms R1–R7 below, using the existing `ok`/`bad`/`want`
   helpers and the existing `$ROOT` convention. Append them **after** the last existing arm and
   before the `echo "==== $PASS passed, $FAIL failed ===="` footer.
3. Snapshot into `.snap/M1/`.

**Acceptance criteria**

| # | Command | Expected | At base |
|---|---|---|---|
| A1.1 | `bash tools/launchd/test_mission_routing.sh` | `==== 44 passed, 0 failed ====`, rc=0 (36 existing + 8 new: R1,R2,R3,R4,R4b,R5,R6,R7) | 36/0 — **must not lose an existing arm** |
| A1.2 | `/bin/bash -n tools/launchd/resolve-role-spawn.sh` | rc=0 | RED (file absent) |
| A1.3 | `MISSION_EXECUTOR_MODEL=codex:gpt-5.6-luna bash tools/launchd/resolve-role-spawn.sh executor` | `recipe codex:gpt-5.6-luna declared:provider-pin` | RED |
| A1.4 | `env -u MISSION_EXECUTOR_MODEL bash tools/launchd/resolve-role-spawn.sh executor` | `refuse fail-closed:executor-model-missing` | RED |
| A1.5 | `grep -c 'declare -A\|\${[A-Za-z_]*,,}\|mapfile' tools/launchd/resolve-role-spawn.sh` | `0` | RED (file absent) |
| A1.6 | `PATH=/usr/sbin:$PATH make test-launchd-drivers` | rc=0 | GREEN — must stay green |

**Test plan**

| Arm | Setup | Assertion | Kills which mutation |
|---|---|---|---|
| R1 | `MISSION_EXECUTOR_MODEL=codex:gpt-5.6-luna`, role `executor` | `recipe codex:gpt-5.6-luna declared:provider-pin` | delete the `:`-detection branch → **only R1 dies** (docs-10 replay at Layer 1; design test 9) |
| R2 | `MISSION_DESIGNER_MODEL=fable`, role `designer` | `agent-tool fable declared:alias-pin` | make every pin a `recipe` → only R2 dies |
| R3 | `MISSION_EVALUATOR_MODEL=sonnet`, `MISSION_EXECUTOR_MODEL=codex:gpt-5.6-sol`, role `evaluator` | `agent-tool sonnet declared:alias-pin` | over-eager collision check (always re-routing) → only R3 dies; this is R4's **false-positive control** (design test 5) |
| R4 | `MISSION_EVALUATOR_MODEL=sonnet`, `MISSION_EXECUTOR_MODEL=sonnet`, `MISSION_EVALUATOR_FALLBACK=pi:ollama/minimax-m3:cloud,pi:openrouter/minimax/minimax-m3`, role `evaluator` | `reroute pi:ollama/minimax-m3:cloud generator-equals-judge` | delete the generator≠judge check → only R4 dies (docs-9 replay; design test 10) |
| R4b | as R4 but `MISSION_EVALUATOR_FALLBACK` unset | `refuse fail-closed:evaluator-collision-no-fallback` | invent a re-route target when the chain is empty → only R4b dies |
| R5 | `MISSION_PLANNER_MODEL=pi:ollama/kimi-k3:cloud`, role `planner`, doc `tools/launchd/testdata/planner-lane/c-clean-infra.md` | `recipe pi:ollama/kimi-k3:cloud declared:codex-ok` — the reason token is `derive-planner-lane.sh`'s, copied through | re-derive the planner lane instead of consuming → only R5 dies (proves composition, not overlap) |
| R6 | `MISSION_PLANNER_MODEL=codex:gpt-5.6-sol`, role `planner`, doc `tools/launchd/testdata/planner-lane/a-unlisted-language-path.md` | `agent-tool opus fail-closed:path-not-in-codex-allowlist` | swallow or rewrite the derive reason token → only R6 dies |
| R7 | role `judge` | `refuse fail-closed:role-unknown` | default an unknown role to anything → only R7 dies (design test 11's sibling) |

**Anti-vacuity**: R3 and R4 are a matched pair (same evaluator alias, different executor). If either
passes for the wrong reason the pair disagrees.

---

### M2 — PreToolUse hook, its suite, and its wiring

**Files**: `tools/launchd/spawn-pin-hook.sh` (new, ~130 LOC),
`tools/launchd/test_spawn_pin_hook.sh` (new, ~190 LOC), `make/test.mk` (one line),
`.claude/settings.json` (one PreToolUse entry), `tools/launchd/test_mission_routing.sh` (+1 arm).

**Do**
1. Write the hook to §3.2–§3.6. Decision order is normative.
2. Write `tools/launchd/test_spawn_pin_hook.sh` in the same style as `test_mission_routing.sh`
   (`PASS`/`FAIL` counters, `ok`/`bad`/`want`, `==== N passed, M failed ====` footer,
   `[ "$FAIL" -eq 0 ]` as the last line). Each arm builds a payload with `jq -nc`, pipes it into the
   hook with the arm's env, and asserts `.hookSpecificOutput.permissionDecision` plus the reason
   **token** as a substring. Export `HOME="$tmp"` in every arm so the log lands in a temp dir.
3. Wire it into `make/test.mk`, immediately after the `test_mission_routing.sh` line:
   `@/bin/bash tools/launchd/test_spawn_pin_hook.sh`
4. Add the hook to `.claude/settings.json` under `hooks.PreToolUse` as a **second entry** in the
   array (leave the existing `coordinator_hook.sh` entry byte-identical and FIRST):
   ```json
   {
     "matcher": "Agent|Task",
     "hooks": [
       {
         "type": "command",
         "command": "\"$CLAUDE_PROJECT_DIR\"/tools/launchd/spawn-pin-hook.sh",
         "timeout": 5
       }
     ]
   }
   ```
5. Add arm W to `test_mission_routing.sh` (see table).
6. Snapshot into `.snap/M2/` (cumulative: M1's files too).

**Acceptance criteria**

| # | Command | Expected | At base |
|---|---|---|---|
| A2.1 | `bash tools/launchd/test_spawn_pin_hook.sh` | `==== 13 passed, 0 failed ====`, rc=0 (arms 1,2,3,4,5,6,7,7ctl,7a,7b,8j,L,7c) | RED (file absent) |
| A2.2 | `/bin/bash -n tools/launchd/spawn-pin-hook.sh` | rc=0 | RED |
| A2.3 | `grep -c 'test_spawn_pin_hook.sh' make/test.mk` | `1` | `0` — **RED** |
| A2.4 | `python3 -c "import json,sys; json.load(open('.claude/settings.json'))"` | rc=0 (settings.json stays valid JSON) | GREEN — must stay green |
| A2.5 | `python3 -c "import json;d=json.load(open('.claude/settings.json'));p=d['hooks']['PreToolUse'];print(len(p), p[0]['hooks'][0]['command'].endswith('coordinator_hook.sh'), p[1]['matcher'])"` | `2 True Agent|Task` | `1 True` + IndexError — **RED** |
| A2.6 | `grep -c 'set -e' tools/launchd/spawn-pin-hook.sh` | `0` (a `set -e` abort is an allow) | RED (file absent) |
| A2.7 | `bash tools/launchd/test_mission_routing.sh` | `==== 45 passed, 0 failed ====`, rc=0 (44 after M1, +1 for arm W) | 36/0 |
| A2.8 | `PATH=/usr/sbin:$PATH make test-launchd-drivers` | rc=0, and the new suite's `==== N passed ====` line appears in the output | GREEN — must stay green |

**Test plan** (`tools/launchd/test_spawn_pin_hook.sh` unless stated)

Common env for arms 1–8: `MISSION_CONTROL_ACTIVE=1`, `MISSION_NAME=test`, `HOME=$tmp`.

| Arm | Setup | Assertion | Kills which mutation |
|---|---|---|---|
| 1 | `MISSION_EXECUTOR_MODEL=codex:gpt-5.6-luna`; payload `tool_name=Agent`, `model=sonnet`, `subagent_type=general-purpose`, prompt first line `MISSION-ROLE: executor` | `deny`, reason contains `Agent-tool alias spawn refused` and the pin `codex:gpt-5.6-luna` | delete the provider-pin (`:`) deny → **1 and 2 die**; arm 1 alone is the docs-10 replay |
| 2 | identical to arm 1 but `model=opus` | byte-identical `deny` reason to arm 1 | make the deny depend on WHICH alias (e.g. only deny `sonnet`) → **only arm 2 dies**; proves no silent retry through another alias |
| 3 | `MISSION_EXECUTOR_MODEL` **unset**; prompt `MISSION-ROLE: executor` | `deny`, reason contains `fail-closed:executor-model-missing` | apply any default when the pin is unset → **only arm 3 dies** |
| 4 | `MISSION_EVALUATOR_MODEL=sonnet`, `MISSION_EXECUTOR_RESOLVED=sonnet`; prompt `MISSION-ROLE: evaluator`, `model=sonnet` | `deny`, reason contains `generator!=judge` | delete the evaluator collision branch → **only arm 4 dies** |
| 5 | `MISSION_EVALUATOR_MODEL=sonnet`, `MISSION_EXECUTOR_RESOLVED=codex:gpt-5.6-sol`; prompt `MISSION-ROLE: evaluator` | `allow`, reason contains `allow:alias-pin` | make the collision check unconditional → **only arm 5 dies** (arm 4's false-positive control) |
| 6 | `MISSION_EXECUTOR_MODEL=codex:gpt-5.6-luna`; prompt = `Evaluate whether the sprint-planner's plan was followed by the sprint-executor` (two role skills named, **no token**) | `deny`, reason contains `fail-closed:role-missing` | reintroduce prose-based role inference → **only arm 6 dies** (design M8's measured false positive) |
| 7 | no token, `subagent_type=general-purpose`, marker **present** | `deny`, reason contains `fail-closed:role-missing` | delete the role-missing deny → **only arm 7 dies** |
| 7ctl | byte-identical payload to arm 7, `MISSION_CONTROL_ACTIVE` **unset** | `allow`, reason contains `passthrough:marker-absent` | delete the `MISSION_CONTROL_ACTIVE` check → **only arm 7ctl dies** (the attended-session status quo) |
| 7a | no token, `subagent_type=Explore`, marker present | `allow`, reason contains `explore-readonly` | delete the Explore exception → **only arm 7a dies**; arm 7 is its paired control (same call, `general-purpose`) |
| 7b | prompt first line `MISSION-ROLE: judge`, marker present | `deny`, reason contains `fail-closed:role-unknown` | accept any token value → **only arm 7b dies** |
| 8j | marker present, arm-1 payload, but `PATH` stubbed to a directory with **no `jq`** | `deny`, reason contains `fail-closed:jq-missing`, exit 0 | turn the missing-parser branch into an allow (or an early `exit 1`) → **only arm 8j dies** |
| L | after arms 1 and 7a, read `$tmp/.ailang/state/mission-test-spawn-hook.log` | exactly 2 lines; line 1 has 7 tab-separated fields ending `deny\tdeny:provider-pin`; line 2 ends `allow\texplore-readonly` | drop the log append, or change the field order/count → **only arm L dies** |

Plus, in `test_mission_routing.sh`:

| Arm | Assertion | Kills which mutation |
|---|---|---|
| W | `grep -q 'test_spawn_pin_hook.sh' "$ROOT/make/test.mk"` → ok | delete the make wiring → **only arm W dies**. This is the delivery-path guard: a suite that exists but is never invoked is green forever while enforcing nothing — the same class as the existing "driver EXPORTS the planner allowlist" arm |

**Arm 7c (settings.json, two overlapping hooks)** — implement inside `test_spawn_pin_hook.sh` as a
**static** assertion against the **repo's real** `.claude/settings.json` (not a `--settings` file);
the live two-hook behaviour was already measured in the design's M7, so the test's job is to pin the
wiring, not re-measure the platform:

```
PreToolUse array has >= 2 entries;
entry[0].hooks[0].command ends with coordinator_hook.sh   (unchanged, still first)
entry[1].matcher == "Agent|Task"
entry[1].hooks[0].command ends with tools/launchd/spawn-pin-hook.sh
```
Kills the mutation "replace the coordinator hook instead of adding beside it" and "register the hook
with a matcher that never fires". Parse with `jq -r` (already a dependency of this suite).

**Bounded-run note**: M2 is the largest milestone. If the executor's ≤30-minute budget is exhausted,
land the hook + arms 1–5 + 8j and stop; arms 6/7/7ctl/7a/7b/L/W and the settings.json wiring become
**M2b**. Do NOT ship the settings.json wiring without arms 7/7ctl — an un-controlled marker check is
how the fleet denies every attended spawn.

---

### M3 — Driver exports + docs env widening

**Files**: `tools/launchd/mission-control.sh` (~+20 LOC),
`tools/launchd/mission-env/mission-docs.env` (one alternative appended),
`tools/launchd/test_mission_routing.sh` (+3 arms).

**Do**
1. In `mission-control.sh`, **immediately before line 1003** (the
   `log "=== mission iteration starting …"` line — anchor on that string, not the number), insert:
   ```bash
   # Layer 3 (M-SPAWN-PIN-ENFORCEMENT): export the RESOLVED plan, post-degradation.
   # MUST be here and not beside the MISSION_<ROLE>_MODEL exports at 544/569/586/793:
   # the codex lane loop rewrites those vars in place at :722 and the pi loop at :770/:779,
   # and the one-shot override at :904 — exporting earlier would publish a plan the driver
   # then silently changed, which is the exact silent-degradation class this closes.
   export MISSION_CONTROL_ACTIVE=1
   for _role in DESIGNER PLANNER EXECUTOR EVALUATOR; do
     _mv="MISSION_${_role}_MODEL"; _rv="${!_mv:-}"
     printf -v "MISSION_${_role}_RESOLVED" '%s' "$_rv"; export "MISSION_${_role}_RESOLVED"
     case "$_rv" in
       *:*) printf -v "MISSION_${_role}_PATH" '%s' 'recipe' ;;
       "")  printf -v "MISSION_${_role}_PATH" '%s' 'unset'  ;;
       *)   printf -v "MISSION_${_role}_PATH" '%s' 'agent-tool' ;;
     esac
     export "MISSION_${_role}_PATH"
   done
   unset _role _mv _rv
   ```
   `${!var}` and `printf -v` are bash-3.2-safe and already used at `:713-717`.
2. In `tools/launchd/mission-env/mission-docs.env`, append `|scripts/*` to the **end of the
   `MISSION_PLANNER_ALLOWLIST` default value**, inside the `${…:-…}` braces. Do NOT touch
   `.claude/skills/docs-sync/scripts/*`, which is a different entry. Add a two-line comment above it
   naming the docs-10 incident and PR #1010. **Do not create or edit
   `~/.config/ailang/mission-docs.env`** — that live file is outside the repo and is the
   controller's deploy step; say so in the comment.
3. Add arms D1, D2 and arm 12 to `test_mission_routing.sh`.
4. Snapshot into `.snap/M3/` (cumulative).

**Acceptance criteria**

| # | Command | Expected | At base |
|---|---|---|---|
| A3.1 | `grep -c 'export MISSION_CONTROL_ACTIVE=1' tools/launchd/mission-control.sh` | `1` | `0` — **RED** |
| A3.2 | `MISSION_PROFILE=v1 MISSION_DRY_RUN=1 bash tools/launchd/mission-control.sh` | rc=0, `DRY RUN ok:` line unchanged in shape | GREEN — must stay green (the dry-run path must not break) |
| A3.3 | `grep -n 'MISSION_PLANNER_ALLOWLIST' tools/launchd/mission-env/mission-docs.env \| grep -c '\|scripts/\*'` | `1` | `0` — **RED**. Control: the same grep for `\|docs/\*` → `1` at base and after |
| A3.4 | `bash tools/launchd/test_mission_routing.sh` | `==== 49 passed, 0 failed ====`, rc=0 (45 after M2, +4 for D1, D2, 12-widened, 12-control) | 36/0 |
| A3.5 | `PATH=/usr/sbin:$PATH make test-launchd-drivers` | rc=0 | GREEN — must stay green |

**Test plan**

| Arm | Setup | Assertion | Kills which mutation |
|---|---|---|---|
| D1 | line-order assertion, same shape as the existing allowlist-order arm: `_res_line` = line number of `export MISSION_CONTROL_ACTIVE=1`; `_deg_line` = line number of `codex ${role_lc} lane -> falling back to` | `_res_line > _deg_line` → ok | move the Layer-3 exports up beside the role exports at 544/569/586 → **only D1 dies**. This is the arm that encodes correction §1.1(2) |
| D2 | run the driver's Layer-3 block in a child `/bin/bash` with `MISSION_EXECUTOR_MODEL=codex:x` and `MISSION_EVALUATOR_MODEL=sonnet`, then `/usr/bin/env \| grep -E '^MISSION_(EXECUTOR\|EVALUATOR)_(RESOLVED\|PATH)='` | `MISSION_EVALUATOR_PATH=agent-tool`, `MISSION_EVALUATOR_RESOLVED=sonnet`, `MISSION_EXECUTOR_PATH=recipe`, `MISSION_EXECUTOR_RESOLVED=codex:x` | assign without `export` → **only D2 dies**; `/usr/bin/env` is the process boundary, exactly as the existing `#696` arm does it. Include that arm's `unset` preamble and a `MC_EXPORT_CONTROL=sentinel` known-positive |
| 12 | build a temp design doc declaring `` `scripts/verify_examples.go` ``; run `derive-planner-lane.sh` twice — once with the allowlist read out of `mission-env/mission-docs.env`, once with the **pre-widening** list as the control | widened → `codex:gpt-5.6-luna declared:codex-ok`; control (no `\|scripts/*`) → `opus fail-closed:path-not-in-codex-allowlist` | drop `\|scripts/*` → **only arm 12's widened half dies**; drop the allowlist gate entirely → only the control half dies. Both halves measured at base (B11) |

---

### M4 — SKILL.md Gate-3 edit (SHRINK, keep it small)

**File**: `.claude/skills/mission-control/SKILL.md` — **exactly one paragraph replaced, nothing else.**

**Replace lines 1099–1101**, the paragraph whose opening words are
`Spawn pattern (heavy roles): ` and which currently reads verbatim:

> Spawn pattern (heavy roles): `Agent(subagent_type="general-purpose", model="<the role's env value>",
> prompt="invoke the <skill> for <doc>/<worktree> …")` — resolve the env value first via
> `echo $MISSION_EXECUTOR_MODEL`. These are in-session Agent-tool model **aliases**.

with:

> **Spawn pattern (heavy roles) — run the resolver, follow its output VERBATIM.** For each role run
> `tools/launchd/resolve-role-spawn.sh <role> [<design-doc>]`. Its output is exactly one line,
> `<path> <value> <reason-token>`; do not second-guess it. `agent-tool <alias>` → spawn
> `Agent(subagent_type="general-purpose", model="<alias>", …)`. `recipe <provider:model>` → the
> Agent tool is NOT a valid path for that role; use the cross-provider recipe below.
> `reroute <alias> generator-equals-judge` → spawn the named re-route target and say so in the
> Gate-4 row. `refuse …` → that role is a routing **FAILURE**, not a FLAG: record the reason token
> VERBATIM and continue the iteration without the role rather than spending un-budgeted opus.
> **Every role prompt MUST begin with the line `MISSION-ROLE: <designer|planner|executor|evaluator>`**
> — that token is the ONLY input the spawn-pin hook uses to map a spawn to a role, and while
> `MISSION_CONTROL_ACTIVE=1` an unlabelled Agent/Task call is DENIED at the tool boundary. A
> read-only reality-check needs no token: spawn it as `subagent_type: Explore`, the one
> machine-readable exception.

**Do not touch** the `⚠ CORRECTED 2026-08-20` fable paragraph (1103–1134), Step 1b (1136–1149), the
cross-provider recipe (1151+), or the chain paragraph at 1316–1330. They stay as data.

**Acceptance criteria**

| # | Command | Expected | At base |
|---|---|---|---|
| A4.1 | `grep -c 'MISSION-ROLE' .claude/skills/mission-control/SKILL.md` | `>= 1` | `0` — **RED** |
| A4.2 | `grep -c 'resolve-role-spawn.sh' .claude/skills/mission-control/SKILL.md` | `>= 1` | `0` — **RED** |
| A4.3 | `git diff --numstat .claude/skills/mission-control/SKILL.md` (READ-ONLY: `git diff` is not a write op) | insertions ≤ 20, deletions ≤ 5 — the edit SHRINKS the rule surface and touches one paragraph | n/a |
| A4.4 | `grep -c 'the Agent tool.s .model. enum in this build lists' .claude/skills/mission-control/SKILL.md` | `1` (the fable paragraph survives untouched) | `1` — must stay `1` |
| A4.5 | `wc -l .claude/skills/mission-control/SKILL.md` | between 2819 and 2839 | `2819` |

**Test plan**

| Arm | Assertion | Kills which mutation |
|---|---|---|
| S1 (add to `test_mission_routing.sh`) | `grep -q 'resolve-role-spawn.sh' .claude/skills/mission-control/SKILL.md` **and** `grep -q 'MISSION-ROLE:' .claude/skills/mission-control/SKILL.md` | revert the skill edit while keeping the tooling → **only S1 dies**. Without this the whole design degrades to "code exists, nothing calls it" |
| S2 (add to `test_mission_routing.sh`) | `grep -q "the Agent tool's \`model\` enum in this build lists" .claude/skills/mission-control/SKILL.md` | over-delete the fable capability paragraph while editing → **only S2 dies** (the September-2025 import-disaster class: deleting "unused" prose that is load-bearing) |

---

## 5. Out of scope (do not do these)

- **The live `~/.config/ailang/mission-docs.env`.** Outside the repo, outside the worktree. Only the
  versioned copy `tools/launchd/mission-env/mission-docs.env` is edited here. Deploying it to the
  rig is the controller's step and belongs in the Gate-4 record.
- **The `.agents/skills` mirror (#544).** This change edits the AUTHORITATIVE
  `.claude/skills/mission-control/SKILL.md` only. Mirror drift is #544's battle.
- **Filing GitHub issues.** The design's Bookkeeping section (an issue for the Agent-tool alias-only
  mechanism, cross-linked to #493 and #902) is the controller's Gate-5 work, not the executor's. The
  executor runs no `gh` command.
- **The Agent tool's model enum itself.** Not our surface — Claude Code platform. The design already
  rejected extending it.
- **`scripts/hooks/coordinator_hook.sh`** and every other existing hook. Read them; change nothing.
- **Removing the opus tail.** It is the designed last resort. The bug is the shortcut past healthy
  rungs, not the tail.
- **`~/.claude/settings.json`** (the user-global file). Only the repo's `.claude/settings.json`.
- **Retro-fitting `MISSION-ROLE:` tokens into past mission logs.** Design M8 measured that the real
  docs-9/docs-10 prompts were never banked; there is nothing to retro-fit.

## 6. Rollout guard (verbatim from the design; the executor must honour it)

> **Do not ship Layer 1 or the skill edit as enforcement unless the PreToolUse hook passes these
> tests. If hook enforcement is unavailable, retain the current behavior and open a blocked
> platform-gap issue; Layer 1 may land only as diagnostics and must not be claimed to convert silent
> fallback to loud failure.**

Operationally: **M4 must not be committed before M2's acceptance criteria A2.1–A2.8 are green.**
If M2 fails, land M1 and M3 as diagnostics only, leave the skill text unchanged, and report the
blockage — do not describe M1 alone as enforcement.

## 7. Success metrics

- `bash tools/launchd/test_mission_routing.sh` → `==== 51 passed, 0 failed ====` after M4. Running total: 36 baseline → 44 (M1: R1,R2,R3,R4,R4b,R5,R6,R7) → 45 (M2: W) → 49 (M3: D1,D2,12-widened,12-control) → 51 (M4: S1,S2). **If your count differs, report the real number — never adjust an arm to hit a number in this plan.**
- `bash tools/launchd/test_spawn_pin_hook.sh` → `==== 13 passed, 0 failed ====` (arms 1,2,3,4,5,6,7,7ctl,7a,7b,8j,L,7c).
- `PATH=/usr/sbin:$PATH make test-launchd-drivers` → rc=0, and the new suite's footer appears.
- Every hook branch (deny role-missing · deny role-unknown · deny provider-pin alias · deny re-route ·
  allow Explore · allow marker-absent · deny jq-missing) has exactly one arm that is its **sole** killer.
- `.claude/settings.json` is still valid JSON and `coordinator_hook.sh` is still the FIRST PreToolUse entry.
- No git write command was run by the executor; `.snap/M1..M4/` exist and are cumulative.
