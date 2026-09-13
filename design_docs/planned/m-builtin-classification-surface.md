# M-BUILTIN-CLASSIFICATION-SURFACE: Producer-Visible Builtin Classification

**Status**: Planned (tooling/interface surface — no language-semantics change)
**Target**: v0.39.0
**Priority**: P1
**Estimated**: 2 days
**Dependencies**: None
**Origin**: GitHub issue #901 (public-feedback inbox, triaged; verified still valid at HEAD
2026-09-13 — `ailang iface --help` exposes only `-compact`, no builtins mode, no
machine-readable builtin list).

## Problem Statement

Motoko extension authors need a **static effect classifier**: given an `.ail` source file,
decide which effects a package declares vs. which it actually uses, so `ailang publish`
sandboxes run with the correct `effects` ceiling. That classifier needs a closed, classified
inventory of language builtins. Today it can get one only for **underscore** builtins — by
scraping `internal/builtins/registry.go` (not producer-visible, and scraping is brittle). For
**non-underscore** language builtins (`show`, `toText`, `println`, `intToFloat`,
`floatToInt`) there is no stable answer:

**Current State (verified at HEAD, base v0.38.5):**

- `ailang iface <module> [--compact]` (`cmd/ailang/main.go:261`, `cmd/ailang/check.go`)
  dumps a *user module's* interface only. There is no mode that reports the language's own
  builtins, in any format.
- The compiler-side registry `internal/builtins/registry.go` (`BuiltinMeta{Name, NumArgs,
  IsPure, GoCodegen}`) holds purity (`IsPure`) and stdlib-folding data
  (`GoCodegen.StdlibName`/`StdlibModule`) for the underscore builtins **and** `intToFloat` /
  `floatToInt` — but nothing exports it.
- `show`, `toText`, `println` are registered only via `env.Set(...)` in
  `internal/eval/eval_simple.go` and `internal/eval/eval_typed_helpers.go` — **outside both
  registries**, with no purity metadata anywhere.
- `internal/iface/builtin_freeze.go` is a **hand-maintained** copy of the `$builtin`
  interface (name/type/arity/category per export). It is exactly the staleness anti-pattern
  this doc must not extend: it already drifts by construction whenever a registration site
  changes.

**Impact:** a static effect classifier is blocked for **9 of 15** extension packages. The
author either hard-codes a builtin list into their classifier (goes stale), or falls back to
dynamic observation (defeats "static"). Per the mission's default bias this is a
**tooling/interface** fix, not a language-semantics change: the classification data already
exists in the compiler; it just has no producer-visible surface.

## The Three Proposed Shapes (and the Ruling)

The issue reporter proposed three shapes:

1. **An `iface` builtins mode** — `ailang iface --builtins` prints the language builtin
   interface.
2. **A prelude interface document** — a checked-in markdown/reference document listing every
   builtin with its class.
3. **A machine-readable builtin inventory** — a JSON (or equivalent) artifact enumerating
   builtins with name, type, arity, and classification, derived from compiler source.

### Ruling: shape 3 is the deliverable; shape 1 is its CLI door; shape 2 is generated, never authored

**Pick: machine-readable inventory, exposed through `ailang iface --builtins [--json]`.**

Reasons:

- **The consumer is a program, not a person.** The effect classifier needs a parseable,
  closed set. Shape 2 alone (a doc) is for humans and rots; shape 1 alone (a print mode)
  invites ad-hoc text formats that drift from whatever the classifier needs. Shape 3 with a
  `--json` flag subsumes shape 1: the same command with no `--json` can render a
  human-readable table.
- **One source, many views.** The doc (shape 2) should be *generated from* the inventory
  into `docs/` on release (like other docs-sync flows), not hand-maintained. The existing
  `internal/iface/builtin_freeze.go` proves hand-maintained copies drift; we do not add
  another.
- **No new subcommand sprawl.** `iface` already answers "what does this module export?" —
  "what does the *language* export?" is the same question with a different subject. A
  `--builtins` flag reuses the existing command and help surface (CLI doc-maintainer skill
  applies when implemented).

## Classification Classes and Where They Live Today

Every inventory entry carries exactly one class:

| Class | Meaning | Where it lives today (HEAD) |
|---|---|---|
| `pure` | No effects; deterministic value→value | `BuiltinMeta.IsPure == true` in `internal/builtins/registry.go` (e.g. `add_Int`, `intToFloat`) |
| `effect` | Performs I/O or requires a capability | `BuiltinMeta.IsPure == false` (e.g. `_io_println`, `_net_httpGet`); interpreter side: `BuiltinFunc.IsPure == false` in `internal/eval/builtins_io.go` |
| `stdlib` | Foldable to a stdlib function; pure for classification purposes | `BuiltinMeta.GoCodegen.StdlibName != ""` (e.g. `_str_trim` → `std/string` `trim`) |

`stdlib` entries are also `pure`; the class field is the *strongest* statement (stdlib
refines pure). `show`, `toText`, `println` have **no classification anywhere today** —
closing that gap is part of this design (they must be registered in the meta registry or
annotated at their `env.Set` sites so the derivation can see them). Working classification
(ratified by implementation review, not by this doc alone): `show`/`toText` pure;
`println` effect (Console/IO).

## Staleness Guarantees

1. **The inventory derives from the same source the compiler uses.** The emitter walks the
   live `internal/builtins` Registry plus the prelude `env.Set` registrations (once those
   carry metadata) at *invocation time* — the JSON a consumer gets is what the binary in
   front of them actually enforces. **Never a hand-maintained list.** No new
   `builtin_freeze.go`-style copy.
2. **Regression gate:** a test asserts inventory output is non-empty, total (every Registry
   key and every prelude `env.Set` name appears), and that every `IsPure == false` entry
   maps to a known capability. Adding a builtin without classification metadata fails the
   build — staleness becomes a compile-time error, not a runtime surprise.
3. **Version stamping:** the JSON carries the binary's version (`std/VERSION`), so consumers
   can detect classifier/binary skew explicitly rather than misclassifying silently.

## Consumer Contract (Effect Classifier)

A consumer (the extension effect classifier, or any future linter) may rely on:

- **Stability:** within a version, `ailang iface --builtins --json` output is deterministic
  (sorted by name, stable field set). Breaking field changes require a format-version bump.
- **Closure:** any identifier in `.ail` source that is *not* in the inventory and *not* a
  module import is a user-defined binding — the classifier may treat "not in inventory" as
  "not a language builtin" without fear of an invisible builtin.
- **Class semantics:** `pure`/`stdlib` ⇒ contributes no effects; `effect` ⇒ contributes the
  named capability to the package's required-effects set.
- **Unknown-name behavior:** the classifier must **not** silently default unknown names to
  pure (no silent fallbacks, CLAUDE.md §2) — an unknown non-import identifier is a hard
  classification error the classifier surfaces.

## Alternatives Considered

- **Classifier scrapes Go source (`registry.go`) directly** — rejected: couples consumers to
  internal layout; breaks silently on refactors; not available to installed binaries.
- **Extend `internal-internal-dump-iface`** — rejected: internal-only command shape; the
  consumer surface must be a documented, versioned CLI.
- **Hand-maintained prelude doc only** — rejected: measured anti-pattern
  (`builtin_freeze.go`); docs may be *generated* from the inventory instead.

## Acceptance Criteria

- [ ] `ailang iface --builtins --json` emits the full inventory (underscore + prelude
      builtins) with name, arity, class, and (for `stdlib`) the stdlib target.
- [ ] `ailang iface --builtins` renders the same data human-readably.
- [ ] A regression test fails if any registered builtin lacks classification metadata.
- [ ] Generated prelude doc lands in `docs/` via docs-sync; no hand-authored builtin list is
      added anywhere.
- [ ] CLI help (`help.go`) updated per cli-doc-maintainer conventions.

**DESIGN_DOC_PATH:** `design_docs/planned/m-builtin-classification-surface.md`
