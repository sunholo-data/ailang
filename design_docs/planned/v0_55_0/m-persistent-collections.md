# M-PERSISTENT-COLLECTIONS: Collections That Are Cheap to Update (Set, structured keys, batch accumulate, persistent Map/Array)

**Status**: Phase 1a–1c implemented 2026-10-28 (see [Phase 1 implementation record](#phase-1-implementation-record-2026-10-28)); Phase 1d (prompt) follows; Phases 2–3 not started (D1, D3 undecided). Quorum round 1 was BLOCKED (3 rejects); all three objections are fixed in this revision; it has not been re-quorumed (see [Quorum log](#quorum-log)).
**Target**: v0.55.0 (Phase 1); Phases 2–3 v0.56.0
**Priority**: P1. This is the largest confirmed AILANG-vs-Python algorithmic delta in the 2026-10-28 external-benchmark spike.
**Estimated**: Phase 1 is 3 days, Phase 2 is 4–5 days, Phase 3 is 4 days. That is about 11–12 days sequential, already doubled for honesty.
**Dependencies**: None hard. **Coordination**: [m-list-cons-cells-decomposition](../m-list-cons-cells-decomposition.md) (D-19 programme, lists only, see [Distinction from the list-cons programme](#distinction-from-the-list-cons-programme)).
**Planner-Lane**: opus-required for Phases 2–3 (runtime value representation in two engines). Phase 1 is eligible for cheaper routing.
**Quorum trigger**: #1 fired, because the doc has design-freeze items (persistent representation, key-order contract). See the [Quorum log](#quorum-log).

## Routing Lane (per design_docs/PROGRAM.md §4)

**Lane: AILANG fix.** PROGRAM.md §1 gives AILANG the "gaps get fixed as the loop finds them" trajectory, and §4 routes a missing builtin or runtime capability to an AILANG fix. No motoko extension can fix this. A prompt-shaping extension could tell a model "use a visited set", but no visited set exists, and every update primitive that does exist is O(n) (V1–V5). The prompt half (cost-model teaching) goes to the **prompt-manager lane** as Phase 1d and depends on the runtime half landing first. Teaching an O(n) `Map.insert` as the visited-set idiom would teach a quadratic algorithm.

## Problem Statement

Every AILANG collection that can be updated costs **O(n) per update**:

| Structure | Update | Cost at HEAD | Evidence |
|---|---|---|---|
| `std/map` | `insert`, `remove` | O(n) copy-on-write | V1 (measured 3.9× per doubling), V6 |
| `std/array` | `set`, `append` | O(n) copy | V2 (measured 3.6–4× per doubling at n ≥ 40k) |
| list | `nth`, `++` (left) | O(n) | std/list cost header; the D-19 programme owns `::` |
| set | — | **does not exist** | V3 |
| int accumulate into an array | — | **does not exist**: `scatterAdd` is `Array[float]` only | V4 |

There are also two **check-passes, runtime-fails** seams on the very structures an algorithm needs:

- `Map` keys that are tuples or records type-check, but `insert` fails at runtime with `unsupported map key type: *eval.TupleValue (only int, string, bool supported)`. Both engines behave the same way (V7, V8).
- `lookup` and `member` with such a key **silently return `false`/`None`** in both engines, with no error (V9). This violates CLAUDE.md §2 ("no silent fallbacks").

The result is that the standard competitive-programming toolkit (visited set, BFS frontier, DP table, difference array, union-find parent array) has no form in AILANG that runs in the same complexity class as Python.

### Measured impact (2026-10-28 spike, AetherCode ICPC/CCPC 2024, Claude Haiku 4.5 agent mode, hidden tests)

| Problem | Python | AILANG | Mechanism, confirmed by patching only one variable |
|---|---|---|---|
| cp_60159 | 3/3 trials pass small hidden tests, 0.05s | 0/3 | Python stops at the first repeated state with `visited = set()`. AILANG has **no visited set** and instead runs a `4·n²` step cap: about 40M steps on a 100×100 grid with 1,000 queries, so it times out. Changing **only** the grid to `std/array` (O(1) `get`) leaves the result unchanged: **12/47 hidden tests either way** (V11). Indexing was not the cause. |
| cp_60164 | fast trial 0.11s (mutable 2D **difference array**) | 0/4 | AILANG used O(n³) brute force. Two Python trials did too and take 9.5s and 49s. Replacing **only** the `++` appends with cons still times out above 60s on both the evaluator and the VM (V12). The cheap algorithm needs int accumulate into an array, which does not exist (V4). |
| cp_60153 | 1/1 (deque flood fill) | 0/4 (one trial got 16/23 with timeouts) | Weaker evidence: a grid flood fill was replaced by a wrong "flat interval" heuristic. |
| cp_60170 | 1/3 | 0/4 | Weaker evidence: BFS with sets and dicts was replaced by subset enumeration. |

**Structure census** (V13, heuristic grep):

| Run | AILANG `std/array` | AILANG `std/map` | Python sets/dicts/deques/grids |
|---|---|---|---|
| First run | 1/20 | 0/20 | 18/20 |
| Confirm trials | 0/21 | 2/21 (both small histogram maps, not visited sets) | 11/21 |

The teaching prompt never mentions `std/set` (it does not exist). It also never states that `Map.insert` is unsuitable inside a search loop, and never mentions the key-type restriction (V14).

**Honesty notes on the evidence:**

- cp_60159 also has a **set-free** solution today: Floyd or Brent cycle detection works on the deterministic `(r, c, dir)` transition with O(1) memory. A strong model could pass it without this doc. It is still evidence, because the idiomatic Python solution (and the one Haiku wrote) needs a cheap set, and the AILANG model avoided the set because no cheap one exists. It did not choose Floyd.
- cp_60164's cheap path could in principle be built today by sorting the 4n corner updates and folding. That is a much harder algorithm than `diff[x][y] += 1`, so it is the "algorithm downgrade" pattern again.
- cp_60153 and cp_60170 rest on 2 or fewer valid Python trials, so treat them as supporting, not load-bearing.

## Goals

**Primary goal:** every structure in the competitive-programming toolkit has a pure AILANG form whose update costs O(1) amortized or O(log n), with no change to AILANG's pure, deterministic semantics.

**Success metrics:**

1. `v/mapins.ail` (V1, 20,000 sequential inserts) drops from 5.25s to **under 0.2s** on the evaluator, and similarly on `--bytecode`.
2. `v/arrset.ail` (V2, 160,000 sequential `set`s) drops from 24.2s to **under 1s**.
3. A visited-set version of cp_60159 written in AILANG (written as the Phase 2 acceptance fixture) passes **all 47 hidden tests** within 20s each. The baseline is 12/47.
4. A difference-array version of cp_60164 using `accumulate` (Phase 1) passes the 15KB small test in under 2s. The baseline is a timeout above 60s.
5. **No regression** on read-heavy workloads: `Map.lookup` and `Array.get` microbenchmarks are within 1.5× of HEAD (kill criterion for Phases 2 and 3).

## Distinction from the list-cons programme

The [D-19 cons-cells programme](../m-list-cons-cells-decomposition.md) (and its parked predecessor [m-list-cons-quadratic](../m-list-cons-quadratic.md)) makes **`x :: xs` O(1)** by changing `ListValue`'s representation. It owns lists only. This doc **does not touch `ListValue`**, `::`, `++`, or `nth`. Lists stay sequential structures, and the list-cons doc already names `ArrayValue` as the documented alternative for indexed access (its row N13). This doc owns **Map, Set (new), and Array**, the structures that are supposed to be the cheap alternative and currently are not cheap to update. The two programmes touch **disjoint symbols but overlapping files**. The list-cons blast radius is `ListValue`/`.Elements`, and this one is `MapValue`/`MapObj`/`ArrayValue`/`ArrayObj` (V15, V16). However, **12 of the 15 files this doc modifies also reference list values** (V21): `eval/value.go`, `bytecode/value.go`, `vm/convert.go`, `vm/builtins.go`, `builtins/xml_query.go` and others. At git-merge granularity they will conflict.

**Sequencing rule:**
- Phase 1a touches the key encoders. `bytecode/value_map.go` has 0 list references, but `eval/value.go` has 4. Phase 1a may run alongside D-19 piece work only if it lands as a small, separate commit, rebased before the D-19 piece that edits `eval/value.go`.
- Phases 2–3 must **not** run concurrently with D-19's LC-1 or LC-4 representation pieces. Whichever programme reaches a shared file first lands first; the other rebases.

It also completes two deferrals that are already on record:

- [m-std-map-and-array-gaps](../../implemented/v0_11_0/m-std-map-and-array-gaps.md) (v0.11.0) chose O(n) copy-on-write "honest about cost; XLSX is build-once-read-many; **persistent HAMT deferred to v1.0**".
- [m-hash-collections](../../implemented/v0_9_2/m-hash-collections.md) Phase 2 specified `std/set` with O(1) insert. Map shipped, Set never did (V3).

## Design Options

| Option | What it buys | Cost / risk | Verdict |
|---|---|---|---|
| **A. Persistent HAMT/CHAMP for Map and Set** | O(log₃₂ n) insert/remove with structural sharing. API and semantics are unchanged, and every existing program keeps working | `lookup` goes from 1 hash probe to about 2–4 trie hops; more memory per entry; two engines to implement (one shared Go package fixes that) | **Chosen (Phase 2)** |
| **B. Persistent vector (32-way trie or RRB) for `Array[a]`** | O(log₃₂ n) `set`/`append`, with `get` still effectively constant at 2–4 hops | Packed `Array[float]` feeds the numerics builtins (`dot`, `axpy`, encode/decode) as flat `[]float64`, so a trie would cost every numeric kernel a flatten | **Chosen for boxed arrays only (Phase 3)**. Packed float arrays stay flat (see Decision 3) |
| **C. Scoped local mutation (transient/builder: `A.withMutable(arr, \b. …)`)** | O(1) writes inside the scope, and pure from the outside | Needs either an escape or linearity analysis (new type-system surface, A5/A8) or a runtime frozen-flag check (fails at runtime, A11). It is also a **new idiom models must discover**, and the census shows they do not even discover `std/array` (V13). It adds no complexity class that A+B lack | **Deferred**: revisit only if Phase 2/3 constant factors miss the success metrics |
| **D. Uniqueness-based in-place update** (Koka/Lean "functional but in place") | O(1) `set` when the array is uniquely referenced, invisibly | AILANG runs on Go's GC, with no reference counts, so this needs whole-program ownership analysis in both engines. Performance would depend on aliasing the programmer cannot see, which works against A9 | **Rejected for now** |
| **E. `std/set` (over the Map representation)** | The missing visited-set structure | Thin; inherits whatever Map costs | **Chosen (Phase 1)**, initially over today's Map, becoming O(log n) automatically when A lands |
| **F. Structured map/set keys** (tuples, records, ADTs of hashable leaves via one canonical encoder) | Fixes the check-passes, runtime-fails seam (V7–V9); `visited: Set[(int, int, Dir)]` becomes writable | Must define a canonical, deterministic encoding and order (Decision 2); function values stay unhashable | **Chosen (Phase 1)** |
| **G. `std/array.accumulate(arr, writes, combine)`** (generic batch combine, O(n + k), one copy) | Difference arrays, histograms and counting sorts in one pure call, independent of persistence | Small; generalises the float-only `scatterAdd` | **Chosen (Phase 1)** |
| **H. Teaching-prompt cost-model section** (prompt-manager lane) | Models learn which structure to reach for | Useless before E/F/G exist (it would teach O(n) loops) | **Chosen (Phase 1d, after 1a–1c)** |

**Recommendation:** start with **E + F + G + H** (Phase 1: cheap, no representation change, unblocks the toolkit's *expressibility* now), then **A** (Phase 2: Map and Set become O(log n)), then **B** for boxed arrays (Phase 3). Hold C and D unless Phase 2/3 miss the kill criteria.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| D1: Persistent representation for Map/Set: CHAMP (compressed HAMT) vs HAMT vs balanced tree | Sets constants, memory and iteration cost for every map in every program, in both engines | human | design | high |
| D2: Canonical key encoding and **iteration order** for structured keys. Preserve today's string-sorted order, in which int keys iterate **lexically** (`[10, 2, 9]`, V10)? | `keys`/`toList`/`show` output is observable. Changing it changes the output of existing programs and breaks replay (A2) | human | design | high |
| D3: `Array[float]` stays flat-packed (O(n) `set`), or becomes a trie of packed leaves | Numerics kernels (`dot`, `axpy`, `encodeF64LE`) assume a flat `[]float64` | human | design | high |
| D4: One shared Go package (`internal/pcoll`) used by the evaluator and the VM, vs a per-engine implementation | There are two map-key encoders today (`eval/value.go:286`, `bytecode/value_map.go:37`, V8). A third copy would be the bug factory this repo keeps paying for | agent (recommend shared) | design | med |
| D5: Function-typed keys: a type error (new constraint) vs a loud runtime error | Choosing a type error adds type-system surface (A5); a runtime error keeps it out, but must be loud for `lookup`/`member` too, not just `insert` | human | compile | med |
| D6: `accumulate` signature: `(Array[a], [(int, a)], (a, a) -> a)` vs separate `scatterAddInt` | API surface (A8) vs generality | agent | compile | low |

### Design Freeze

- [ ] D1: persistent representation (recommended: **CHAMP**. It has canonical structure for equal maps, which keeps structural `==` cheap and deterministic.)
- [x] D2: iteration order (recommended: **preserve** the current canonical-string sort exactly, including lexical int order, and record a separate decision on whether to ever change it). **Taken as the default on 2026-10-28 for the Phase 1 implementation, pending human confirmation.** Int/string/bool bytes are unchanged; the new kinds' encoding is in [Phase 1 implementation record](#phase-1-implementation-record-2026-10-28).
- [ ] D3: float arrays (recommended: **stay flat**. Numeric code is bulk/kernel-shaped, and `updateMany`/`accumulate` cover its batch writes.)
- [ ] D5: function keys (recommended: **loud runtime error on every operation**, with no new constraint in this doc). Phase 1 implements the recommendation (insert, lookup, member, remove all error); the box stays open for a human to confirm that no type-level constraint is wanted.

## Solution Design

### Phase 1 — Expressibility (3 days, v0.55.0, no representation change)

**1a. One canonical key grammar, widened.** Today `eval.MapKey` and `bytecode.MapKey` are two hand-kept copies of one encoding, and both handle only int, string and bool (V8).

**The import-cycle constraint and how it is met:**
- `internal/bytecode` imports no `internal/` package, and `internal/eval/value.go` imports only `internal/builtins` and `internal/types` (V22).
- So a leaf package `internal/pcoll` can be imported by both only if it imports **neither**. It therefore never sees `*eval.TupleValue` or `bytecode.Value`.

The mechanism is dependency inversion:
- `pcoll` owns the **byte grammar** through a `KeyWriter` builder (`Int(int64)`, `Str(string)`, `Bool(bool)`, `Float(uint64 bits)`, `Unit()`, `BeginTuple(n)`, `BeginRecord(fieldNames)`, `BeginCtor(adt, tag, n)`, `End()`), and returns the canonical string.
- Each engine keeps a small walker (~40 LOC) that walks its own value type and calls the writer.
- That leaves one grammar and two walkers. Divergence between the walkers is caught mechanically: a shared golden test builds equivalent values in both engines (via the existing `vm/convert.go` bridge) and asserts byte-identical keys for every type, including nested ones.
- The same inversion carries Phases 2–3: `pcoll`'s CHAMP and pvec are Go generics (`champ.Map[V any]` keyed by the canonical string, `pvec.Vec[V any]`), which each engine instantiates with its own value type. Add tuples, records (field-name sorted), ADT constructors (tag plus payload), unit and float (bit-exact, with NaN rejected loudly), all recursively. The existing int, string and bool encodings stay **byte-identical** (`i:%d`, `s:…`, `b:%t`), so existing maps keep their order (D2). Any other value (functions, effect handles) returns an error naming the type, and **`lookup`/`member` propagate it** instead of returning `false` (V9 fix).

**1b. `std/set`.** `Set[a]` with `empty`, `fromList`, `insert`, `remove`, `member`, `size`, `toList` (canonical order), `union`, `intersect`, and `difference`. It is represented over the Map runtime (`Map[a, unit]`), so it inherits 1a's keys and, later, Phase 2's O(log n).

**1c. `std/array.accumulate`.** `accumulate[a](arr: Array[a], writes: [(int, a)], combine: (a, a) -> a) -> Array[a]` makes one copy and folds the writes in list order. It is O(n + k) and pure, and out-of-bounds indexes fail loudly. `scatterAdd` stays as the packed-float fast path.

**1d. Teaching prompt (prompt-manager lane).** Add a 15–25 line "Choosing a structure for algorithmic work" section that covers:
- the cost table;
- a visited `Set[(int, int)]`, a BFS frontier as two lists, a DP table via `Array` plus `accumulate`/`updateMany`, and a difference array;
- an explicit statement that **single-element `set`/`insert` inside a hot loop is O(n) until Phase 2/3 ship**.

Measure it with an A/B on the new eval benchmarks before keeping it (PROGRAM.md 5c: "measure what each section earns").

### Phase 2 — Persistent Map/Set (4–5 days)

Implement CHAMP in the shared package from D4, with O(log₃₂ n) insert/remove/lookup and structural sharing. `keys`/`values`/`toList` stay sorted by canonical key (they are already O(n log n), V6 header), so iteration semantics do not change. `fromList` builds through a transient internal builder, which is invisible from AILANG. Equality is structural over canonical entries, as it is today. The evaluator's `MapValue` and the VM's `MapObj` both wrap the shared trie, and the cross-engine conversion in `convert_map.go` becomes pointer-sharing instead of copying.

**Kill criterion:** if `lookup` regresses more than 1.5× on the read-heavy microbenchmark, or the V1 fixture is not under 0.2s, stop and re-open option C.

### Phase 3 — Persistent boxed Array (4 days)

A 32-way persistent vector with a tail buffer (Clojure-style) backs non-float `ArrayValue`/`ArrayObj`. `get` takes about 2–4 hops and `set`/`append` are O(log₃₂ n). Packed `Array[float]` stays flat (D3). `fromList`, `make`, `updateMany` and `accumulate` build through internal transients. The kill criterion is the same as Phase 2, measured on `Array.get`.

### Files to Modify/Create

**New files:**
- `internal/pcoll/key.go` — the one canonical key encoder for both engines (Phase 1a, ~150 LOC)
- `internal/pcoll/champ.go` — CHAMP map (Phase 2, ~400 LOC)
- `internal/pcoll/pvec.go` — persistent vector (Phase 3, ~350 LOC)
- `std/set.ail` — Set API (Phase 1b, ~80 LOC)
- `examples/persistent_collections.ail` — visited set, difference array, structured keys (~80 LOC). *Shipped instead as four concept-named examples in `examples/runnable/`: `visited_set_bfs.ail`, `cycle_detection_visited_set.ail`, `map_tuple_record_keys.ail`, `difference_array_accumulate.ail`.*

**Modified files:**
- `internal/eval/value.go` — `MapKey` delegates to `pcoll`; `MapValue`/`ArrayValue` wrap `pcoll` types (Phases 1–3)
- `internal/bytecode/value_map.go` — `MapKey` delegates; `MapObj` wraps the CHAMP map
- `internal/bytecode/value.go` — `ArrayObj` wraps `pvec` for boxed arrays
- `internal/builtins/map.go` — insert/remove/lookup through `pcoll`; propagate key errors from lookup/member
- `internal/builtins/array.go` — set/append through `pvec`; new `_array_accumulate`
- `internal/vm/builtins_map.go`, `internal/vm/builtins_map_bulk.go`, `internal/vm/builtins_array_poly.go`, `internal/vm/builtins.go` — the VM halves of the same builtins
- `internal/vm/convert_map.go`, `internal/vm/convert.go` — share instead of copy across the engine bridge
- `internal/eval/show.go`, `internal/eval/show_bounded.go`, `internal/eval/eval_expressions.go`, `internal/builtins/xml_query.go` — `.Entries` readers move to an iterator (V15)
- `internal/bytecode/builtin_names.go`, `internal/pipeline/testdata/builtin_types.golden` — register `_array_accumulate`
- `std/map.ail`, `std/array.ail` — cost-model headers (Phase 2/3)
- the teaching prompt source (prompt-manager lane, Phase 1d)

## Conflict Surface

1. **Positions extended:** the runtime representation of `Map`/`Array` values (eval and VM), the map-key domain, and the `std/array` API (one new function). There is no parser, lexer, AST or type-system change. `Set` is a new stdlib type name, and no `type Set` exists in `std/` (V17).
2. **Other constructs living there:**
   - `Array[float]` packing feeds the numerics kernels (D3: untouched).
   - `show` of maps (the `Map{…}` format, sorted).
   - Structural `==` on maps.
   - The eval↔VM interop bridge, which copies maps today (`convert_map.go`).
   - `keys`/`values`/`toList` ordering.
   - `std/json` and `std/xml` producing maps (`xml_query.go` reads `MapValue`).
   - The `.Entries` field is read directly in 10 eval-side and 7 VM-side files (V15, V16). Every one moves to an accessor in Phase 2, enforced by deleting the exported field.
3. **Disambiguation:** none is needed syntactically. Semantically, the canonical key bytes for int/string/bool are **unchanged**, so ordering and `show` output are identical for every map that runs at HEAD (D2).
4. **Programs that must still work:**
   - `examples/` map and array programs (enumerate them at sprint start with `grep -l "std/map\|std/array" examples/*.ail`).
   - V10's `[10, 2, 9]` ordering.
   - The numerics `dot`/`axpy` examples.
   - `make verify-examples` green with both engines.
5. **Deliberate changes:**
   - Tuple, record and ADT keys go from a runtime error to working (V7). A program that queries such a key, such as V9's `v/silent.ail`, still prints `false` after Phase 1, now correctly, because the key is encodable and absent.
   - `lookup`/`member` with a **non-encodable** key (a function or effect handle) go from a silent `false`/`None` to a loud error. That is the one change in the output of a program that runs successfully at HEAD. Such a program queries a map that can never contain that key, because `insert` already rejects it at HEAD, so its `false` answer could not have come from data. The change is **intentional and listed**. It is a correctness fix under CLAUDE.md §2, not a replay-compatible change.
   - The O(n) cost headers change to O(log n).

   **Every other program that runs successfully at HEAD produces byte-identical output**: the int, string and bool key bytes and the iteration order are unchanged (D2).

## Axiom Compliance

| Axiom | Score | Justification |
|---|---|---|
| A1: Determinism | +1 | Iteration stays canonically sorted; CHAMP has a canonical structure per key set; the silent-false lookup is removed |
| A2: Replayability | 0 | Output is byte-identical for every program that runs today **except** a lookup/member on a non-encodable key, which changes from silent `false` to a loud error (Conflict Surface item 5, intentional). D2 preserves order. It is not scored +1 because of that one listed break |
| A3: Effect Legibility | 0 | All operations stay pure; transients are internal and never observable |
| A4: Explicit Authority | 0 | No capability change |
| A5: Bounded Verification | +1 | Closes the check-passes, runtime-fails key seam for every hashable type |
| A6: Safe Concurrency | +1 | Persistent structures are immutable after publication, so sharing across `Fork` stays safe with no copy |
| A7: Machines First | +2 | Models get the toolkit they already reach for in Python (Python 18/20 vs AILANG 1/20 structure use, V13); measured complexity-class wins |
| A8: Minimal Syntax | 0 | No syntax; +1 module (`std/set`) and +1 function (`accumulate`) |
| A9: Cost Visibility | +1 | Cost headers become true *and* small; no hidden aliasing-dependent costs (option D rejected for this reason) |
| A10: Composability | +1 | Structured keys compose with records, tuples and ADTs |
| A11: Structured Failure | +1 | Unhashable keys fail loudly on every operation |
| A12: System Boundary | 0 | Unchanged |

**Net Score: +9** → **Decision: Move forward.** No −1 on A1/A3/A4/A7.

## Examples

**Before (cp_60159-shaped, the closest AILANG equivalent of Python's visited set):**
```ailang
-- type-checks, then fails at runtime: unsupported map key type: *eval.TupleValue
let seen = insert(empty(), (r, c, dirToInt(d)), true)
-- and member() on such a key silently returns false
```

**After (Phase 1 makes it work; Phase 2 makes it O(log n)):**
```ailang
import std/set as S
letrec walk = \r c d seen moves.
  if S.member(seen, (r, c, d)) then -1
  else match step(r, c, d) {
    Exit => moves + 1,
    Move(r2, c2, d2) => walk(r2, c2, d2, S.insert(seen, (r, c, d)), moves + 1)
  };
```

**Difference array (cp_60164-shaped, Phase 1c):**
```ailang
let diff = A.accumulate(A.make(w * h, 0), cornerWrites(rects), \x y. x + y)
```

(These After snippets use APIs that do not exist yet; they are illustrative, not verified, and become the Phase 1 example file, which **must** pass `ailang check` at sprint end.)

## Success Criteria

- [x] V7/V8 tuple and record keys round-trip through `insert`/`lookup`/`member` in **both** engines; lookup with a function key errors loudly (V9 inverted) — V23, V24
- [x] V10 ordering is unchanged (`[10, 2, 9]`) in both engines — V23
- [x] `std/set` exists with the API in 1b; the examples (`examples/runnable/visited_set_bfs.ail`, `cycle_detection_visited_set.ail`, `map_tuple_record_keys.ail`, `difference_array_accumulate.ail`; `runnable/` is where `verify-examples` looks) pass `ailang check` and `make verify-examples` — V26, V29
- [~] `accumulate` passes a difference-array golden test (yes); the cp_60164 difference-array fixture passes its 15KB tests in 0.52s on `--strict-bytecode` but **9.3s on the evaluator** — V28. The 2s target is met on the VM only, and the evaluator time is the prefix-sum sweep, not `accumulate` (~0.1s)
- [ ] Phase 2: V1 fixture under 0.2s; lookup microbenchmark within 1.5× of HEAD
- [ ] Phase 3: V2 fixture under 1s; get microbenchmark within 1.5× of HEAD
- [ ] The cp_60159 visited-set fixture passes 47/47 hidden tests at 20s each. The repo pack holds 17 of them (the ≤64KB selection, V27); the fixture passes **17/17** already in Phase 1 (V27), so the remaining 30 large tests are the Phase 2 evidence
- [ ] Eval: the cp benchmarks from the companion eval-suite work show the AILANG pass rate moving, measured with `ailang eval-paired`
- [ ] All tests passing; `make freeze-stdlib` for the std/ interface change; changelog fragment; LIMITATIONS updated

## Testing Strategy

**Unit:** key-encoder golden tests (every type, including nested); CHAMP and pvec property tests against a reference `map`/slice model (random op sequences with equality after every step); structural equality of maps built in different insertion orders.

**Integration:** each builtin run under the evaluator and `--strict-bytecode` with identical output; the interop bridge passes maps both ways without copying (allocation test, per [feedback: memory tests assert allocation]).

**Performance:** V1/V2 fixtures plus read-heavy microbenchmarks banked through `perf-sweep`.

## Deferred Decisions

- Internal node width and the tail-buffer size of pvec: agent may choose, by measurement.
- Whether `fromList` dedups keys via the transient or via a sort: agent may choose.
- Whether `std/set` gets `map`/`filter`/`fold`: agent may add them if under 30 LOC.

## Non-Goals

- Lists, `::`, `++`, `nth`: owned by the D-19 cons-cells programme.
- Scoped mutation (option C) and uniqueness typing (option D): deferred and rejected above.
- Changing map iteration order: needs its own decision if ever wanted.
- Go codegen (`--emit-go`): neither array nor map builtins carry a `GoCodegenSpec` today (V18), so there is nothing to keep in sync.
- `Array[float]` persistence (D3).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Read constants regress on the lookup-heavy XLSX/DocParse workloads that motivated v0.11.0's copy-on-write choice | High | Kill criterion of 1.5× on lookup and get; the ailang-parse benchmark is run before merge |
| A third implementation diverges between the engines | High | D4: one shared package; both engines' `MapKey` deleted, not wrapped |
| The canonical encoding of new key types is accidentally order-changing for old types | High | Byte-identical golden test for the int, string and bool encodings |
| The prompt section teaches O(n) loops before Phase 2 ships | Med | 1d states the current cost explicitly and is A/B-measured |
| Memory growth from trie nodes | Med | Measured by allocation per op in perf-sweep |

## Verification Log

All rows were run first-party on 2026-10-28. Binary `AILANG v0.54.19-1-g379a9e20e-dirty` (built 2026-10-27, commit `379a9e2`); tree HEAD `51b00832f`. Fixtures are under the session scratchpad `pc/v/` and will be reproduced as repo fixtures in Phase 1.

| # | Claim | Command | Observed |
|---|---|---|---|
| V1 | `Map.insert` is O(n) per call | `v/mapins.ail`: n sequential inserts, `ailang run` | n=5k/10k/20k → 0.35/1.36/5.25s (eval), 0.42/1.51/6.16s (`--bytecode`), ×3.9 per doubling |
| V2 | `Array.set` is O(n) per call | `v/arrset.ail`: n sequential `set`s | n=40k/80k/160k → 1.73/6.02/24.17s (eval) |
| V3 | No `std/set` | `ailang check` on `import std/set (empty)`; `ls std/set.ail` | `stdlib module not found: std/set`; no such file |
| V4 | `scatterAdd` is float-only | `scatterAdd(zeros(3), …)` with `zeros: int -> Array[int]` | `cannot unify type constructors: float vs int` |
| V4b | (side finding) an untyped int literal array checks as float, then crashes | `scatterAdd(make(3, 0), [0,0], [1,2])` | `✓ No errors found!`, then runtime `must be float, got *eval.IntValue`. **Out of scope here**: same family as the float/int-literal soundness hole, filed separately |
| V5 | `updateMany` (batch, O(n+k)) exists and is overwrite, not combine | `updateMany(make(3,0), [(0,1),(2,5),(0,7)])` | `[7, 0, 5]` (later write wins) |
| V6 | Cost headers state O(n) insert/remove/set | `sed -n 1,30p std/map.ail std/array.ail`; `internal/builtins/map.go` LongDesc | "insert O(n) copy-on-write", "set O(n) copies entire array"; LongDesc "O(n) copy-on-write" |
| V7 | Tuple and record keys type-check but fail at runtime | `v/tupkey.ail`, `v/reckey.ail` | check ✓; run: `unsupported map key type: *eval.TupleValue` / `*eval.RecordValue (only int, string, bool supported)`; same with `--bytecode` |
| V8 | Two key encoders exist, one per engine | `grep -rn "unsupported map key type" internal` | `internal/eval/value.go:286`, `internal/bytecode/value_map.go:37` |
| V9 | `member` with an unhashable key silently returns false | `v/silent.ail`: `member(e(), (1, 2))`, `e: () -> Map[(int,int), bool]` | prints `false` under eval and `--strict-bytecode`; `value.go:291-295` returns `nil, false` on a key error |
| V10 | Int keys iterate lexically | `keys(fromList([(9,"a"),(10,"b"),(2,"c")]))` | `[10, 2, 9]` in both engines |
| V11 | cp_60159: the indexing fix alone does not help | sed-patch `[string]` → `Array[string]`, `nth` → `getOpt`; all 47 hidden tests at 20s | original 12 pass/35 TLE; patched 12 pass/35 TLE. Python (visited set): 0.05s on the 17KB test |
| V12 | cp_60164: the `++` fix alone does not help | patch `cells ++ [x]` → `x :: cells` and the two coordinate folds | 15KB test: TLE above 60s for eval, `--bytecode` and `--strict-bytecode`; Python brute force 9.48s/48.96s; Python difference array 0.11s |
| V13 | Structure census | `grep -l "import std/array"` / `std/map` over the saved solutions; a Python regex for `set(`/`dict`/`deque`/grids | first run: AILANG 1/20 array, 0/20 map, Python 18/20; confirm trials: 0/21, 2/21 (cp_60164 t3 histogram and cp_60197 t2), Python 11/21. The Python regex is a heuristic |
| V14 | The teaching prompt omits `std/set`, the key restriction and loop cost | `ailang prompt > prompt.txt`; grep | `std/set`=0; the Map section (line 1989) says "copy-on-write inserts" without any loop caveat or key-type rule |
| V15 | Eval-side `MapValue` readers | `grep -rln MapValue internal cmd \| grep -v _test` | 10 files (listed under Files to Modify) |
| V16 | VM-side `MapObj` readers | `grep -rln "AsMap()\|MapObj" internal cmd \| grep -v _test` | 7 files |
| V17 | `Set` is a free stdlib type name | `grep -rn "type Set\b\|type Set\[" std/` | empty |
| V18 | No Go-codegen spec for array/map builtins | `grep -n GoCodegen internal/builtins/array.go internal/builtins/map.go` | empty |
| V20 | Phase 3 blast radius: Array readers | `grep -rln "ArrayValue\b" internal cmd \| grep -v _test \| wc -l`; same for `ArrayObj\|AsArray()` | 12 eval-side files, 7 VM-side files. They are enumerated at Phase 3 start; the private `elems`/`floats` fields (`internal/eval/value.go:111`) already force accessors on the eval side |
| V21 | File overlap with the D-19 lists programme | for each of the 15 files listed under Files to Modify: `grep -cE 'ListValue\|TagList\|AsList\(' <file>` | 12 of 15 non-zero (`eval/value.go` 4, `bytecode/value.go` 7, `vm/convert.go` 4, `vm/builtins.go` 14, `builtins/xml_query.go` 20, `builtins/map.go` 4, `builtins/array.go` 2, `vm/builtins_map_bulk.go` 3, `show.go`/`show_bounded.go`/`eval_expressions.go` 1 each); 0 in `vm/convert_map.go`, `vm/builtins_map.go`, `vm/builtins_array_poly.go`, `bytecode/value_map.go` |
| V22 | Import directions allow a leaf `pcoll` only via inversion | `grep -h "sunholo-data/ailang/internal/" internal/bytecode/*.go \| grep -v _test`; same for `internal/eval/value.go` | bytecode: **no** internal imports; `eval/value.go`: `internal/builtins`, `internal/types` only |
| V19 | `mapAccumL` exists (the D-19 doc's list-building primitive), so it is not re-proposed here | `grep mapAccumL std/list.ail` | `std/list.ail:76` |

**Re-verified at implementation (2026-10-28, tree `57b0b8f20`, binary built in the worktree).** V1, V7, V8, V9 and V10 reproduce as written. **V22 is wrong as written:** the two halves are swapped. `internal/eval/value.go` imports no `internal/` package (only `fmt`, `sort`, `strings`), and `internal/bytecode` *does* import `internal/builtins` and `internal/types` (from `builtin_adapt.go`), so bytecode reaches eval transitively. The conclusion still holds: a leaf `pcoll` that imports neither engine is importable by both. **The "47 hidden tests" for cp_60159 (V11, metric 3) are the full AetherCode set; the repo pack `benchmarks/hidden_tests/cp_60159/` holds 17** (the ≤64KB selection, `AETHERCODE.md`).

| # | Claim | Command | Observed |
|---|---|---|---|
| V23 | Structured keys work, identically, in both engines; legacy order unchanged | `go test ./internal/vm -run 'StructuredMapKeys\|TupleKeyAbsent'` (program under evaluator and `--strict-bytecode`, twice) | tuple, record, ADT, unit, float and nested keys round-trip; `keys(fromList([(9,…),(10,…),(2,…)]))` is still `[10, 2, 9]`; both engines byte-identical across runs; V9's `member(e(), (1, 2))` prints `false` in both, now because the key is absent |
| V24 | Unencodable keys are loud on every operation | `go test ./internal/vm -run 'Unencodable\|NaNKey\|SetUnencodable'` | `insert`/`lookup`/`member`/`remove` with a function key fail in both engines with `unsupported map key type: function (supported: …)`; a NaN key fails naming NaN |
| V25 | The two walkers agree byte for byte | `go test ./internal/vm -run MapKeyCrossEngine`; `go test ./internal/pcoll` | golden keys for every kind incl. nesting identical from `eval.MapKey` and `bytecode.MapKey`; an evaluator-built tuple-key map crosses the bridge and the VM recomputes the same keys. pcoll: legacy bytes frozen, injectivity on separator-collision shapes, element-wise tuple order, numeric float order |
| V26 | `std/set` API in both engines | `go test ./internal/vm -run SetOperations` | all ten operations, identical output in both engines across two runs |
| V27 | cp_60159 visited-set fixture (`benchmarks/frontier_refs/cp_60159_set.ail`, `Set[(int, int, int)]` per query) on the 17 repo hidden tests at 20s | ad-hoc runner (the shape of m-eval-hidden-tests V9's `alltests.py`): each `.in` → `ailang run [--strict-bytecode] --caps IO`, compare `.ans` line-wise, 20s cap | **17/17 on both engines.** Evaluator total 7.2–7.8s, max 3.6–3.7s (test 33: N=100, Q=1000, 53,576 inserts); `--strict-bytecode` total 1.3–1.8s, max 0.38–0.55s. The longest single path is 4,852 states (test 20: 0.63s eval, 0.38s VM, Σ L² = 23.5M entry copies). At these sizes the O(n) insert is not the bottleneck: evaluator per-step overhead is (test 33's ~68µs/step against the VM's ~5µs) |
| V28 | cp_60164 difference array via `accumulate` (`benchmarks/frontier_refs/cp_60164.ail`) on the 8 repo hidden tests | same runner, 60s cap | **8/8 on both engines.** The four 15KB tests: 9.1–9.4s evaluator, 0.51–0.53s `--strict-bytecode` (baseline: >60s TLE on all engines, V12). Building the 1001×1001 difference array with `accumulate` alone takes 0.09s eval / 0.06s VM; the evaluator's time is the list-HOF prefix sweep (≈5M closure calls) |
| V29 | Example and gates | `make verify-examples`, `make test-core`, `make check-file-sizes`, `make check-boundaries`, `make verify-stdlib`, `make check-architecture-closure` | all green; `make freeze-stdlib` adds `set` and changes only `array` |
| V30 | Set insert cost today (D1 evidence) | `S.insert` of `(i/100, i%100)` n times | n=5k/10k/20k → 0.38/1.39/5.56s eval; 20k → 6.48s `--strict-bytecode` (×4 per doubling, same as V1's int-key Map) |
| V31 | Mutation tests | scratch copies of each file, sed the mutation, run the tests, copy back | caught: VM ctor key by tag instead of name; evaluator and VM `Lookup` swallowing the key error; `i:` prefix change; NUL not escaped; `-0.0` not folded; `difference` without `not`; `member` always false; `accumulate` combine args swapped; missing negative-index bound |

## Phase 1 implementation record (2026-10-28)

**1a (structured keys).** `internal/pcoll/key.go` owns the grammar; `eval.MapKey` (`internal/eval/value_mapkey.go`) and `bytecode.MapKey` (`internal/bytecode/value_map.go`) are now ~40-line walkers that drive a `pcoll.KeyWriter` and no longer encode anything themselves. The names are kept because `builtins/map.go`, `builtins/xml_query.go` and `vm/builtins_map_bulk.go` call them; they are the per-engine walkers the plan describes, not wrappers around a second encoder.

Grammar: legacy atoms byte-identical (`i:%d`, `s:` + raw, `b:true`/`b:false`). New: `u:` unit; `f:` + 16 hex of the order-preserving float transform (NaN rejected, `-0.0` folded into `0.0` because the two are `==`); `t:` tuple, `r:` record (fields name-sorted, name then value), `c:` constructor (name then fields), where each element is `escape(enc) + "\x00"` and `escape` maps NUL to `\x00\xff`. Injective, and composite keys sort element by element with each element ordered as it would be at top level. So int elements inside tuples stay lexical, consistent with D2.

**Deviations from the planned `KeyWriter` API, with reasons:**
- `BeginCtor(name)` instead of `BeginCtor(adt, tag, n)`. The VM's `ADTObj` carries no type or module name, and its `Tag` is a per-type ordinal the evaluator's `TaggedValue` does not have. The constructor name is the only identity both engines share. It is unique within one ADT, and a well-typed map holds one ADT type, so the key stays injective where it matters. Constructors therefore sort **by name**, not by declaration order.
- `BeginTuple()` and `BeginRecord()` + `Field(name)` take no arity or name list: the terminated-element grammar is self-delimiting. `Field` rejects out-of-order names, so a walker that forgets to sort fails loudly instead of minting a second key.
- `Float(float64)` instead of `Float(uint64 bits)`, so NaN rejection and `-0.0` folding live in `pcoll` rather than in both walkers.
- `-0.0` folding is a refinement of "bit-exact": without it, `member(insert(empty(), 0.0), -0.0)` would be `false` although `0.0 == -0.0`.

Still not keys (a loud error naming the type): lists, arrays, maps, bytes, functions. Lists are a natural next kind (`Set[[int]]`), but they were left out of scope here to stay clear of D-19's `ListValue` work.

**1b (`std/set`).** `type Set[a] = Set(Map[a, ()])` in `std/set.ail`, calling the `_map_*` builtins directly. `union`, `intersect` and `difference` are one `fromList` bulk build, not repeated O(n) inserts. `insert`/`remove` are O(n) and the module header says so. Type syntax has no qualified names, so programs use `import std/set as S` plus `import std/set (Set)`. That is a teaching point for 1d.

**1c (`accumulate`).** `_array_accumulate` is native in both engines: an evaluator builtin through `FnCallerN`, and a VM HOF builtin (`__array_accumulate`, appended to `HOFBuiltinNames`/`HOFBuiltinTable`). `scatterAdd` is unchanged.

**For D1 (human):** at the repo pack's sizes, Phase 1 already passes cp_60159 (17/17) and cp_60164 (8/8). The O(n) insert does not decide those results (V27). The evidence for Phase 2 is therefore the V1/V30 microbenchmark (×4 per doubling) and the 30 large cp_60159 tests that are not in the repo. If Phase 2 is to be justified by a benchmark result rather than a microbenchmark, a ≥64KB cp_60159 test (or a synthetic large-N fixture) has to be added first.

### Unverified / needs measurement

- CHAMP lookup constants in Go relative to the current single `map[string]` probe. These are **not measured**; they are the Phase 2 kill criterion.
- That a Floyd/Brent AILANG solution to cp_60159 passes today. This is a technique claim, not run.
- cp_60153 and cp_60170 mechanisms: 2 or fewer valid Python trials each.

## Quorum log

**Round 1 (2026-10-28T09:26:40Z): BLOCKED.** Cost $0.1967; artifact `.ailang/state/mission-quorum/m-persistent-collections-2026-10-28T09-26-40Z.json`.

| Reviewer | Verdict | Objection |
|---|---|---|
| oc-glm-5-3 | reject | The A2 claim "byte-identical output" is contradicted by the V9 change |
| oc-kimi-k3 | reject | "Shares no files" with D-19 is unverified, and is false at file granularity |
| gemini-4-argon (reserve) | reject | One leaf key encoder cannot inspect both engines' value types without an import cycle; no mechanism specified |
| gpt6-1-sol | absent | unreachable (quorum degraded to N−1) |
| controller | pass | — |

All three objections were **accepted and fixed in this revision**, not argued:

| Objection | Fix | Evidence |
|---|---|---|
| A2 claim | The single output change is now scoped and listed (Conflict Surface item 5). V9's tuple-key program in fact keeps printing `false` after Phase 1 | — |
| File overlap | Replaced with the measured overlap (12 of 15 files) and a sequencing rule | V21 |
| Import cycle | Specified dependency inversion: `pcoll` owns the grammar via `KeyWriter`, each engine has a walker, a cross-engine golden test catches divergence, and CHAMP/pvec are generics | V22 |

**Not re-run** in this session (the authoring brief allowed one round). The next quorum round is the planner's step, and it should pass `--max-cost-usd 0.30`.

## Related Documents

- [m-list-cons-cells-decomposition](../m-list-cons-cells-decomposition.md) — D-19 lists programme (distinct; see above)
- [m-list-cons-quadratic](../m-list-cons-quadratic.md) — parked predecessor, lists only
- [m-std-map-and-array-gaps](../../implemented/v0_11_0/m-std-map-and-array-gaps.md) — shipped O(n) copy-on-write, deferred persistent HAMT to v1.0
- [m-hash-collections](../../implemented/v0_9_2/m-hash-collections.md) — specified `std/set`, never shipped
- [m-foldl-cons-cost-model](../../implemented/v0_52_0/m-foldl-cons-cost-model.md) — `mapAccumL`, cost-honesty precedent

## References

- [Design Axioms](/docs/references/axioms)
- 2026-10-28 external-benchmark spike (AetherCode v1_2024, HF `m-a-p/AetherCode`, CC-BY-4.0); companion eval-suite work adds cp_60159/cp_60164 as tracked benchmarks
- Steindorfer & Vinju, "Optimizing Hash-Array Mapped Tries for Fast and Lean Immutable JVM Collections" (CHAMP, OOPSLA 2015)

## Future Work

- Option C (transients) if the Phase 2/3 constants are not enough.
- A persistent deque for BFS frontiers (today two lists work, in amortized O(1)).
- Sorted persistent map (ordered iteration by value order, not canonical string order).

---

**Document created**: 2026-10-28
**Last updated**: 2026-10-28 (Phase 1a–1c implemented)
