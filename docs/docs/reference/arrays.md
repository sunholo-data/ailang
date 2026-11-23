---
sidebar_position: 5
title: Arrays
description: Fixed-size arrays in AILANG (std/array)
reviewed: 2026-11-23
reviewBy: 2027-05-23
---

# Arrays

AILANG provides **fixed-size immutable arrays** (`Array[T]`) in `std/array` as a complement to lists (`[T]`). Arrays give O(1) indexed reads and contiguous storage (including an unboxed 8-byte-per-element layout for `Array[float]`).

## Arrays vs Lists

| Feature | Arrays `Array[T]` | Lists `[T]` |
|---------|-------------------|-------------|
| **Access** | O(1) indexed (`get`, `getOpt`) | O(n) traversal |
| **Update** | O(n) copy-on-write (`set`), or O(n + k) batch (`updateMany`, `accumulate`) | Structural / `map` / `filter` |
| **Syntax** | `#[1, 2, 3]` or `fromList([1, 2, 3])` | `[1, 2, 3]` |
| **Use case** | Grids, lookup tables, numeric vectors | Recursive processing, pipelines |

## Array Syntax & Construction

Use `#[...]` for array literals, or construct arrays with `std/array` functions (`empty`, `make`, `fromList`):

```ailang
import std/array as A

func constructionDemo() -> int {
  let numbers: Array[int] = #[1, 2, 3, 4, 5]
  let names: Array[string] = #["Alice", "Bob", "Charlie"]
  let emptyArr: Array[int] = A.empty()
  let zeros: Array[int] = A.make(4, 0)
  let fromXs: Array[int] = A.fromList([10, 20, 30])
  A.length(numbers) + A.length(names) + A.length(emptyArr) + A.length(zeros) + A.length(fromXs)
}
```

Nested arrays represent 2D grids:

```ailang
import std/array as A

func gridCenter() -> int {
  let grid: Array[Array[int]] = #[
    #[1, 2, 3],
    #[4, 5, 6],
    #[7, 8, 9]
  ]
  A.get(A.get(grid, 1), 1)
}
```

## Reading Elements & Length

Indexing is 0-based via `get` (runtime error if out of bounds) or `getOpt` (returns `Option[a]`):

```ailang
import std/array as A

func readDemo() -> (int, Option[int], Option[int], int) {
  let arr = #[10, 20, 30, 40, 50]
  let first = A.get(arr, 0)        -- 10
  let inBounds = A.getOpt(arr, 2)  -- Some(30)
  let oob = A.getOpt(arr, 99)      -- None
  let len = A.length(arr);         -- 5
  (first, inBounds, oob, len)
}
```

## Updating Arrays (Copy-on-Write vs Batch Updates)

Arrays are immutable. Single-element `set` and `append` copy the whole array (**O(n)**), so updating in a loop is O(n²). For multiple writes, use `updateMany` or `accumulate` (**O(n + k)**, one copy):

```ailang
import std/array as A

func updateDemo() -> (Array[int], Array[int], Array[int]) {
  let base = A.make(4, 0)

  -- Single update (O(n))
  let one = A.set(base, 1, 42)

  -- Batch writes in one copy (O(n + k); later write to same index wins)
  let many = A.updateMany(base, [(0, 10), (2, 20), (0, 30)])

  -- Batch fold per index in one copy (O(n + k); e.g. histogram)
  let hist = A.accumulate(base, [(0, 1), (2, 5), (0, 2)], \a b. a + b);

  (one, many, hist)
}
```

## Transforming Arrays via `std/list`

`std/array` intentionally keeps its surface minimal. To map, filter, or fold an array, convert with `toList` / `fromList` and use `std/list`:

```ailang
import std/array as A
import std/list (map, filter, foldl)

func transformDemo(arr: Array[int]) -> (Array[int], Array[int], int) {
  let xs = A.toList(arr)
  let doubled = A.fromList(map(\x. x * 2, xs))
  let evens = A.fromList(filter(\x. x % 2 == 0, xs))
  let total = foldl(\acc x. acc + x, 0, xs);
  (doubled, evens, total)
}
```

## 2D Grid Pattern

```ailang
import std/array as A
import std/list (map, range)

type Cell = Empty | Wall | Player | Enemy
type Grid = Array[Array[Cell]]

func createGrid(width: int, height: int) -> Grid {
  A.fromList(map(\y.
    A.fromList(map(\x.
      if x == 0 || x == width - 1 || y == 0 || y == height - 1
        then Wall
        else Empty,
      range(0, width)
    )),
    range(0, height)
  ))
}

func getCell(grid: Grid, x: int, y: int) -> Cell {
  A.get(A.get(grid, y), x)
}

func setCell(grid: Grid, x: int, y: int, cell: Cell) -> Grid {
  let row = A.set(A.get(grid, y), x, cell)
  A.set(grid, y, row)
}
```

## Float Kernels & Binary Codecs (`@experimental`)

An `Array[float]` uses an unboxed 8-byte-per-element representation. `std/array` provides native numeric kernels (`dot`, `axpy`, `scale`, `add`, `sub`, `mul`, `sum`, `argmax`, `scatterAdd`) and packed little-endian binary codecs (`encodeF64LE`, `decodeF64LE`, `encodeF32LE`, `decodeF32LE`):

```ailang
import std/array as A

func vectorDemo() -> (float, int, int) {
  let x = #[1.0, 2.0, 3.0]
  let y = #[4.0, 5.0, 6.0]
  let d = A.dot(x, y)            -- 32.0
  let z = A.axpy(2.0, x, y)      -- #[6.0, 9.0, 12.0]
  let bestIdx = A.argmax(z)      -- 2
  let packed = A.encodeF64LE(z)
  let decodedLen = match A.decodeF64LE(packed) {
    Ok(v) => A.length(v),
    Err(_) => 0
  };
  (d, bestIdx, decodedLen)
}
```

## Cost Model Summary

| Function | Complexity | Notes |
|----------|------------|-------|
| `empty()`, `length(arr)`, `get(arr, i)`, `getOpt(arr, i)` | O(1) | `get` fails on OOB; `getOpt` returns `None` |
| `make(n, v)`, `fromList(xs)`, `toList(arr)` | O(n) | Contiguous allocation |
| `set(arr, i, v)`, `append(arr, v)` | O(n) | Full copy — avoid in tight loops |
| `updateMany(arr, ws)`, `accumulate(arr, ws, f)` | O(n + k) | One copy for `k` writes |
| `dot`, `axpy`, `scale`, `add`, `sub`, `mul`, `sum`, `argmax` | O(n) | `@experimental`; strict equal-length check on binary ops |

## Related Resources

- [Language Syntax](/docs/reference/language-syntax) — Complete syntax reference
- [Limitations](/docs/reference/limitations) — Known runtime and representation bounds
