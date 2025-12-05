# M-PROMPT-RAND: Update Teaching Prompt with Rand Effect API

**Version**: v0.5.1
**Priority**: Medium
**Estimated Effort**: 30 minutes
**Status**: Planned

## Problem Statement

The v0.5.0 teaching prompt doesn't document the Rand effect and `std/rand` module, making it harder for AI models and users to generate code that uses random number generation for games and simulations.

The Rand effect was added for game development (stapledons_voyage) but isn't visible in the prompt that AI models use to learn AILANG.

## Current State

### Effect List (v0.5.0 prompt line 136)
```markdown
- Effect: `! {IO, FS, Net, Env, Debug, AI}` after return type
```

**Missing:** `Rand` effect

### Effect Summary (v0.5.0 prompt line 404)
```markdown
effects (`! {IO, FS, Net, Env, Debug, AI}`)
```

**Missing:** `Rand` effect

### Standard Library Table (v0.5.0 prompt lines 1005-1016)

| Module | Functions | Import Example |
|--------|-----------|----------------|
| `std/io` | `print`, `println`, `readLine` | ... |
| `std/fs` | ... | ... |
| `std/net` | ... | ... |
| `std/json` | ... | ... |
| `std/env` | ... | ... |
| `std/clock` | `now`, `sleep` | ... |
| `std/string` | ... | ... |
| `std/debug` | `log`, `check` | ... |
| `std/ai` | `call` | ... |
| `std/prelude` | ... | ... |

**Missing:** `std/rand` row

## Available Rand Functions (from internal/builtins/rand.go)

| Function | Signature | Description |
|----------|-----------|-------------|
| `_rand_int` | `(int, int) -> int ! {Rand}` | Random integer in [min, max] inclusive |
| `_rand_float` | `(float, float) -> float ! {Rand}` | Random float in [min, max) |
| `_rand_bool` | `() -> bool ! {Rand}` | Random boolean (coin flip) |
| `_rand_seed` | `(int) -> () ! {Rand}` | Set seed for deterministic results |

## Proposed Changes

### 1. Update Effect List (line 136)

```markdown
- Effect: `! {IO, FS, Net, Env, Debug, AI, Rand}` after return type
```

### 2. Update Effect Summary (line 404)

```markdown
effects (`! {IO, FS, Net, Env, Debug, AI, Rand}`)
```

### 3. Add std/rand to Standard Library Table

| Module | Functions | Import Example |
|--------|-----------|----------------|
| `std/rand` | `_rand_int`, `_rand_float`, `_rand_bool`, `_rand_seed` | `import std/rand (_rand_int)` |

### 4. Add Rand Effect Section (after AI Effect section, around line 990)

```markdown
## Rand Effect (std/rand) - Random Number Generation

**For deterministic games and simulations with reproducible randomness.**

**Import**: `import std/rand (_rand_int, _rand_float, _rand_bool, _rand_seed)`
**Effect**: `! {Rand}`
**Runtime**: `--caps Rand`

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `_rand_int(min, max)` | `(int, int) -> int ! {Rand}` | Random integer in [min, max] inclusive |
| `_rand_float(min, max)` | `(float, float) -> float ! {Rand}` | Random float in [min, max) |
| `_rand_bool()` | `() -> bool ! {Rand}` | Random boolean (50/50 chance) |
| `_rand_seed(seed)` | `(int) -> () ! {Rand}` | Set seed for deterministic generation |

### Example: Dice Rolling Game

```ailang
module game/dice

import std/rand (_rand_int, _rand_seed)

-- Roll a d6 (1-6)
export func rollD6() -> int ! {Rand} =
  _rand_int(1, 6)

-- Roll multiple dice
export func rollDice(n: int, sides: int) -> [int] ! {Rand} {
  if n <= 0
  then []
  else _rand_int(1, sides) :: rollDice(n - 1, sides)
}

-- Deterministic game session
export func main() -> () ! {IO, Rand} {
  _rand_seed(42);  -- Same seed = same results every time
  let roll = rollD6();
  print("You rolled: " ++ show(roll))
}
```

**Running:**
```bash
ailang run --caps IO,Rand --entry main game/dice.ail
```

### Determinism for Game Replays

```ailang
-- Replay a game with the same seed
export func replayGame(seed: int, moves: [Move]) -> GameState ! {Rand} {
  _rand_seed(seed);  -- Initialize with replay seed
  executeGame(initialState, moves)
}
```

**Key design:**
- Same seed produces identical random sequences
- Use `_rand_seed` at game start for reproducible behavior
- Track seed with game state for replays/saves
```

### 5. Add to "What Works" Section (line 402-406)

Update to include `Rand`:

```markdown
## What Works (v0.5.1)

Modules, functions, lambdas, pattern matching, ADTs, records, effects (`! {IO, FS, Net, Env, Debug, AI, Rand}`), ...
```

## Files to Modify

1. **cmd/ailang/prompts/v0.5.1.md** (create new version)
   - Update effect lists
   - Add std/rand to stdlib table
   - Add Rand Effect section with examples

2. **cmd/ailang/prompts/versions.json**
   - Add v0.5.1 entry

## Success Criteria

1. AI models can generate code using `std/rand` functions
2. Rand effect appears in all effect lists
3. Examples compile and run correctly
4. Teaching prompt validates with `ailang prompt --version v0.5.1`

## Notes

- Function names use `_rand_` prefix (underscore convention for builtins)
- Rand effect requires `--caps Rand` at runtime
- Deterministic by default with seed 0 (can be overridden)
