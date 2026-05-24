# Running Local Ollama Models in the AILANG Eval Rig — Operations Runbook

Hard-earned lessons from the 2026-05-22/23 debugging sessions. Consolidates
findings that are currently scattered across commits + design docs + chat
history into one document you can hand to a new person (or your future
self) for onboarding a new model.

## TL;DR setup checklist (new model, e.g. qwen3-coder:30b)

```bash
# 1. Pull the base model
ollama pull qwen3-coder:30b

# 2. Build an AILANG-tuned variant (sampling fixes + output cap)
ollama create qwen3-coder:30b-ailang \
  -f tools/ollama/qwen3-coder-30b-ailang.modelfile      # adapt from gemma4-26b-ailang.modelfile

# 3. Register in opencode.jsonc with options.name="Ollama" (CRITICAL — see #971 below)
#    and per-model options block (temperature, max_tokens, etc.)
$EDITOR ~/.config/opencode/opencode.jsonc

# 4. Add models.yml entry routing to the variant
$EDITOR internal/eval_harness/models.yml

# 5. (Re)install ailang — auto-symlinks to ~/.local/bin for opencode-compat
make install

# 6. Sanity-check: opencode should see ailang AND your variant
opencode mcp list                                       # ailang-docs should be ✓ connected
opencode models | grep ollama
opencode run --format json --dangerously-skip-permissions \
  --model "ollama/qwen3-coder:30b-ailang" "Run: ailang --version. Output DONE."

# 7. Warm up the rig
.claude/skills/local-ollama-eval/scripts/warmup_rig.sh ollama/qwen3-coder:30b-ailang

# 8. Single-benchmark smoke
make eval-smoke MODELS=opencode-qwen3-coder-30b-ailang \
  EXTRA="-trials 1 -parallel 1 -agent-timeout 1200 -langs ailang \
         -benchmarks fizzbuzz -output /tmp/quick_check"

# 9. If it passes, full smoke tier with N=3 trials
make eval-smoke MODELS=opencode-qwen3-coder-30b-ailang \
  EXTRA="-trials 3 -parallel 1 -agent-timeout 1200 -langs ailang \
         -output eval_results/rotation/$(date +%Y-%m-%d)/$(date +%H%M)_<model>_smoke_n3"
```

## The mental model

```
    eval-suite (Go)
       │
       │  spawns ONE opencode subprocess per benchmark trial (parallel = N from -parallel flag)
       ▼
    opencode CLI (Node)
       │
       │  reads ~/.config/opencode/opencode.jsonc (provider config, options, MCP servers, bash permissions)
       │  reads ~/.config/opencode/agents/*.md (per-agent system prompts; we use default "build" agent)
       │  spawns tool subprocesses (bash, read, write, edit) — sanitized PATH, NOT inheriting ~/go/bin
       │  sends OpenAI-compat /v1/chat/completions to provider
       ▼
    ollama serve (Go) — process under launchd
       │
       │  loads model into VRAM (kept resident via OLLAMA_KEEP_ALIVE=-1)
       │  serves one request at a time (OLLAMA_NUM_PARALLEL=1)
       │  Modelfile params apply: top_k, repeat_penalty, min_p, num_predict
       ▼
    gemma4:26b-ailang (the variant we built)
```

Four configuration surfaces, each in a different place:

| Surface | What it controls | Lives at |
|---|---|---|
| Modelfile | ollama-native params (top_k, repeat_penalty, num_predict, min_p) | `tools/ollama/<model>-ailang.modelfile` |
| opencode.jsonc model options | OpenAI-compat params (temperature, top_p, frequency_penalty, max_tokens) + MCP wiring + bash perms | `~/.config/opencode/opencode.jsonc` |
| models.yml | harness routing (api_name, agent_cli, timeouts, budgets) | `internal/eval_harness/models.yml` |
| ollama plist env | server-wide config (NUM_PARALLEL, KV_CACHE_TYPE, FLASH_ATTENTION, KEEP_ALIVE) | `~/Library/LaunchAgents/dev.ollama.serve.plist` |

A mistake at any surface is silent in the others. The most painful traps below.

## Hard-won lessons — DO NOT re-discover these

### 1. opencode's child-shell PATH does NOT include `~/go/bin`

**Symptom**: bash tool fails with `command not found: ailang`. Model falls into `find / -name ailang`, `ls -R /`, or guessing `/usr/local/bin/ailang` / `/opt/homebrew/bin/ailang` paths. Sessions hang for many minutes per pathological command.

**Why**: opencode launches bash subprocesses with a sanitized PATH (`~/.local/bin:/usr/local/bin:/usr/bin:/bin:...:/opt/homebrew/bin`). Setting PATH in the executor env doesn't help — opencode overrides.

**Fix**: `make install` now auto-symlinks `~/.local/bin/ailang -> ~/go/bin/ailang`. If you suspect the symlink is missing, `ls -la ~/.local/bin/ailang` to check. If gone, just `make install` again.

### 2. opencode silently drops model options without `options.name`

**Symptom**: opencode.jsonc has `provider.ollama.models["foo"].options = {temperature: 0.5, ...}` but generation behaves like default sampling. Temperature, max_tokens, frequency_penalty all ignored.

**Why**: [sst/opencode#971](https://github.com/sst/opencode/issues/971) — `@ai-sdk/openai-compatible` provider requires `options.name` to be set OR all per-model options are dropped. Easy trap because the option block looks correct.

**Fix**: 
```jsonc
"provider": {
  "ollama": {
    "options": {
      "name": "Ollama",                              // <-- THIS is required
      "baseURL": "http://localhost:11434/v1"
    },
    ...
  }
}
```

**Verify**: run the same prompt twice through opencode at `temperature: 0.5`. If output is identical, options pass through. If wildly different, they're being dropped.

### 3. ollama's `/v1/chat/completions` only honors a subset of sampling params

OpenAI-compat surface passes: `temperature`, `top_p`, `frequency_penalty`, `presence_penalty`, `max_tokens`, `seed`, `stop`, `stream`.

Silently drops: `top_k`, `repeat_penalty`, `min_p`, `num_predict`, `repeat_last_n`.

**Fix**: put dropped params in a Modelfile and `ollama create` a variant. The Modelfile params apply at model-load time and aren't affected by request-level options.

### 4. gemma4:26b is prone to token-repetition collapse at default sampling

**Symptom**: a single LLM response generates 30k+ tokens of repeated garbage (e.g., `v_h_y_f_d_v_h_y_f_d_...`). Per-call wall time blows up to 9+ minutes.

**Why**: Google's default `temperature=1.0` with no repetition penalty is a known bug ([google-deepmind/gemma#622](https://github.com/google-deepmind/gemma/issues/622)) on the 26b MoE variant. Affects tool-argument generation particularly.

**Fix**: 
- Modelfile: `repeat_penalty 1.1`, `min_p 0.05`, `num_predict 4096` (hard cap)
- opencode.jsonc options: `temperature 0.5`, `frequency_penalty 0.3`, `max_tokens 4096`

Trade-off: lower temperature reduces peak coding ability for Gemma 4 (per Unsloth's HF #21 thread) but eliminates collapses. We trade peak for reliability.

### 5. Bash permission denylist works EVEN with `--dangerously-skip-permissions`

Deny rules always deny, regardless of the dangerously flag. So you can run unattended (no permission prompts) AND still block pathological commands.

```jsonc
"permission": {
  "bash": {
    "*": "allow",
    "find / *": "deny",
    "find /System*": "deny",
    "ls -R /*": "deny",            // trailing * catches pipelines like `ls -R / | grep X`
    "grep -r /*": "deny",
    "rm -rf /*": "deny",
    "sudo *": "deny"
  }
}
```

**Wildcard gotcha**: `"ls -R /": "deny"` matches the literal but NOT `ls -R / | grep ailang` (pipeline form). The trailing `*` in `"ls -R /*"` is required.

### 6. `-agent-parallel` was a DEAD flag (removed 2026-05-23)

The harness used to advertise `-agent-parallel` as a concurrency knob; it was decoration only. The real dispatch semaphore is `-parallel` (default 10). Mismatching these silently oversubscribes ollama's NUM_PARALLEL=1 queue, causing 15-minute TTFT timeouts.

The flag has been deleted as of [commit 98c0c408](commits/98c0c408). The status banner now prints `Dispatch parallelism: N (-parallel flag)` to unambiguously identify the correct knob.

### 7. opencode default agent has compaction baked in

opencode's "build" agent auto-compacts when context is "full" — meaning at the model's max context window (e.g. 262k for gemma4:26b). Our typical agentic sessions hit ~36k input tokens stably (verified across 56 prior sessions in chains DB), well below the threshold. Compaction does NOT save us when individual responses go runaway (the 32k repetition case). Output capping via Modelfile `num_predict` is the right control there.

### 8. AILANG MCP server vs CLI fallback

The remote MCP at `mcp.ailang.sunholo.com` exposes 14+ structured tools (`ailang.prompt_get`, `ailang.stdlib_search`, `ailang.example_for_concept`, etc.). With `options.name="Ollama"` set and the MCP block in opencode.jsonc, **opencode picks up MCP tools automatically and adds them to the agent's tool catalog**.

Verified live: with the v0.10.0-slim seed prompt that names the MCP tools, gemma4:26b uses them — observed `ailang-docs_stdlib_search` calls with queries like `"if"` and `"for"`. The agent is genuinely discovering syntax dynamically.

If MCP is unreachable, opencode logs the error and continues; agent falls back to shelling out to `ailang docs` / `ailang examples` CLI.

## Healthy-rig signatures (what "working" looks like)

Watch the ollama log (`/tmp/ollama-serve-launchd.log`) during a run:

| Pattern | Interpretation |
|---|---|
| Per-call durations 1s — 60s, most under 30s | ✓ Normal agentic loop |
| Occasional 60-120s call (warm cache miss after gap) | ✓ Normal — model loading or large prefill |
| Sub-second call (`1.7s`, `2.3s`) | ✓ Best — cache hit, no marginal prefill |
| Same 200 status, slowly-growing durations (5s → 10s → 30s → 5min) | ✗ Queue oversubscription — check `-parallel` |
| 200 status with `15m0s` duration | ✗ TTFT timeout — model didn't start producing for 15min |
| 500 status | ✗ Real error — check ollama server log for OOM or model unload |

Watch chains DB (`observatory.db`):

```bash
sqlite3 -readonly ~/.ailang/state/observatory.db <<'SQL'
SELECT
  duration_ms,
  json_extract(attributes, '$.opencode.input_tokens') AS in_tok,
  json_extract(attributes, '$.opencode.output_tokens') AS out_tok
FROM spans WHERE name = 'opencode.step'
ORDER BY start_time DESC LIMIT 10;
SQL
```

| Pattern | Interpretation |
|---|---|
| input_tokens stable around 30-40k across steps | ✓ opencode compaction working |
| output_tokens < 4096 (the Modelfile cap) per step | ✓ num_predict cap honored |
| output_tokens jumping to 30k+ in one step | ✗ Sampling collapse — check Modelfile + opencode.jsonc options |
| input_tokens growing each step (40k → 60k → 80k) | ⚠ History accumulating; eventual prefill ceiling |

## Pre-flight checklist before a long rotation

```bash
# 1. Rig precondition (verify_setup.sh wraps these)
curl -sf http://localhost:11434/api/tags >/dev/null            # ollama UP
curl -sf http://localhost:1957/health >/dev/null               # ailang server UP
launchctl list | grep dev.ailang.rig-watchdog                  # watchdog registered
ollama list | grep <your-model>                                # variant exists

# 2. Verify env config is sane
ollama show <your-model> --modelfile | grep -E "PARAMETER"      # sampling params baked in
opencode mcp list | grep "ailang-docs.*connected"               # MCP up
opencode models | grep <your-model>                             # opencode sees model

# 3. Warmup
.claude/skills/local-ollama-eval/scripts/warmup_rig.sh ollama/<your-model>

# 4. Single-trial sanity (1 benchmark, agent mode)
make eval-smoke MODELS=<your-model> \
  EXTRA="-trials 1 -parallel 1 -agent-timeout 1200 -langs ailang \
         -benchmarks fizzbuzz -output /tmp/preflight_check"
# Expect: opencode subprocess produces a solution.ail, ailang check / run ran, exits in <10 min

# 5. Full rotation
make eval-smoke MODELS=<your-model> \
  EXTRA="-trials 3 -parallel 1 -agent-timeout 1200 -langs ailang \
         -output eval_results/rotation/$(date +%Y-%m-%d)/$(date +%H%M)_<model>_smoke_n3"
```

## Adding a model — checklist of all surfaces

When you add a new model to the rig, you must touch all four configuration surfaces. Missing any one is silent failure.

| Surface | What | Where | Failure mode if missing |
|---|---|---|---|
| 1 | Modelfile (sampling) | `tools/ollama/<model>-ailang.modelfile` | runs with bad defaults, may collapse |
| 2 | Build variant | `ollama create <model>-ailang -f <modelfile>` | model not available |
| 3 | opencode provider model | `~/.config/opencode/opencode.jsonc` `provider.ollama.models["<model>-ailang"]` with options block | opencode doesn't know about it |
| 4 | models.yml entry | `internal/eval_harness/models.yml` with `agent_cli: "opencode"`, `agent_model_name: "ollama/<model>-ailang"` | harness can't route |

## When something goes wrong — diagnostic flowchart

**Sessions hang for many minutes**:
1. Check `~/.local/bin/ailang` symlink exists. If not: `make install`.
2. Check opencode bash logs for `find /`, `ls -R /`, or thrashing on PATH discovery. Update bash denylist.
3. Check `/tmp/ollama-serve-launchd.log` for individual call durations. If 15m timeouts, check `-parallel` value (must be 1 to match server NUM_PARALLEL=1).

**Sessions complete but produce 30k+ token outputs**:
1. Check Modelfile has `repeat_penalty` and `num_predict` baked in: `ollama show <model> --modelfile`.
2. Check opencode.jsonc has `options.name="Ollama"` set (sst/opencode#971).
3. Check `temperature` is not 1.0 (Google default) for models with known repetition issues.

**Pass rate is 0% on all benchmarks**:
1. Read solution.ail from a result file — does it look like syntactically-valid AILANG?
2. If no — it's a model capability ceiling. The prompt is being honored, the model just isn't good enough yet. Try a stronger model (qwen3-coder:30b, or wait for AILANG-finetuned variants).
3. If solution.ail looks reasonable but the harness reports compile_error — check `eval_mode` in the result JSON. If `standard` not `agent`, the harness ran in standard mode (probably forgot `-agent` flag — should be auto-injected by `make eval-smoke` per [eval.mk fix](commits/07cdcbbd)).

**MCP server unreachable**:
1. `curl -sf https://mcp.ailang.sunholo.com/mcp/ -X POST -H "Content-Type: application/json" -H "Accept: application/json, text/event-stream" -d '{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"probe","version":"0"}}}'`
2. If down, agent falls back to CLI tools (`ailang docs`, `ailang examples`) — slightly worse but functional.
3. Cloud Run cold-start takes 1-5s; `warmup_rig.sh` pre-warms it.

**ollama VRAM is way bigger than expected**:
1. Check `OLLAMA_NUM_PARALLEL` env var — should be `1` for our rig. Higher values multiply KV-cache by N.
2. Check `OLLAMA_KV_CACHE_TYPE` — should be `q8_0` to halve KV memory footprint.
3. Check `OLLAMA_FLASH_ATTENTION` — required for q8 KV to actually take effect.

## Why we made each choice (one-liners)

- **`-parallel 1`** — single-GPU rig; ollama serializes anyway; oversubscription causes 15min TTFT
- **`OLLAMA_KEEP_ALIVE=-1`** — keeps the ~100k-token opencode framework prefix in KV cache across trials
- **`OLLAMA_NUM_PARALLEL=1`** — per-slot KV-cache pre-allocation at 110k context = 6 GB per slot; we want one
- **`OLLAMA_KV_CACHE_TYPE=q8_0`** — halves KV memory (~25 GB → ~12 GB) for ~5% gen slowdown
- **Modelfile `repeat_penalty=1.1`** — prevents repetition collapses; kept ≤1.1 because >1.1 hurts code (variable names legitimately repeat)
- **opencode `temperature=0.5`** — Gemma 4 26b MoE collapses at the Google default 1.0; 0.5 is the agentic-loop middle ground
- **`max_tokens=4096`** — hard cap per LLM response (matches Modelfile num_predict); bounds worst-case wall time at ~68s
- **`options.name="Ollama"`** — sst/opencode#971 trap; without this, all per-model options are silently dropped
- **`~/.local/bin/ailang` symlink** — opencode's child shell PATH excludes `~/go/bin`; symlink makes ailang visible to the bash tool
- **bash denylist `find /*`** — prevents pathological filesystem exploration even with `--dangerously-skip-permissions`
- **AILANG MCP wiring** — gives the agent first-class structured tools for syntax discovery, better than parsing prose CLI output

## Acknowledged limits we don't yet have a fix for

- **gemma4:26b doesn't converge on fizzbuzz** within a 5-min single-trial budget. Code is structurally close (recursive, uses map, has the right shape) but has small API errors each iteration. Model-capability ceiling. Try qwen3-coder:30b for direct comparison.
- **Per-turn latency variance is real**. A "cold" session with a fresh framework prefix prefill is ~30-90s; a "warm" session with cache hits is sub-second. The first benchmark of a rotation pays the cold cost.
- **opencode session-end is not always clean**. Sessions that hit token limits or agent iteration caps may not exit immediately. The `agent-timeout` wall-clock cap is the final safety net.

## Hardware fit analysis (added 2026-05-24)

The rig is a 128 GB Apple M4 Max Mac Studio. The single integrated 40-core GPU runs models serially (verified empirically; NUM_PARALLEL>1 thrashes). The constraint that matters for selecting a model is **memory bandwidth (~546 GB/s)**, not memory size.

### The decode-throughput formula

```
decode_tok_per_s ≈ memory_bandwidth / (active_params × bytes_per_param)
```

For Q4 quants (~0.5 bytes/param effective with K_M tuning):

| Model | Active params | Active Q4 size | Theoretical tok/s | Observed |
|---|---|---|---|---|
| gemma4:26b (dense) | 26B | 17 GB | 32 | 60-67 |
| qwen3-coder:30b (dense) | 30B | 18 GB | 30 | (expected) 50-60 |
| qwen3:72b (dense) | 72B | 36 GB | 15 | (expected) 25-35 |
| Mixtral 8x22B (MoE) | 39B active / 141B total | 20 GB active / 75 GB total | 27 | 45-55 |
| Qwen3 235B/22B MoE | 22B active / 235B total | 12 GB active / 118 GB total | 45 | 70-80 |

**Active params, not total**, drives decode speed. An MoE that fits in 128 GB but activates only 22B per token is **faster than a dense 30B**.

### Memory-budget axes (three things sharing 128 GB)

1. **Model weights** — `total_params × bytes/param` (Q4 ≈ 0.5)
2. **KV cache** — `ctx × layers × kv_heads × head_dim × 2 × bytes_per_value`
   - q8 KV cache (our config) ≈ 1 byte/value, halves vs FP16
   - GQA shrinks kv_heads vs full MHA (~4-8x smaller KV)
   - MLA (DeepSeek-style) is ~10x smaller still
3. **Activations + workspace** — typically 5-15 GB during inference

For your typical agentic session (~36k context, gemma4:26b GQA + q8 KV): ~17 GB weights + ~3.7 GB KV = ~21 GB total (matches observed).

At max 262k context on gemma4:26b: KV alone ~26 GB at q8 → total ~43 GB (still fits comfortably).

### Decision matrix when picking a new model

| Metric | What it tells you | Good fit if... |
|---|---|---|
| Active params × bytes/param | Decode bandwidth cost | < 25 GB → ≥30 tok/s |
| Total params × bytes/param | Memory footprint | < 90 GB safe; < 115 GB with wired-limit bump |
| Architecture (MHA / GQA / MLA) | KV cache density at agentic context lengths | GQA preferred, MLA ideal |
| Active / total ratio (MoE only) | Capacity per byte of bandwidth | Lower = more capability per tok/s |
| Max context length | Long-context headroom | ≥ 128k for opencode-style agentic work |
| Quantization quality (Q4_K_M / Q5_K_M / Q6_K) | Capability per byte | Q4_K_M is the standard sweet spot |
| Tool-following baseline | Does it actually call `ailang check`/`run`? | Run 1 benchmark via OpenRouter to baseline before downloading |

### Model classes ranked by fit

| Tier | Best candidates | Why |
|---|---|---|
| **Current sweet spot** | gemma4:26b, qwen3-coder:30b, glm-5:30b | 17-20 GB Q4, GQA, ~50-60 tok/s |
| **Bigger but viable** | qwen3:72b | 36 GB Q4, ~30 tok/s, GQA — 2x slower per trial |
| **MoE candidates** | Qwen3 MoE (22B active / 235B total), older Mixtral 8x22B | Use most of 128 GB, decode faster than 70B dense |
| **Long-context specialist** | DeepSeek-Coder-V3 family (MLA) | KV cache 10x smaller — opencode sessions much cheaper |

### The architecture worth watching: MoE + MLA

MoE (low active params → fast decode) combined with MLA (low KV per context token → fits long sessions cheaply) is the design that maximizes both axes on bandwidth-constrained hardware. DeepSeek's recent models, Qwen3 MoE family, and Apple's own MLX-tuned MoEs are in this space.

### Recommended workflow when adding a new model

1. **Cloud baseline first**: trial via OpenRouter (existing `opencode-or-*` entries in `models.yml`). Establishes a "best case" pass rate at cloud-grade sampling, no local-rig variables.
2. **Hardware fit check**: total + active params + KV/token cost vs the 128 GB budget. Skip if it won't fit (don't waste a download).
3. **Pull to ollama**: `ollama pull <tag>` — only after cloud baseline justifies it.
4. **Make AILANG variant**: `ollama create <tag>-ailang -f tools/ollama/...modelfile` with sampling tune (repeat_penalty 1.1, min_p 0.05, num_predict 4096, top_k 64).
5. **Sanity check**: single-fizzbuzz trial via the harness, confirm rig agrees with cloud baseline.
6. **Slot into leaderboard**: with M-EVAL-RATING-EFFICIENCY (planned), 8 trials at the rating band suffice to slot into ELO rankings.

This workflow turns "should we run this model?" into a $0.50-$2 OpenRouter API call before any local commitment.
