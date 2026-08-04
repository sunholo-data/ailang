---
name: mission-brief
description: Read-only steering brief for attended mission sessions — dashboard + directive channel + live loop health + inbox, then hand the wheel to Mark. Use when the user says "mission brief", "catch me up on the mission", "steering session", or opens a fresh thread for in-person steering or meta-loop updates. NOT for running iterations — that is mission-control.
---

# Mission Brief — the attended steering entry point

READ-ONLY: this skill gathers and reports. No acks, no edits, no commits — steering actions
come after the brief, at Mark's direction. mission-control is the machine's entry point into
the loop; this is the human's. Never fold brief logic into mission-control (it is live by
symlink for every mission the instant it is saved — small blast radius matters).

## Gather (all four; they are independent, so run in parallel)

1. **Dashboard**: Read `design_docs/mission-dashboard.md` (≤40 lines, the full snapshot).
   Staleness check: if its `Updated` stamp is older than ~2× the v1 cadence on its Loops
   line (90min default) while the loop claims armed, FLAG IT — Gate-4 isn't refreshing,
   which is itself a finding (wedged loop, quota dry-out, or launchd off).
   Normalize to UTC before ANY timestamp comparison: issue comments are UTC (`Z` suffix),
   the dashboard stamp is local time. A stamp in the future is itself suspect — the
   2026-08-04 bootstrap wrote 14:30Z as "~16:30" via exactly this mixup.

2. **Directives** — the channel the dashboard only *points* at:
   ```bash
   gh api "repos/sunholo-data/ailang/issues/$(cat ~/.ailang/state/mission-gh-issue)/comments" \
     --jq 'reverse | .[0:5][] | "\(.created_at) \(.user.login): \(.body | split("\n")[0])"'
   ```
   (Last 5 comments, newest first, first line each.) Any comment newer than the dashboard's
   `Updated` stamp is an UNPROCESSED DIRECTIVE — surface it verbatim. The pointer file
   rotates Mondays; if the issue is closed or missing, report that rather than skipping.

3. **Live loop health** — the dashboard's "armed" is a snapshot, not now:
   ```bash
   launchctl list | grep -E 'dev\.ailang\.mission-(control|world)'
   ```
   Column 1 = PID if running right now, column 2 = last exit status. Then the last logged
   iteration: `grep "^## " design_docs/v1-mission-log.md | tail -1` — compare its date
   against the cadence, but read the dashboard's quota-posture section before calling a gap
   "wedged" (a dry-out can be expected posture, not breakage).

4. **Inbox**: usually already injected by the SessionStart hook — summarize what it showed,
   don't re-fetch. Mid-session: `ailang messages list --compact`. Do NOT ack — Mark's call.

## Report

One compact brief, in this order: Now (from dashboard) · unprocessed directives · loop
health (live) · inbox highlights · Parked-on-Mark items. Then ask: **what do you want to
steer?**

## Steering levers — route Mark's decision through an existing channel, never improvise one

- **Durable directive** → comment on the bookkeeping issue (Gate-1 reads it next iteration)
- **Charter change** → edit `design_docs/v1-mission.md` per its attended-stamp conventions
- **Message a loop/agent** → `ailang messages send` (see `ailang messages --help`)
- **Do it now** → this attended session does the work directly
- **Meta/skill fix** → edit the skill IN THE MAIN CHECKOUT — a worktree commit reaches
  origin but never the running loop (mission-control's Gate-5 symlink note has the details)
