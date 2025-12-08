# M-GAME-D: Sim Example & Integration

**Status**: Planned
**Target**: v0.5.0 (Phase 4 of M-GAME-ENGINE - Final Phase)
**Priority**: P1 - Medium
**Estimated**: 1.5 weeks (~800 LOC)
**Dependencies**: M-GAME-A, M-GAME-B, M-GAME-C (all previous phases)

## AI-First Alignment Check

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Working example eliminates guesswork for developers |
| Preserve Semantic Clarity | + | +1 | ABI documentation makes contracts explicit |
| Increase Determinism | + | +1 | CI validates deterministic output with same seed |
| Lower Token Cost | 0 | 0 | Documentation, not code reduction |
| **Net Score** | | **+3** | **Decision: Move forward** |

## Problem Statement

**Current State:**
- No working end-to-end example of AILANG → Go game workflow
- No CI validation that the Go codegen pipeline works
- ABI is undocumented - developers don't know what's stable
- Migration path from v0.4.x to v0.5.x unclear

**Impact:**
- Game developers can't verify their setup works
- Regressions in codegen pipeline go unnoticed
- Breaking changes to ABI surprise users
- Adoption blocked by uncertainty

## Goals

**Primary Goal:** Create a complete, working example demonstrating AILANG's game development workflow with CI validation and ABI documentation.

**Success Metrics:**
- `examples/sim_stub/` runs 10 ticks deterministically with same seed
- CI catches any codegen regressions
- `docs/guides/go-interop.md` documents all stable interfaces
- ABI freeze announced in CHANGELOG for v0.5.x

## Solution Design

### Overview

Create a minimal simulation example that proves the entire pipeline works, add CI tests to prevent regressions, and document the stable ABI for game developers.

### Architecture

**Components:**
1. **Sim Stub Example**: Complete AILANG + Go example
2. **CI Integration**: GitHub Actions job for codegen validation
3. **ABI Documentation**: Stable interface documentation

### Design Decisions (Locked)

#### D1: Sim Example is Minimal

The example should be trivially simple - enough to prove the pipeline works, not a real game.

**Includes:**
- World type with a counter
- FrameInput/FrameOutput types
- init_world(seed) and step(world, input) functions
- main.go that runs 10 ticks

**Excludes:**
- Graphics/rendering
- Complex game logic
- Multiple files

#### D2: ABI Stability is "Preview"

Mark interfaces as "stable preview" - breaking changes allowed until v0.6.0 with notice.

### Implementation Plan

**Milestone D1: Sim Stub Example** (~4 days, 400 LOC)
- [ ] Create `examples/sim_stub/world.ail` with types
- [ ] Implement `init_world(seed: int) -> World`
- [ ] Implement `step(world, input) -> (World, FrameOutput)`
- [ ] Create `examples/sim_stub/main.go` driver
- [ ] Add `Makefile` for build workflow
- [ ] Verify output is deterministic with same seed
- [ ] Add `README.md` with usage instructions

**Milestone D2: CI Integration** (~3 days, 200 LOC)
- [ ] Add `make test-sim-stub` target
- [ ] Add GitHub Actions job `test-game-codegen`
- [ ] Test: compile AILANG → Go → build Go → run
- [ ] Compare output to expected (golden file)
- [ ] Fail CI if output differs

**Milestone D3: Documentation & ABI Freeze** (~3 days, 200 LOC)
- [ ] Create `docs/guides/go-interop.md`
- [ ] Document ADT → Go mapping rules
- [ ] Document exported function ABI
- [ ] Document effect handler interface
- [ ] Add ABI stability promise to README
- [ ] Add migration guide v0.4.x → v0.5.x
- [ ] Update CHANGELOG with ABI freeze announcement

### Files to Modify/Create

**New files:**
- `examples/sim_stub/world.ail` - AILANG game types (~50 LOC)
- `examples/sim_stub/main.go` - Go driver (~80 LOC)
- `examples/sim_stub/Makefile` - Build workflow (~20 LOC)
- `examples/sim_stub/README.md` - Documentation (~50 LOC)
- `examples/sim_stub/expected_output.txt` - Golden file (~20 lines)
- `.github/workflows/test-game-codegen.yml` - CI job (~40 LOC)
- `docs/guides/go-interop.md` - ABI documentation (~200 LOC)

**Modified files:**
- `Makefile` - Add test-sim-stub target (~10 LOC)
- `README.md` - Add ABI stability section (~20 LOC)
- `CHANGELOG.md` - Add v0.5.x announcement (~50 LOC)

## Examples

### Example 1: Sim Stub Structure

```
examples/sim_stub/
├── world.ail          # AILANG game types and logic
├── main.go            # Go driver that calls AILANG
├── gen/               # Generated Go code (gitignored)
│   └── game/
│       └── world.go
├── expected_output.txt  # Golden output for CI
├── Makefile           # Build workflow
└── README.md          # Usage instructions
```

### Example 2: world.ail

```ailang
module examples/sim_stub

-- Minimal world state
type World = { tick: int, value: int }

-- Frame input (empty for this example)
type FrameInput = { }

-- Frame output
type FrameOutput = { message: string }

-- Initialize world with seed
export func init_world(seed: int) -> World ! {Rand} {
  rand_seed(seed)
  { tick = 0, value = rand_int(0, 100) }
}

-- Advance one tick
export func step(world: World, input: FrameInput) -> (World, FrameOutput) ! {Rand, Clock} {
  let new_tick = world.tick + 1
  let delta = rand_int(-5, 5)
  let new_world = { tick = new_tick, value = world.value + delta }
  let output = { message = "Tick " ++ int_to_string(new_tick) ++ ": value=" ++ int_to_string(new_world.value) }
  (new_world, output)
}
```

### Example 3: main.go

```go
package main

import (
    "fmt"
    "examples/sim_stub/gen/game"
)

func main() {
    // Initialize with fixed seed for determinism
    world := game.InitWorld(42)

    // Run 10 ticks
    for i := 0; i < 10; i++ {
        var output game.FrameOutput
        world, output = game.Step(world, game.FrameInput{})
        fmt.Println(output.Message)
    }
}
```

### Example 4: CI Job

```yaml
name: Test Game Codegen

on: [push, pull_request]

jobs:
  test-game-codegen:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Build AILANG
        run: make build

      - name: Generate Go code
        run: make -C examples/sim_stub generate

      - name: Build and run
        run: make -C examples/sim_stub run > actual_output.txt

      - name: Compare output
        run: diff expected_output.txt actual_output.txt
```

## Success Criteria

- [ ] `ailang compile --emit-go` generates valid Go for sim_stub
- [ ] `go run main.go` executes 10 ticks without error
- [ ] Output is deterministic with same seed
- [ ] README documents full workflow
- [ ] CI job runs on every PR
- [ ] CI fails if Go codegen broken
- [ ] CI fails if generated Go doesn't compile
- [ ] CI fails if output differs from expected
- [ ] `docs/guides/go-interop.md` complete
- [ ] API stability promise in README
- [ ] CHANGELOG announces ABI freeze
- [ ] All tests passing

## Testing Strategy

**Unit tests:**
- Golden file comparison for generated Go code
- Output determinism with fixed seed

**Integration tests:**
- Full pipeline: AILANG → Go → compile → run → verify output
- Cross-platform (Linux in CI)

**Manual testing:**
- Run on macOS locally
- Verify README instructions work for new user

## Non-Goals

**Not in this feature:**
- Real game logic - Keep it trivial
- Graphics/rendering - Not needed to prove pipeline
- Multiple AILANG files - One file is enough
- Performance benchmarks - Focus on correctness

## Timeline

**Days 1-4** (D1):
- Create sim_stub example
- Test locally
- Verify determinism

**Days 5-7** (D2):
- Add CI job
- Set up golden file comparison
- Test on GitHub Actions

**Days 8-10** (D3):
- Write go-interop.md
- Update README and CHANGELOG
- Final review

**Total: ~10 days across 1.5 weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| ABI freeze premature | High | Mark as "stable preview", allow changes until v0.6.0 |
| CI flaky tests | Medium | Use deterministic seed, golden file |
| Cross-platform issues | Low | Test on Linux (CI), verify on macOS |

## References

- [M-GAME-ENGINE Sprint Plan](M-GAME-ENGINE-sprint-plan.md) - Master sprint plan
- [M-GAME-A Design Doc](../implemented/v0_5_0/M-GAME-A-go-codegen-foundation.md) - Phase 1
- [M-GAME-B Design Doc](../implemented/v0_5_0/M-GAME-B-effects-for-games.md) - Phase 2
- [M-GAME-C Design Doc](M-GAME-C-compiler-ux-extern.md) - Phase 3

## Future Work

- Performance benchmarks
- More complex examples
- Game project template generator
- WASM compilation target

---

**Document created**: 2025-12-01
**Last updated**: 2025-12-01
