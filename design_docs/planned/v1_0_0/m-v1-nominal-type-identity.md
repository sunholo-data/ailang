# M-V1-NOMINAL-TYPE-IDENTITY: a type is its name *and* its module, and a name must resolve to exactly one thing

**Status**: Planned — M0 spike done (result in §9, from draft PR #2205, not merged); M1b canonical module identity done (§9)
**Target**: v0.58.x (the last breaking window before the 1.x promise)
**Priority**: P0 (release-readiness review §3.1 a and d; "the two findings I would not ship 1.0 with")
**Estimated**: 9–14 working days across 6 milestones (an estimate, not a plan; see §11). Revised by the M0 spike to 11.5–17 d with M1b added (§9)
**Dependencies**: none to start. Consumes [m-v1-downstream-compat-gate](m-v1-downstream-compat-gate.md) for
acceptance. Interacts with the exhaustiveness fix (#2146, review §3.1b), which needs the qualified ADT
identity introduced here to find the right constructor set.
**Planner-Lane**: opus-required (type representation, unifier, interfaces, cache key)
**Supersedes (type namespace only)**: M4 of [m-type-name-shadow-and-cache](../v0_44_0/m-type-name-shadow-and-cache.md);
[m-unbound-type-names](../v0_57_1/m-unbound-type-names.md) (#1485); the type/constructor half of
[m-import-vs-import-collision](../v0_54_20/m-import-vs-import-collision.md) and
[m-import-alias-collision](../v0_55_0/m-import-alias-collision.md) (#1980). Their verification logs stay
the evidence; this doc unifies them behind one type-name resolver so they are not built three times.
**Measured at**: `dev` = `51f6636` (binary built from it), 2026-11-10.

## 1. Problem

AILANG identifies a nominal type by its bare name. `types.TCon` is `struct { Name string }` and
`Equals` compares only `Name` (`internal/types/types.go:57-69`). Every module that declares `type T`
mints the same `TCon{"T"}`, so two unrelated ADTs unify. Four defects follow from the same root, or from
the absence of one authoritative "what does this name mean here" step:

| # | Defect | Repro (all run on `51f6636`) | Today |
|---|---|---|---|
| a | Two same-named ADTs from different modules are one type | `q/a: type T = A(int)`, `q/b: type T = A(string)`; `main` imports `mk` from a and `T, A` from b, matches `A(s) => toUpper(s)` | `check` ✓, `run` → `_str_upper: arg 0 - expected string, got int`; `--strict-bytecode` → `vm: BUILTIN_CALL: __str_upper: expected string` |
| a′ | Same, with disjoint constructors | b's `T = B(string)`, match `B(s)` on a's value | `check` ✓, `run` → `no pattern matched` |
| a″ | A **local** ADT captures an imported one | `main` declares `type T = A(string)` and calls a's `mk()` | `check` ✓, `run` → `_str_upper` error. `LIMITATIONS.md:145` says local capture "retains TC_TYPE_SHADOW_001"; that is true only for aliases |
| a‴ | Structural twins cross silently | a and b each `type Color = Red \| Green`; `b.name(a.pick())` | `check` ✓, prints `green` — works by accident; this is the class the fix deliberately rejects |
| d1 | Undeclared type names (#1485) | `func f(r: Nope) -> int`, `type W = { x: Nada }` | `check` ✓ |
| d2 | Duplicate top-level funcs (#1278) | `f(x) = x + 1` then `f(x) = x * 100` | `check` ✓, `f(2)` = 200 (last wins) |
| d3 | Same type name imported from two modules (#1980) | `import a (Leg)` + `import b (Leg)` | `check` ✓, last import wins |
| d4 | Duplicate type declarations in one module (new) | `type T = A(int)` twice | `check` ✓ |
| d5 | Two local ADTs declare the same constructor (new) | `type U = A \| B`, `type V = A \| C`; `f(A)` | `check` ✓, `A` silently means V's |

Probe sources: `scratchpad/dd-nominal/r/{q,v2,v3,v6,v8,v9,v10,v12,v13,v14,v15}` in this session's
scratchpad. A legal program that uses two same-named ADTs *disjointly* (`v15`: `Ma.desc(Ma.mk())`,
`Mb.desc(Mb.mk())`, different constructor orders) prints the right answers on both the evaluator and
the VM today, and must keep doing so.

**Where bare names are the identity today** (line numbers at `51f6636`):

- **Type representation:** `TCon{Name}` (`internal/types/types.go:57`). About 126 sites read a
  TCon's name across `internal/` (40 in `types`, 12 in `pipeline`, 7 each in `repl` and `gen/lower`).
- **Annotation → type:** the elaborator's `astTypeToInternalType` default arm returns
  `&types.TCon{Name: typ.Name}` for any unknown name (`internal/elaborate/file_funcs.go:353-359`); this
  is both d1 and the minting point for a.
- **Imports:** `resolveSelectiveImports` binds types, aliases and constructors by bare name, and
  auto-imports *every* alias (`pipeline_module_imports.go:278`) and *every* constructor (`:297`, first
  write wins) of each direct dependency; transitive aliases are pulled from the whole import closure
  (`:151`). `resolveConstructorImport` names the factory `$adt.make_<T>_<C>` (`:316`), so `q/a.T.A` and
  `q/b.T.A` share one `ExternalTypes` key and the later import overwrites the earlier scheme.
- **Checker tables:** `ctorTypes` (ctor → ADT name) and `adtTypeParams` (ADT name → arity), local
  overriding imported (`pipeline_module_compile.go:90-111`).
- **Interfaces:** `iface.ConstructorScheme{TypeName, CtorName, …}` and `TypeExport{Name, Arity}`
  (`internal/iface/iface.go:26-47`). The JSON renders a TCon as `typ.Name` (`internal/iface/json.go:255`);
  the registry hash projection (`iface/hash_projection.go:23`) and the stdlib goldens hash that string.
  The cache digest (`iface/digest.go:30`) also renders bare names.
- **Instances:** derived Eq is keyed by lowercased bare name (`types/dictionaries.go:422`), so two
  modules deriving Eq on `Row` collide and the duplicate is swallowed (audit row 9 of the M-TYPE-NAME-SHADOW doc).
- **Run time:** `eval.TaggedValue` already carries `ModulePath` (`internal/eval/value.go:429`) and the
  constructor resolver reports `ambiguous constructor … found in multiple modules`
  (`internal/runtime/resolver.go:203`), but pattern matching and equality compare `CtorName` only
  (`eval_patterns.go:187`, `eval_equality.go:68`). Bytecode ADT tables are keyed by bare name
  (`bytecode/compiler/collections.go:82`, `bridge_safety.go:121`).
- **Merges by bare name:** SMT `adtTypes`/`recordAliases`, first write wins (`internal/smt/verify.go:92-126`);
  REPL/WASM module registry (`internal/repl/module_registry_load.go:82, 241-256`).
- **Duplicates:** `checkDuplicateModuleBindings` (MOD007) checks let-vs-func and let-vs-let and assumes
  "funcs are declared, so unique by the parser's own rules" (`internal/elaborate/file.go:16-18`). They are not.

**Impact.** The pitch is that the checker makes AI-written code trustworthy. a/a′/a″ are a checker that
says ✓ and a program that dies with a type error. Agents reuse `Row`, `Item`, `Config`, `Error`, `Event`
constantly (the M-TYPE-NAME-SHADOW doc's words, and the registry agrees: `Event` is exported by three
packages, `Verdict` by `std/stream/bridge` and `motoko_ext_compose`).

## 2. Goals

**Primary goal:** every program `check` accepts can no longer fail at run time because two types or two
definitions shared a name; every type name in a signature resolves to exactly one declaration or is a
coded error at its own line.

**Success metrics:**
- All ten rows of the §1 table reject at `check` with a coded error (a‴ included), and `v15` still runs.
- Zero change in `check` verdicts across the in-repo corpus and the downstream gate, apart from the
  rows listed in §8 (measured blast radius).
- `interface_hash` of every published package unchanged (no republish forced by identity).
- One resolver: d1, d3 and a share one type-name scope table, not three.

## 3. High-Impact Decisions

| Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|
| D1 Identity = (defining module, name), stored as `TCon.Module` | Representation used by unifier, interfaces, cache, codegen | human | design | high |
| D2 Display stays the bare name; diagnostics qualify only when two distinct types print the same | Error text and goldens are user-facing | agent | compile | low |
| D3 Registry hash projection keeps rendering bare names; the internal cache digest renders qualified | Decides whether every published package must republish | human | design | high |
| D4 Builtin-minted nominal types get a home module from one table (`Json` → `std/json`, …) | A missed entry is a false rejection of a valid program | agent | compile | med |
| D5 Auto-import channels stay; resolution order is explicit and ambiguity at use is an error | Removing them rejects about 500 annotation sites downstream | human | design | high |
| D6 Duplicates (func, type, constructor) in one module are MOD007; two explicit imports of one bare type/ctor name are MOD015 | Error-code registry is public | human | design | low |
| D7 Errors, not warnings, in dependency code too | Decides whether the `ailang_parse` package must republish | human | design | med |

### Design freeze

- [ ] D1 qualified identity (vs Alternative B, §5)
- [ ] D3 hash projection unchanged
- [ ] D5 auto-import channels kept, order fixed
- [ ] D6 MOD015 (not MOD016) for import-vs-import
- [ ] D7 hard error everywhere, republish `ailang_parse`

### Deferred decisions

- Spelling of the qualifier in diagnostics (`T (from q/a)` vs `q/a.T`) — agent, consistent with `TC_TYPE_SHADOW_001`.
- Whether `TaggedValue` matching also compares `ModulePath` (defence in depth) — agent; not required for soundness.
- Go-codegen type-name mangling for same-named ADTs — agent; `--emit-go` is not Stable.
- Levenshtein suggestion thresholds for `TC_UNBOUND_TYPE_001` — agent, per m-unbound-type-names D5.

## 4. Design

### 4.1 Representation

`types.TCon` gains `Module string`. `Equals` compares `(Module, Name)`. `String()` returns `Name`.
`Module == ""` is legal only for primitives (`int`, `string`, …) and builtin structural heads (`Array`,
`Map`, `List`, which already normalise to `TArray`/`TMap`/list). After elaboration a nominal TCon with an
empty module is an internal error: a test-only hook in `internal/types` panics on it, and release
builds report it as an internal compiler error. A forgotten minting site therefore fails loudly instead
of re-opening the hole. (No new environment variable: new ones must go through the `internal/config`
registry, and a test hook is enough.)

Keyed struct literals keep compiling (`TCon{Name: "x"}`), which is what makes the change incremental:
M0 adds the field, then each minting site is fixed under a counter that reports empty-module nominal
TCons reaching the unifier.

### 4.2 Minting sites

1. **Elaborator** — `astTypeToInternalType` resolves a bare name through a `TypeScope` (§4.3) and stamps
   the defining module. Type-application heads (`T[a]`) go through the same lookup.
2. **Interface builder** — ADT `ResultType`, constructor `FieldTypes` and export schemes carry the
   defining module (an extension of M-ALIAS-BODY-CLOSURE, which already closes alias bodies in their
   defining module). Interfaces are the only way a type crosses a module, so a qualified interface
   is enough for an importer that never names the type (row a).
3. **Builtins** — `types.Con(name)` (builder) looks the name up in one `nominalHome` table:
   `Option` → `std/option`, `Result` → `std/result`, `Json` → `std/json`, `XmlNode` → `std/xml`, and so on;
   names with no `.ail` declaration (`DOMEvent`, `StreamConn`, `ProcessHandle`, …) get `$builtin`. A unit
   test walks every registered builtin signature and fails on a nominal TCon without a home, and checks
   that each home module really declares the type.

### 4.3 Type-name resolution (one table for a, d1, d3)

At each annotation, in order:

1. a type parameter in scope;
2. a type declared in this module;
3. a type bound by an explicit selective import (`import M (T)`, `import M (T as U)` binds `U`);
4. the prelude (`Option`, `Result` from `internal/loader/prelude_imports.go`);
5. a type exported by a **direct** dependency through the auto-import channels (M-CTOR-AUTO, M-TYPE-ALIAS);
6. an alias reached transitively (M-TRANSITIVE-ALIAS-ENV-IMPORT).

The first rule that matches wins, which keeps today's local-wins and explicit-beats-auto behaviour
(`examples/alias_body_closure/main.ail` depends on it). If rule 5 or 6 matches more than one *distinct*
`(Module, Name)`, the use is `TC_AMBIGUOUS_TYPE_001`; nothing is ambiguous until it is used, so a module
that merely imports two packages exporting `Event` is unaffected. No rule matching is
`TC_UNBOUND_TYPE_001` (#1485), with the suggestion set from m-unbound-type-names D5.

Constructors resolve the same way in their own namespace (types and constructors stay separate:
`mk-main/src/eval/journal/scan.ail` imports record type `Interaction` from one module and constructor
`Interaction` from another, and that stays legal). `ConstructorPattern.TypeName` and the `$adt` factory
key become qualified (`make_<module>_<T>_<C>`, escaped), which removes the `ExternalTypes` overwrite.

### 4.4 Duplicates

MOD007 grows three cases in `checkDuplicateModuleBindings`: func declared twice, type declared twice,
and one constructor name declared by two local types. Two explicit imports binding the same bare type or
constructor name are MOD015 at the import site, as [m-import-vs-import-collision](../v0_54_20/m-import-vs-import-collision.md)
proposes (its registry text already says "or another import"). Values keep that doc's use-site rule.

### 4.5 Everything else that keys by name

- **Unifier diagnostics:** a mismatch whose two sides print identically becomes `TC_NOMINAL_001` and
  names both modules.
- **Derived Eq / dictionaries:** key `prelude::Eq::<module>::<name>`; duplicates become impossible rather
  than swallowed.
- **SMT:** `adtTypes`/`recordAliases` keyed by qualified name; first-write-wins merge removed.
- **REPL/WASM:** the registry loader builds its scope with the same `TypeScope`. `repl` is a Stable
  command, so the deferral noted in `LIMITATIONS.md:148` ends here.
- **Bytecode:** ADT tables keyed by qualified name. This also narrows the LIMITATIONS row "Cross-ADT
  constructor tag collisions in unresolved bytecode switches", since the scrutinee's static type now
  names one ADT.
- **Re-exports:** AILANG has none (`MOD003`). Identity is the *defining* module, so a future re-export
  (Rust `pub use`, OCaml `type t = M.t`) preserves identity with no further change.
- **Aliases:** stay structural and transparent. Two record aliases `R = {x: int}` from different
  modules remain the same type (closed issue #904 was the record variant and is unaffected). An alias
  to an ADT resolves to the ADT's qualified TCon.

### 4.6 Interfaces, cache and packages

| Artifact | Change | Consequence |
|---|---|---|
| iface JSON `type` strings, stdlib goldens, registry `interface_hash` (`sha256:ifacev2:`) | **none** (TCon renders `Name`) | no republish for identity; `make freeze-stdlib` unchanged |
| iface JSON constructors/types | new `module` field on nominal type refs, excluded from `HashProjection` | consumers can see identity without the hash changing |
| internal cache digest (`iface/digest.go`) | renders qualified | an upstream switching `a.T`→`b.T` invalidates importers |
| compile cache key | `cacheKeyVersion` `v10` → `v11` (`pipeline/cache_key.go:50`) | every entry misses once |

Not covered by D3, and recorded as a follow-up rather than done: the hash cannot tell a package that
changes which `T` an export returns. Fixing that is an `ifacev3` identity change and every publisher on
an older binary would hit `PUB005` (the reason `hash_projection.go:11-15` pins the projection schema).

## 5. Alternatives considered

**B. Closure-wide name uniqueness (reject, don't distinguish).** Keep `TCon{Name}`; reject any module
whose import closure contains two exported nominal types with the same name. Cheaper (no representation
change, about 3 days) and, on today's corpus, it rejects the same zero programs (§8). It loses because:
(1) it rejects legal programs like `v15` and any consumer of two packages that both export `Event`
(three registry packages do today) even when they never mix; (2) a package's *private* dependencies
would decide whether its users compile, which is hostile to the registry; (3) it does nothing for d1,
the cache digest or SMT/REPL merges, so most of §4 is still needed. It is, however, strictly more
restrictive than A, so relaxing B to A later would be additive. **Fallback:** if the M0 spike shows A
cannot land in the window, ship B for v0.58.x and A in 1.x, and say so in the release notes.

**C. Runtime tag check.** Compare `ModulePath` in pattern matching so a/a′ fail loudly at run time. It
turns a wrong answer into a crash but leaves `check` lying; rejected as the fix, kept as an optional
defence (deferred decision).

**D. Qualified names in `Name` (`"q/a.T"`).** Breaks every `Name == "Json"` test in 126 sites and changes
every rendered signature, hence every `interface_hash`. Rejected.

## 6. Diagnostics

Codes follow the existing scheme: namespaced `TC_*_NNN` for checker errors
(`TC_TYPE_SHADOW_001`, `TC_ARITY_001`), `MODnnn` for module namespace errors. Each gets a page under
`docs/docs/reference/errors/`.

```
TC_NOMINAL_001: expected T (defined in q/b), but this value has type T (defined in q/a).
  They are different types that share a name.
  Fix: use the T from one module on both sides — import q/a (T) — or convert between them explicitly.

TC_UNBOUND_TYPE_001: type 'Nope' is not in scope (at v9/main.ail:2:23).
  Fix: declare it, or import it by name: import <module> (Nope).
  Types in scope here: Option, Result, W.

TC_AMBIGUOUS_TYPE_001: type 'Leg' is ambiguous here: q/a and q/b both export a type Leg and
  neither is imported by name. Fix: import q/a (Leg), or import q/b (Leg as BLeg).

MOD007: func 'f' is declared twice (at 2:1 and 3:1) — a module-scope name may have only one binding.
  Fix: remove or rename the duplicate.
MOD007: type 'T' is declared twice (at 2:1 and 3:1).
MOD007: constructor 'A' is declared by both type U (at 2:10) and type V (at 3:10).
MOD015: 'Leg' is imported twice (from q/a at 2:1 and from q/b at 3:1). Fix: import q/b (Leg as BLeg).
```

The `MOD007` registry description changes from "(let/func same name)" to "Duplicate module-scope
declaration"; v3's current message ("V's constructors are: " with an empty list) disappears because
the constructor table is qualified.

## 7. Conflict surface

**Positions touched:** `TCon.Equals` (every unification); `astTypeToInternalType` default and
`TypeApp` arms; `resolveSelectiveImports`/`resolveConstructorImport`; `ctorTypes`/`adtTypeParams`;
`iface.Builder.Build`; `computeDigest`; `RegisterDerivedEq`; SMT/REPL merges; bytecode ADT tables.

**What else lives there and must still work** (each becomes a fixture in M2/M3):

| Program | Why it is at risk |
|---|---|
| `examples/alias_body_closure/main.ail` | explicit `(Item)` must beat the auto-import channel (rule 3 before 5) |
| `examples/runnable/visited_set_bfs.ail` | `import std/set as S` + `import std/set (Set)`; `Set` wraps the builtin `Map` head |
| `examples/runnable/nested_match_simple.ail` | local `type Option` shadows the prelude's (rule 2 before 4) |
| `mk-main/src/eval/journal/scan.ail` | type `Interaction` and constructor `Interaction` from two modules |
| registry `crew_core` 0.6.0 `session.ail` | same, in a published package (`Applied`) |
| `stapledons-godot/sim/protocol_worlds.ail` | `Planet` plus `Planet as ObservedPlanet` |
| `docparse/tests/verify_attribution.ail` | writes `-> Json` without importing the name (rule 5) |
| `v15` repro | two same-named ADTs used disjointly, evaluator and VM |
| `std/net.ail` `urlEncodeForm(params: List[…])` | `List[...]` head must stay builtin |

**Disambiguation:** the six-rule order in §4.3 is today's de-facto precedence made explicit; the only
new outcomes are errors where today one candidate wins silently.

**What deliberately changes:** rows a–d5 of §1 become errors; a‴ (structural twins crossing a module
boundary) is rejected even though it runs today.

## 8. Blast radius, measured

Method: a regex sweep (`sweep.py`, `unbound2.py` in the session scratchpad; not the real parser),
resolving `module`/`import` lines, relative imports and `pkg/` imports against `ailang-packages`, the
`ailang-parse` checkout and the latest version of each of the 61 packages in this machine's local
registry cache (`~/.ailang/cache/registry/sunholo`, not the registry itself). Corpus: this repo
(`std`, `examples`, `benchmarks`, `tests`, `internal` fixtures; `.claude/` worktrees excluded), `daneel`,
`mk-main`, `ailang-packages`, `ailang-parse`, `docparse`, `email-parse`, `ailang-world`,
`stapledons-godot` (caches and builds excluded), `stapledons_voyage`: **2,073 files**, plus **561 files**
in the 61 cached packages. Import resolution after the package mapping: at least 99.9% in every
downstream root, 1,327 / 1,350 in this repo (the misses are experimental examples). Positive controls: the sweep
flags every §1 repro. Local checkouts can be behind their remotes.

| Class | Measured | Real after reading each hit |
|---|---|---|
| Two exported same-named ADTs in one import closure (a, a′, a‴) | 0 | **0** |
| Local ADT named like a reachable imported ADT (a″) | 2 (`nested_match_*` local `Option`) | **0 expected to break**: neither mixes values with `std/option`; confirm in M2 |
| Duplicate top-level funcs (d2) | 5 pairs in 4 files | **5, all real, all identical bodies** (one differs only in a binder name): `daneel ext/writer/register.ail` (`slen`, `cut`); `mk-main scripts/verify_mot136_dagr_producer.ail` (`sfield`), `packages/motoko-ext-herdr/types.ail` (`length_of`); `ailang-parse docparse/services/zip_extract.ail` (`readDocxStyles`) — **and every cached `ailang_parse` version from 0.39.2 to 0.55.5 (22)**, which `docparse` (pins 0.55.5) and `email-parse` (pins 0.50.0) depend on |
| Same type name explicitly imported twice (d3) | 8 | **0**: 4 are `as` renames, 4 are type-vs-constructor namespace pairs (legal; one is in the published `crew_core` 0.6.0) |
| Undeclared type names (d1), known set = §4.3 rules | 25 | **≈1**: `DOMEvent` in `std/dom.ail` (builtin, needs a home entry); the rest are `[T]` parameters, `List[..]`, fixtures in `tests/` that already fail, or a regex miss |
| Annotation sites relying on the auto-import channel (would break if D5 went the other way) | ≈500 in ≈60 files | `Json`, `XmlNode` written without importing the name (docparse, ailang-parse, packages, email-parse) |

**Summary:** the soundness change itself breaks no program found; the duplicate-func rule breaks one
published package that two flagship downstreams depend on, fixed by deleting one duplicated function and republishing.
**Not measured:** non-exported ADTs leaking through exported signatures; programs that match a builtin's
`Option` result against a local `Option`; anything outside these checkouts. The real number comes from
running the M2 binary through the downstream gate, which is an acceptance criterion below.

## 9. Milestones

Ordered so the riskiest unknown — how many places mint an unqualified nominal TCon — is measured first.

**M0 Spike: count the minting sites (1.5 d).** Add `TCon.Module`, stamp it in the elaborator and iface
builder only, treat `""` as a wildcard, and count wildcard matches by `(name, call site)` through
the test hook across `make test`, `make verify-examples` and the downstream gate.
*Accept:* a ranked list of sites; rows a, a′, a″, a‴ already rejected; `v15` passes; go/no-go on
Alternative B recorded in this doc.

> **Spike result (2026-11-10, branch `v1/nominal-identity-m0-spike`, base `dev` = `470b6ea59`).**
> **Verdict: GO on A, with two conditions the doc did not have.** (i) A **module is not one identity
> today**: a package's own modules load a second time under their `module_prefix` path, so the same
> `type Block` arrives as `docparse/types/document.Block` *and* `pkg/sunholo/ailang_parse/types/document.Block`
> and the spike rejects 14 of docparse's 44 files. A new milestone (M1b, canonical module identity) must
> land before M2; it blocks B equally. (ii) M2 cannot remove the wildcard until M3's resolver exists, so
> M2 and the resolver half of M3 merge, and three hard cases the counter cannot see join M2. Revised M2:
> **5.5–7 d** (was 3–4) plus M1b **1–1.5 d**; M3 shrinks to ~1 d (diagnostics only); whole doc
> **11.5–17 d** (was 9–14).
>
> *What the spike is.* `TCon.Module` (wildcard `""`); one comparison, `types.NominalSame`, used by
> `TCon.Equals`, `SafeEquals`, `equalHead` and the unifier's TCon case; stamping only where this doc
> says — the elaborator (local declarations and explicit `import M (T)`, via a name→module table the
> pipeline builds) and the iface builder (ctor result/field types, alias bodies, export schemes, for
> every type the module declares). Every non-primitive `&TCon{Name: …}` literal (47 sites) goes through
> `types.SpikeCon`, which records its call site; `AILANG_NOMINAL_SPIKE_LOG=<file>` logs each wildcard
> use as (kind, name, minting site, comparison site). Two serialisers had to learn the field for warm and
> cold runs to agree: the type JSON in the iface cache, and the CoreTI DAG codec, whose pinned field-count
> test (`TestCoreTIDAG_FieldCountsPinned`) failed as soon as the field appeared, which is what it is for.
> Instrument: `internal/types/nominal_spike.go`, `internal/pipeline/nominal_spike.go`.
>
> *Verdicts.* In-repo nothing changed: `make test-core` green (17 core packages, 45 s); `make
> verify-examples` 339 passed / 0 failed / 12 skipped; `make verify-stdlib` all 49 interface hashes
> unchanged (D3 holds: Module never reaches a hashed rendering); full `make test` (5 min): 163 packages
> ok, 4 failed — 5 s fake-binary timeouts in `executor/{codex,opencode,pi}` and `mission/dispatch` under
> concurrent load, all green on rerun and on `dev`; it found no new minting site and no nominal mismatch. Downstream
> (check-only, 1,492 files, baseline `dev` vs spike): **14 flips, all in docparse, one cause** — see
> "Downstream" below.
>
> *a-rows, evaluator and `--strict-bytecode`.* With nothing but the two stamping sites, `check` rejects
> a (`q`), a′ (`v2`), a″ (`v6`) and a‴ (`v14`) with `cannot unify type constructors: T (from q/b) vs T
> (from q/a)`; `run` and `--strict-bytecode` reject identically (the check runs first). `v15` passes on
> both back ends. Warm cache agrees with cold.
>
> *Ranked unqualified minting sites.* Events = wildcard uses where one side was qualified and the other
> was minted at this site (test-core + verify-examples, power-of-two sampled, so within 2x low): each is a
> false rejection M2 would cause if the site stayed unqualified. A further 39k uses had *both* sides
> empty; they come from the same six families.
>
> | Rank | Site | What it mints | Events | Class |
> |---|---|---|---|---|
> | 1 | `types/typechecker_patterns.go:190,195` `checkPattern` | a constructor pattern's ADT from `ctorTypes[ctor]` (bare) | 64.1k | **trivial** once `ctorTypes` values carry the module (local → modID, imported → iface ctor's module) |
> | 2 | `types/constructor_factory.go:106,108` ← `pipeline.buildConstructorFactoryTypes` | the result type of this module's own constructor factories | 49.0k | **trivial**: pass `modID`. `runtime/runtime.go:557` builds the same scheme at run time — same fix |
> | 3 | `types/builder.go:70,77,81` `Builder.Con/App` ← 162 call sites in 52 builtin files | builtin signatures: 28 names (`Result` 90, `XmlNode` 27, `Option` 21, `StreamConn` 14, `NetError` 7, … `DOMEvent`, `BridgeFrame`) | 29.8k | **module-less** → one `nominalHome` table inside `Con`/`App` fixes all 162 (D4) |
> | 4 | `types/json.go:318` `UnmarshalType` (and gob, and the DAG codec) | pass-through of cached types | 6.8k | **trivial**, done in the spike; the origin is upstream. A missed serialiser strips identity on *warm* runs only, so M2 needs a warm-vs-cold test |
> | 5 | `elaborate/file_funcs.go:359,441` (default arm, `TypeApp` head) | annotation names that are neither local nor explicitly imported: prelude `Option`/`Result`, auto-imported `Json`/`XmlNode`, builtin-only `DOMEvent`, the structural head `Map` | 0.67k | **needs design**: this *is* §4.3 rules 4–6. Without it, removing the wildcard rejects every `-> Option[int]` written against `std/option` |
> | 6 | `elaborate/patterns.go:254-260`, test literals | capitalised primitives `Int`/`Bool`/`Float`/`String` | 0 | **module-less**: treat as primitives, not nominal |
> | 7 | `iface/type_conversion.go:32,103` (downstream only) | ctor field types and alias bodies in an *interface* that name an *imported* type (`LedgerEvent`, `WaitDescriptor`, `Json`, `Option`) | 11.9k downstream | **needs design**: the iface builder re-converts AST with its own converter instead of reusing the elaborator's resolved types. There are four AST→type converters (`elaborate/file_funcs.go`, `iface/type_conversion.go`, `types/typechecker.go:190` `astTypeToType`, `smt/from_ast.go`); M2 should make it one |
>
> Downstream the order is the same with more weight on the resolver: 2.16M wildcard uses, led by pattern
> types (1.41M), factories (0.30M), cache decode (0.22M), and the elaborator's auto-imported `Json` and
> `XmlNode` (0.16M, rank 5) — the names docparse, ailang-parse and email-parse write without importing.
>
> Comparison sites: all wildcard uses arrive through three comparators (`SafeEquals`, `equalHead`,
> `TCon.Equals`), called from `SolveConstraints`, `solveEqualitiesInto` and `typesIdentical`. Bare-name
> logic that bypasses `NominalSame` and that the counter cannot see (static census): 13 literal checks
> (`Name == "Array"|"Map"|"List"|"Option"|"Result"|…`, fine once builtins have homes);
> `unification_types.go:241` (diagnostic only); and the name-keyed tables below. The other 37 rewritten
> literal sites never minted a compared TCon in these runs (instance/dictionary keys, `derived_eq.go`,
> `typechecker.go:213/285`, `link/linker.go`, `eval/eval_patterns.go`, `smt/from_ast.go`, `repl/*`).
>
> *Hard cases* (repros `h1`–`h7` in the PR; "base" = `dev`, "spike" = this branch):
>
> - **Aliases vs ADTs — a hole in A as specified.** The unifier expands aliases by *bare name* before
>   any nominal comparison, and the iface builder registers every single-constructor ADT wrapping a
>   record as an *alias* (M-STREAM-DX/M4, for field access). So `a: type W = W({x:int})` passed where
>   `b: type W = W({x:int}) | Z` is expected checks on base *and on the spike* (`h4c`) — identity is lost
>   before `NominalSame` runs. M2 must key `aliasEnv`/`ImportedTypeAliases` by `(module, name)` and stop
>   treating newtype-record ADTs as aliases for identity. Plain record aliases stay structural and work
>   (`h4`); an alias to an imported ADT works (`h5`).
> - **`$adt` factory keys — a legal program crashes today.** `import h2/b (getb)` then
>   `import h2/a (A, get)`; `get(A(7))`: `check` ✓ on both, **evaluator** `ambiguous constructor T.A found
>   in multiple modules`, **VM** prints `7`. Not a soundness hole, a run-time failure of a correct program,
>   and the back ends disagree. Fixed by the qualified factory key M2 already plans.
> - **Bytecode ADT tables — a legal program fails under `--strict-bytecode` today.** `h7` (`a.T = A(int)`,
>   `b.T = B(string)`, used disjointly via `Ma.`/`Mb.`): evaluator prints `a 1 / b x`; VM `BC001 …
>   compiler: unknown ADT tag T.A` because `bytecode/compiler/compiler.go:65` keeps the last `T` declared
>   in the program. `v15` passes only because its two `T`s share constructor names. So M4's bytecode part
>   is a fix for valid programs, not hardening.
> - **Derived Eq keys** (`h1`, two modules deriving Eq on `Row`): correct on both back ends, base and
>   spike. The bare-name key collision exists but is not observable here; keep the M2 item, lower risk.
> - **Pattern `TypeName`**: rank 1 above — the pattern's ADT is rebuilt from a bare name.
> - **iface v2 hashing**: unchanged by construction; verified by `verify-stdlib`.
> - **SMT**: the doc's "first write wins" is stale — `smt/verify.go:94-121` now records ambiguous ADT
>   names and skips the function as unresolved (fails closed). M4's SMT part shrinks to keying, no
>   soundness work.
> - **REPL/WASM**: `repl/module_registry_load.go` is a second module loader that never calls
>   `compileFreshModule`, so it got no stamping and stays all-wildcard. It must share the resolver.
> - **Re-exports**: none exist (`MOD003`); nothing to measure.
> - **Cache**: the key already includes the compiler identity (commit, plus a per-build fingerprint when
>   dirty), so a pre-change verdict cannot be served across binaries in practice; the v11 bump stays as
>   belt-and-braces and was *not* done here.
>
> *Why A over B.* The counter says unqualified minting is concentrated: seven families, three trivial,
> one table, two that are the resolver this doc already designs. Nothing in the corpus needed the
> wildcard for a reason that A cannot model once module identity is canonical. B (ban duplicate names across a closure) is cheaper but
> rejects `h7`/`h2`/`v15`-shaped programs and two packages exporting `Event`, and still needs the alias,
> factory-key and bytecode fixes to stop the legal-program failures above *or* to reject them. B stays
> the fallback only if the merged M2 slips past the 0.58.x window.
>
> *New M1b, canonical module identity (1–1.5 d, before M2):* one module ID per source file — a
> dependency package's modules keep their `pkg/<owner>/<name>/…` identity whether reached from inside the
> package (via `module_prefix`, m-dx-module-prefix-resolution) or from outside; the file compiles once.
> *Test:* docparse's `tests/verify_ai_cache.ail` loads `types/document` once; the spike's 14 flips go to 0.
>
> *Revised M2 (5.5–7 d):* representation + `NominalSame` + serialisers with a warm-vs-cold test (0.5);
> qualified `ctorTypes`/factories/pattern types (0.5); `nominalHome` + completeness test (0.5–1);
> elaborator resolver for §4.3 rules 4–6, shared with the REPL loader (1–1.5); alias env keyed by
> `(module, name)` and newtype-record ADTs out of the alias channel (0.5–1, **new**); qualified `$adt`
> keys + runtime resolver (0.5); derived-Eq/instance keys (0.5); digest + cache-key bump (0.5); wildcard
> removal, internal-error hook, downstream gate and fallout (1).
>
> *Ordering findings.* (0) M1b (canonical module identity) before M2, and before B if B is chosen.
> (1) M3's resolver must land with or before M2's wildcard removal. (2) The
> exhaustiveness doc's universe registry does not need all of M2: the iface constructor result type is
> already stamped with its module in this spike (`iface/builder.go`, four lines), so
> `adtUniverse[(module, name)]` can be built from `ConstructorScheme` as soon as that stamp lands; its
> wiring milestone can start on that alone — keyed by the canonical module (M1b), or a package's ADT has
> two universes. (3) `h7` and `h2` belong in
> [m-bytecode-vm-parity-bugs](m-bytecode-vm-parity-bugs.md) as well: they fail on valid programs today.
>
> *Downstream.* Check-only sweep, baseline (`dev` @ `470b6ea59`) vs spike, over the scratch snapshots
> of docparse, email-parse, daneel, ailang-packages, ailang-parse, ailang-world and mk-main (1,492 files;
> 1,280 pass on baseline; 4 min per arm; snapshots taken 2026-11-10 08:20, isolated HOME/cache, no writes
> to the snapshots). **14 verdict flips, all docparse, all `Block (from pkg/sunholo/ailang_parse/types/document)
> vs Block (from docparse/types/document)`** at three call sites (`convert.ail:463` `generateHtml`,
> `mcp_paging.ail:134` `renderMarkdown`, `tests/verify_ai_cache.ail:21`). Not an a-row: `--debug-compile`
> on the baseline shows the one file compiled twice, as `docparse/types/document` (ailang_parse's own
> imports, `module_prefix = "docparse"`) and as `pkg/sunholo/ailang_parse/types/document` (docparse's
> imports). It is a false rejection that A — and B — would ship unless M1b lands first. The doc's §8
> regex sweep resolved imports by path, so it could not see that two paths name one file. No other
> corpus flipped; email-parse also depends on ailang_parse but never meets both paths in one value.

**M1b Canonical module identity (1–1.5 d, before M2) — DONE 2026-11-10** (branch
`v1/canonical-module-identity`, PR #2224). One module ID per package module file, whichever
spelling imported it; the file compiles and evaluates once.

> **Measured before the change** (`dev` = `b517ebe09`; fixture `internal/testutil/prefixpkg`: package
> `acme/shapes` with `module_prefix = "shp"`, whose `ops/area` imports `shp/types/shape`, consumed by
> `acme/app`, which imports `pkg/acme/shapes/types/shape`). `ailang run --debug-compile` compiled
> `types/shape.ail` twice, as `pkg/acme/shapes/types/shape` and as `shp/types/shape`; inside the package
> itself an entry using the prefix, the bare self-import and the published path compiled it **three**
> times. It is not only a type-checking artefact: a module-level `let` that logs ran **twice at run
> time** (evaluator and `--bytecode`), because the runtime keeps one instance per import spelling.
>
> *Where the identities came from.* (1) `loader.Load` keyed its cache and `LoadedModule.Path` by
> `CanonicalModuleID(<spelling>)` (`internal/loader/loader.go:132-139`, `:322-333` on `b517ebe09`),
> so each spelling was a cache miss; (2) the prefix branch (`:218-247`) and the bare self-import branch
> (`:202-217`) resolved a spelling to the package's file but kept the spelling as the ID; (3)
> `extractImports` (`:534-554`) normalised only `./x` imports to `pkg/…`, so `imp.Path` — read by
> `pipeline.resolveModuleImports` (`pipeline_module_imports.go:124`), `link/topo.go:107`,
> `cache_dep_closure.go:35`, the elaborator's `GetExport` (`elaborate/file.go:219`) and the runtime
> (`runtime/runtime.go:196,232`, instances keyed by import path) — carried the spelling everywhere; (4)
> the compile cache indexes entries by `(moduleID, key)` and digests dependencies by module ID
> (`pipeline_module_cache.go:28-51,120`), so the two IDs had two cache entries and two interface
> digests. `detectModulePathCollisions` (`pipeline_module.go:26-135`) already knew: it de-duplicated
> "the same file loaded under two canonical IDs … routinely with `module_prefix`-aliased packages".
>
> *Design (as revised after two reviews; the exact rule is "The import resolution rule, as implemented" below).* `ModuleLoader.canonicalImportPath`
> (`internal/loader/canonical_identity.go`) resolves every spelling the package resolver answers —
> `<prefix>/<m>`, bare `<vendor>/<name>/<m>` inside the root package, `pkg/<vendor>/<name>/<m>`, `./x`,
> and in a package without `[exports]` any other path that names a file — to the **package and the file**,
> then derives the module path from the FILE (not the spelling): prefix, `src/` or checkout
> `<vendor>/<name>/` stripped, kept only if the resolver maps it back to the same file. A prefix spelling is
> read in the **importing package's scope** (its own prefix, then its declared dependencies'; nothing
> outside that scope). It names the result with **one ID per build**: a **dependency's**
> module is `pkg/<vendor>/<name>/<m>`; the **root package's** own module is the spelling the package itself
> writes, `<prefix>/<m>` when it declares a `module_prefix`, else `<vendor>/<name>/<m>`. `Load` keys and names the module by it; `extractImports` rewrites `imp.Path` to
> it and keeps the author's spelling in `ImportDecl.WrittenPath`; every downstream consumer already reads
> `imp.Path`/`LoadedModule.Imports`, so no consumer changed. Decisions:
>
> - **A dependency is named by its published path, not its prefix.** `pkg/<vendor>/<name>/<m>` is unique
>   in a build (package names are unique in the lock; a `module_prefix` is not — two dependencies may share
>   one), it is what consumers write, and it is already the identity `./x` imports inside a dependency get.
>   Naming dependencies by their prefix would also have renamed every ordinary `pkg/` import of every
>   package without one.
> - **The root package keeps its own spelling.** A first cut used `pkg/…` for the root too, so a package
>   had the same identity in its own repository and installed. The downstream gate showed the cost: no
>   verdict moved, but 15 motoko-fork diagnostics went from `effect checking failed in
>   src/core/test/dst_harness` to `… in pkg/local/motoko_agent/core/test/dst_harness` — the root's own
>   modules renamed to a path their authors never write, and the same rename reaches serve-api
>   `CanonicalID`s and traces. Identity only has to be unique *within a build*, and the root's spelling is:
>   MOD013 already forbids a dependency to share the root's prefix. So the root's modules keep the ID they
>   always had under their usual spelling (also the path-derived ID of the same file checked as an entry);
>   only the other spellings (`pkg/<root>/…`, the bare form of a prefixed root, `./x` in an unprefixed
>   root) fold into it. Consequence: a package's module has one identity in its own repository and
>   another when installed — never two in one build.
> - **Kept as before:** the **entry file** (its path-derived ID — the entry imported back by its own
>   closure is an import cycle under either ID, reported as LDR002), and the ROOT's **unexported** modules
>   (the resolver refuses them, so the root's project-relative path is used exactly as before — there is only
>   one working spelling). A ROOT spelling the resolver rejects fails with the error it failed with before; a
>   DEPENDENCY's fails in the dependency's scope (round 2).
> - **Logical, not realpath — and checked.** The mapping is package name + module path, so two vendored
>   copies (distinct files) of different packages stay two modules. A file reached by a path no resolver
>   governs (a symlink into a package, a file-path import of a package file) would be a second identity;
>   the pipeline's file↔identity check makes that an error (LDR003), so "one file, one identity" holds in
>   every build that compiles.
> - **Two versions of one package in one build: impossible today, by an explicit error at lock time.**
>   `ailang lock` rejects a diamond with `VersionConflictError` (`pkg/resolver.go`) and the lock holds one
>   entry per name (`LockFile.FindPackage`), so `pkg/<vendor>/<name>/…` is version-free and still unique.
>   If multi-version builds are ever allowed, the version must join the identity.
> - **Unchanged by construction and verified:** the Stable v2 `interface_hash` (the hashed module is the
>   entry's *declared* path, `canonical_json.go:103-111`, and type strings carry no module) — `ailang
>   internal-dump-iface` output byte-identical for all 2,053 exported modules of 184
>   cached package versions that build (444 fail to build in both arms, identically), and
>   `make verify-stdlib` unchanged; `ailang iface` rendering; `cacheKeyVersion` stays `v10` — the cache
>   is indexed by module ID, so a renamed module simply misses once; no format changed.
> - **Diagnostics name both.** `loader.DisplayModule` / `DeclaredModuleDisplay` are wired into IMP010
>   (`symbol 'Triangle' not exported by 'pkg/acme/shapes/types/shape (imported as shp/types/shape)'`),
>   LDR002 (`… -> pkg/acme/shapes/cyc/b (imported as shp/cyc/b) -> …`, and the cycle is no longer printed
>   twice) and the type / effect errors of a dependency module (`type error in pkg/acme/shapes/ops/area
>   (module shp/ops/area)`). LDR001 needs nothing: a spelling that does not resolve keeps its spelling.
>   The root's own modules and entry files are named exactly as before.
> - **Shared prefixes are scoped, not just deterministic.** Two dependencies may share a `module_prefix`;
>   each one's own prefix spellings name its own modules. The build-wide table (now sorted, so it no
>   longer follows Go's map order) only answers a prefix no package in scope claims.
>
> *A behaviour change that was a bug — and whose ceiling bounds a dependency.* `isOwnPackageModule`
> (`pipeline/effect_ceiling.go:67`) treats every non-`pkg/` module ID as the root's own, so a
> **dependency** module reached by its prefix spelling was judged WHOLE against the **root's**
> `[effects].max` — every function in it, called or not ("effect ceiling violation in package acme/app" for
> code acme/app does not own) — while the same file under its `pkg/` spelling was exempt. It is now exempt
> under every spelling. This is not a loosening of authority (independent review, 2026-11-10): in a
> consumer build only the ROOT's ceiling is ever enforced (a dependency's own `[effects].max` is not
> checked there, on `dev` or here), and it bounds a dependency through the root's code — rule (a) on the
> root's declared rows, rule (b) on every reference the root's code makes to a global it does not own
> (`isAuthorityEntry`), which charges the referenced function's whole effect row. A root capped at `[IO]`
> that calls, returns, stores in a record, or returns a transitively-effectful dependency function (FS,
> Net) is rejected under both spellings on `check`, and on `run` (evaluator and `--strict-bytecode`) before
> any effect (`TestCanonicalModuleIdentity_RootCeilingBoundsDependencyReferences`,
> `TestCanonicalModuleIdentity_RunRefusesDependencyEffectBeyondRootCeiling`). After M1b, rule (b) is the
> ONLY thing bounding a prefix-spelled dependency, which is why those tests exist: a mutant that stops
> charging `pkg/` references survived every test before them and fails 8 subtests now.
>
> *Review fixes (independent review of PR #2224, 2026-11-10; each reproduced with a failing test first).*
> (1) the ceiling tests above. (2) Identity was built from the spelling: in a package without `[exports]`,
> `pkg/acme/shapes/types/shape` and `pkg/acme/shapes/shp/types/shape` reach one file and compiled it twice
> (its module-level code ran twice). Fixed by deriving the module path from the resolved file; the loader
> now records file↔identity for the whole build and `pipeline.Result.ModuleIdentityConflicts` lists any
> violation (tests assert none). (3) Prefix lookup was build-wide, so when two dependencies share a prefix
> one package's own `px/types/t` loaded the other's file — `dev` did so by map order, the first cut of this
> PR deterministically. Fixed by scoping; a dependency's own prefix spelling that does not resolve fails
> naming that dependency. Measured on the reviewer's fixture in both orders. The real
> `ailang_parse`/`docparse` pair no longer shares a prefix (`docparse` uses `docparse_api`), and no other
> pair in the cached registry does, so no real build was mis-resolving. (4) A prefixed package that is only
> a TRANSITIVE dependency could not load its own prefix imports (LDR001); scoping fixes it. (5) The display
> helpers were not wired; they now are (above), and the unused `ImportDisplay` is gone. Pre-existing on
> both, not changed: with a hand-written lock, a root that depends on an old copy of itself has its own
> files replaced by the old copy (`ResolveImport` consults the lock before the self-reference).
>
> *Tests* (`internal/loader/canonical_identity_test.go`, `internal/pipeline/canonical_module_identity_test.go`,
> `internal/runner/canonical_module_identity_test.go`): every spelling → one `*LoadedModule` and one cache
> entry; a spelling-first `Load` is still stored canonically; unresolvable spellings unchanged; memo
> invalidated by each setter; shared prefix deterministic; in a full pipeline no two loaded modules share
> a source file (`assertOneIdentityPerFile` — the test that fails if a second identity reappears) and
> every import names a loaded ID, both from the consumer and inside the package; the module-level binding
> runs once; IMP010 names both. Each was mutation-checked (18 mutants, scratch copy → mutate → test → copy back + `cmp`; all killed — two survived the first round and got `TestLoad_SpellingFirstStillCachedCanonically` and a per-setter memo test).
>
> *Measured after.* Downstream (`make downstream-check`, baseline = a `dev` build of `b517ebe09`, candidate =
> this branch; 2,099 entries): **0 NEWLY_FAILING, 0 fail_changed, 0 newly_passing in every corpus**; exit 4
> only for the 66 entries the *baseline* cannot measure (daneel 53, motoko-fork 11, email-parse 1,
> ailang-packages 1 — unresolved private dependencies, not this change). 6 `fail_diag_changed`, all in
> ailang-packages and all the same intended rename: a root package without a prefix that reached its own
> module by `./reply_runner` used to report it as `pkg/sunholo/crew_lab/reply_runner`; it now reports the
> name the file declares, `sunholo/crew_lab/reply_runner` (likewise `sunholo/motoko_ext_a2a/a2a`). The first
> cut (root under `pkg/` too) had produced 15 such renames in motoko-fork instead, in the other
> direction — the reason for the root rule above. Same-file probe (scratch instrument running the
> pipeline per corpus file and grouping loaded modules by realpath): on `dev`, **16 of docparse's 44 files** loaded at least one file under two IDs (129 same-file pairs) and one ailang-packages file did (a bare and a `pkg/` self-import); on this branch **0** in docparse, ailang-packages (600 files), ailang-parse, email-parse and daneel; motoko-fork and world (1,515 corpus files in all; 1,190 check in both arms, the probe sees module sets of those). Remaining realpath-level double loads: none. docparse's
> `tests/verify_ai_cache.ail` now compiles 68 modules instead of 70 — `types/document` and
> `services/output_formatter` once each — which is the spike's M1b acceptance. docparse's serve-api
> (corpus copy) dropped 127 out-of-basePath module loads before and 95 after (32 `ailang_parse` files
> were loaded twice); its registered surface (`/health`, `/api/_meta`, MCP `tools/list`) is byte-identical. `check` over `examples/` and `std/` (cold, isolated
> caches, `dev` vs branch): 607 files, 521 pass on both, **0 verdict flips, 0 differences in output text**.
>
> *Measured after the review fixes* (baseline = a fresh `dev` build, candidate = this branch rebased on
> it). Census — a scratch instrument run by the old head (`07b4a4bb0`) and the new one over every
> downstream corpus copy, recording each import edge's resolved file and ID: **3,780 distinct import
> edges in 7 corpora, 0 resolved to a different file, 0 changed ID, 0 verdict flips**. The holes the review
> found (items 2–4) are latent; no current downstream build hits them. `make downstream-check`: **0
> NEWLY_FAILING, 0 fail_changed, 0 newly_passing in every corpus**; exit 4 only for 4 baseline-unmeasurable
> entries (email-parse 1, motoko-fork 1, ailang-packages 2); the same 6 intended `fail_diag_changed`
> renames as before; no write-audit (exit 3) false positive in this run. `internal-dump-iface` over the
> cached registry: 0 differ of 2,497 modules (2,053 identical, 444 fail identically); `examples/` + `std/`
> `check`: 607 files, 0 flips, 0 text differences; `verify-stdlib` 49/49; `cacheKeyVersion` still `v10`.
> Mutation: 24 mutants over the new code, all killed.
>
> *Review fixes, round 2 (second independent review, 2026-11-10).* On real code the first round was
> already clean (the reviewer compared 18,844 import edges over 4,887 files: 0 resolved differently); five
> constructed holes remained, all present on `dev` too. (1) The file↔identity record was only read by tests;
> it is now a production check — the pipeline fails the build with **LDR003** after the cycle check — and a
> root package's project-relative spelling of its own file (`import util` beside `acme/app/util`, fxJ: init
> ran twice) is canonicalised to that module. (2) `Load` re-resolved a dependency's identity in the ROOT's
> scope, so a dependency's miss could silently run the root's file (fxC) or another package's (fxA, dev);
> an identity resolved in an importer's scope is now loaded as resolved, and a dependency's miss fails in its
> own scope (**LDR001** naming the package, the published path tried and the spelling written). (3) An import
> could be satisfied by a package the importer does not declare (fxB prefix table, fxB2 `pkg/` path, fxG a
> dependency importing the root): **LDR006**. Also: two declared dependencies providing one prefix spelling
> is **LDR005** naming both (fxD; it used to pick one by map order, then alphabetically); the bare
> self-import is scoped to the importer (fxM: a dependency whose prefix is the root's vendor got the root's
> `util`); a spelling whose letter case differs from the file on disk is **LDR001** on every host (fxI:
> `cc/Init` was a second identity of `cc/init.ail` on macOS and "not found" on Linux); LDR002 shows how the
> first module of a cycle was imported.
>
> **The import resolution rule, as implemented** (`internal/loader/canonical_scope.go`). A spelling S written
> by a module of package P is resolved in P's scope — P, the packages P declares in `[dependencies]`, std:
> 1. `std/…` and file paths (`…/x.ail`, `../x`) are not package spellings.
> 2. `pkg/<vendor>/<name>/<m>` names that package; it must be P or one P declares, else **LDR006**.
> 3. `<vendor>/<name>/<m>` with `<vendor>/<name>` = P is P's bare self-import.
> 4. `<prefix>/<m>`: P's own module if `<prefix>` is P's `module_prefix` and P has it; else the module of the
>    ONE declared dependency with that prefix that has it; two such dependencies is **LDR005**.
> 5. Otherwise, for a DEPENDENCY P: **LDR006** if a package outside P's scope (or the root) would have
>    provided it, else **LDR001**; never retried in another scope. For the ROOT: a path to a file of the root
>    package is canonicalised to that module; any other path is loaded by path, as before.
> The resolved (package, file) is then named — module path derived from the file, so every spelling of one
> file agrees; a dependency as `pkg/<vendor>/<name>/<path>`, the root by its own spelling — and a spelling
> whose case differs from the file on disk is **LDR001**. Finally the pipeline checks the whole build's
> file↔identity record: a file read under two identities, or an identity read from two files, is
> **LDR003** (only reachable through a path no resolver governs, e.g. a symlink into a package; two
> DIFFERENT packages that are literally the same files through a symlink are therefore an error, while
> vendored copies — distinct files — stay distinct packages). Without a resolver that can describe package
> layouts, or outside any package, the pre-scoping rule applies unchanged.
>
> *Measured after round 2* (baseline = `origin/dev`, candidate = this branch): __C2__
>
> **What M2 can now assume** — each a property the build enforces, not a convention.
> (1) **One identity per file per build.** Every file the loader reads has exactly one module ID and every ID
> names exactly one file, or the build fails with LDR003 before anything is compiled. That ID is
> `LoadedModule.Path` = `iface.Module` = the `modID` the pipeline compiles it under = `GlobalRef.Module` of
> its exports, so stamping `TCon.Module` from any of them gives one identity per declaration. The entry file
> keeps its path-derived ID; another spelling of it is an import cycle, reported as LDR002 first.
> (2) **The ID is canonical, not a spelling.** After `Load`, every `ImportDecl.Path` and `LoadedModule.Imports`
> entry is that ID; the author's spelling survives only in `WrittenPath` (diagnostics). The ID is
> `pkg/<vendor>/<name>/<m>` for a dependency and the package's own spelling for the root — compare IDs
> within one build only; never read "dependency or not" out of the string, and never compare across builds.
> (3) **A name means one thing in one importer.** A prefix or self-import spelling resolves only within its
> importer's declared scope; nothing resolves through a package the importer does not declare (LDR006), and
> an ambiguous prefix is an error (LDR005), so a type reached through an import is a type of a package the
> importer chose. (4) Nothing in M1b needs a cache-key bump; `cacheKeyVersion` stays `v10` and M2's v11 covers
> its qualified digest. (5) The REPL/WASM loader has no packages; it has no second spelling to fold, and it
> still needs the M3 resolver.
>
> **The four-line constructor stamp** the exhaustiveness wiring needs (from the spike, on `dev` lines of
> `internal/iface/builder.go`): the constructor result type is built from a bare name in four places —
> `:330` `Constructor: &types.TCon{Name: ctorInfo.TypeName}`, `:334` `resultType = &types.TCon{Name:
> ctorInfo.TypeName}`, `:410` `Constructor: &types.TCon{Name: typeDecl.Name}`, `:414` `resultType =
> &types.TCon{Name: typeDecl.Name}` — each gains `Module: b.module`. `b.module` is the `modID` passed by
> `buildAndRegisterInterface`, which after M1b is canonical, so `adtUniverse[(module, name)]` built from
> `ConstructorScheme` has one universe per ADT. (`TCon.Module` itself is M2's representation change; it
> is not in this milestone.)

**M1 Duplicates (0.5–1 d, independent of M0).** MOD007 func/type/constructor cases.
*Tests:* `v8`, `v12`, `v13` rejected with both positions; MOD007 let cases unchanged; a func defined
once in each of two modules is fine. *Mutation:* delete the func-vs-func loop, `TestMOD007_DuplicateFunc`
must fail. *Downstream:* PRs (by their owners) removing the five duplicates; `ailang_parse` republished
and the docparse/email-parse pins moved before this lands in a tag.

**M2 Nominal identity (3–4 d).** `Equals` on `(Module, Name)`; `nominalHome` table and its completeness
test; qualified `$adt` factory keys, `ctorTypes`, `adtTypeParams`, `ConstructorPattern.TypeName`, derived
Eq keys; `TC_NOMINAL_001`; qualified cache digest; `cacheKeyVersion` v11; wildcard removed.
*Tests:* every §1 a-row rejected with `TC_NOMINAL_001` naming both modules; `v15` on evaluator and
`--strict-bytecode`; two modules deriving Eq on `Row` both work; an upstream changing which `T` it
returns invalidates the importer's cache (`TestCacheKey_NominalOriginEditInvalidates`); every
`interface_hash` in `.stdlib-golden/` and of the 61 cached packages byte-identical before/after.
*Mutation:* revert `Equals` to name-only, the q repro test must fail.

**M3 One resolver (2 d).** `TypeScope` with the §4.3 order; `TC_UNBOUND_TYPE_001`;
`TC_AMBIGUOUS_TYPE_001`; MOD015 for import-vs-import types/constructors.
*Tests:* `v9`, `v10` rejected; every §7 fixture passes; auto-imported `Json` resolves; an `as`-renamed
type is unbound under its original name (#1485's second case). *Mutation:* make rule 5 return the
first candidate, the ambiguity test must fail; drop the unbound error, `v9` must pass `check` again
(proving the test sees the change).

**M4 Back ends and merges (1.5–2 d).** SMT and REPL/WASM keyed qualified; bytecode ADT tables keyed
qualified. *Tests:* a contract over `a.T` verified against a's constructors when `b.T` is loaded; a REPL
session importing both modules; the VM tag-collision fixture from the LIMITATIONS row.

**M5 Promise, docs, gate (1 d).** Stability clause (§10); error pages; teaching prompt; LIMITATIONS;
changelog. *Accept:* downstream gate green with moved pins; `make test`, `make verify-examples`,
`make freeze-stdlib` produces no diff.

Testing notes: `ailang check` checks only its first file argument, so multi-module tests use Go
pipeline tests or `check` on the entry module; every new error code gets a positive and a negative test.

## 10. The soundness clause for `stability.md`

Proposed text, to go under "What Stable means in 1.x":

> **Soundness fixes are not breaking changes.** A Stable program is one the documented rules accept. If
> `ailang check` accepts a program those rules reject, that is a bug, whether it shows as a run-time type
> error, a name that resolves to the wrong declaration, or an effect the checker missed. Fixing it may
> make a previously accepted program fail to check, and the fix may ship in any 1.x release. Such a fix:
> - only rejects programs; it never changes what an accepted program computes;
> - is listed in the CHANGELOG under `Fixed — … (soundness)`, with its error code and the migration;
> - is run against the downstream gate first, and the number of affected programs is published;
> - ships as a warning for at least one minor before it becomes an error, where that is feasible —
>   but immediately as an error when the hole lets a program reach behaviour its types forbid.
>
> Experimental items need no carve-out; they may change in a minor anyway. Internal items carry no promise.

Sources adapted: Rust's 1.0 promise ("we reserve the right to fix compiler bugs, patch safety holes, and
change type inference in ways that may occasionally require new type annotations") and RFC 1122's
process (crater run, future-incompatibility warning where possible); Go 1's carve-out for bugs that
violate the specification ("a program that depends on the buggy behavior may break if the bug is
fixed"). Quoted from memory, not re-fetched. AILANG's version is narrower than Rust's: it does not
reserve inference changes, only rejections of programs the rules already forbid.

## 11. Migration

**What a downstream author changes:** delete duplicate definitions (MOD007); import the type you use
by name when two direct dependencies export it (`TC_AMBIGUOUS_TYPE_001`); stop passing one module's `T`
to another module's `T` (`TC_NOMINAL_001`) — convert explicitly or use one type; declare or import
typo'd type names (`TC_UNBOUND_TYPE_001`). No source change is needed for programs that only use
same-named types disjointly. No republish is needed for identity; `ailang_parse` needs one for d2.

**Changelog fragment draft** (`changelogs/unreleased/YYYY-MM-DD-nominal-type-identity.md`):

```markdown
### Fixed — Types are identified by name and module; two `T`s from different modules no longer unify (soundness, BREAKING)

`check` accepted programs that passed one module's `type T` where another module's `type T` was
expected, and they then failed at run time (`_str_upper: arg 0 - expected string, got int`). A
nominal type is now its defining module plus its name. Mixing them is `TC_NOMINAL_001`; an
undeclared type name is `TC_UNBOUND_TYPE_001` (#1485); a bare name two dependencies export is
`TC_AMBIGUOUS_TYPE_001`; the same type imported twice is `MOD015` (#1980); a func, type or
constructor declared twice in one module is `MOD007` (#1278). Interface hashes are unchanged.
Migration: design_docs/planned/v1_0_0/m-v1-nominal-type-identity.md §11.
```

**Docs:** `LIMITATIONS.md` — delete "Same-named types across imports" (lines 139-148) and narrow the
bytecode tag-collision row; the canonical `docs/docs/reference/limitations.md` has no entry today, so
nothing to remove there. Teaching prompt "Module Import Rules": add two lines — a type belongs to the
module that declares it, and import a type by name if two imports export it — and remove nothing (the
prompt never taught the unsound behaviour). Version the prompt through prompt-manager. Error pages for
the three new TC codes; update `mod007.md` and `mod015.md`. `stability.md`: §10 clause; drop the
bare-name note if one is added meanwhile.

## 12. Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| A minting site left unqualified → false rejection | High in M2 | M0 counter; debug-build panic; `nominalHome` completeness test; downstream gate |
| Cache serves a pre-change verdict | Low | `cacheKeyVersion` v11 |
| `interface_hash` drifts and publishing breaks (`PUB005`) | Med | M2 byte-identity test over goldens and 61 packages |
| Interacts with #2146 exhaustiveness work landing in parallel | Med | exhaustiveness reads constructor sets by qualified ADT; land M2 first or agree the key |
| Window slips | Med | Alternative B fallback named in M0 |

**Non-goals:** module-qualified type *syntax* (`S.Set[int]`, the LIMITATIONS parser papercut — additive,
1.x); re-export syntax; changing alias structural equality; removing the auto-import channels;
`ifacev3`; Go codegen correctness beyond name mangling.

## 13. Axiom compliance

A1 +1 (verdicts no longer depend on import order); A2 0; A3 0; A4 +1 (a name means what its module
says, not what is nearby); A5 +1 (a module's typing depends only on its own declarations and imports);
A6 0; A7 +1 (a coded error at the cause instead of a distant run-time crash); A8 0 (no syntax); A9 0;
A10 +1 (two packages can each export `Event`); A11 +1 (four coded errors replace silent behaviour);
A12 +1 (identity checked at the module boundary). **Net +7.** No hard violation on A1/A3/A4/A7.

## 14. Decisions for Mark

1. **Qualified identity (A) over closure-wide uniqueness (B)?** Recommend A, with B as the named
   fallback if M0 says A will not fit the window.
2. **Duplicates in dependency code: error, or warning for `pkg/` modules during 0.58.x?** Recommend
   error everywhere and republish `ailang_parse` (delete one duplicated function) before the tag; a capped-lint
   mode would be a second promise to keep.
3. **MOD015 or MOD016 for import-vs-import?** Recommend MOD015; the registry has promised it since v0.52.0.
4. **Keep `interface_hash` on display names (no republish), with identity in a non-hashed field?**
   Recommend yes; `ifacev3` is a 1.x follow-up.
5. **Adopt the §10 clause, including warn-first-where-feasible?** Recommend yes.
6. **Fold the #1485 and #1980 type-namespace docs into this one?** Recommend yes; the value-namespace
   half of #1980 stays in its own doc.

## 15. Findings that differ from the review

- Row a is not new: M4 of m-type-name-shadow-and-cache (2026-09-26, audit row 8) recorded it, at P2.
- The review lists three silent acceptances; there are five (d4 duplicate types, d5 duplicate
  constructors), plus a″, which `LIMITATIONS.md` describes as loud.
- #1485 and #1980 already have planned docs, and #1980 has two that disagree on the error code.
- The measured break from the soundness fix is zero; the measured break from #1278 falls on a published
  package two flagship downstreams pin. The review's "rejects programs that pass today" is true for
  d2, not (on this corpus) for a.
