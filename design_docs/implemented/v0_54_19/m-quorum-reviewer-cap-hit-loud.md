# M-QUORUM-REVIEWER-CAP-HIT-LOUD — a reviewer that spends its whole output budget is a truncation, not "malformed JSON"

**Status**: Planned · **Ticket**: `quorum:oc-glm-5-3-output-cap-invalid` (stapledon, 2026-10-21) · **Lane**: mechanical, `internal/mission/**` only · **Size**: ~30 lines Go + 1 test file

## Problem (measured)

Two design-quorum rounds on `kerr-black-holes` banked seat `oc-glm-5-3` (provider `ollama`,
`max_output_tokens: 65536`, `default_thinking: always_on`) as `present=false`, `tokens_out=65536`,
`absent_reason=invalid`, error `reviewer returned non-JSON or malformed response: unexpected end of
JSON input at byte 0 of 0`. Re-measured here: 23 banked artifacts under `~/.ailang/state/mission-quorum{,-repos}`
seat `oc-glm-5-3`; exactly two have `tokens_out: 65536` (`kerr-black-holes-2026-10-21T04-39-59Z`,
`…T04-45-36Z`); the third run of the same doc (`…T22-46-13Z`) was present at 56,487.

`runReviewerWith` (`internal/mission/quorum/run.go:173`) already rewrites a parse failure into a
truncation error — but only when `resp.FinishReason == "length"`. The ollama legacy `Generate`
(`internal/ai/ollama/client.go:230`) builds its `*ai.Response` without a `FinishReason`; only
`step.go:484` and `streamstep.go:295` set one. So for ollama seats the branch is dead and the operator
has to spot the 65536 by eye. Fixing the provider is outside fleet authority (`internal/ai/**`), and a
provider fix would leave the next provider that omits the field silently in the same hole. The fix
belongs where the budget is known: the reviewer, which set `MaxTokens = mc.MaxOutputTokens` itself
(`call.go:84`, `call.go:131`).

## Premise verification (call-path trace)

The fix rests on one premise: for an ollama seat, the response reaches `runReviewerWith` with
`FinishReason == ""` and with `OutputTokens` reported by the daemon. If the premise were wrong, the
inferred disjunct would never fire. Round 1 rejected the doc because the premise was asserted
without a trace. Below is the trace. Every command is read-only and was run in this worktree on
2026-10-27. Only the relevant lines are quoted.

**1. The reviewer calls `Provider().Generate`, not `Step`/`StreamStep`.**

```text
$ grep -n 'Generate(' internal/mission/quorum/call.go
43:	resp, err := c.handler.Provider().Generate(context.Background(), &ai.Request{
```

`handlerCaller.CallJSON` (`call.go:39-55`) passes `MaxTokens: c.maxTokens` and
`ResponseFormat: "json"`, and its result is returned unchanged as the `*ai.Response` that
`runReviewerWith` reads. The provider comes from `factory.NewProvider(mc.Provider, …)` (`call.go:121`).
`NewProvider` returns `c.Provider` unwrapped (`factory.go:97-103`), and `Handler.Provider()` returns
the stored provider as-is (`handler.go:231-233`). No retry or normalising wrapper sits between the
reviewer and the client.

**2. Provider `"ollama"` resolves to a bare `*ollama.Client`.**

```text
$ sed -n 240,275p internal/ai/factory/factory.go
	case ai.ProviderOllama:
	...
		client, err := ollama.NewClient(copts...)
	...
		return &Client{Provider: client, Type: typ, Lane: LaneLocal}, nil
```

`oc-glm-5-3` is `provider: "ollama"`, `api_name: "glm-5.3:cloud"`, `default_thinking: "always_on"`,
`max_output_tokens: 65536` (`internal/modelreg/models.yml:2854-2861`).

**3. `Client.Generate` builds its response without a `FinishReason`.**

```text
$ sed -n 120,125p internal/ai/ollama/client.go
func (c *Client) Generate(ctx context.Context, req *ai.Request) (*ai.Response, error) {

$ sed -n 225,240p internal/ai/ollama/client.go
	out := &ai.Response{
		Text:   response.String(),
		Model:  req.Model,
		WallMS: wallMS,
		TTFTMS: ttft.Milliseconds(),
	}
	tally.apply(out)
	return out, nil
```

`tally.apply` writes only `InputTokens`, `OutputTokens` and `TotalTokens` (item 4). Nothing else in
`Generate` touches `out`, so `FinishReason` keeps its zero value `""`.

**4. `OutputTokens` is the daemon's own `eval_count` and is never estimated from text.**

```text
$ sed -n 30,60p internal/ai/ollama/tokens.go
func (t *tokenTally) observe(m ollamaapi.Metrics) {
	...
	if m.EvalCount > 0 {
		t.out = m.EvalCount
	}
}
...
func (t tokenTally) apply(r *ai.Response) {
	r.InputTokens = t.in
	r.OutputTokens = t.out
```

The type comment (`tokens.go:25-27`) says Ollama emits the counts only on the terminal chunk, and
`observe` keeps the last non-zero value. No code path derives a count from `response.String()`.
`tokens_out: 65536` on a 0-byte body is therefore the daemon reporting that it generated 65,536
tokens. It is not an estimate taken from the empty text.

**5. The budget sent on the wire is the reviewer's `MaxTokens`.**

```text
$ sed -n 158,165p internal/ai/ollama/client.go
	// Set max output tokens if specified
	if req.MaxTokens > 0 {
		options["num_predict"] = req.MaxTokens
```

`num_predict = 65536 = mc.MaxOutputTokens`, and the daemon reported `eval_count = 65536`. The
generation stopped at exactly the ceiling we set.

**6. Only the agentic Step paths set `FinishReason`, and the reviewer uses neither.**

```text
$ grep -rn 'FinishReason' internal/ai/ollama/*.go | grep -v _test
internal/ai/ollama/step.go:192:		"finish_reason": resp.FinishReason,
internal/ai/ollama/step.go:226://   - response carries ToolCalls + FinishReason="tool_calls" when the model
internal/ai/ollama/step.go:227://     decides to call a tool; otherwise FinishReason="stop".
internal/ai/ollama/step.go:484:		FinishReason: finish,
internal/ai/ollama/streamstep.go:295:			resp.FinishReason = "tool_calls"
```

Every hit is in `step.go` or `streamstep.go`. Those are `Client.Step`/`StreamStep`, the tool-calling
agent loop. Item 1 shows the reviewer never enters them. Round 1's question was whether the live path
might normalise `done_reason` to `"stop"`. It does not: the stream's `DoneReason` field
(`ollama@v0.40.9/api/types.go:1025`) is never read in `client.go`.

**7. The banked artifact has no `finish_reason` field at all.**

```text
$ jq '.reviewers[] | select(.model=="oc-glm-5-3")' ~/.ailang/state/mission-quorum-repos/sunholo-data/stapledons-godot/kerr-black-holes-2026-10-21T04-45-36Z.json
{
  "model": "oc-glm-5-3",
  "cost_usd": 0.31375020000000003,
  "tokens_in": 18137,
  "tokens_out": 65536,
  "present": false,
  "absent_reason": "invalid",
  "error": "reviewer returned non-JSON or malformed response: unexpected end of JSON input at byte 0 of 0 (window [0:0]: \"\")",
  "landed": null
}
```

`ReviewerOutcome` does not persist the reason, so the artifact cannot prove `""` directly. Items 1–6
prove it from the code: for this provider on this path, no assignment to the field exists.

**Why the body is 0 bytes at 65,536 tokens.** `Client.Generate` consumes the stream like this
(`client.go:195-203`):

```go
err := c.client.Generate(ctx, genReq, func(resp ollamaapi.GenerateResponse) error {
	...
	response.WriteString(resp.Response)
	tally.observe(resp.Metrics)
	return nil
})
```

It appends only `resp.Response`. `ollamaapi.GenerateResponse` has a separate
``Thinking string `json:"thinking,omitempty"` `` field (`api/types.go:1019`), and no code in
`client.go` reads it. `grep -n 'Think' internal/ai/ollama/client.go` returns nothing: the request sets
no `think` value and the callback drops `Thinking`. `oc-glm-5-3` is `default_thinking: always_on`. If
the daemon routes its reasoning into `thinking`, a run whose reasoning uses up all of `num_predict`
before the answer starts produces 65,536 counted tokens and 0 `response` bytes. That matches the
artifact exactly.

- **Measured:** the artifact shows eval_count 65536, `num_predict` 65536 and a 0-byte `Text`.
- **Code-proven:** `Client.Generate` discards `Thinking`.
- **Inferred, not measured on the wire:** that the daemon actually put the 65,536 tokens in
  `thinking` for this cloud model. No raw stream was captured.

The inference does not change the fix. The predicate depends only on the measured count reaching
the budget with the reason unreported.

### 8. Which identifier names the models.yml entry (added round 2)

```
$ sed -n 43,50p internal/mission/quorum/call.go
	resp, err := c.handler.Provider().Generate(context.Background(), &ai.Request{
		Model:          c.handler.Model(),
		SystemPrompt:   sysPrompt,
		UserPrompt:     userPrompt,
		MaxTokens:      c.maxTokens,
$ grep -n 'ai.NewHandler(provider' internal/mission/quorum/call.go
130:	handler := ai.NewHandler(provider, mc.APIName, ai.WithMaxTokens(maxTokens))
$ grep -n 'out := &ReviewerOutcome\|^func runReviewerWith' internal/mission/quorum/run.go
112:	out := &ReviewerOutcome{Model: modelID}
137:func runReviewerWith(caller JSONCaller, mc *eval_harness.ModelConfig, out *ReviewerOutcome, docPath, docBody string, maxCostUSD float64) *ReviewerOutcome {
```

So the wire carries `mc.APIName` (`glm-5.3:cloud`), while `runReviewerWith` receives the models.yml id
already bound into `out.Model` by `RunReviewer`. The banked artifact's `"model": "oc-glm-5-3"` (item 7)
is that same `out.Model`, serialised by `json:"model"` on `ReviewerOutcome` (`run.go:50`). The acceptance
tests pass the outcome in the same way (`&ReviewerOutcome{Model: "oc-glm-5-3"}`), so they exercise the
production identifier rather than a stubbed one.

## Decision 1 — the detection predicate

```go
func capHit(resp *ai.Response, budget int) (hit, inferred bool) {
	if resp.FinishReason == "length" {
		return true, false
	}
	hit := resp.FinishReason == "" && budget > 0 && resp.OutputTokens >= budget
	return hit, hit
}
```

`inferred` is never true without `hit` (round-2 fix from oc-glm-5-3, applied verbatim): a non-hit
returns `(false, false)` whatever the reason, so the Decision 3 table's inferred label can only appear
on the `N >= budget > 0` row.

- **Reported wins.** Every provider that reports a reason normalises truncation to `"length"`
  (gemini `MAX_TOKENS` at `gemini/generate.go:169`, anthropic `max_tokens` at `anthropic/step.go:429`,
  openai `MapChatFinishReason`). That branch is unchanged.
- **Inference only on silence.** A provider that reported `"stop"` (or anything non-empty) is believed:
  `"stop"` with `OutputTokens == cap` means the model finished on the last token, and a malformed body
  there is a real malformed body. Inference applies only when `FinishReason == ""`, i.e. the provider
  told us nothing.
- **`>=`, not `==`.** Some backends count one or two tokens past `num_predict`; a count at or over the
  ceiling we set cannot be a voluntary stop.
- **`budget > 0` guard.** `cheapModel()` in the tests and any hand-built `ModelConfig` carry
  `MaxOutputTokens == 0`; without the guard every empty-reason parse failure (`OutputTokens >= 0`) would
  be relabelled. Production cannot reach 0 (`reviewerMaxTokens` refuses it, `call.go:81`).
- **Parse first, predicate second.** The predicate runs only inside the existing `perr != nil` branch.
  A verdict that parses at exactly the cap stays `Present=true`: the cap is a reason to explain a
  failure, never a reason to discard a valid vote.

## Decision 2 — keep `AbsentReason = "invalid"`

No new reason. The outcome is still produced by `recordInvalid` (so salvage of a cut-off reject keeps
working), and only `Err` changes. Consumers checked:

- `internal/mission/quorum/salvage.go:95` — `recordInvalid` sets `ReasonInvalid` and runs salvage; a new reason would need a second setter or would lose salvage.
- `internal/mission/quorum/quorum.go:196` (`foldAbsent`) and `quorum.go:51` (`absent_reviewers`) — copy the reason verbatim; the salvaged-reject veto is keyed on `Salvaged`, not the reason.
- `internal/mission/quorum/tier2.go:83` — copies the reason verbatim.
- `internal/mission/quorum/artifact.go:157`, `artifact.go:172` — render the reason as text.
- `internal/mission/quorum/seating.go:143`, `escalate.go:83`, `escalate.go:113` — key on `Present` only.
- `cmd/ailang/design_review.go:132` — prints reason and `Err`; the new text shows up here with no change.
- `cmd/ailang/design_quorum.go:195`, `claude_reviewer.go:110`, `agentic_caller.go:203` — producers, untouched.
- `.claude/skills/mission-control/resources/gate-2-pick.md:275-287` (mirror `.agents/skills/…:279-281`) — the remedy table is keyed on `invalid`, and its rule for `invalid` without a salvaged verdict ("never raise the cap; at most one solo re-run at the same cap") is already the correct handling for a cap-hit. A new reason would fall through that table to no rule.
- `tools/launchd/lib/quorum-artifacts.sh`, `tools/launchd/test_quorum_artifacts.sh` — resolve artifact paths only; their `invalid` hits are namespace validation, not the reason field. `scripts/` — no artifact readers (grep empty, rc=1).
- Tests pinning the reason on a truncation: `reviewer_test.go:221`, `salvage_test.go:201` and `:229`.

A distinct reason would mean editing the skill, the tests and every future reader for no routing
difference. The distinction belongs in `Err`, which is what a human reads.

## Decision 3 — the error text

Every message has the form `reviewer output truncated at N tokens (<how>) — <limit>. Parse error: …`.
`<how>` says where the signal came from. `<limit>` says which ceiling was hit, and it claims the
declared budget was used up only when the count shows it was:

| Case | `<how>` | `<limit>` |
|---|---|---|
| reported, `budget > 0 && N >= budget` | `finish_reason=length, reported by provider` | `the response consumed the declared max_output_tokens (M) of models.yml entry "X"` |
| reported, `budget > 0 && N < budget` | same | `provider reported finish_reason=length at N tokens, below the declared max_output_tokens (M) of models.yml entry "X" — a provider-side or context-window limit` |
| reported, `budget == 0` | same | `provider reported finish_reason=length at N tokens; models.yml entry "X" declares no max_output_tokens — a provider-side or context-window limit` |
| inferred (always `N >= budget > 0`) | `finish_reason not reported; inferred from tokens_out N >= max_output_tokens M` | `the response consumed the declared max_output_tokens (M) of models.yml entry "X"` |

Examples:

- reported, consumed: `reviewer output truncated at 65536 tokens (finish_reason=length, reported by provider) — the response consumed the declared max_output_tokens (65536) of models.yml entry "oc-glm-5-3". Parse error: …`
- reported, below budget: `reviewer output truncated at 40000 tokens (finish_reason=length, reported by provider) — provider reported finish_reason=length at 40000 tokens, below the declared max_output_tokens (65536) of models.yml entry "oc-glm-5-3" — a provider-side or context-window limit. Parse error: …`
- inferred: `reviewer output truncated at 65536 tokens (finish_reason not reported; inferred from tokens_out 65536 >= max_output_tokens 65536) — the response consumed the declared max_output_tokens (65536) of models.yml entry "oc-glm-5-3". Parse error: …`

The `<limit>` choice is one small helper, `capLimitClause(n, budget int, model string) string`, in
`run.go`. The `budget == 0` row cannot occur in production, because `reviewerMaxTokens` refuses 0. It
is still the row that `cheapModel()` reaches, and that includes the existing truncation tests.

The entry is `out.Model`, which is the reviewer OUTCOME's field, not the provider response's: inside
`runReviewerWith`, `out` is the `*ReviewerOutcome` that `RunReviewer` built as
`out := &ReviewerOutcome{Model: modelID}` (`run.go:112`) from the models.yml id it was called with, and
nothing on the provider path writes it (see trace item 8). The provider-side identifier is the
request's `Model: c.handler.Model()` (`call.go:44`), i.e. `mc.APIName` (`glm-5.3:cloud` for this
seat), and the fix never reads it. Every reported row keeps the
substrings `truncated` and `finish_reason=length`. `reviewer_test.go:224` and `salvage_test.go:213`
assert exactly those two. Both tests use `cheapModel()` (budget 0) with `FinishReason: "length"`
(`OutputTokens` 16384 and 20), so they land on the `budget == 0` row and pass unmodified. They also
assert `AbsentReason == invalid` and a salvaged reject, which only `recordInvalid` controls, and this
change leaves `recordInvalid` alone.
The old suffix "raise it in models.yml" is dropped. Changing the budget is a routing/quota decision
(see Out of scope), so the error names the entry and does not prescribe a fix.

## Acceptance criteria — new `internal/mission/quorum/cap_hit_test.go`

All tests use `stubCaller` (`reviewer_test.go:15`) and a `cappedModel()` (= `cheapModel()` with
`MaxOutputTokens: 65536`), and call `runReviewerWith` with model id `oc-glm-5-3`.

| # | Input (`raw`, `FinishReason`, `OutputTokens`) | Asserts | Mutation it kills |
|---|---|---|---|
| a | `""`, `""`, 65536 | absent, `invalid`, `Salvaged==nil`, Err has `truncated`, `inferred`, `65536`, `"oc-glm-5-3"`, `max_output_tokens` | delete the inferred disjunct (back to today's code) |
| a2 | `""`, `""`, 65537 | Err has `truncated at 65537` | `>=` → `==` |
| b | cut-off reject, `"length"`, 65536 | Err has `finish_reason=length`, `reported by provider` and `consumed the declared max_output_tokens (65536)`; salvaged reject | delete the `"length"` disjunct, or hardcode the label to `inferred` |
| b2 | cut-off reject, `"length"`, 40000 (cap 65536) | Err has `finish_reason=length`, `below the declared max_output_tokens (65536)` and `provider-side or context-window limit`; Err does **not** contain `consumed`; salvaged reject | make `capLimitClause` always return the "consumed" text (drop its `n >= budget` check) |
| b3 | cut-off reject, `"length"`, 16384, cap 0 (`cheapModel()`) | Err has `declares no max_output_tokens`; contains neither `consumed` nor `below the declared` | delete the `budget > 0` conjunct in `capLimitClause`. 16384 >= 0, so the mutant prints "consumed the declared max_output_tokens (0)" |
| c | `{"verdict":`, `"stop"`, 65536 | Err == `ParseReviewResult` error, no `truncated` | delete the `FinishReason == ""` conjunct |
| d | `{"verdict":`, `""`, 65535 | Err has no `truncated` | delete the `OutputTokens >= budget` comparison |
| f | `{"verdict":`, `""`, 20, cap 0 (`cheapModel()`) | Err has no `truncated` | delete the `budget > 0` guard |
| e | valid pass JSON, `""`, 65536 | `Present`, `AbsentReason==""`, `Err==""`, `Result.Verdict==pass` | move the cap check ahead of `ParseReviewResult` |

The existing `TestRunReviewerWith_TruncatedOutputFailsLoudly` and `TestSalvageProduction` must
pass unmodified. Each mutation is to be applied to a scratch copy and the matching test watched
failing before it is reverted.

## Files

- internal/mission/quorum/run.go
- internal/mission/quorum/cap_hit_test.go
- changelogs/unreleased/2026-10-27-quorum-reviewer-cap-hit-loud.md

## Out of scope (policy, parks for Mark)

These are the three waste-reduction options from the ticket. Each changes a budget or a seating
decision, so each is a routing/quota decision row and not part of this fix:

- **Lower `max_output_tokens` for `oc-glm-5-3`.** This would touch `internal/modelreg/models.yml`, which the fleet may not edit. It also goes against the full-ability policy recorded on `reviewerMaxTokens` (`call.go:57-80`).
- **Add a reasoning budget for this seat.** This would touch `internal/mission/quorum/call.go` (`ai.Request` thinking fields) and possibly `internal/ai/ollama`. Restricting thinking is reserved to evals.
- **Stop re-seating a reviewer after one cap-hit in a fire.** This would touch `internal/mission/quorum/seating.go` and `gate-2-pick.md`. Who sits on the quorum is a seating policy, and the third kerr run shows a re-seat can succeed.

Also out of scope: setting `FinishReason` from ollama's `done_reason` in `internal/ai/ollama/client.go`
(no authority). If someone lands that later, the reported branch takes over with no change here.

## Verification

```bash
go test ./internal/mission/quorum/...
go vet ./internal/mission/quorum/...
grep -c 'FinishReason' internal/ai/ollama/client.go   # premise check (g): must print 0
grep -n 'Model:' internal/mission/quorum/call.go       # premise check (h): request Model is c.handler.Model() (api_name)
grep -n 'out := &ReviewerOutcome' internal/mission/quorum/run.go   # premise check (h): out.Model is the models.yml id
```

**(g) Premise check, not a test.** A stub cannot exercise the real ollama response shape without a
running daemon, so the unit tests stop at `runReviewerWith`. The premise from the call-path trace is
re-checked by hand when this is implemented: `grep -c 'FinishReason' internal/ai/ollama/client.go`
must print `0` (it printed `0` on 2026-10-27). If it prints a non-zero count, someone has mapped
`done_reason` onto the legacy path. In that case the reported branch handles ollama seats, the
inferred branch never fires for them, and nothing breaks. Record the count in the sprint log either
way, so the evaluator can see which branch production actually uses.

## Quorum verification log

**Round 1, 2026-10-27, BLOCKED.** Artifact:
`~/.ailang/state/mission-quorum/m-quorum-reviewer-cap-hit-loud-2026-10-27T08-12-52Z.json`.

- **oc-kimi-k3: reject.** Same objection as glm, plus one question: why was the body 0 bytes at
  65,536 tokens?
- **oc-glm-5-3: reject.** The premise that `FinishReason == ""` reaches `runReviewerWith` is
  asserted, not traced. `step.go:484` and `streamstep.go:295` do set it, and the doc does not show
  that `OutputTokens` is provider-reported.
- **gemini-4-argon: pass.** Non-blocking note: the reported-branch text "consumed the declared
  max_output_tokens" is false when the budget is 0 or `OutputTokens < budget`.
- **gpt6-1-sol: absent** (`unreachable`, OpenAI 429 "no credits remaining").

What this revision changed:

1. Added `## Premise verification (call-path trace)` with seven quoted command outputs. They show
   that the reviewer calls `Provider().Generate`, which resolves to a bare `*ollama.Client`. That
   `Generate` never assigns `FinishReason`, and `OutputTokens` is the daemon's `eval_count`. The
   only `FinishReason` writers are the Step paths, which the reviewer never calls.
2. Explained the 0-byte body: `Client.Generate` appends only `resp.Response` and drops `Thinking`.
   The claim that the daemon routed the reasoning tokens into `thinking` is marked as inferred.
3. Rewrote Decision 3 so that the "consumed" text appears only when `N >= budget > 0`. Added test
   rows b2 and b3, each with its killing mutation, and showed that the two existing truncation tests
   land on the `budget == 0` row and pass unmodified.
4. Added premise check (g) to Verification.

**Round 2, 2026-10-27, BLOCKED (1 reject).** Artifact:
`~/.ailang/state/mission-quorum/m-quorum-reviewer-cap-hit-loud-2026-10-27T08-18-19Z.json`.

- **oc-kimi-k3: reject.** The trace never showed which identifier lands in the error's model slot; it
  argued `out.Model` might be the provider's `req.Model` (`glm-5.3:cloud`).
- **oc-glm-5-3: pass**, with the same catch plus a `capHit` tail fix (`inferred` true on non-hits).
- **gemini-4-argon: pass.** Note: claude_reviewer.go and agentic_caller.go are not traced; they do not
  call `runReviewerWith` (they set their own outcomes, Decision 2's producer list), so the predicate does
  not reach them.
- **gpt6-1-sol: absent** (OpenAI 429).

Controller's bounded second revision under the mission-control narrow-refinement carve-out (every
remaining blocking objection carries a reviewer-authored `proposed_fix` and none disputes the
direction). Applied: kimi's trace request (quote the request literal's `Model` field and how the id
reaches `runReviewerWith`) as trace item 8 plus premise check (h); glm's alternative "if you keep
`out.Model`, add trace item 8 … and state where the artifact's `model` field is populated from",
verbatim; glm's `capHit` tail `hit := …; return hit, hit`, verbatim. NOT applied: kimi's replacement
sentence "`out.Model` is provider-side and, for ollama, carries the api_name", because trace item 8
refutes its premise (`out` is the `*ReviewerOutcome` from `run.go:112`, never written by a provider).
