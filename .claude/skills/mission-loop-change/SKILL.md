---
name: mission-loop-change
description: Change or add a mission loop safely — routing/lanes, schedule and cadence, driver code, plists, and adding a whole new mission. Use when editing tools/launchd/mission-control.sh, any dev.ailang.mission-* plist or mission-*.env, changing a mission's models/fallbacks/interval, porting a fix to the world fork, bootstrapping a new mission loop (e.g. for ailang-parse), or when asked why a mission-loop change "didn't take". Covers reach, live-edit hazards, drought simulation and reload.
---

# Changing a Mission Loop

The mission loops are four long-running agents on the rig. They are easy to *edit* and easy to
edit **without effect** — the recurring failure is not a broken change, it is a change that
lands somewhere nothing reads. Every rule here is a measured incident.

> **Read this first:** the friction is catalogued, with counts, in
> [`design_docs/planned/v0_36_0/m-mission-loop-workbench.md`](../../../design_docs/planned/v0_36_0/m-mission-loop-workbench.md).
> Once that ships, most of this file collapses into `ailang mission install|doctor`. Until then
> the procedure below **is** the control.

## Gate 0 — Which surface actually runs?

There are six places a mission's behaviour can be declared, and they are not kept in sync by
anything. Before editing, decide which one you need:

| Want to change | Edit | Reaches the fleet via |
|---|---|---|
| Driver logic (probes, gates, chains) | `tools/launchd/mission-control.sh` in **the ailang repo** | the driver pin — committed `origin/dev`, so **it must be pushed** |
| A mission's models / lanes / allowlist | `~/.config/ailang/mission-<name>.env` | sourced per fire. The repo copy under `tools/launchd/mission-env/` is **reviewable, not deployed** |
| Schedule / PATH / env | `~/Library/LaunchAgents/dev.ailang.mission-<name>.plist` | requires copy + reload. Installed plists are **copies, not symlinks** |
| Anything on `world` | the **`ailang-world` repo** | world runs a FORK and no pin — see Gate 2 |

**The trap, measured 2026-09-05:** the versioned `mission-docs.env` widened the planner
allowlist and its own comment declared itself "the sprint's only deployment surface". It was
never copied to `~/.config`. The docs mission has been routing that work to opus instead of
codex ever since, with a green CI arm — because the test asserts against the repo copy.

```bash
# ALWAYS check before and after editing an env file:
for m in v1 docs motoko world; do
  cmp -s tools/launchd/mission-env/mission-$m.env ~/.config/ailang/mission-$m.env \
    && echo "$m: in sync" || echo "$m: *** DIVERGED — the repo copy is not what runs"
done
```

## Gate 1 — Does the change reach the mission you mean?

A mission runs the driver its plist's `ProgramArguments` points at. Three of four re-exec from
the driver pin (committed `origin/dev` of *their own clone's* repo); world does not.

```bash
for m in v1:ailang docs:ailang-docs motoko:ailang-motoko world:ailang-world; do
  n=${m%%:*}; P=~/dev/sunholo-data/${m##*:}/tools/launchd/mission-control.sh
  printf "%-7s re-execs-from-pin=%s\n" "$n" \
    "$(grep -c '^\s*\. "\$REPO/tools/launchd/lib/pin-root.sh"' $P 2>/dev/null | sed 's/^0$/NO/;s/^[1-9].*/YES/')"
done
```

- `YES` → the clone's own contents are irrelevant; it runs committed `origin/dev`. **Push or it
  does not ship.** A "N behind" line about the source clone is expected and harmless.
- `NO` (world) → it runs the working tree directly. Committing is not enough and not required;
  the file on disk is what executes.

## Gate 2 — World is a fork

`world` lives in a separate GitHub repo with its own ~1000-line copy of the driver and no
`pin-root.sh`. It has silently missed every routing fix landed in `sunholo-data/ailang`.

- Any driver change you make for the fleet **must be made twice** until the de-fork lands.
- Port **verbatim**, then prove it:

```bash
for fn in _mc_set_controller select_model; do
  diff <(sed -n "/^${fn}()/,/^}/p" ~/dev/sunholo-data/ailang/tools/launchd/mission-control.sh) \
       <(sed -n "/^${fn}()/,/^}/p" ~/dev/sunholo-data/ailang-world/tools/launchd/mission-control.sh) \
    >/dev/null && echo "$fn: identical" || echo "$fn: DIFFERS"
done
```

Byte-identical is what lets you claim the upstream suite covers the port. Anything else and you
owe world its own tests.

## Gate 3 — Never edit a driver that is executing

World executes its driver straight from the working tree, and **bash reads a script
incrementally** — truncating it under a live interpreter feeds it garbage mid-iteration.

```bash
for f in ~/.ailang/state/mission-*.pid; do
  [ -f "$f" ] || continue; p=$(cat "$f")
  kill -0 "$p" 2>/dev/null && echo "RUNNING: $(basename $f) pid=$p"
done
```

If it is running, either wait, or write via **atomic replace** so the running process keeps its
inode (`python3 -c "..."` with `tempfile` + `os.replace`, or `sed > tmp && mv tmp file`).
`open(w)` and in-place `sed -i` are unsafe. The pinned missions are exempt — the pin runs from a
separate worktree.

## Gate 4 — Prove it, before it fires for real

**Nothing here spends tokens except where noted.** `MISSION_DRY_RUN=1` exits before the
iteration but *after* the probes, so it reports real resolved lanes.

```bash
# Working tree instead of the pinned copy — REQUIRED or you are testing the old driver:
AILANG_DRIVER_PIN=0 MISSION_PROFILE=<name> MISSION_DRY_RUN=1 \
  /bin/bash tools/launchd/mission-control.sh 2>&1 | tail -3
```

**Simulate the Friday Anthropic drought instead of waiting for Friday.** Point a role at a model
that cannot answer; the pre-flights degrade it down its chain and `lanes=` reports the result:

```bash
AILANG_DRIVER_PIN=0 MISSION_PROFILE=<name> MISSION_DRY_RUN=1 \
  MISSION_DESIGNER_MODEL='claude:claude-drought-sim' \
  MISSION_EVALUATOR_MODEL='claude-drought-sim' \
  /bin/bash tools/launchd/mission-control.sh 2>&1 | grep -E 'anthropic|lanes='
# healthy -> lanes=ok, roles unchanged.  drought -> lanes=DEGRADED(n) with each handoff named.
```

Force the memory gate the same way (`MISSION_MIN_AVAIL_GB=9999 MISSION_MEM_WAIT=2`); it must log
`STILL SHORT … yield` and **never reach `DRY RUN ok`**. A gate you have only seen pass is a gate
you have not tested.

## Gate 5 — Config declared is not config walked

A `MISSION_<ROLE>_FALLBACK` only does something if a pre-flight loop walks that role **and**
matches the value's provider prefix. `MISSION_EVALUATOR_FALLBACK` sat dead for ten days because
the loops iterated `PLANNER EXECUTOR` and the skill reads no `*_FALLBACK` var at all.

Before believing a role survives an outage, check all three:

```bash
grep -n 'for role in' tools/launchd/mission-control.sh        # is the role in the list?
grep -n 'case "$val" in' tools/launchd/mission-control.sh     # does an arm match its prefix?
grep -c 'MISSION_.*_FALLBACK' .claude/skills/mission-control/SKILL.md   # 0 = the skill reads none
```

Then prove it with the drought simulation above. Do not reason about it.

## Gate 6 — Tests, and the ones that do not run

```bash
make test-launchd-drivers        # must be exit 0; bash 3.2, the rig has no 4.x
```

The suites are grep-based structural assertions over `$HERE/mission-control.sh`, so **copying a
suite into another repo tests that repo's driver** — that is how world gets coverage today.

Two suites were orphaned (present but absent from `make/test.mk`) and CI never ran them. When
you add a suite, wire it, and check nothing else has drifted out:

```bash
for f in tools/launchd/test_*.sh; do
  grep -q "$(basename $f)" make/test.mk || echo "ORPHAN (CI never runs): $(basename $f)"
done
```

## Gate 7 — Installing and reloading

```bash
cp tools/launchd/dev.ailang.mission-<name>.plist ~/Library/LaunchAgents/
plutil -lint ~/Library/LaunchAgents/dev.ailang.mission-<name>.plist   # never skip
launchctl bootout   "gui/$(id -u)/dev.ailang.mission-<name>"
launchctl bootstrap "gui/$(id -u)" ~/Library/LaunchAgents/dev.ailang.mission-<name>.plist
launchctl print     "gui/$(id -u)/dev.ailang.mission-<name>" | grep -E 'state|properties'
```

- **Check Gate 3 first** — bootout kills a running iteration.
- `RunAtLoad` means bootstrap starts an iteration immediately. Expect it.
- `rig-watchdog` re-bootstraps missing jobs every 60s, so a bare `bootout` does not stick. To
  hold a mission off you need the kill switch (`~/.ailang/state/mission-<name>.disabled`), not
  `bootout`.
- Keep the plist **versioned in the repo** and copy from there. World's lived only in
  LaunchAgents for months, so its schedule was unreviewable.

## Scheduling: what the knobs actually do

`StartInterval` **re-arms from the job's EXIT**, not its start — measured across four gaps on two
missions (5403/5372/5379s against 5400s; 14390s against 14400s). So:

```
StartInterval:                  period = duration + interval   (a full interval idle, every cycle)
KeepAlive + ThrottleInterval:   period = max(duration, throttle)
```

`ThrottleInterval` is measured from the **start**, which is what makes it a floor rather than a
gap. It is load-bearing under `KeepAlive`: the yield paths (kill switch, overlap guard, memory
gate) exit in seconds, and bare `KeepAlive` would respawn them in a tight loop.

**Cadence is a measured decision, not a preference.** On 2026-09-02 three loops were tightened in
the same hour: fleet stall rate 6% → 33%, and total starts did **not** rise. Change one loop,
hold the others, and re-read the stall rate the next day — signature `claude idle with a
descendant alive >=2400s`. Durations and verdicts are in
`~/.ailang/state/mission-<name>-slot-verdicts.log` (durable; `/tmp` driver logs do not survive a
reboot).

## Adding a whole new mission

Until the registry lands, this is the manual path (`tools/launchd/mission-template.plist` and
`docs/docs/guides/mission-bootstrap.md`):

1. `sed s/__NAME__/<name>/g < tools/launchd/mission-template.plist > ~/Library/LaunchAgents/dev.ailang.mission-<name>.plist`
   — and **commit a copy to the repo**.
2. Write `~/.config/ailang/mission-<name>.env` (`MISSION_NAME`, `MISSION_REPO`, `MISSION_DOC`,
   `MISSION_WORKDIR`) **and** the reviewable copy under `tools/launchd/mission-env/`. Keep them
   identical (Gate 0).
3. Seed `~/.ailang/state/mission-<name>-gh-issue` with the bookkeeping issue number.
4. Add a boot-stagger offset arm for the name in `_mc_boot_offset` — an unknown mission silently
   gets 0 and joins the boot stampede.
5. Give the plist a PATH that includes `/usr/sbin`, or rely on the driver's own append. Omitting
   it disabled the boot stagger on two missions.
6. Dry-run it (Gate 4) **before** bootstrapping, then Gate 7.

Prefer running the new mission's driver from a clone that sources `pin-root.sh`. A fork is how
world ended up invisible.

## The five-line pre-flight

Before claiming a mission-loop change is done:

- [ ] Edited the surface that actually runs (Gate 0), and repo/installed copies match
- [ ] Confirmed reach — pushed if pinned; on disk if not (Gate 1); ported to world if fleet-wide (Gate 2)
- [ ] Dry-ran healthy **and** degraded (Gate 4); any new gate proven non-vacuous
- [ ] `make test-launchd-drivers` exit 0, new suites wired (Gate 6)
- [ ] Reloaded and verified state, with nothing mid-iteration (Gates 3, 7)
