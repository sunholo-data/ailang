# Motoko Harness Analysis Log

An append-only log of motoko-harness eval failure analyses. The goal is iterative
self-improvement: each analysis characterizes *why* motoko underperforms vs other
coding harnesses (pi, opencode) on the same model/benchmarks, classifies the fix
lever, and records what observability we have vs need. When we have ~3+ entries and
a repeatable method, codify into a `motoko-analyzer` skill.

## Analysis template (use for every entry)

```
### YYYY-MM-DD — <data source>
- Headline: motoko AILANG X% (+ per-lang); vs pi Y%, opencode Z%. Coverage: N/total.
- Failure taxonomy (count): <category> = n ...
- Root cause: <the dominant failure + hypothesis>
- Lever classification (pick): HARNESS-ERROR | AILANG-INTEGRATION | PROMPT |
  AILANG-LANGUAGE | NEW-PACKAGE-EXTENSION | MODEL-CAPABILITY
- Observability: HAVE <…> | MISSING <…>
- Next action: <…>
- Prior-action status: <closes which earlier item>
```

**Lever definitions:**
- **HARNESS-ERROR** — motoko/eval plumbing bug (PATH, env, step budget, crash). Fix the harness.
- **AILANG-INTEGRATION** — the AILANG↔model glue (e.g. `internal/ai/ollama` tool-calling, provider routing). Fix AILANG's integration code.
- **PROMPT** — motoko's system/agent prompt doesn't elicit the right behaviour (e.g. doesn't compel tool use on small models). Fix the prompt (a `motoko_ext_*` package or SYSTEM.md).
- **AILANG-LANGUAGE** — the model writes plausible code but AILANG's compiler/semantics reject it where a fix to the *language* (or docs) would help. Fix AILANG.
- **NEW-PACKAGE-EXTENSION** — a missing AILANG-aware capability (docs, μRAG, verification) the agent needs. Build a `motoko_ext_*`.
- **MODEL-CAPABILITY** — the model just can't (logic errors on correct-compiling code). Not fixable by us short of a better model.

---

### 2026-06-17 — overnight OS rotation (os-rolling, qwen3.6, 3 trials)

- **Headline:** motoko AILANG **26%** (Py 70% / JS 89% / Go 79%) vs **pi 96%** / **opencode 79%** on AILANG. Coverage: motoko ~6/39 benchmarks (late start — failing until the ~22:20 PATH/key fix), pi/opencode ~32/39. Numbers are early/partial but directionally clear.
- **Failure taxonomy** (14 AILANG failures / 19 runs):
  - `api_error` **12** — of which **10 = "non-agentic result: 1 turn, 0 tool calls, 0 code"**, 1 = `step budget exhausted`, 1 = `motoko CLI not found` (pre-fix residual).
  - `logic_error` **2** (model wrote compiling-but-wrong AILANG).
- **Root cause:** motoko's ollama tool-calling is **unreliable** — qwen3.6 frequently produces **0 tool calls** (one prose turn, no `WriteFile`/`RunTests`), so no solution is written (`code_len=0`, `turns=None`). This is **language-agnostic** (motoko is below pi/opencode on Python/JS/Go too), and motoko fails benchmarks pi/opencode pass (symbolic_diff, type_unify, recursion_fibonacci) → it's the *harness*, not the language or benchmark difficulty.
- **Lever classification:** **AILANG-INTEGRATION (primary)** + **PROMPT (secondary)**. NOT AILANG-language, NOT a new extension, NOT (mostly) model-capability. opencode/pi don't hit this because they use their own battle-tested ollama tool-calling; motoko routes through AILANG's `internal/ai/ollama` (tool-calling only enabled this session) which qwen3.6 doesn't reliably trigger.
- **Observability: HAVE** — per-run `error_category`, `stderr` (carries the "0 tool calls" signature), `code`, `compile_ok/runtime_ok/stdout_ok`, cross-harness pass rates. **MISSING** — for *failing* runs, `agent_turns`/`finish_reason` are null and `code` is empty; we do **not** retain the motoko **session JSONL** (the raw model responses), so we can't yet see *why* tool calls are 0 (qwen emitting Hermes/XML `<function>` tool blocks the native parser misses? vs pure prose? vs tools not advertised?). **This is the #1 tooling gap to close before we can fix it.**
- **Next action:**
  1. **Capture observability** — retain the motoko `session_*.jsonl` (or at least the raw first model response) on eval failure, so we can inspect qwen's actual tool-call output.
  2. From that, root-cause the 0-tool-calls: (a) malformed/Hermes-XML tool calls → add tolerant parsing in `internal/ai/ollama/step.go`; (b) tools not advertised / prompt too weak → strengthen tool advertisement + motoko's prompt to compel tool use on small models.
  3. Re-measure on the rotation; expect AILANG to jump if 0-tool-calls is fixed (10/14 failures).
- **Prior-action status:** closes the "is the integration working?" question (it runs, but tool-calling is unreliable) and the PATH/key harness fix (done; 1 residual CLI-not-found failure pre-fix). Supersedes the assumption that motoko's low AILANG = model can't write AILANG (it mostly never gets to write any).

#### Root cause CONFIRMED via pi's source (the reference harness)
pi = `@mariozechner/pi-coding-agent` → `@mariozechner/pi-ai`. pi-ai has **no native
ollama provider** — it drives local ollama through the **OpenAI-compatible
`/v1/chat/completions`** endpoint (`openai-completions` provider, baseURL
`localhost:11434/v1`), using OpenAI-style `tools`/`tool_calls`. qwen3.6 emits tool
calls reliably over `/v1` (ollama's compat layer normalizes the model's tool-call
format) → pi 96%.

motoko routes through AILANG `internal/ai/ollama`, which uses ollama's **native
`/api/chat`** Tools API (`github.com/ollama/ollama/api`). qwen3.6 does not reliably
emit *native* tool calls over `/api/chat` → 0 tool calls → 26%. (Ironic: motoko's
own `.motoko/config/ollama/config.json` already sets `openai_base_url:
localhost:11434/v1`, but the AILANG ollama provider ignores it and uses native.)

**Concrete fix (highest-value next work item):** make AILANG drive ollama tool-calling
over the OpenAI-compat `/v1/chat/completions` endpoint (like pi/opencode), instead of
native `/api/chat`. Either (a) add a `/v1` tool path to `internal/ai/ollama`, or
(b) route `ollama/…` models through AILANG's OpenAI provider with baseURL
`localhost:11434/v1`. This is the first design-doc→sprint→execute item for the mission.
