# Docs iteration17 external issue inventory

50 open issues enumerated (limit50; not a complete repository sweep). Known-present control #1089 matches the corpus; fresh negative control fired. Counts: charter / log / STATUS archive / dashboard. Non-doc issues belong to V1; this inventory does not declare them triaged or clean.

| Issue | Counts | Title |
|---|---|---|
| #1436 | 0 / 0 / 0 / 0 | Docs mission bookkeeping — week of 2026-09-28 |
| #1420 | 0 / 0 / 0 / 0 | [stapledons_godot] Bytecode compiler: nested cons pattern a :: b :: rest is evaluator-only (unbound variable) |
| #1419 | 0 / 0 / 0 / 0 | [stapledons_godot] VM/interpreter divergence: NaN > x is true in the interpreter |
| #1383 | 0 / 0 / 0 / 0 | ailang unpublish: --force after the package argument is silently ignored, then the prompt fails |
| #1380 | 0 / 0 / 0 / 0 | Fleet mission bookkeeping — week of 2026-09-28 |
| #1375 | 0 / 0 / 0 / 0 | verify: examples/runnable/contracts/hof_verify.ail no longer verifies — the callee-sort gate skips HOFs before inlining |
| #1374 | 0 / 0 / 0 / 0 | ailang iface --json prints Go type names (<*types.TRecord2>) in function signatures |
| #1328 | 0 / 0 / 0 / 0 | ailang test is 18-24x slower than v0.33 on motoko's session.ail (98s -> 1751s -> 2336s) |
| #1326 | 0 / 0 / 0 / 0 | Effect leak: a pure function can perform IO by passing an effectful function to any higher-order function (check passes, side effect runs) |
| #1307 | 0 / 0 / 0 / 0 | [daneel] Feature: task attachments + image-returning read so a plane agent can see images (Daneel design lane) |
| #1306 | 0 / 0 / 0 / 0 | mission rotate-log: registry resolved against CWD, and index regeneration drops orphan-iteration rows |
| #1305 | 0 / 0 / 0 / 0 | pkg quality --no-run reports the un-run smoke as PUB015 failed (exit 2) on a passing package |
| #1278 | 0 / 0 / 0 / 0 | [ailang-parse] Duplicate top-level function names in a module are accepted silently; error surfaces at an unrelated call site |
| #1276 | 0 / 0 / 0 / 0 | [ailang-parse] messages send: unknown --body-file flag silently becomes the message body (content lost, agents dispatched on it) |
| #1268 | 0 / 0 / 0 / 0 | [daneel] Bug: publish smoke gate runs in a temp copy — path dependencies vanish (PUB015) — Daneel use case |
| #1267 | 0 / 0 / 0 / 0 | [daneel] Feature: the executor holds sunholo/gemini_agents (Deep Research) as a tool — Daneel use case |
| #1262 | 0 / 0 / 0 / 0 | [user] Feedback gate -> sunholo/decisions (shadow first, with fallback) |
| #1261 | 0 / 0 / 0 / 0 | [sprint-executor] ailang test evaluates module code differently from ailang run (flat env re-injection): wrong-arity calls, Unit params, nil deref |
| #1259 | 0 / 0 / 0 / 0 | [sprint-executor] std/ai has no HTTP deadline; named-test bodies do not round-trip escaped strings / integral floats |
| #1164 | 0 / 0 / 0 / 0 | check_no_personal_email.sh: five of six exclusion clauses are substring matches — attacker-noreply@<personal domain> reads clean |
| #1160 | 0 / 0 / 0 / 0 | Gate 0: a second, no-authority read for the driver's own crash notices (measured on World, row 73) |
| #1134 | 0 / 0 / 0 / 0 | [daneel] IFC label refinements are lost across module boundaries — a library cannot enforce a taint sink |
| #1132 | 0 / 0 / 0 / 0 | [daneel] Daneel's concrete asks on IFC Phase 2 and label-aware tracing |
| #1130 | 0 / 0 / 0 / 0 | [daneel] std/cognition Msg effect has no CLI handler — messaging cannot be capability-gated |
| #1043 | 0 / 0 / 0 / 0 | mission_pi_run.sh wires neither containment extension the pi recipe mandates — pi roles run unfenced |
| #981 | 0 / 0 / 0 / 0 | Gate-0 directive watermark is advanced by fires that die before processing — two of Mark's answers were silently lost |
| #959 | 0 / 0 / 0 / 0 | [dx] test --package should also run inline test blocks (two test conventions) |
| #941 | 0 / 0 / 0 / 0 | design-quorum: a reviewer recorded absent for reason 'invalid' answered fine — its JSON failed to parse because the objection quoted the literals under discussion |
| #934 | 0 / 0 / 0 / 0 | [model-manager-session] Parser: one bad token emits a 342-error cascade, drowning the real error for model self-repair |
| #905 | 0 / 0 / 0 / 0 | [motoko_agent] Optional / defaultable record fields, so an extension ABI can grow additively |
| #904 | 0 / 0 / 0 / 0 | [motoko_agent] Same-named exported record types in two sibling packages collide; the error names the correct module but the other package's definition |
| #903 | 0 / 0 / 0 / 0 | AI effect cannot bill an OpenAI ChatGPT subscription — and the `codex` model prefix silently resolves to the metered OpenAI lane |
| #901 | 0 / 0 / 0 / 0 | [motoko_agent] No producer-visible way to classify non-underscore language builtins (`show`, `intToFloat`) — blocks a static effect classifier |
| #800 | 0 / 0 / 0 / 0 | generate-extension-registry emits [effects] max as the dispatch row, over-declaring and breaking consumers |
| #752 | 0 / 0 / 0 / 0 | [email-parse] IFC: Declassify is whole-body authority, and positive param labels aren't enforced |
| #715 | 0 / 0 / 0 / 0 | [world] v0.30.0 inline tests: a multi-argument row does not parse, and a tuple-valued input is COLLECTED then dies at runtime with "no pattern matched" |
| #713 | 0 / 0 / 0 / 0 | [world] ADT equality rejected inside a conjunction but accepted as a top-level postcondition — and the error's suggested fix (`import std/prelude`) does not parse (IMP012) |
| #680 | 0 / 0 / 0 / 0 | LIST_TAKE_AFTER_FLATMAP misses nested traps: only the outermost take-after-flatMap fires |
| #676 | 0 / 0 / 0 / 0 | [email-parse] Quadratic memory in hand-rolled list recursion — 6,400 conses = 2.6 GB (minimal repro, no DB) |
| #669 | 0 / 0 / 0 / 0 | `ailang test` reports FALSE FAILURES: a stdlib export that delegates to another same-module export cannot be destructured by pattern matching (works under `ailang run`) |
| #662 | 0 / 0 / 0 / 0 | [ailang-parse] WASM type-checker budget is wall-clock, making module loading hardware-dependent (2x slowdown breaks a working demo) |
| #651 | 0 / 0 / 0 / 0 | design-quorum: the zero-signal guard is vacuous whenever a controller verdict is supplied — 3 docs shipped on 'proceed' with 0 of 2 reviewers present |
| #636 | 0 / 0 / 0 / 0 | ailang publish --dry-run truncates all three digests to 68 bits, and no surface prints the full values |
| #624 | 0 / 0 / 0 / 0 | ailang test: top-level forall properties never evaluate — 'empty program' on the simplest property, synthesized-program parse error with function calls |
| #616 | 0 / 0 / 0 / 0 | Effect row variables parse but never unify — `! {e}` silently becomes a phantom concrete effect, with a blank error message |
| #615 | 0 / 0 / 0 / 0 | Agent-mode cost/provenance union path has zero test coverage (#545 landed at 63.4% coverage on new code) |
| #614 | 0 / 0 / 0 / 0 | ailang test: nested blocks discard all but the last expression — the #604 vacuous pass one level down |
| #612 | 0 / 0 / 0 / 0 | Replace the textual proxy-boundary completeness gate with a go/packages AST analyzer (D-6 Option A follow-up) |
| #604 | 0 / 0 / 0 / 0 | ailang test: named test blocks check only the LAST expression — earlier failing checks are discarded and the block reports 'All tests passed!' |
| #589 | 0 / 0 / 0 / 0 | [motoko_agent] ailang test: cluster harness nondeterministically fails a passing test (6/10) with 'record has no field' naming a field outside the call's dependency closure |
