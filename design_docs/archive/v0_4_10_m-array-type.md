# M-ARRAY-TYPE: O(1) Indexed Array Type

**Status**: Planned
**Target**: v0.5.0
**Priority**: P1 (Medium) - Enables efficient game data structures
**Estimated**: 1 week
**Dependencies**: None (can be developed in parallel with M-GAME-ENGINE)
**Reported by**: stapledons_voyage (agent inbox)

## AI-First Alignment Check

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Direct indexing instead of list traversal |
| Preserve Semantic Clarity | 0 | 0 | Arrays are well-understood semantically |
| Increase Determinism | 0 | 0 | Same determinism as lists |
| Lower Token Cost | + | +1 | Shorter code for grid operations |
| **Net Score** | | **+2** | **Decision: Move forward** |

## Problem Statement

For game grids (64x64 tiles), AILANG's lists are O(n) for indexed access. Games need `Array[T]` with O(1) get/set for tile lookups, collision detection, and other hot-path operations.

**Current State:**
- Lists `[T]` are linked lists - O(n) index access
- No array type exists in the language
- Original placeholder code used `Array<Tile>` but this type doesn't exist

**Example (current workaround - slow):**
```ailang
-- O(n) list access - too slow for 64x64 grid
func getTile(grid: [Tile], idx: int) -> Tile {
  match grid {
    [] => Tile.Empty,
    h :: t => if idx == 0 then h else getTile(t, idx - 1)
  }
}
```

**Impact:**
- Blocks efficient game grid implementations
- Forces O(n²) algorithms for 2D operations
- Makes AILANG impractical for simulation/game use case

## Goals

**Primary Goal:** Provide `Array[T]` type with O(1) indexed read/write operations.

**Success Metrics:**
- `Array.get(arr, i)` is O(1)
- `Array.set(arr, i, val)` returns new array in O(n) or O(1) with persistent data structures
- 64x64 grid operations complete in <1ms
- Type-safe: `Array[Tile]` prevents storing wrong types

## Solution Design

### Overview

Add `Array[T]` as a built-in type backed by Go slices. Provide stdlib functions for array operations.

### Design Options

**Option A: Immutable Arrays (Pure)**
- `set` returns new array (copy-on-write)
- Purely functional semantics
- Performance: O(n) for set, O(1) for get
- Fits AILANG's functional model

**Option B: Persistent Arrays (RRB Vector)**
- Relaxed Radix Balanced tree structure
- O(log32 n) ≈ O(1) for both get and set
- Complex implementation (~500 LOC)
- Used by Clojure, Scala

**Option C: Mutable Arrays with Effect**
- `set!` mutates in place with `Mut` effect
- Requires effect tracking
- Best performance but changes language semantics

**Recommendation:** Start with **Option A** (immutable), optimize to **Option B** if needed.

### Architecture

**Type System:**
```ailang
-- Array[T] is a built-in parameterized type
type Array[T]  -- Opaque, no constructors exposed

-- Std library operations
module std/array

-- Creation
func make[T](size: int, default: T) -> Array[T]
func fromList[T](xs: [T]) -> Array[T]
func toList[T](arr: Array[T]) -> [T]

-- Access (O(1))
func get[T](arr: Array[T], idx: int) -> Option[T]
func unsafeGet[T](arr: Array[T], idx: int) -> T  -- Panics on out-of-bounds

-- Update (O(n) for immutable, O(log n) for persistent)
func set[T](arr: Array[T], idx: int, val: T) -> Array[T]

-- Properties
func length[T](arr: Array[T]) -> int
func isEmpty[T](arr: Array[T]) -> bool

-- Transformations
func map[T, U](arr: Array[T], f: T -> U) -> Array[U]
func foldl[T, A](arr: Array[T], init: A, f: (A, T) -> A) -> A
```

**Runtime Representation:**
```go
// internal/eval/value.go
type ArrayValue struct {
    Elements []Value  // Go slice for O(1) access
}

func (a *ArrayValue) Get(i int64) (Value, bool) {
    if i < 0 || i >= int64(len(a.Elements)) {
        return nil, false
    }
    return a.Elements[i], true
}

func (a *ArrayValue) Set(i int64, v Value) *ArrayValue {
    if i < 0 || i >= int64(len(a.Elements)) {
        return a  // Or panic for unsafeSet
    }
    // Copy-on-write
    newElements := make([]Value, len(a.Elements))
    copy(newElements, a.Elements)
    newElements[i] = v
    return &ArrayValue{Elements: newElements}
}
```

### Implementation Plan

**Phase 1: Core Type** (~4 hours)
- [ ] Add `Array` to type system (`internal/types/types.go`)
- [ ] Add `ArrayValue` to runtime (`internal/eval/value.go`)
- [ ] Add array literal syntax `#[1, 2, 3]` to parser

**Phase 2: Builtins** (~6 hours)
- [ ] Register `array_make`, `array_get`, `array_set`, `array_length`
- [ ] Implement in `internal/builtins/array.go`
- [ ] Wire to evaluator

**Phase 3: Stdlib** (~4 hours)
- [ ] Create `std/array.ail` module
- [ ] Implement `map`, `foldl`, `fromList`, `toList`
- [ ] Add type signatures

**Phase 4: Testing** (~4 hours)
- [ ] Unit tests for all operations
- [ ] Performance benchmarks
- [ ] Example: 64x64 grid operations

**Phase 5: Go Codegen** (~4 hours)
- [ ] Array → Go slice codegen
- [ ] Ensure M-GAME-ENGINE compatibility

### Files to Create/Modify

**New files:**
- `internal/builtins/array.go` - Array builtins (~150 LOC)
- `stdlib/std/array.ail` - Stdlib wrapper (~50 LOC)
- `internal/builtins/array_test.go` - Tests (~100 LOC)
- `examples/array_grid.ail` - Example usage (~30 LOC)

**Modified files:**
- `internal/types/types.go` - Add Array type (~20 LOC)
- `internal/eval/value.go` - Add ArrayValue (~50 LOC)
- `internal/lexer/lexer.go` - Array literal token (~10 LOC)
- `internal/parser/parser.go` - Array literal syntax (~30 LOC)

## Examples

### Example 1: 2D Grid (Game Use Case)

```ailang
import std/array as A

type Tile = | Empty | Wall | Floor | Entity(int)

-- Create 64x64 grid
func makeGrid(width: int, height: int) -> Array[Array[Tile]] {
  A.make(height, A.make(width, Tile.Floor))
}

-- O(1) access
func getTile(grid: Array[Array[Tile]], x: int, y: int) -> Tile {
  A.unsafeGet(A.unsafeGet(grid, y), x)
}

-- O(n) but single row copy, not whole grid
func setTile(grid: Array[Array[Tile]], x: int, y: int, tile: Tile) -> Array[Array[Tile]] {
  let row = A.unsafeGet(grid, y)
  let newRow = A.set(row, x, tile)
  A.set(grid, y, newRow)
}
```

### Example 2: Collision Detection

```ailang
import std/array as A

type Entity = { id: int, x: int, y: int, solid: bool }

-- Check all entities at position (O(n) but with O(1) grid lookup)
func entitiesAt(entities: Array[Entity], x: int, y: int) -> [Entity] {
  A.foldl(entities, [], \acc. \e.
    if e.x == x && e.y == y then e :: acc else acc
  )
}
```

### Example 3: Array Literal Syntax

```ailang
-- Array literal with #[...]
let primes = #[2, 3, 5, 7, 11, 13]
let first = A.get(primes, 0)  -- Some(2)
let updated = A.set(primes, 0, 1)  -- #[1, 3, 5, 7, 11, 13]
```

## Success Criteria

- [ ] `Array[T]` type works in type system
- [ ] `A.get(arr, i)` is O(1)
- [ ] `A.set(arr, i, v)` returns new array
- [ ] 64x64 grid access completes in <1ms
- [ ] Works with polymorphic functions
- [ ] Go codegen produces `[]T` slices
- [ ] All tests passing
- [ ] Examples and docs added

## Testing Strategy

**Unit tests:**
- Array creation (make, fromList)
- Access (get, unsafeGet, out-of-bounds)
- Update (set, immutability)
- Transformations (map, foldl)

**Performance benchmarks:**
- 64x64 grid access: 1000 random reads
- 64x64 grid update: 1000 random writes
- Compare to list-based implementation

**Integration tests:**
- Game grid scenario
- Collision detection scenario

## Non-Goals

- **Not in this feature:**
  - Mutable arrays with `Mut` effect
  - Persistent vector implementation (can optimize later)
  - 2D array syntax sugar (`arr[x][y]` instead of `A.get(A.get(arr, y), x)`)
  - Negative indexing

## Timeline

**Week 1:**
- Days 1-2: Core type and runtime
- Days 3-4: Builtins and stdlib
- Day 5: Testing and documentation

**Total: ~22 hours across 1 week**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Performance of copy-on-write | Medium | Profile and optimize hot paths |
| Type system complexity | Low | Follow existing list type pattern |
| Go codegen compatibility | Medium | Test with M-GAME-ENGINE early |

## References

- stapledons_voyage feature request (agent inbox, 2025-11-28)
- M-GAME-ENGINE sprint plan (Go codegen)
- Clojure persistent vectors (potential future optimization)

## Future Work

- Persistent array implementation (RRB vectors)
- Mutable arrays with effect tracking
- 2D array convenience functions
- Array slicing and views

---

**Document created**: 2025-11-28
**Last updated**: 2025-11-28
