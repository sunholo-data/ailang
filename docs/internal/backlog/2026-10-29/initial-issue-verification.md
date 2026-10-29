# Initial attended issue backlog verification — 2026-10-29

Historical starting snapshot:66 then-open issues were checked against the exact source baselines named below. Subsequent fixes/merges/closures are recorded in the [sweep summary](summary.md); the historical table is intentionally preserved. No issue closures preceded user approval.

# Language/effects/testing backlog verification

Verified against `origin/dev` `0796ec8e0a00e30e95e9cee83bfa39f344206ba3` (2026-10-29). Read issue bodies/comments via `gh`; searched issue-number and independent keyword history, planned/implemented docs, and relevant code paths. Reproductions use an archived-source binary and matching stdlib. No GitHub mutations or repository edits.

| # | title (≤60 chars) | verdict | evidence |
|---|---|---|---|
| #1980 | [stapledons_godot] Duplicate type import from two modules s… | KEEP-PLANNED | pipeline_module_imports.go:244 last-wins. Duplicate plans PR #1984 (MOD015) / #1985 (MOD016); choose one before executor launch. |
| #1961 | verify soundness: contract-only callee calls collapse to on… | KEEP-PLANNED | Repro: gap falsely VERIFIED on 0796ec8e0; callee_resolver.go:126 shared result. m-smt-well-sorted-preflight.md M0; quorum before planning. |
| #1960 | verify soundness: Int / and % encoded as Euclidean div/mod,… | KEEP-PLANNED | Repro parity(x)=x%2 falsely VERIFIED; smt/types.go:269–272 Euclidean div/mod. m-smt-well-sorted-preflight.md M0; no PR/task. |
| #1958 | Effect soundness: row unification drops labels when an open… | KEEP-PLANNED | Repro pure viaMap checks+prints leak. row_unification.go:97–121. v0_55_0/m-effect-row-open-closed-soundness.md F1–F3 ruled; no PR/task. |
| #1956 | [ailang-explainers] run --bytecode --verify-contracts silen… | NEEDS-DECISION | VM+verify-contracts prints -5 exit 0; evaluator errors. m-runtime-contract-parity.md F3 OPEN; quorum also required before planning. |
| #1955 | [ailang-explainers] runtime contracts: a bounded forall in … | NEEDS-DECISION | Repro test fails: core.Forall unsupported. m-runtime-contract-parity.md M3; F1–F4 OPEN; eval_expressions.go has no Forall arm. |
| #1954 | [ailang-explainers] runtime contracts: derived Eq on a reco… | NEEDS-DECISION | Repro runtime errors: derived Eq expects TaggedValue, gets RecordValue. m-runtime-contract-parity.md M2; F1–F4 OPEN. |
| #1953 | [ailang-explainers] verify: string <= / >= in a body is enc… | KEEP-PLANNED | smt/types.go string maps exist, body operator path not type-directed. m-smt-well-sorted-preflight.md M1; quorum gate/no PR. |
| #1950 | [ailang-explainers] verify: a contract over a sha256Hex cal… | KEEP-PLANNED | SMT helper name path emits unresolved callee. m-smt-well-sorted-preflight.md M2–M3; same root cluster #1953; no PR/task. |
| #1947 | [ailang-explainers] Consing onto an accumulator is still O(… | KEEP-PLANNED | builtins/list.go:100 copies whole tail per ::. m-list-cons-cells-decomposition.md LC-4/5 remain; prompt now documents cost, no fix. |
| #1921 | [stapledons_godot] ailang test: out-of-contract property sk… | NEEDS-DECISION | testing/result.go excludes out_of_contract from vacuous count. D-TRIAGE-25 OPEN threshold/zero-accepted/retain semantics; no implementation. |
| #1914 | [stapledons_godot] Parser: cons operator :: fails to parse … | KEEP-PLANNED | Repro check passes, test fails PAR_NO_PREFIX_PARSE(::). PR #1916 plan→executor/task-8b94f341; avoid duplicate implementation. |
| #1888 | [cli] DX: std/jsonschema — validate a Json against a JSON S… | KEEP-ORPHAN (P2) | No std/jsonschema. Open PR #1889 is triage-only std-jsonschema-validator.md; root design/execution absent, coordinator owns triage lane. |
| #1885 | [cli] DX: effectful lambda passed to std/list map — diagnos… | KEEP-PLANNED | Repro still leads with import collision on effect mismatch. ailang-core-triage/effectful-callback-misread-as-name-collision.md, design needed. |
| #1826 | Effect checker: mapE with an annotated effectful callback →… | KEEP-PLANNED | Concrete ε_annot only closed at direct call path. Open PR #1963 implements jointly with #1718; merge after #1790. |
| #1719 | IFC: aliasing or higher-order passing of a sink bypasses Ch… | KEEP-PLANNED | ifc_check.go:378 name-keyed sig; aliases bypass. m-ifc-function-value-sinks.md frozen 8e634480f; no implementation PR/task; precedes #1771. |
| #1718 | Effect leak: an effectful callback inside a list, tuple or … | KEEP-PLANNED | typechecker_latent_mask.go:23 top-level arrows only. PR #1963 implements R1–R9 nested coverage; do not duplicate. |
| #1714 | Inline tests: rows using an imported constructor pass check… | KEEP-ORPHAN (P2) | Repro imported Some row passes check then test undefined Some. m-inline-test-expected-values.md explicitly defers V15; dedicated design absent. |
| #1485 | Type checker accepts undeclared type names in signatures (f… | KEEP-PLANNED | Repro f(r: Nope) checks clean. ailang-core-triage/unknown-type-name-in-signature.md; no root design/implementation PR. |
| #1443 | Type checker accepts `pure func` with a declared effect row… | KEEP-PLANNED | Repro pure func !{IO} accepted. PR #1790 M1–M3 implements; M4 remains in sprint plan; no closure yet. |
| #1375 | verify: examples/runnable/contracts/hof_verify.ail no longe… | KEEP-PLANNED | Repro hof_verify.ail: incrementAll/positives still SKIPPED. ailang-core-triage/verify-hof-inline-behind-callee-sort-gate.md. |
| #1278 | [ailang-parse] Duplicate top-level function names in a modu… | KEEP-PLANNED | Repro duplicate pick checks clean, returns last (3). ailang-core-triage/duplicate-func-binding-silent.md; MOD007 only lets checked. |
| #1134 | [daneel] IFC label refinements are lost across module bound… | KEEP-PLANNED | IFC sig table remains module-local; PR #1771 implements summaries M1–M2 only, not import enforcement. m-ifc-cross-module-labels.md. |
| #1132 | [daneel] Daneel's concrete asks on IFC Phase 2 and label-aw… | KEEP-PLANNED | Roadmap asks not all delivered; cross-module #1771 partial, label-aware tracing still planned m-trace-label-aware.md; no full closure. |
| #959 | [dx] test --package should also run inline test blocks (two… | KEEP-PLANNED | cmd/ailang/test.go:224 discovers *_test.ail only. m-package-test-discovery.md; no open implementation PR/task. |
| #934 | [model-manager-session] Parser: one bad token emits a 342-e… | KEEP-PLANNED | parser_error.go appends and returns all errors, no cap. m-parser-error-cascade-suppression.md; keyword search finds design only. |
| #905 | [motoko_agent] Optional / defaultable record fields, so an … | KEEP-PLANNED | No field-default grammar/AST; v1_1_0/m-record-defaultable-fields.md remains planned; deliberate language feature, no PR/task. |
| #800 | generate-extension-registry emits [effects] max as the disp… | NEEDS-DECISION | ext_registry_gen.go:245 still Effects.Max; m-ext-registry-effect-row-ruling.md retains ceiling, conflicting reporter precise-row ask; ruling needed. |
| #715 | [world] v0.30.0 inline tests: a multi-argument row does not… | KEEP-PLANNED | Repro tuple row collected then no-pattern-matched. m-inline-test-multiarg-tuple-rows.md; expected-values work only fixed result side. |
| #713 | [world] ADT equality rejected inside a conjunction but acce… | KEEP-PLANNED | Eq diagnostic hint fixed (instances.go eqInstanceHint), but body-vs-contract instance asymmetry persists; m-adt-equality-position-independence.md. |
| #614 | ailang test: nested blocks discard all but the last express… | KEEP-PLANNED | Repro nested false;true passes. m-named-test-body-check-semantics.md scoped top-level; nested handling requires decision D-24. |
| #604 | ailang test: named test blocks check only the LAST expressi… | KEEP-PLANNED | Repro false;true passes. test_body_lowering.go:132 legacy fold discards nonfinal bool. v0_33_1/m-named-test-body-check-semantics.md. |
| #539 | Wrong-dialect keywords (case/switch/return/data/enum/struct… | KEEP-PLANNED | Repro return x+1 gives PAR020 add-semicolon. v1_0_0/m-dialect-keyword-diagnostics.md PARKED at quorum; reserved-keyword fix narrower. |

- No safe CLOSE-FIXED/OBSOLETE/DUPLICATE candidates among these 33 issues. Fix references in design commits are not implementation evidence.
- Highest-priority unowned, decision-ready fix: #1958, approved F1/F2/F3 on 2026-10-26. Prepare sprint for std/ai polymorphic callbacks plus symmetric row guards; no overlap with #1963 production files.
- #1980 has two competing plan PRs (#1984/#1985); merging both launches duplicate implementations and inconsistent diagnostics.
- #1960/#1961 remain false-VERIFIED P0s. Their design explicitly requires quorum before sprint planning. #1954/#1955/#1956 runtime parity has four OPEN freeze items and the same quorum gate.
- Reproduction artifacts: `language-repros/`, `language-repro-results.json`, `ailang-language-od` under this scratch directory. Original archived source is `language-od/`. #1443 corrected rerun accepts an imported println; #1375 rerun still yields two SKIPs.


# Platform issue triage — 2026-10-29

Verified source at origin/dev `0796ec8e0a00e30e95e9cee83bfa39f344206ba3`; current ref advanced during review to release `e863a1e3dc075dd4acc0fdb26a467f4258898588`. The intervening diff is only changelog consolidation and version files, so all code verdicts apply to the latter too. Read all 33 issue bodies/comments using `gh issue view --json ...comments`; snapshots are in `/tmp/ailang-backlog-20261029/platform/`. No GitHub mutations, code edits, acknowledgements, or dispatches performed.

| # | title | verdict | evidence |
|---|---|---|---|
| 1974 | Triage bookkeeping, week 2026-10-26 | KEEP-PLANNED | Current weekly directive thread; predecessor #1892. Latest comments preserve D-TRIAGE decisions. Never close the current thread. |
| 1965 | Confined docs search escapes through gh fallback | KEEP-ORPHAN (P1) | docs_search.go:144 creates GitHub backend without confinement guard; docsearch/github/github.go:37 spawns gh auth token; cache.go writes HOME. Deferred from #1964. |
| 1949 | Speech default still names failing preview TTS | KEEP-ORPHAN (P2) | internal/ai/speech.go:24 still defaults gemini-2.5-flash-preview-tts; std/ai.ail:170 and ai-effect.mdx:226 document it. No PR/task/design found. |
| 1945 | Fleet bookkeeping, week 2026-10-26 | KEEP-PLANNED | Current weekly thread, latest iteration 68; predecessor #1891. Current directives/17 OPEN fleet decisions remain. |
| 1902 | Add Faber Romanus to syntactic language camp | KEEP-ORPHAN (P3) | three-camps-comparison.md:22,34–39 lists X07/NERD/Magpie/Laze; no Faber/Romanus row. No PR/task found. |
| 1900 | Publish key from protected file | KEEP-ORPHAN (P1) | pkg_publish.go:24–47 has no key-file; :268 uses config.RegistryAPIKey only. Help/publishing docs recommend env export. No specific design/task found. |
| 1880 | ProcessError does not state pre-start guarantee | KEEP-PLANNED | process.go:208 cmd.Run merges start/wait; permission substring/catch-all SpawnFailed unchanged. Process-result design 050b39c14; plan PR #1883/task-5a2f63e8. |
| 1879 | WaitDelay after successful command is SpawnFailed | KEEP-PLANNED | process.go:182 WaitDelay; :208 cmd.Run; :225–257 no ErrWaitDelay branch. Same design and plan PR #1883; merging plan launches executor. |
| 1878 | Process timeout/output cap drop captured output | KEEP-PLANNED | process.go:218–228 returns count/duration only; limitedWriter discards bytes without killing. Same root-cause design and PR #1883/task-5a2f63e8. |
| 1798 | Ollama embed lacks Result and std wrappers | KEEP-PLANNED | _ollama_embed remains list[float] !{IO}; std/embedding only documents underscore calls. m-ollama-embed-result-ai-effect plan; PR #1867/task-9c11d207 owns implementation. |
| 1796 | Persistent shared stores and configured embedding | KEEP-PLANNED | runner/handlers.go:174,272 still in-memory with Future flags comments. m-cross-run-content-library plan; implementation PR #1862/task-7f6a2223 needs rebase. |
| 1773 | Publish smoke stages broken path-dependency lock | KEEP-PLANNED | publish_validator.go:59–79 copies then runs; :185–190 copies lock unchanged. No rewrite/relock. m-pkg-smoke-staged-lock.md planned, no sprint/PR/task in current inventories. |
| 1720 | Package git cache and direct HTTP bypass confinement | KEEP-PLANNED | pkg/gitcache.go:20–26 creates HOME cache; :95 clones; pkg_provenance.go:89 raw http.Get. Authority implementation PR #1964 owns root cause. |
| 1711 | Extra-repo credentials versus task inputs | CLOSE-FIXED | 939beb6fd/#1753 shipped cloud inputs; live site-rda declares inputs_allow daneel-memory; daneel#347 merged 37772e55. Explicit deploy+migration hold satisfied; H-6 separate. |
| 1631 | MCP and REST numeric argument binding diverge | KEEP-PLANNED | mcp.go:378/417 raw float64→CallPreserveFloats:456; REST separate binder. serveapi-mcp-int-args-bind-float64.md triage design recommendation, no implementation PR/task. |
| 1618 | mcp_files friction bundle | KEEP-PLANNED | Prompt units and test-string extraction fixed (f610b897b/b6c9f592e), but multiline and IFC/function-field work remains. PR #1789/task-002d8bc0 owns first slice; later slices separate. |
| 1536 | Cross-provider calls and catchable image errors | KEEP-PLANNED | ai_handlers.go:200 still rejects foreign provider; no callImageBase64Result. v0_51_2/m-ai-cross-provider-per-call-routing plan awaits execution approval, no current task/PR. |
| 1436 | Docs mission bookkeeping | KEEP-PLANNED | Still current docs directive thread; no successor. Latest record 17 parked on judge transport. D-TRIAGE-12 explicitly preserves issue. |
| 1307 | Task images and vision-aware plane reads | KEEP-PLANNED | Task inputs now exist, but pi_assets/ailang-exec.ts:304 always text; models.yml has no typed input modalities. task-input-attachments-and-image-read.md; partial completion only. |
| 1276 | Unknown messages-send flag becomes body | NEEDS-DECISION | messages_send.go:541–544 treats unknown flags as positional, no body-file. Proposed both reject and support body-file; overlaps systemic #534. |
| 1267 | Deep Research tool on Daneel executor | NEEDS-DECISION | m-daneel-deep-research.md freezes D1 Env admission/D2 credential delivery; restricted resolve.go still excludes Env. Do not duplicate generic Gemini work or skip freezes. |
| 1262 | Enable feedback-gate decision shadow | NEEDS-DECISION | 298bf970e shipped shadow off; feedbackgate/shadow.go checks off. Explicit TypeSafe/OpenRouter data-boundary ruling + ~200 message comparison still required. |
| 1259 | AI calls have no general deadline | KEEP-PLANNED | httpjson.go:71/openrouter/client.go:162 DefaultClient; handler.go:101/184/298/319 Background. m-ai-call-context.md unifies #578/#231; interface/interrupt freezes remain. |
| 1130 | CLI runtime has no Msg handler | KEEP-PLANNED | effects/msg.go ErrNoMsgHandler persists; runner has no Msg setup. Planned v0_38_6/m-cli-msg-handler.md; no current implementation PR/task. |
| 981 | Gate-0 can lose directives on crash | KEEP-PLANNED | mission_directives.sh:89/101 exclusive createdAt>since remains. m-gate0-watermark-crash-safety; PR #1962 owns fix and needs rebase. Fleet also tracks ticket; avoid parallel owner. |
| 903 | Codex subscription provider | KEEP-PLANNED | Phase1 fixed: config.go routes codex to ProviderCodex; factory.go:119 refuses; chatgpt timeout 10m. CLI-backed provider absent; m-codex-subscription-lane plan/report remains. |
| 662 | Deterministic WASM type-check ceiling | KEEP-PLANNED | d5831af9b shipped config/metrics; typecheck_budget.go still wall-clock 2s and observes steps only. wasm-typecheck-step-ceiling-662.md awaits corpus-derived ceiling. |
| 612 | Durable AST proxy-boundary analyzer | NEEDS-DECISION | No go/packages analyzer; proxy-boundary M2–M4 not completed. Prior plan-only auto-close was reopened. Explicit resume/drop decision outstanding. |
| 584 | Size/boundary gates miss top-level Go packages | KEEP-PLANNED | code-health.mk:167 still find internal cmd; check_boundaries.sh:45–58 fixed package arrays omit apiserver/serveapi. size-and-boundary-gates-scope.md triage recommendation. |
| 581 | Planner fence parse and remaining drift debt | KEEP-ORPHAN (P3) | 03d801f17/7f4a38240/#1977 fixed items1–2; sprint-plan-planner-lane-fences.md explicitly excludes item3 (drift detector), which remains. Retitle/split rather than close whole issue. |
| 578 | Cancellable AI handler context | KEEP-PLANNED | effects/ai.go AIHandler no context; handler.go:298 Background stream call. m-ai-call-context.md also covers #1259/#231. Consumer scope correction already recorded. |
| 534 | Systemic silent flags after positional argument | NEEDS-DECISION | test parseTestArgs is fixed, but other FlagSet commands still drop trailing flags. cli-flags-after-positional-systemic.md recommends shared helper with forwarding semantics. |
| 231 | Interrupt blocked AI step | KEEP-PLANNED | handler.go:319 uses Background and AIHandler no ctx. m-ai-call-context.md core plumbing + m-agent-step-cancellation LaneB; stdin cannot be polled during blocked evaluator. |

- Ready closure candidate: **#1711**. Live registry read and merged downstream migration independently satisfy the issue’s explicitly recorded hold. Keep H-6 in M-EXECUTOR-UID-SPLIT; closing #1711 does not resolve that audit.
- Small genuinely unowned items: **#1949** (replace preview default with verified GA model and update two docs), **#1902** (one language-table addition). These still require the repository’s exact-change trivial-fix gate; existing issue text is concrete enough for operator review.
- High priority unowned: **#1965** (confined docs fallback escape) and **#1900** (protected registry key file). Both require design approval rather than ad-hoc code; #1965 must coordinate with #1964’s authority seam.
- Best existing-design execution candidate: **#1773**, `design_docs/planned/v0_54_7/m-pkg-smoke-staged-lock.md`, no current owner/PR/task and no sprint JSON. Needs design approval→planner→execution gate before implementation. **#1536** already has a plan but needs refreshed review/approval and cross-provider credential care.
- Avoid duplicate work: #1878/#1879/#1880→PR1883, #1798→1867, #1796→1862, #1720→1964, #1618→1789, #981→1962; merges can start downstream coordinator work. #1259/#578/#231 are one shared context root cause, not three fixes. #581 remains open solely for item3 despite newly landed parser correction.
