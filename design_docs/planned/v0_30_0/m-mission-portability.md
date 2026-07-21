# M-MISSION-PORTABILITY: extract the mission loop into a portable template — bootstrap kit for the Ailang World mission

**Status**: **SPLIT-AUTHORIZED by Mark 2026-07-21 (option a)** — iter-68 flagged the live-driver
self-modification hazard and declined headless execution (correct). Mark's split: **M1 (driver
parameterization) = ATTENDED-ONLY — executed interactively with the session 2026-07-21** (atomic
mv-replace, never in-place on the running script); **M2 (skill repo/verify profiles) + M3
(bootstrap kit + dry-run) = HEADLESS-GREENLIT for the loop** (plain-file work, no live-driver
risk). Original ask: Mark 2026-07-18, precedes the **Ailang World** mission launch (separate
repo, same rig/keys/fleet).
**Target**: v0.30.x — mission infrastructure, P1 (gates the World mission's launch; zero language
surface). **M2+M3 PRIORITY (Mark 2026-07-21: "could we feasibly launch Ailang World now?"): pick
NEXT after the currently in-flight item — ahead of the backlog triage — so World can bootstrap
this week.**
**Estimated**: ~1–1.5d (M1 driver ~0.5d · M2 skill templating ~0.5d · M3 bootstrap kit + dry-run ~0.5d)
**Dependencies**: none new — parameterizes what exists. The fleet substrate (design-quorum, exec
lanes incl. clone-over-egress, messages, coordinator) ships IN the `ailang` binary and ports with
`ailang install`; billing guards (`~/.zshenv`, `claude-sub`) and gh auth are machine-level.
**Author**: interactive session (Fable) with Mark, 2026-07-18

---

## Problem statement

The mission loop (driver + mission-control skill + inner-loop skills + charter conventions) was
built inside sunholo-data/ailang and is wanted for a second, concurrent mission in a different
repo (**Ailang World**). Measured hardcoding (2026-07-18, HEAD):

- `tools/launchd/mission-control.sh`: **3** `sunholo-data/ailang` refs, **2** `design_docs/v1-mission.md`
  refs (the PROMPT + a comment), and **6** `~/.ailang/state/mission-*` files that would COLLIDE if two
  missions run on one rig (`mission-control.disabled/.pid`, `mission-model`, `mission-model-last`,
  `mission-executor-model-once`, `mission-gh-issue`; plus `mission-329-last-seen` +
  `mission-designer-rotation` in the skill).
- `.claude/skills/mission-control/SKILL.md`: **3** repo-slug refs, hardcoded CI workflow names
  ("CI", "Build and Release", "Deploy Documentation…"), and a Go-repo verification profile
  (`make quick-install`, `make test`, `bin/ailang`) that does not fit an AILANG-code repo.
- Two concurrent 2h loops would also contend for fire slots and double Anthropic burn if unplanned.

Everything else already ports: the fleet lanes are `ailang` CLI features, the charter FORMAT
(STATUS rotation, queue tags, evidence rows, ruled-out ledger) is repo-agnostic, and separate
checkouts eliminate the shared-working-tree hazard class entirely.

## Design

### M1 — driver parameterization — ✅ DONE (attended, 2026-07-21; per Mark's option-a split)
Landed exactly as specced with ONE deliberate amendment: **no state migration** — instead,
`MISSION_NAME=v1` (the default) keeps the LEGACY paths bit-for-bit (`mission-control.pid`,
`/tmp/ailang-mission-control.log`, …) while any other name gets fully namespaced paths
(`mission-<name>.pid`, `/tmp/ailang-mission-<name>.log`, …). Rationale: migration would have
violated the acceptance criterion (defaults reproduce today bit-for-bit) AND risked an
overlap-guard blind window if renamed mid-run. Installed via same-dir atomic `mv` while a live
iteration ran (old inode preserved; verified unharmed). Acceptance evidence (3 dry-runs,
2026-07-21): (1) default v1 read the LIVE legacy pidfile and yielded — compat proven;
(2) `MISSION_NAME=world` ignored v1's live pid, namespaced pidfile — isolation proven;
(3) `MISSION_PROFILE=worldtest` sourced `mission-worldtest.env` (name/repo/doc all flowed) —
profile mechanism proven. `tools/launchd/mission-template.plist` added (`__NAME__`/`__WORKDIR__`
placeholders, RunAtLoad=true, staggered-offset note). gh-comment fallback `329` is now V1-ONLY;
other missions must seed their gh-issue state file (comments skip with a warning otherwise —
never cross-post to the wrong repo).
- `MISSION_NAME` (default `v1`) → legacy paths for v1, `~/.ailang/state/mission-${MISSION_NAME}-*` otherwise (amended from the original migration design, see above).
- `MISSION_REPO` (default `sunholo-data/ailang`) → replaces the 3 hardcoded `--repo` args.
- `MISSION_DOC` (default `design_docs/v1-mission.md`) → the PROMPT line.
- `MISSION_WORKDIR` (default: derived from script path, as today).
- launchd: `dev.ailang.mission-control` keeps working; a `tools/launchd/mission-template.plist`
  with `__NAME__` placeholders is added for new missions (label `dev.ailang.mission-<name>`).
Profile file: `~/.config/ailang/mission-<name>.env`, sourced by the driver when
`MISSION_PROFILE=<name>` is set by the plist — one plist + one env file per mission.

### M2 — skill templating (repo profile + verify profile)
`mission-control/SKILL.md` gains a **REPO PROFILE** block read from the mission doc's charter
header (single source of truth, versioned with the mission): repo slug, bookkeeping-issue state
key, CI workflow names for Gate 3b, and a **VERIFY PROFILE** naming the build/test commands.
Two canonical verify profiles:
- `go-compiler` (this repo): `make quick-install && make build`, `make test`, binary staleness rules.
- `ailang-code` (Ailang World): `ailang check` / `ailang test` / `ailang ai-check --json` — the
  binary provides the whole gate; note `ailang ai-check` is the unified check+verify (do not
  reinvent split gates). Inner-loop skills reference the profile instead of `make` literals where
  they currently assume Go.
The directive-author allowlist (`MarkEdmondson1234`), quorum-at-pick, billing tripwire, pidfile
guard, rotation designer, and weekly issue rotation all port UNCHANGED — they are already
repo-agnostic once the state keys are namespaced.

### M3 — bootstrap kit + dry-run acceptance
**(Mark 2026-07-21: the manual MUST publish to the public website — it is the getting-started
recipe for launching a mission, and doubles as public documentation of the mission-loop story.)**
`docs/docs/guides/mission-bootstrap.md` (Docusaurus tree → ships via Deploy-Docs automatically;
write it PUBLIC-READER-quality: assume the reader has ailang installed and a repo, not our
context) + `design_docs/mission-charter-TEMPLATE.md`:
1. New repo prerequisites: **CI workflows exist** (Gate 3b is meaningless without them — hard
   precondition), gh auth = `sunholo-voight-kampff`, `ailang install`, same key env.
2. Copy driver + plist from the template, write `mission-<name>.env`, create the bookkeeping
   issue, seed state files.
3. **Iteration 0 = ratify the new mission's charter via the quorum** (bar, queue, guardrails —
   the format from the template, the content from the mission).
4. Concurrency rules for same-rig missions: **staggered StartInterval offsets** (e.g., World fires
   on odd hours), per-mission state namespace (M1), shared quota plan — World's executor defaults
   to a NON-Anthropic lane (codex) from day one; rig.lock stays GPU-only and global.
**Acceptance**: a scratch repo boots the templated loop with `MISSION_DRY_RUN=1` (wiring-only,
zero tokens) proving no state collision with the live v1 mission (both pidfiles/state distinct),
plus one gate-walk of Gate 0–1 against the scratch repo's own CI.

## Conflict surface
Touches the driver + mission-control skill (the LIVE loop's own files — edit in the MAIN checkout,
launchd reads on-disk; same discipline as M1a/pidfile fixes). MUST be backward-compatible at every
step: defaults reproduce today's behavior bit-for-bit; the state-file migration is one-shot and
guarded. Must NOT: fork the skill per mission (ONE skill, parameterized — divergence would undo
the self-improvement loop, since Gate-5 retro fixes must benefit all missions); move fleet logic
out of the `ailang` binary into repo scripts; widen the directive allowlist.

## Non-goals
- Authoring the Ailang World charter/bar/queue (that is World's iteration 0, with Mark).
- Multi-rig or cloud scheduling (launchd-on-this-rig only, as today).
- Changing quota policy (the routing table + fleet lanes already govern it).
- A public "mission framework" product (internal template; productizing is post-v1.0 at best).

## Verification log
| Claim | Method | Result |
|---|---|---|
| 3+3 repo-slug refs, 2 mission-doc refs | grep 2026-07-18 HEAD | Confirmed |
| 6 driver state files would collide cross-mission | grep `\$HOME/.ailang/state/mission*` | Confirmed (plus 2 skill-side keys) |
| Fleet substrate ships in the binary | `ailang` help: design-quorum/exec/messages/chains present | Confirmed |
| Billing guards are machine-level | `~/.zshenv` unset + `~/.local/bin/claude-sub` (no repo paths) | Confirmed |
| Skill already accepts a mission-doc argument | SKILL.md line 8 "(or the mission doc passed as argument)" | Confirmed |
| ai-check is the unified AILANG verify gate | memory `project-ailang-contracts-not-typechecked` | Confirmed (do not reinvent) |

## Related
- [m-mission-adaptive-multiprovider-routing](m-mission-adaptive-multiprovider-routing.md) — the fleet the template carries
- [m-mission-agentic-provider-routing](m-mission-agentic-provider-routing.md) — per-role pinning (ports as-is)
- `design_docs/v1-mission.md` — the charter whose FORMAT becomes the template

---
**Document created**: 2026-07-18 (interactive; expect quorum-at-pick when the loop picks this — no
creation-time quorum was run)
