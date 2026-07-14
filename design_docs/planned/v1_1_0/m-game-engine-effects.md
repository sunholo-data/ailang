# M-GAME-ENGINE-EFFECTS: the game engine as typed effects — Stapledon's Voyage revived as v1.1's flagship

**Status**: Planned (v1.1 arc spine — approved direction, Mark 2026-07-14; full design via
design-doc-creator post-1.0)
**Target**: v1.1.0
**Priority**: P1 (v1.1 headline: "the bytecode VM grows up, proven by a game")
**Estimated**: TBD at full design (engine-effects host ~1wk; game port ~1wk; VM flagship wiring ongoing)
**Dependencies**: bytecode VM maturation (m-perf4-bytecode-interpreter, m-bytecode-vm-parity-bugs);
the DrawCmd bridge lineage (internal/gen/golang DrawCmd conversion, test-game-codegen.yml — the
OLD approach this supersedes); effect system (Render/Input/Clock as new host effects)

---

## The inversion (the decision of record)

Stapledon's Voyage v1 was built as *"compile AILANG to Go so the game is fast"* — the Go-codegen
path the v0.11 design committee subsequently demoted as non-convergent. This doc inverts it:

**AILANG is the game-logic and orchestration brain; the engine substrate (rendering, input,
timing, audio) is Go host code exposed as typed effects** — `!{Render, Input, Clock}` — exactly
the `!{AI, Net, FS}` pattern. Not a slowness workaround: the language's identity applied to
games. Deterministic, replayable game logic (replay-verified game state is a novel property no
mainstream engine offers) with frame-critical inner loops living in the host where they belong.

## Performance posture (grounded, 2026-07-14)

- Evaluator today: fast enough for turn-based/event-driven logic and likely 10–30Hz logic ticks
  (post perf6/goroutine-id work), NOT for per-frame inner loops (6,900×+ native gap is
  architectural).
- **The game is the bytecode VM's flagship consumer and forcing function**: frame-time is the
  crispest perf KPI the VM could get. The game ships on evaluator+effects first, then gets
  faster as the VM lands underneath WITHOUT the game changing.
- emit-go-v2: FROZEN (Mark 2026-07-14, ratify at release gate) — contracts projection stays live.

## Scope sketch (full design post-1.0)

1. `std/game` or host-effect package: Render (DrawCmd successor), Input, Clock[mode=frame] —
   routed through the extension lane (apps/ or pkg/ citizen, NOT compiler core).
2. Port Stapledon's Voyage game logic to pure AILANG over these effects; delete its Go-port
   dependency.
3. VM flagship wiring: frame-budget benchmark in the eval rotation; `--bytecode` A/B on the
   game's logic tick as the VM's standing regression+progress metric.

## Non-Goals
- Resurrecting Go source codegen for game speed (committee decision stands).
- A general-purpose engine competing with Unity/Godot — this is the AILANG-native
  deterministic-logic engine + demo, and the VM's proving ground.

---
**Document created**: 2026-07-14 (strategic audit follow-up; Mark: "action this")
