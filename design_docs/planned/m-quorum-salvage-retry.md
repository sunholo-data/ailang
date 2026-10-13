# M-QUORUM-SALVAGE — Never Discard a Reviewer's Reject; Locate the Parse Break

**Status:** implemented on branch `fleet/i36-quorum-salvage` (fleet iteration 36, 2026-10-14; M1-M3 re-verified outside the sandbox by the controller; moves to `implemented/` after landing)
**Type:** mechanical quorum-transport fix (`internal/mission/quorum` + one mission-control skill file). No AILANG language, stdlib, motoko-core, routing, lane-order, quota/ration threshold or billing surface.
**Source:** GitHub issue sunholo-data/ailang#941, the 2026-09-01 refresh comment (mission-world iter 143), which refuted the original premise.
**Fleet tickets:** `quorum:invalid-absent-on-quoted-literals` (charter P2 #12, `design_docs/fleet-mission.md:223`) and `quorum:reviewer-invalid-json-discarded`. This one doc closes both.
**Version context:** `v0.54.9-9-g42652c19e` (`git describe --tags`, 2026-10-14)

## Premise correction

The first version of this doc said reviewers broke JSON by quoting literals, and proposed a JSON-repair
pass plus a re-ask. The #941 refresh **refuted that by measurement**: the cited literals were escaped
correctly, and the break sat at byte 256, outside the published window. This revision drops the repair
and the re-ask. It fixes three defects, all re-measured below at HEAD:

1. **Verdict discarded (correctness).** On a parse failure the reviewer is recorded ABSENT
   (`run.go:159-174`, `agentic_caller.go:201-206`, `claude_reviewer.go:121-126`). The raw text often
   still holds a complete `"verdict":"reject"`. **7 of 7** `invalid` absences on disk carry one; the
   control (all other absence reasons) has **0 of 90** (V3). Salvageable by the strict scan: **4/4**
   unexpected-end rows by construction; **3/3** syntax rows confirmed; **0** unconfirmable from the
   retained window (V21).
2. **Diagnosis lost (observability).** `ParseReviewResult` publishes `(raw: %.200q)`, the first 200
   bytes (`reviewer.go:135`). `*json.SyntaxError.Offset` appears nowhere, because `.Error()` omits it
   (V5) and all three sites store only `perr.Error()` (`run.go:172`, `agentic_caller.go:204`,
   `claude_reviewer.go:124`).
3. **Wrong remedy (skill).** The absent-reviewer rule says *"re-run each absent reviewer alone with a
   raised cap"* (`.claude/skills/mission-control/resources/gate-2-pick.md:185-189`, mirrored byte-identically
   in `.agents/`). That rule targets `budget`. For `invalid` it spends more and fixes nothing.

### What the zero-signal guard (#1767, `a5f9af982`) already covers, measured at HEAD

`synthesize` now counts present EXTERNAL reviewers only, and blocks on zero (`quorum.go:153-187`).
Measured against HEAD with a probe test (V7):

| Shape | HEAD synthesis | Covered? |
|---|---|---|
| G1: every external reviewer absent `invalid` (hidden reject), controller `pass` | `blocked` | **Yes.** All 3 measured vacuous proceeds had this shape (V4) |
| G2: one external `pass` plus one absent `invalid` (hidden reject), controller `pass` | `proceed` | **No** |
| G3: `RunSeatedQuorum`, seats=3: A pass, B invalid (hidden reject), C pass, reserve D pass | `proceed` (4 outcomes; B listed absent) | **No.** This is the production path (`cmd/ailang/design_quorum.go:89`) |

So the guard closed every measured instance. The live hole is the **N-1 shape**: a discarded reject
next to at least one present pass. It has 0 measured instances on disk (V4), but at HEAD it can be
built with no assumptions. The reserve refill (`seating.go:114-120`) makes it *more* likely: it
replaces the silent rejecter with a fresh seat that may pass.

## Goals / Non-goals

**Goals.** (G1) No reviewer-emitted `reject` can be dropped on the way to the Tier-1 synthesis
(`synthesize`), on either twig of the invalid branch (plain parse failure or `finish_reason=length`
truncation, V18). `synthesize` is the only synthesis with a production caller (V19). (G2) Every JSON
parse failure records the byte offset and a window centred on it. (G3) The skill names a correct remedy
for `invalid`.

**Scoped out: the Tier-2 wing.** `SynthesizeWithTier2` has the same hole at HEAD (V19 probe: `proceed`),
but its only caller `RunQuorumWithEscalation` has **zero non-test callers**, and `RunAgenticReviewer` is
reached only through it (V19). The `tier2.go` fold and `agentic_caller.go` hook move to follow-up
`quorum:tier2-salvage-fold`, triggered by the first non-test caller of
`RunQuorumWithEscalation`; with `foldAbsent` in place it is a ~3-line call.

**Non-goals.** No retry or re-ask (see *Retry: dropped*). No JSON repair. No model escalation. No change
to seating, reserve order, roster, caps, `DefaultMaxCostUSD`, quota/ration thresholds or billing; those
belong to Mark under HD-2a and are untouched. No change to `ShouldEscalate`. No change to the reviewer
schema, prompts or `ValidateReviewResult`. **The fix is mechanical:** it changes how bytes the reviewer
already returned are read and recorded, never what is asked or who is asked.

## Design

### D1. Salvage = a strict token scan, never a repair

New `salvageVerdict(raw string) *SalvagedVerdict` in a new `salvage.go`. It runs `json.Decoder.Token()`
over `stripCodeFences(strings.TrimSpace(raw))`, the same bytes `ParseReviewResult` saw, and records
only what the reviewer emitted as **complete JSON tokens before the break**:

- The first token must be `{`. Otherwise it returns `nil`. Leading prose is not salvaged, because a quoted
  `"reject"` in prose may be the doc's own text.
- It tracks depth (`{`/`[` increment, `}`/`]` decrement) and records `verdict` and
  `strongest_objection` **only at depth 1**. A nested value is skipped, not a stop. A `verdict` at depth
  2 or deeper is never read, yet a top-level `verdict` emitted after a nested field is still salvaged
  (V20: `{"meta":{"k":1},"verdict":"reject",...` gives reject; `{"x":{"verdict":"reject"}}` gives nil).
  Stopping at the first nested opener, as round 0 proposed, would have dropped that reject.
- `verdict` counts only on an exact `"pass"`/`"reject"` string token. **Any** recovered `reject` wins,
  including duplicate keys, so the asymmetry favours blocking.
- A free-text field (`strongest_objection`) counts only when it is **confirmed**: the next key or the
  closing `}` must decode after it. Fixture F2 shows why. A bare quote ends a string early (`"the test
  greps "`), and the tokenizer returns that prefix as a complete string before failing (V8).
- The scan inserts, deletes and rewrites nothing. It cannot produce a field the reviewer did not emit.

```go
// SalvagedVerdict is what a strict token scan recovered from a response that
// ParseReviewResult rejected. It is audit plus a veto, never a vote.
type SalvagedVerdict struct {
	Verdict            Verdict `json:"verdict"`
	StrongestObjection string  `json:"strongest_objection,omitempty"` // confirmed tokens only
	Offset             int64   `json:"offset"`                         // -1 when the JSON was valid (gate violation)
}
```

### D2. Representation: still ABSENT, carrying a salvaged veto

The outcome keeps `Present=false`, `AbsentReason="invalid"` and `Result=nil`, and gains one additive field,
`Salvaged *SalvagedVerdict` (JSON tag `salvaged,omitempty`). Every existing reader of `.present` or
`.result` (`ShouldEscalate`, `tier1Split`, `presentCount`, `MarkdownBlock`, the skill's `jq` reads,
`cmd/ailang/design_review.go:131`) therefore sees exactly what it sees today. **No consumer can
mistake a salvaged verdict for a validated one.** `AbsentNote` gains
`SalvagedVerdict Verdict` (JSON tag `salvaged_verdict,omitempty`), so
`.synthesis.absent_reviewers[]` shows it.

Rules, folded through one helper `foldAbsent(&s, o, prefix)` that `synthesize` calls on its absent branch
(`quorum.go:159`). `SynthesizeWithTier2` calls `synthesize` first (`tier2.go:73`), so Tier-1 salvage
already reaches the two-tier verdict. Only its own Tier-2 absent branch (`tier2.go:82-84`) is the
follow-up:

| Salvaged verdict | Effect on synthesis | Counts toward `externalPresentCount` / `presentCount`? |
|---|---|---|
| `reject` | `blocked`. Adds objection `"<model> [salvaged from invalid response]: <objection>"`, or `"... strongest_objection unrecoverable, see reviewer error"` when unconfirmed | **No** |
| `pass` | **None.** Recorded for audit only | **No** |
| none (`nil`) | None (today's behaviour) | No |

**Why a salvaged pass never counts.** The contract makes a pass valid only with a stated closest
concern and a `catch` (`reviewer.go:151-163`), and a broken response cannot show those survived.
Counting it would let transport noise satisfy the zero-signal guard and the seat count. The costs are
asymmetric. Discarding a real pass costs at most one reserve seat or one re-quorum. Counting a wrong
pass costs a vacuous `proceed`, the failure this program has now patched twice. **Why a salvaged reject
blocks with no objection.** Under reject-by-default a reviewer's `reject` is the veto. A missing
objection makes it a weaker report, not a pass (`ValidateReviewResult`'s own wording, `reviewer.go:158`).
A salvaged reject is *not* a present seat. `RunSeatedQuorum` therefore still fills its seat from the
reserve, costing at most one extra seat at the existing cap. The synthesis is already `blocked`, so the
extra seat cannot change the outcome. The seating code is not touched.

The scan also runs on gate violations (valid JSON, e.g. `{"verdict":"reject","strongest_objection":""}`),
with `Offset=-1`. That is the same rule: a reviewer's reject is never dropped.

### D3. Diagnostics: the offset plus a centred window

One shared helper, `describeJSONError(err error, b []byte) string`, used by `ParseReviewResult`
(`reviewer.go:135`) and by the claude envelope error (`claude_reviewer.go:111`). When
`errors.As(err, &*json.SyntaxError)` holds, it renders:

`<err> at byte <Offset> of <len> (window [<a>:<b>]: %q)`, where `a = max(0, Offset-diagHalfWindow)`,
`b = min(len, Offset+diagHalfWindow)`, and `const diagHalfWindow = 120`. Both ends snap to UTF-8 rune
starts. Any other error keeps the current leading `%.200q`. The offset counts bytes of the *trimmed*
text, the same bytes `json.Unmarshal` saw. Because `ParseReviewResult` builds the message, all three
`out.Err = perr.Error()` sites, and the `%v perr` inside the `finish_reason=length` message
(`run.go:170`), inherit it with no per-site edits. The measured #941 break at byte 256 falls inside
`[136:376]` by construction.

### D4. Wiring: one helper, two production call sites

`recordInvalid(out *ReviewerOutcome, raw string)` sets `out.AbsentReason = ReasonInvalid` and
`out.Salvaged = salvageVerdict(raw)`. It replaces the bare `out.AbsentReason = ReasonInvalid` line at
the **top** of each `if perr != nil` block: `run.go:161` and `claude_reviewer.go:123` (passing
`env.Result`). Measured control flow at `run.go:159-175` (V18):

```
159  result, perr := ParseReviewResult(raw)
160  if perr != nil {
161      out.AbsentReason = ReasonInvalid          // ← recordInvalid(out, raw) here
169      if resp.FinishReason == "length" {        // truncation twig: Err text only
170          out.Err = "reviewer output truncated ... Parse error: %v"
171      } else { out.Err = perr.Error() }         // plain twig
174      return out
```

The length test sits **inside** the invalid block and only picks the `Err` text, so one call at 161
covers both twigs; T6b proves it (kills M8b). V18 shows a `finish_reason=length` response lands in the
`perr != nil` branch, and 0/53 `unreachable` rows carried a reject (V3). The claude **envelope** failure
(`claude_reviewer.go:109-112`) is CLI transport: D3 diagnostics, no salvage. `agentic_caller.go:203` is
Tier-2 only (follow-up).

### Retry: dropped from scope (justified)

- **Re-ask benefit is measured-zero for every row the strict scan provably recovers (the unexpected-end
  rows plus any syntax rows confirmed in V21); residual unconfirmed rows are bounded by the D5 remedy (one
  same-cap re-run when salvaged_verdict is absent) and the AC10 re-open trigger (salvaged == null and the
  absence changed the outcome), so retry stays out of scope.** V21 confirms all 7 (4 + 3), so the residual
  set is empty today. This holds only if the 4 `unexpected end` rows would still be
  `ReasonInvalid` at HEAD. All 4 predate the truncation guard `885725f06` (V3b). At HEAD a truncated
  reject is `ReasonInvalid` on the `finish_reason=length` twig (V18) and reaches `recordInvalid`. The
  condition holds. If D4 is ever moved below the twig split, this justification falls with it.
- **A second mechanism already exists.** `RunSeatedQuorum` refills an absent seat from the reserve
  (`seating.go:114-120`). A same-model re-ask would add another path for the same need, with its own
  cost accounting and its own interaction with the length-truncation branch.
- Re-open as a follow-up only if a quorum is recorded where `salvaged == null` and the absence changed
  the outcome. The `jq` read in AC10 finds it.

### D5. Skill remedy, in scope (fleet allowlist)

The remedy lives in `.claude/skills/mission-control/resources/gate-2-pick.md` and its byte-identical mirror
`.agents/skills/mission-control/resources/gate-2-pick.md`. Grep found no remedy text in design-doc-creator
(V12), so no out-of-scope follow-up is needed. Edits:

- Lines 185-189 split the rule by reason. `budget`: re-run alone with a raised cap (unchanged).
  `invalid` with `salvaged_verdict: reject`: the synthesis is already blocked; treat the objection as
  real and revise; no re-run. `invalid` with no salvage: **never** raise the cap. At most one re-run
  alone at the same cap; a second `invalid` is a durable absence, quoted "PROCEED at N-1, `<model>` absent (invalid)".
- Lines 233-238 fix stale text. The guard cited at `quorum.go:168` landed in #1767 (`a5f9af982`) and is
  now `quorum.go:183`, so *"until it lands"* becomes a landed note.
- Sync: `tools/launchd/sync-agents-skills.sh`, then `--check` exits 0.

## Files

- `internal/mission/quorum/salvage.go`: NEW. `SalvagedVerdict`, `salvageVerdict`, `recordInvalid`, `describeJSONError`, `diagHalfWindow` (~110 LOC)
- `internal/mission/quorum/salvage_test.go`: NEW. Fixtures F1-F3 and arms T1-T8 plus T6b (~240 LOC)
- `internal/mission/quorum/reviewer.go`: `ParseReviewResult` error via `describeJSONError` (~3 LOC)
- `internal/mission/quorum/reviewer_test.go`: `TestRunReviewerWith_TruncatedOutputFailsLoudly` gains one assertion that `Salvaged.Verdict=="reject"` (T6b)
- `internal/mission/quorum/run.go`: `Salvaged` field on `ReviewerOutcome`; `recordInvalid` at line 161, above the truncation twig (~8 LOC)
- `internal/mission/quorum/claude_reviewer.go`: `recordInvalid(out, env.Result)`; envelope error via `describeJSONError` (~3 LOC)
- `internal/mission/quorum/quorum.go`: `AbsentNote.SalvagedVerdict`; `foldAbsent` used by `synthesize` (~25 LOC)
- `internal/mission/quorum/artifact.go`: `MarkdownBlock` absent line adds `; salvaged REJECT — blocks`, or `; salvaged PASS — audit only (not counted)` when Salvaged.Verdict==pass (~6 LOC)
- `.claude/skills/mission-control/resources/gate-2-pick.md`: D5 remedy split and stale-guard note
- `.agents/skills/mission-control/resources/gate-2-pick.md`: byte-identical mirror via the sync script
- `changelogs/unreleased/2026-10-14-quorum-salvage-verdict.md`: `### Fixed` fragment
- `design_docs/planned/m-quorum-salvage-retry.md`: this doc; moves to `implemented/` on landing

Nothing outside the allowlist changes. `cmd/ailang/design_quorum.go` and `design_review.go` compile
unchanged (the fields are additive), and they pick up D3 through `o.Err`. `tier2.go` and
`agentic_caller.go` are **not** edited (Tier-2 follow-up); the latter still inherits D3.

## Fixtures (verified at HEAD with `go test`, V6)

```go
// Go raw strings, verbatim from the probe that produced V6.
const F1 = `{"verdict":"reject","strongest_objection":"the table row a \| b is unescaped","catch":"x","proposed_fix":""}`
const F2 = `{"verdict":"reject","strongest_objection":"asserts \"GOOS\" absent","catch":"the test greps "GOARCH" in ci.yml","proposed_fix":""}`
const F3 = `{"verdict":"reject","strongest_objection":"cut off mid-sent`
```

| ID | `json.Unmarshal` error | `SyntaxError.Offset` / len |
|---|---|---|
| F1 | `invalid character '\|' in string escape code` (pipe after a backslash) | 61 / 108 |
| F2 | `invalid character 'G' after object key:value pair` | 94 / 130 |
| F3 | `unexpected end of JSON input` | 59 / 59 |

`Decoder.Token` reports F3's break as `unexpected EOF`, worded differently from `Unmarshal`. D3 always
uses the `Unmarshal` error. Salvage ignores the wording.

## Acceptance Criteria (each arm names the mutant it kills)

1. **T1 offset in error.** For F1/F2/F3, `ParseReviewResult` errors contain `at byte 61`, `at byte 94`
   and `at byte 59`. *Kills M1: offset dropped from the message* (today's code fails this).
2. **T2 centred window.** F2 padded with a 300-byte confirmed objection puts the break past byte 300.
   The error window contains `GOARCH`. *Kills M2: reverting to the leading `%.200q`.*
3. **T3 salvage table.** F1 gives reject with empty objection. F2 gives reject with objection
   `asserts "GOOS" absent`. F3 gives reject with empty objection. Gate violation
   `{"verdict":"reject","strongest_objection":"","catch":""}` gives reject with `Offset=-1`. A broken pass
   gives pass. Prose `I would reject this` gives nil. Duplicate keys pass then reject give reject.
   Depth (V20): `{"x":{"verdict":"reject"}}` gives nil;
   `{"meta":{"k":1},"verdict":"reject","strongest_objection":"after nested","catch":"x` gives reject with
   objection `after nested`; `{"notes":["a",{"verdict":"reject"}],"verdict":"reject",...` gives reject
   with its top-level objection. *Kills M3: an unconfirmed free-text value accepted
   (`{"verdict":"reject","strongest_objection":"the test greps "GOARCH" x"}` must give objection `""`),
   M4: depth ignored (a depth-2 `verdict` read), M4b: scan stops at the first nested opener (the
   `meta` case gives nil), M5: prose scanned for `{`.* Key/value state (gemini-4-argon, round 2):
   `{"other":"verdict","reject":"pass","verdict":"pass","catch":"x` gives `pass`, not `reject`. *Kills M5b:
   object keys not distinguished from string values at depth 1.*
4. **T4 N-1 hole closed (G2).** `synthesize([present pass, absent invalid + Salvaged reject], ctrl pass)`
   gives `blocked`, with one objection containing `[salvaged from invalid response]`. *Kills M6:
   `foldAbsent` not called.* HEAD returns `proceed` (V7).
5. **T5 salvaged pass is not signal.** `synthesize([absent invalid + Salvaged pass], ctrl pass)` gives
   `blocked` with the zero-signal objection. *Kills M7: a salvaged pass counted toward
   `externalPresentCount`.*
6. **T6 production path (G3).** `RunSeatedQuorum` with a runner that calls `runReviewerWith` and a
   `stubCaller` returning F2 for seat B, and passes for the rest, gives `blocked`. *Kills M8:
   `recordInvalid` missing in `run.go`.*
   **T6b truncation twig (G1).** `runReviewerWith` with a `stubCaller` returning F3 and
   `ai.Response{FinishReason:"length"}` gives `Present==false`, `AbsentReason=="invalid"`, an `Err`
   containing `finish_reason=length`, and `Salvaged.Verdict=="reject"`. The same stub fed through
   `RunSeatedQuorum` for seat B gives `blocked`. `TestRunReviewerWith_TruncatedOutputFailsLoudly` also
   asserts `Salvaged` is set. *Kills M8b: `recordInvalid` placed only on the non-truncation twig.*
   **T6c gate-violation path (oc-glm-5-3 + oc-kimi-k3, round 2).** `runReviewerWith` with a stub returning
   the gate-violation body `{"verdict":"reject","strongest_objection":"","catch":"","proposed_fix":""}`
   yields `Present==false`, `AbsentReason=="invalid"`, `Salvaged.Verdict=="reject"`, `Salvaged.Offset==-1`.
   If the probe shows gate violations do not flow through `perr != nil`, extend `recordInvalid` to that
   branch or delete the `Offset=-1` semantics and D2's gate-violation paragraph. *Kills M8c: `recordInvalid`
   placed on a parse-only branch.*
7. **T7 claude seam wired.** `RunClaudeSubscriptionReviewer` with `claudeReviewCall` stubbed to
   `{"is_error":false,"result":<F2>}` gives `Salvaged.Verdict=="reject"` and `Present==false`; stubbed to
   a broken envelope `{"verdict":"reject","result":"x` (`claude_reviewer.go:109`) gives `Salvaged==nil`
   (envelope, not verdict). *Kills M10: claude site not
   wired; M10b: envelope error salvaged.* No Tier-2 arm (scoped out, V19).
8. **T8 backward compatibility.** Marshalling a `budget` absence and an `AbsentNote` without salvage emits
   no `salvaged` or `salvaged_verdict` key. A HEAD-shaped artifact JSON unmarshals into the new structs
   unchanged. *Kills M12: `omitempty` dropped.* `MarkdownBlock` renders a salvaged reject as
   `; salvaged REJECT — blocks` and a salvaged pass as `; salvaged PASS — audit only (not counted)`. The existing
   `TestRunReviewerWith_MalformedResponseIsInvalidAbsence`, `TestRunAgenticReviewer_MalformedOutputIsInvalidAbsence`
   and all `tier2_test.go` tests pass unedited. `TestRunReviewerWith_TruncatedOutputFailsLoudly` keeps
   its existing assertions and gains only the T6b one.
9. **Mutation proof.** For M1, M4b, M5b, M6, M8, M8b and M8c the implementer reverts the change in a scratchpad copy (never
   `git checkout <file>`), observes the named test fail, and restores it. The output goes in the sprint log.
10. **Gates.** `go test ./internal/mission/quorum/ ./cmd/ailang/ -count=1` passes, `make lint` is clean,
    and `tools/launchd/sync-agents-skills.sh --check` exits 0.
    `grep -c salvaged_verdict .claude/skills/mission-control/resources/gate-2-pick.md` is at least 1.
    **Landing precondition (oc-glm-5-3, round 2):** the Tier-2 follow-up `quorum:tier2-salvage-fold` is
    recorded before landing. The fleet cannot file tickets (`ailang mission ticket file` refuses the fleet
    mission: "the fleet mission works tickets; it does not file them", measured iteration 36), so it is
    recorded as a row in `design_docs/fleet-mission.md`'s queue, with this doc's V19 evidence and its trigger.

## Verification Log (run 2026-10-14 in this worktree at `42652c19e`; V3b, V18-V20 at `72d4a639d`, which differs only in this doc)

| # | Claim | Command | Observed |
|---|---|---|---|
| V1 | Version | `git describe --tags` | `v0.54.9-9-g42652c19e` |
| V2 | Leading 200-byte window; three sites keep only `perr.Error()` | `cat -n reviewer.go run.go agentic_caller.go claude_reviewer.go` | `reviewer.go:135` `%.200q`; `run.go:172`, `agentic_caller.go:204`, `claude_reviewer.go:124` `out.Err = perr.Error()` |
| V3 | 7/7 invalid hold a recoverable reject; control 0/90 | `jq` over `~/dev/sunholo-data/{ailang,ailang-world}/.ailang/state/mission-quorum/*.json` (164+157 files), regex `verdict\"\s*:\s*\"reject` on `.error` | ailang invalid 5/5 true, world invalid 2/2 true; budget 26, unreachable 53, auth 5, quota 3, unknown-model 3, all false |
| V3b | Which rows V3 counted, and when (artifacts carry no version field; dated by filename, keys `controller_in_session doc iso_ts reviewers synthesis`) | `jq -r '.reviewers[]? \| select(.absent_reason=="invalid") \| [input_filename, (.error\|test("verdict\\\\?\"\\s*:\\s*\\\\?\"reject")), (.error\|test("unexpected end")), (.error\|test("finish_reason=length"))]'` over both trees. The regex is escape-aware (matches the `%q`-escaped `\"`) | 7 rows, all `reject=true`, `finish_reason=length` false on all. `unexpected end` true on 4: `m-check-strict-fallbacks` ×2 (07-17T07:58Z, 08:03Z), `m-gemini-evaluator-diff-bridge` (07-16T23:03Z), `m-gemini-exec-project-plumbing` (07-16T17:44Z), all before `885725f06` (2026-07-17 10:55Z). The other 3 (08-27, 08-31, 09-17) are syntax errors other than `unexpected end` |
| V4 | The 3 vacuous proceeds had 0 external present (G1 shape); 0 on-disk instances of the G2 shape | same `jq`, printing `.synthesis.verdict` and the present count per invalid artifact | 3× `proceed` present_ext=0, ctrl=pass; 2× `blocked` (present_ext 1 and 2) |
| V5 | `SyntaxError.Error()` omits Offset | probe `strings.Contains(err.Error(), fmt.Sprint(off))` | `false` for F1, F2, F3 |
| V6 | Fixtures yield the three signatures | temp `zz_probe_test.go` (deleted after), `go test -run TestZZProbe` | F1 `'\|' in string escape code` off 61; F2 `'G' after object key:value pair` off 94; F3 `unexpected end of JSON input` off 59 |
| V7 | Guard covers G1, not G2/G3 | same probe calling `synthesize` and `RunSeatedQuorum` | G1 `blocked`; G2 `proceed`; G3 `proceed`, 4 reviewers, absent `[{B invalid}]` |
| V8 | Token scan behaviour on F1-F3 | same probe, `json.Decoder.Token` loop | F1 `{ verdict reject strongest_objection ERR`; F2 `... "asserts \"GOOS\" absent" catch "the test greps " ERR`; F3 `... strongest_objection ERR:unexpected EOF` |
| V9 | Guard is present-external-only | `sed -n 148,190p quorum.go`; `git log -- quorum.go` | `externalPresentCount` at 153/163/183; `a5f9af982 ... (#1767)` |
| V10 | Production quorum uses `RunSeatedQuorum`, refills absent seats | `grep -n RunSeatedQuorum cmd/ailang/*.go`; `sed -n 108,130p seating.go` | `design_quorum.go:89`; reserve loop `seating.go:114-120` |
| V11 | No existing salvage/token-scan helper; no outside-package reader of the new fields | `grep -rn "Token()\|salvag" internal/mission/quorum/`; `grep -rn "quorum\.\(ReviewerOutcome\|AbsentNote\|Synthesis\)" --include=*.go .` | none; only `cmd/ailang/design_review.go:130`, `design_quorum.go:188,193` (read `.Present`/`.Err` only) |
| V12 | Remedy text location; mirror identical; not in design-doc-creator | `grep -rn "raised cap" .claude/skills .agents/skills`; `cmp` on both gate-2-pick.md copies; `grep -rn "max-cost\|absent" .claude/skills/design-doc-creator/SKILL.md` | `gate-2-pick.md:186` in both trees; `IDENTICAL`; no remedy hit |
| V13 | Ticket `quorum:invalid-absent-on-quoted-literals` exists | `grep -rn` in `design_docs/` | `fleet-mission.md:223,235`; `fleet-mission-evidence/iteration14/open-ticket-signatures.json:63` |
| V14 | Ticket `quorum:reviewer-invalid-json-discarded` exists | controller: `ailang mission ticket open --json` (fleet ticket store; no on-disk hit) | present, filed by `attended`, last seen 2026-10-08 |
| V15 | #941 premise refuted (escaped literals, break at byte 256) | `gh issue view 941 --comments` | Refresh comment §1-§2: `ESCAPED \"GOOS\": True`, `Offset = 256` |
| V16 | Quorum tests green at HEAD | `go test ./internal/mission/quorum/ -count=1` | `ok ... 0.584s` |
| V17 | Whether the 4 `unexpected end` rows were provider truncation | n/a | Still **UNMEASURED**: no `finish_reason` was recorded before `885725f06`. The design no longer depends on it, because V18 shows both twigs reach `recordInvalid` |
| V18 | A HEAD truncation reaches the invalid block | `sed -n 150,180p internal/mission/quorum/run.go`; temp `zz_probe_test.go` (deleted): `runReviewerWith(stubCaller{raw:F3, resp:{FinishReason:"length"}})` | The `FinishReason=="length"` test is at :169, inside `if perr != nil` (:160), after `AbsentReason = ReasonInvalid` (:161). Probe: `present=false reason="invalid" err="reviewer output truncated at 16384 tokens (finish_reason=len…"` |
| V19 | Tier-2 at HEAD and every consumer | `sed -n 60,93p tier2.go`; `grep -rn 'SynthesizeWithTier2\|ShouldEscalate\|RunTier2(\|RunQuorumWithEscalation\|RunAgenticReviewer(' --include='*.go' .`; same probe: `SynthesizeWithTier2([present pass], ctrl pass, Tier2{[absent invalid, hidden reject]})` | `tier2.go:73` calls `synthesize`, `:82-84` appends the absent note with no fold. Probe: `verdict=proceed absent=[{a@codex invalid}]` (the hole exists). Non-test callers: `ShouldEscalate` ← `quorum.go:122` only; `SynthesizeWithTier2`/`RunTier2` ← `quorum.go:123,130` only; `RunAgenticReviewer` ← `tier2.go:54` only; `RunQuorumWithEscalation` ← **none** (rest is `*_test.go`). `ShouldEscalate` reads `q.Reviewers[].Present/.Result` (`escalate.go:82-116`) before any Tier-2 synthesis, so it never reads a verdict, `blocked` or otherwise |
| V20 | Depth-tracked token scan (gemini-4-argon refinement) | same probe, prototype scan that records keys only at depth 1 | `{"x":{"verdict":"reject"}}`→nil; `{"x":{…},"verdict":"pass","catch":"c"`→pass; `{"notes":["a",{"verdict":"reject"}],"verdict":"reject","strongest_objection":"top",…broken`→reject obj=`top`; `{"meta":{"k":1},"verdict":"reject","strongest_objection":"after nested","catch":"x`→reject obj=`after nested`; F2→reject obj=`asserts \"GOOS\" absent`; F3→reject obj=``; dup pass,reject→reject; prose→nil |
| V21 | The strict depth-tracked scan recovers a depth-1 reject from each of the 7 stored windows (glm round 2) | controller, iteration 36: `jq` the 7 `invalid` rows' `.error` from both trees, `strconv.Unquote` the `(raw: …)` window, run a `json.Decoder.Token` scan recording `verdict` only at depth 1 with key/value state | 7/7: post-trim leading byte `{`; classes 4× `unexpected end of JSON input`, 2× `'\|' in string escape code`, 1× `'G' after object key:value pair`; `scan_verdict="reject"` on all 7, every scan stopping on window EOF (no syntax break inside the window, so the verdict token precedes the break) |

## Quorum round 2 → bounded controller revision (narrow-refinement carve-out)

Round 2 (artifact `m-quorum-salvage-retry-2026-10-13T23-12-01Z.json`): oc-glm-5-3 reject, oc-kimi-k3 pass,
gemini-4-argon pass, gpt6-1-sol absent (OpenAI API 429, no credits). The one remaining block carries a
concrete reviewer-authored fix and does not dispute the direction (its own fix keeps retry out of scope).
The controller applied the reviewers' fixes verbatim: glm (1) V21 + restated premise, (2) the Retry bullet,
(3) T6c, (4) as a landing precondition, adapted only because the fleet cannot file tickets (AC10);
kimi (1) the `call.go` citation deleted in favour of V18, (2) T6c (merged with glm's), (3) the salvaged-PASS
rendering in artifact.go and T8; gemini the key/value T3 case (M5b).

## Quorum round 1 → revision

| Reviewer | Objection | Resolution |
|---|---|---|
| oc-glm-5-3 (reject) | The `finish_reason=length` twig (`run.go:170`) is unverified, so G1 and the Retry-dropped claim rest on an unmeasured premise | **Measured (V18).** The twig sits inside the invalid block, below line 161, and a HEAD probe records F3+length as `invalid`. D4 now places `recordInvalid` at 161, covering both twigs, so no second site is needed. Added T6b, which kills M8b. The Retry justification is now stated as conditional on V18, and the condition holds. V3b names the 7 rows and dates and confirms the regex is escape-aware |
| oc-kimi-k3 (reject) | The Tier-2 wing is asserted and never verified, and may contradict the `ShouldEscalate` non-goal | **Measured (V19), then scoped out.** At HEAD the Tier-2 hole is real (`proceed`), but `RunQuorumWithEscalation` has zero non-test callers. `ShouldEscalate` reads outcomes before any synthesis and never reads a verdict, so the non-goal stands. The `tier2.go` and `agentic_caller.go` edits and the Tier-2 T7 arm move to follow-up `quorum:tier2-salvage-fold`. G1 is restated for `synthesize`, the production synthesis |
| gemini-4-argon (pass) | Stopping the scan at a nested `{`/`[` drops a later top-level reject | **Adopted (V20).** D1 tracks depth and records only at depth 1. T3 adds the `meta` and `notes` cases, M4b, and the mutation proof |

## Related Documents

- `design_docs/planned/m-quorum-external-zero-signal-guard.md`: the guard this extends from all-absent to N-1.
- `design_docs/fleet-mission.md` P2 #12: the ticket row.
- `.claude/skills/mission-control/resources/gate-2-pick.md`: the absent-reviewer rule (D5).
