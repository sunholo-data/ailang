# M-PARTIAL-ACCESSOR-SHAPE — total accessors, honest denominators, self-describing docs

**Status**: partially implemented (this branch) · two decisions open for Mark (D1, D2)
**Origin**: four external reports from `holosun-decision-envelope`, 2026-08-27, all against
`v0.34.0-57-g816bc80d3` darwin/arm64, filed while building a document validator and an email
parser in AILANG.
**Lane**: AILANG fix (stdlib + CLI). No motoko surface, no core change.

## Why these four are one document

They arrived as three messages and read as unrelated papercuts. They are not. Each is the same
failure: **the toolchain knows something the caller needs and does not say it.**

| Report | What the tool knew | What it said |
|---|---|---|
| `charAt` panics | index is out of range | aborts the process |
| `verify` hides skipped | 14 of 25 exports carry no contract | `11 functions: 11 verified` |
| `docs std/process` mangled | `exec`'s summary line | `}` |
| `prompt --version 0.34.0` | prompt versions are a separate series | `not found in versions.json` |

Grouping them matters because the fixes are individually trivial and collectively a policy: a
partial operation must be partial *in its type or its output*, never only in its runtime
behaviour.

## The failure mode that makes `charAt` worse than a crash

The reporter's account is the important part, and it is not "it crashed":

> I used charAt to test whether a user-supplied date field began with a digit. Absent fields
> arrive as "", so charAt panicked mid-`map` over a list of records. Because the panic exits
> non-zero, my negative tests — which assert "this bad input is rejected" by checking a non-zero
> exit — all still passed. The suite was green while the program crashed on every input.

A panicking accessor **collapses the distinction between "rejected correctly" and "crashed"**
for every test harness that grades on exit code, which is the default way generated code is
graded — including ours. This is the same class of defect as an eval harness that banks
`api_error` for "cause unknown": the signal and the failure are indistinguishable at the
measurement boundary.

`charAt` was the only partial accessor in the stdlib shaped this way:

```
std/list   head(list[a])           -> Option[a]
std/list   nth(list[a], int)       -> Option[a]
std/list   nth_or(list[a], int, a) -> a
std/string stringToInt(string)     -> Option[int]
std/json   asString(Json)          -> Option[string]
std/bytes  slice(bytes, int, int)  -> Option[bytes]

std/string charAt(string, int)     -> string        <-- aborts
```

## What this branch changed

### 1. `std/string` gains total accessors (additive)

```ailang
charAtOpt(s: string, i: int) -> Option[string]
charAt_or(s: string, i: int, default: string) -> string
```

Both are rune-indexed and agree with `charAt` on every in-range index. `charAtOpt("", 0)` is
`None`; negative and `>= length` are `None`. `charAt_or` mirrors `nth_or` in naming. `charAt`
itself is unchanged in behaviour but its doc comment now states the abort and the exit-code
consequence in full, so `ailang docs std/string` carries the warning.

Verified: `Some(e)`, `None`, `None`, `None`, `Some(🎉)`, `?`, `h` for the seven cases in the probe
(in-range, empty string, negative, off-end, multi-byte rune, default taken, default not taken).
`make verify-stdlib` was re-frozen deliberately — `string` was the only module that drifted.

### 2. `ailang verify` reports its denominator

`verify.go` skipped contract-less functions with a bare `continue`, so they never reached
`results` and never reached `total`. A module with 25 exports of which 11 carried contracts
reported `11 functions: 11 verified` — a true statement with a denominator chosen to make it
true.

Now:

```
  ✓ VERIFIED compute  21.7ms
  ○ NO CONTRACTS main

  2 exported functions: 1 verified, 1 without contracts
```

and `--json` gains `uncontracted` and `total_exported`. Names are listed so a function that was
*meant* to carry a contract and silently lost one is auditable without diffing source.

**`--strict` is deliberately unchanged.** An uncontracted function is not a verification
failure; counting it would turn `--strict` red on every module with a helper.

This matters beyond the report: the teaching prompt instructs agents to "maximize the surface
area of verified code", and until now the tool refused to print the bottom half of that ratio.
It is also the same denominator that gates the cost KPI — see the `skipped == 0` predicate note
in `m-contract-verification-coverage.md`.

### 3. `ailang docs` describes exports correctly and documents types at all

Two bugs in `cmd/ailang/docs.go`:

- `currentDoc` was **assigned** on every comment line, so an export's description was the *last*
  line of its comment block. `exec`'s block ends with the `}` of the `match` in its own example,
  so `exec` documented itself as `}`; `spawnProcess`'s ends with a call to `closeProcessStdin`.
  Now the *first* summary line of the block wins, blank lines terminate a block, and section
  banners (`-- =====`) cannot become a description.
- Only `export func` was ever matched, so **no exported type in any stdlib module was
  documented**. `ProcessOutput` — whose `.stdout` is `bytes`, not `string` — was discoverable
  only by printing the record and reading a hex dump. `docs` now renders a `## Types` section
  ahead of `## Exports`, preserving multi-line record and sum-type shapes.

`parseModuleFile` was rewritten from `bufio.Scanner` to an index-based scan because both the
multi-line `export func` signature and the multi-line `export type` declaration need lookahead,
and a Scanner can only over-read.

### 4. `ailang prompt` explains its own version namespace

`--version 0.34.0` still fails — correctly, because there has never been a prompt numbered
`v0.34.0` — but it no longer fails opaquely:

```
Error: "0.34.0" is not a known prompt version

  Prompt versions are their own series and do not track the binary's
  release version — there is no prompt numbered for each AILANG release.

  Active prompt version: v0.16.6  (omit --version, or pass --version latest)
  List every available version: ailang prompt --list
```

The namespace paragraph appears only when the requested version sorts above every prompt on
record; an in-series typo (`v0.16.99`) gets the short form, so the hint does not become noise.
`prompt --help` no longer hardcodes `v0.16.0` as active, and no longer claims each prompt
"corresponds to a specific AILANG language version" — the claim that generated the confusion.

**No silent fallback was added.** Serving a different version than the one asked for, even with
a warning (the reporter's suggestion 2), is exactly the pattern `CLAUDE.md` §2 prohibits.

### 5. One teaching-prompt correction

The "What AILANG Does NOT Have" table offered only `func(a: T, b: U) -> R { body }` for
multi-param lambdas. That advice **does not work for an inline `foldl` callback**, so an agent
following it lands in the parse error the reporter hit. The row now names the curried form:

```
| `\(a, b). body` or `\a, b. body` | Inline: **curry** — `\a. \b. body`. Named: `func(a: T, b: U) -> R { body }` |
```

Verified by running it, not by inspection: `foldl(\acc. \x. ..., 0, xs)` typechecks and returns
`6` for `[1,2,3,-4]`. `v0.16.6` is the single mutable registry entry; both `versions.json` hashes
are updated and the embedded mirror re-synced.

## Not fixed here, with reasons

**The multi-line lambda error location.** The report says the parser loses the good diagnostic.
It does not — it emits it *first* and then buries it:

```
expected '.' after lambda parameter at lam2.ail:4:13     <-- correct
PAR_RESERVED_KEYWORD at lam2.ail:5:5 ...
PAR_UNEXPECTED_TOKEN at lam2.ail:5:5 ...
PAR_UNEXPECTED_TOKEN at lam2.ail:7:28 ...                <-- what the reporter read
```

That is [#934](https://github.com/sunholo-data/ailang/issues/934) — "one bad token emits a
342-error cascade, drowning the real error" — not a lambda-specific defect. Fixing it there
fixes it for every construct. Routing it here would produce a second, narrower implementation of
the same fix.

**`std/yaml` absent from the teaching prompt.** Confirmed: `std/yaml.ail` exists, the prompt
mentions it zero times. This is prompt *curation*, not a defect, and the prompt-curation lane
already has `m-prompt-freeze-mirror-all-versions` queued on this surface. Filed as a backlog item
rather than smuggled into a bug-fix branch, because adding a module section changes what every
eval model is taught and deserves its own baseline decision.

## DECISION FOR MARK — D1: does `charAt` keep its name?

This branch is **additive**: `charAt` still aborts. The reporter's preferred fix, and the one
consistent with `nth`/`head`/`stringToInt`, is to make the *primary* name total:

| Option | Shape | Cost |
|---|---|---|
| **(a) ship as-is** | `charAt` aborts; `charAtOpt`/`charAt_or` total | The landmine stays. Every agent that reaches for the obvious name still gets it. |
| **(b) breaking rename** | `charAt -> Option[string]`, abort form retired or renamed `charAtUnsafe` | Breaks `benchmarks/markdown_reimplement.yml`'s reference solution, the prompt's taught guarded pattern (`if i > 0 && charAt(s, i-1) == "\\"`), and any banked corpus that used it. Needs a prompt version bump, so it lands against a baseline boundary. |

Blast radius for (b) is smaller than it looks — 6 `.ail` files repo-wide use `charAt`, one of
which defines its own local shadow over a list. The real cost is the prompt edit and the
baseline discontinuity, not the code.

**Recommendation: (b), scheduled against the next prompt version bump rather than now.** (a)
unblocks the reporter today, which was the ask; (b) is the fix. Sequencing them means the
breaking change rides a baseline boundary we are taking anyway instead of creating a new one.

## DECISION FOR MARK — D2: `ai-check` has the same blind denominator, and it feeds the KPI

Per CLAUDE.md §3, the denominator bug was checked for a wider pattern. It has one:

```
cmd/ailang/verify.go:273     if len(meta.Contracts) == 0 { continue }   <-- fixed here
cmd/ailang/ai_check.go:289   if len(meta.Contracts) == 0 { continue }   <-- NOT fixed here
```

`ai-check` is a **completely separate implementation** of the same verification walk. It does
not call `printVerifyJSON`, and `internal/eval_harness/verify.go` decodes only
`available/verified/counterexample/skipped/errors` — so this branch's change to `ailang verify`
provably does **not** touch eval banking, the `skipped == 0` predicate, or
`cost-per-verified-success`. Nothing decodes with `DisallowUnknownFields`, so the two added JSON
keys are inert everywhere.

That isolation is why `ai_check.go` was left alone, not an oversight. Correcting *its*
denominator would change what `cost-per-verified-success` divides by, on a metric with a
recorded baseline (`$0.7778187072` on 2026-08-22, corrected to `$0.2121` once the
`no ensures clause` skips were exempted). That is a measurement decision with a baseline
discontinuity attached, and it belongs to whoever owns the KPI — not to a DX fix branch.

**Recommendation: fix `ai_check.go` in the same change that next re-baselines the KPI**, and
have it distinguish "no contract written" from "contract present but unencodable". Those are the
two halves the existing `no ensures clause` correction already had to separate by hand, and the
tool should report them apart rather than making every consumer re-derive the split.

Note that the two implementations drifting is itself the finding: a second copy of the
verification walk means every fix like this one has to be applied twice, and this one was not.

## Provenance

Reports are unread messages `inbox_1787844769465_3003896d`,
`inbox_1787844786100_40dd0a4d`, `inbox_1787844810838_60ab910f` in the canonical prod store,
from `holosun-decision-envelope`. A fourth from the same sender
(`inbox_1787844854611_e0eb9e40`, `std/smt` — expose the embedded Z3 to AILANG programs) is a
feature request on the same session's experience and is **not** covered here.

The reporter asked whether grouping three papercuts into one message was right. It was, but for
a reason they did not give: grouping cost nothing on inbox volume, and two of the three are
one-file fixes while the third is a duplicate of an existing issue — as a single message they
could only be closed together. Per-item closability, not inbox load, is the argument for
splitting.
