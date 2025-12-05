---
sidebar_position: 5
title: Arrays
description: Fixed-size arrays in AILANG
---

# Arrays

AILANG v0.5.6 introduces **fixed-size arrays** as a complement to linked lists. Arrays provide efficient indexed access and are essential for game development and performance-critical code.

## Arrays vs Lists

| Feature | Arrays `Array[T]` | Lists `List[T]` |
|---------|-------------------|-----------------|
| **Size** | Fixed at creation | Dynamic |
| **Access** | O(1) indexed | O(n) traversal |
| **Syntax** | `#[1, 2, 3]` | `[1, 2, 3]` |
| **Memory** | Contiguous | Linked nodes |
| **Use case** | Game grids, buffers | Recursive processing |

## Array Syntax

### Array Literals

Use `#[...]` to create arrays:

```typescript
-- Integer array
let numbers: Array[int] = #[1, 2, 3, 4, 5]

-- String array
let names: Array[string] = #["Alice", "Bob", "Charlie"]

-- Empty array (requires type annotation)
let empty: Array[int] = #[]

-- Nested arrays
let grid: Array[Array[int]] = #[
  #[1, 2, 3],
  #[4, 5, 6],
  #[7, 8, 9]
]
```

### Array Type

The `Array[T]` type is parameterized by element type:

```typescript
-- Type annotations
let xs: Array[int] = #[1, 2, 3]
let ys: Array[string] = #["hello", "world"]
let zs: Array[bool] = #[true, false, true]

-- In function signatures
func processArray(arr: Array[int]) -> int { ... }
```

## Array Operations

### Indexing

Access elements by index (0-based):

```typescript
let arr = #[10, 20, 30, 40, 50]

arr[0]  -- 10
arr[2]  -- 30
arr[4]  -- 50
```

**Bounds checking:** Out-of-bounds access results in a runtime error.

### Length

Get array size:

```typescript
let arr = #[1, 2, 3, 4, 5]
length(arr)  -- 5

let empty: Array[int] = #[]
length(empty)  -- 0
```

### Mapping

Transform each element:

```typescript
let arr = #[1, 2, 3, 4, 5]
let doubled = arrayMap(\x. x * 2, arr)
-- #[2, 4, 6, 8, 10]

let strings = arrayMap(show, arr)
-- #["1", "2", "3", "4", "5"]
```

### Folding

Reduce array to single value:

```typescript
let arr = #[1, 2, 3, 4, 5]

-- Sum
let sum = arrayFold(\acc x. acc + x, 0, arr)
-- 15

-- Product
let product = arrayFold(\acc x. acc * x, 1, arr)
-- 120

-- Max (requires initial value)
let max = arrayFold(\acc x. if x > acc then x else acc, arr[0], arr)
-- 5
```

### Filtering

Select elements matching a predicate:

```typescript
let arr = #[1, 2, 3, 4, 5, 6, 7, 8, 9, 10]

let evens = arrayFilter(\x. x % 2 == 0, arr)
-- #[2, 4, 6, 8, 10]

let large = arrayFilter(\x. x > 5, arr)
-- #[6, 7, 8, 9, 10]
```

### Slicing

Extract subarrays:

```typescript
let arr = #[0, 1, 2, 3, 4, 5, 6, 7, 8, 9]

arraySlice(arr, 2, 5)  -- #[2, 3, 4] (from index 2, length 5)
arraySlice(arr, 0, 3)  -- #[0, 1, 2]
arraySlice(arr, 7, 3)  -- #[7, 8, 9]
```

## Array Patterns

### Iteration

Process each element:

```typescript
func printAll(arr: Array[string]) -> () ! {IO} {
  arrayForEach(\s. println(s), arr)
}

printAll(#["Hello", "World", "!"])
-- Prints:
-- Hello
-- World
-- !
```

### Finding Elements

Search for elements:

```typescript
let arr = #[10, 20, 30, 40, 50]

-- Find first matching element
let found = arrayFind(\x. x > 25, arr)
-- Some(30)

-- Find index
let idx = arrayFindIndex(\x. x == 30, arr)
-- Some(2)
```

### Checking Conditions

Test array properties:

```typescript
let arr = #[2, 4, 6, 8, 10]

-- All elements match predicate
arrayAll(\x. x % 2 == 0, arr)  -- true

-- Any element matches predicate
arrayAny(\x. x > 8, arr)  -- true
```

## Game Development Patterns

### 2D Grids

```typescript
type Cell = Empty | Wall | Player | Enemy

type Grid = Array[Array[Cell]]

func createGrid(width: int, height: int) -> Grid {
  arrayInit(height, \y.
    arrayInit(width, \x.
      if x == 0 || x == width - 1 || y == 0 || y == height - 1
        then Wall
        else Empty
    )
  )
}

func getCell(grid: Grid, x: int, y: int) -> Cell {
  grid[y][x]
}

func setCell(grid: Grid, x: int, y: int, cell: Cell) -> Grid {
  arrayUpdate(grid, y, arrayUpdate(grid[y], x, cell))
}
```

### Entity Arrays

```typescript
type Entity = {
  id: int,
  x: float,
  y: float,
  health: int
}

func updateEntities(entities: Array[Entity], dt: float) -> Array[Entity] {
  arrayMap(\e. {
    id: e.id,
    x: e.x + dt,
    y: e.y,
    health: e.health
  }, entities)
}

func filterAlive(entities: Array[Entity]) -> Array[Entity] {
  arrayFilter(\e. e.health > 0, entities)
}
```

### Buffers

```typescript
type RingBuffer[a] = {
  data: Array[a],
  head: int,
  size: int
}

func push[a](buf: RingBuffer[a], item: a) -> RingBuffer[a] {
  let newHead = (buf.head + 1) % length(buf.data);
  {
    data: arrayUpdate(buf.data, newHead, item),
    head: newHead,
    size: min(buf.size + 1, length(buf.data))
  }
}
```

## Go Codegen

Arrays compile to Go slices with proper typing:

**AILANG:**
```typescript
let scores: Array[int] = #[100, 95, 87, 92]
let total = arrayFold(\acc x. acc + x, 0, scores)
```

**Generated Go:**
```go
scores := []int{100, 95, 87, 92}
total := ArrayFold(func(acc, x int) int { return acc + x }, 0, scores)
```

### ADT Arrays in Go

Arrays of ADTs compile to properly typed slices:

**AILANG:**
```typescript
type Action = Move(int, int) | Attack(int) | Wait

let actions: Array[Action] = #[
  Move(1, 0),
  Attack(5),
  Wait
]
```

**Generated Go:**
```go
type Action interface { isAction() }
type Move struct { X, Y int }
type Attack struct { Damage int }
type Wait struct {}

actions := []Action{
  Move{1, 0},
  Attack{5},
  Wait{},
}
```

## Performance Considerations

### When to Use Arrays

- **Fixed-size data**: Game boards, lookup tables
- **Random access**: Frequent indexing operations
- **Go interop**: Arrays map directly to Go slices
- **Performance critical**: Contiguous memory layout

### When to Use Lists

- **Recursive algorithms**: Pattern matching on head/tail
- **Unknown size**: Building collections incrementally
- **Functional patterns**: Map/filter chains with lazy evaluation

## Limitations

### Current Limitations (v0.5.6)

1. **No array mutation**: Arrays are immutable; `arrayUpdate` returns a new array
2. **No array comprehensions**: Use `arrayInit` with a lambda instead
3. **Bounds checking at runtime**: Out-of-bounds access panics

### Planned Improvements

- **Array patterns**: `match arr { #[x, y, z] => ... }`
- **Array comprehensions**: `#[x * 2 | x <- #[1, 2, 3]]`
- **Static bounds checking**: Compile-time verification where possible

## Related Resources

- [Language Syntax](/docs/reference/language-syntax) - Complete syntax reference
- [Go Codegen](/docs/guides/go-interop) - Compiling to Go
- [Pattern Matching](/docs/reference/language-syntax#pattern-matching) - ADTs and matching
- [Roadmap: Execution Profiles](/docs/roadmap/execution-profiles) - Game development profiles
