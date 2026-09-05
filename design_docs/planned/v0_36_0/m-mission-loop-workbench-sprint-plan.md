# Sprint Plan: M-MISSION-LOOP-WORKBENCH — Phase 1

**Design doc**: [m-mission-loop-workbench.md](./m-mission-loop-workbench.md)
**Created**: 2026-09-05
**Scope**: **Phase 1 ONLY** — registry, renderer, `doctor`. No live path is written.
**Duration**: 4 milestones, ~1,250 LOC (750 impl / 500 test), 2 days
**Risk**: **Low** — by construction. Phase 1 is read-and-render only: it writes no file the fleet
reads, touches no plist, and runs no `launchctl`. `install`/`apply` are Phase 2 and are the point
at which risk appears.
**Decisions**: HD-1(a), HD-3(c), role-passthrough ratified by Mark 2026-09-05; HD-4(b) by Claude.
HD-2 deliberately unratified — Phase 3 only, out of this sprint.

## Why this scope

The design doc's own gate: **a drift detector that reports a clean fleet is indistinguishable
from one that is not looking.** So Phase 1 ships `doctor` and proves it against three
divergences measured on the live rig *before* anything is migrated:

| # | Divergence it must find | Measured 2026-09-05 |
|---|---|---|
| V4 | repo vs installed env drift | docs 4 lines, world 65 lines |
| V5 | that drift changes live routing | repo → `codex:gpt-5.6-luna declared:codex-ok`; installed → `opus fail-closed:path-not-in-codex-allowlist` |
| V8 | v1/docs plists omit `/usr/sbin`, disabling the boot stagger | `sysctl` unreachable → `kern.boottime unreadable` |

If `doctor` cannot reproduce all three, it is not trusted and Phase 2 does not start.

## Velocity basis

Last 7 days across the repo: **+8,170 non-test Go LOC, +9,186 test LOC** (501 files). That is
fleet-wide across several loops plus attended work, so it is an upper bound, not a lane rate.
Planned conservatively at ~600 LOC/day for one attended lane → 2 days for 1,250 LOC.

## Milestones

### M1 — `internal/mission`: registry schema + loader (~200 impl / ~180 test)

One TOML file per mission in `missions/` (HD-1a), parsed with the in-tree
`github.com/BurntSushi/toml` (V11 — already a direct dependency, precedent
`internal/pkg/manifest.go`).

Fields: `name`, `repo`, `workdir`, `doc`, and `[schedule]` (`mode` = `keepalive`\|`interval`,
`throttle_seconds`\|`interval_seconds`, `boot_offset`). **No `[roles]` block** — role/model
config is passthrough (ratified), and the loader must *reject* a `[roles]` key so the cut cannot
silently regress.

**Acceptance criteria**
- Loads a valid entry; round-trips every field
- Rejects: unknown mission name pattern, absent workdir, `mode` not in the enum, `boot_offset`
  colliding with another registered mission, and **any `[roles]` key** (with a message naming
  M-MODEL-REGISTRY-SINGLE-SOURCE as the owner)
- `Load()` on the real `missions/` dir is deterministic and order-independent
- Validation errors name the file and the field, never just "invalid config"

### M2 — Renderer: env + plist, with role passthrough (~250 impl / ~200 test)

Renders both artifacts to **staged paths only** (`*.staged`). Nothing is promoted in Phase 1.

Role/model passthrough (ratified): the renderer reads the mission's *current* env file, carries
every `MISSION_*_MODEL` / `MISSION_*_FALLBACK` / `MISSION_MODEL_PREFS` /
`MISSION_*_ALLOWLIST` line through **verbatim**, and emits a header marking that block as owned
by M8 of the model registry when it unparks. It authors only the schedule/topology lines.

Includes the atomic-write helper the inventory found absent (V15: 11 ad-hoc `os.Rename` sites,
no named helper) — scoped to `internal/mission`, not a repo-wide refactor.

Plist rendering is **new code**, not an extension of `internal/daemon/plist.tmpl` (V17: that
template hardcodes label, `KeepAlive`, log paths and daemon env, and exposes only three
variables). The daemon's *lifecycle* functions are reused in Phase 2, not its template.

**Acceptance criteria**
- **Golden test: rendering each of the four current missions is byte-identical to what is
  installed today**, except the schedule block. This is what makes Phase 2 provably
  behaviour-preserving.
- A mission whose live env has role lines the registry knows nothing about still renders those
  lines unchanged
- `keepalive` renders `KeepAlive`+`ThrottleInterval`; `interval` renders `StartInterval`; never both
- Rendered plists pass `plutil -lint`
- Every rendered PATH contains `/usr/sbin` (the V8 regression, enforced at render time)
- Writes only `*.staged`; a test asserts no non-staged path is ever opened for write

### M3 — `ailang mission doctor` + `list` (~300 impl / ~200 test) — **THE GATE**

Per mission: registry-vs-installed drift (env and plist), which driver the plist's
`ProgramArguments` resolves to, whether that clone re-execs from the pin, and whether the
effective PATH can reach `sysctl`. Non-zero exit on any drift. Replaces the hand-maintained
comment table at `mission-control.sh:752`.

**Acceptance criteria**
- **Reproduces V4, V5 and V8 against the live rig.** Fixtures encode all three; each must fail
  the doctor. A doctor that passes them is rejected.
- Reports world as a fork with no pin, and v1/docs/motoko as pin-backed
- `list` prints name, repo, workdir, schedule, driver, pin status
- Read-only: a test asserts no write syscall to `~/.config` or `~/Library/LaunchAgents`
- Exit codes: 0 clean, 1 drift, 2 registry error

### M4 — Wire it up: CLI, fixtures, CI (~120 impl / ~80 test)

`cmd/ailang/mission.go` following `cmd/ailang/messages_send.go` conventions. Registry entries for
the four current missions authored **from installed state, not the repo copies** — the installed
files are what runs (V4). `make test-mission-registry` added and wired into CI, checked against
the orphan-suite class that let two suites go unrun (V6).

**Acceptance criteria**
- `ailang mission list` and `doctor` work end-to-end on the real fleet
- Four registry entries exist and load
- New target runs in CI; the orphan check (`every test_*.sh appears in make/test.mk`) still passes
- `make test` and `make test-launchd-drivers` both green

## Explicitly NOT in this sprint

- `install` / `apply` — Phase 2. Nothing writes a live path.
- De-forking world — Phase 3, and HD-2 is unratified.
- Any change to role/model assignment — parked at M8 by Mark's ruling.
- Any cadence change. The 09-02 measurement (three loops tightened, stall rate 6%→33%, no extra
  starts) stands.

## Success metrics

- [ ] `doctor` reproduces V4, V5, V8 on the live rig — **the gate**
- [ ] Golden tests prove rendering is byte-identical to installed state
- [ ] Zero live-path writes in Phase 1, asserted by test
- [ ] `make test` + `make test-launchd-drivers` green
- [ ] Loader rejects a `[roles]` block, naming the owning doc

## Risks

| Risk | Mitigation |
|---|---|
| Golden tests calcify today's accidental state | They assert *equality with installed*, which is the point: Phase 2 must be a no-op. Deliberate divergences are reconciled in Phase 2 under HD-3(c), one at a time. |
| `doctor` is vacuous | Its acceptance criteria are three real, measured failures. It cannot pass without finding them. |
| Passthrough block drifts from models.yml | Out of scope by ratification; `doctor` reports the structural half only (driver role list vs `ailang models role` coverage). |
