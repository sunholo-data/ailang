---
name: local-model-onboarding
description: Onboard a new on-device model to the Mac Studio eval rig end-to-end — hardware size/shape feasibility gate, OpenRouter quality pre-screen, ollama pull, opencode+pi cross-harness registration in models.yml, and continuous-rotation wiring. Use when the user wants to add/test/try a new local (Ollama) model on the rig, asks whether a model is the right size/shape for the hardware, or wants to evaluate a candidate before committing rig time.
---

# Local Model Onboarding (the rig)

End-to-end workflow for taking a **candidate model → qualified member of the
on-device eval rotation**. This ties together the pieces that already exist
(`model-manager` for models.yml/API access, `local-ollama-eval` for running on
the rig) and adds the two things that were tribal knowledge: the **hardware
size/shape gate** and the **opencode+pi cross-harness registration**.

The rig: **Mac Studio M4 Max · 128 GB unified · 40-core GPU · ~546 GB/s memory
bandwidth.** One box, shared by every eval job via a mkdir mutex (rig lock).

## Current State

- **ollama**: !'curl -s --max-time 3 http://localhost:11434/api/version >/dev/null 2>&1 && echo "✓ running" || echo "✗ not running — start it first"'
- **Unified memory**: !'echo "$(($(sysctl -n hw.memsize)/1024/1024/1024)) GB  ($(sysctl -n machdep.cpu.brand_string 2>/dev/null || echo Apple Silicon))"'
- **Models on disk**: !'curl -s http://localhost:11434/api/tags 2>/dev/null | python3 -c "import sys,json;[print(\"  \",m[\"name\"],\"%.1fGB\"%(m[\"size\"]/1e9)) for m in json.load(sys.stdin).get(\"models\",[])]" 2>/dev/null || echo "  (ollama down)"'
- **Currently resident (VRAM)**: !'curl -s http://localhost:11434/api/ps 2>/dev/null | python3 -c "import sys,json;d=json.load(sys.stdin);[print(\"  \",m[\"name\"],\"%.1fGB\"%(m.get(\"size_vram\",0)/1e9)) for m in d.get(\"models\",[])] or print(\"  (none loaded)\")" 2>/dev/null || echo "  (ollama down)"'
- **Rotation models**: !'grep -E "OS_FILLER_MODELS|MODELS=" tools/launchd/os-rotation-filler.sh 2>/dev/null | head -1'
- **Rig lock**: !'ls "$HOME/.ailang/state/rig.lock.d" >/dev/null 2>&1 && echo "HELD — a job is running; do heavy pulls/probes around it" || echo "free"'

> Use the state above. If ollama is down, start it before anything. If the rig
> lock is held, a scheduled job is running — `ollama pull` is fine (network/disk)
> but defer GPU probes, or acquire the lock (see `tools/launchd/rig-lock.sh`).

## The hardware envelope (read this first)

Token generation on Apple Silicon is **memory-bandwidth-bound, not
compute-bound**. Two consequences drive every decision in this skill:

1. **`--parallel 1` is the established default.** All three rig jobs
   (`nightly-eval.sh`, `nightly-lang-eval.sh`, `os-rotation-filler.sh`) hard-code
   it. The documented rationale (`tools/launchd/rig-lock.sh`) is operational: the
   box is a single GPU / bandwidth-bound, concurrent streams thrash the ~546 GB/s
   bus, and an Ollama model reload mid-run silently kills a stream — all of which
   bite hard in a *multi-model* rotation. **Evidence caveat (verify before
   changing):** there is **no recorded p=1-vs-p=2 head-to-head**. The only
   measured parallelism data is a p=2-vs-p=4 run on a *single small* model
   (gemma4:26b) that favored p=2 over p=4 — a different question. So treat p=1 as
   the safe operational default, not a benchmarked optimum. If you want to raise
   it, run the head-to-head first (smoke set, p=1 vs p=2, compare wall-clock,
   pass rate, and TTFT-timeout count) — don't just trust the old p=2 note in
   `local-ollama.md`.

2. **Prefer MoE with a small *active* parameter count.** Total params live
   cheaply in unified memory (quality); only the active params compute per token
   (speed). `qwen3.5:35b-a3b` = 35B total / **3B active** is the canonical shape.
   A *dense* model of the same size generates tokens far slower — avoid dense
   above ~14B for the rotation.

**Memory budget (single-stream, p=1):**

| | GB |
|---|---|
| Unified total | 128 |
| Usable by Metal (~75% wired default) | ~96 |
| Reserve (OS + ailang server + opencode/pi subprocs + KV headroom) | ~22 |
| **Safe ceiling for model resident** | **~74** |
| Comfortable (slack at long context) | ≤ ~50 |

Rule of thumb: **on-disk ≤ ~45 GB, total ≤ ~80B, active ≤ ~8B.** Run
`scripts/check_model_fit.sh` to get a verdict (estimate pre-pull, or measure a
pulled model's real resident VRAM).

## Workflow

### 1. Select a candidate (shape gate + generation check)

**Do not pick from memory — fetch live.** The assistant's training is months
stale and model names mislead. Use [`resources/model-watch.md`](resources/model-watch.md):
it lists the canonical sources to fetch (Ollama library, OpenRouter, HF trending,
maintained ranking boards, and our own OS leaderboard) and a dated candidate
shortlist. Refresh that file's table first, then choose from it. The source list
is user-curated — confirm ranking boards are still maintained (Aider was dropped
2026-06 as unmaintained).

Find a recent coding/agentic model that matches the shape rule.

**Verify the generation by RELEASE DATE, not the name** — Qwen naming is a trap:
`Qwen3-Coder-30B-A3B` is the *Qwen3.0* coder (2025) and is **older** than the
rig's incumbent `qwen3.5:35b-a3b` (Qwen 3.5, Feb 2026), despite the higher-looking
"Coder" label. As of mid-2026 the genuinely newer *same-shape* upgrade is
**Qwen 3.6 35B-A3B** (Apr 2026, MoE). `Qwen 3.6-27B` is newer but **dense** (no
A3B) → slower on this bandwidth-bound box. `Qwen 3.7-Max` is API-only (not
on-device). Always web-check the release date and confirm it actually beats the
incumbent on the axis you care about before pulling.

Skip the giant frontier-OS models (GLM-5.1, DeepSeek-V4-Pro, Kimi, MiniMax-m3) —
those stay as **OpenRouter** ceiling references; they don't fit on-device.

```bash
# Fit/shape verdict BEFORE spending rig time (estimate from the published size):
.claude/skills/local-model-onboarding/scripts/check_model_fit.sh --disk 24 --active 3
```

### 2. Quality pre-screen via OpenRouter (cheap, zero rig time)
If the model is also hosted on OpenRouter, screen its *quality* first so you only
pull winners. Use `model-manager` to add a temporary `or-<model>` entry and run a
small standard+agent pass on AILANG + the 4-lang set, then compare to incumbents.

```bash
.claude/skills/model-manager/scripts/test_model_access.sh <openrouter-id>     # access + pricing
.claude/skills/model-manager/scripts/find_model_info.sh  <openrouter-id>
# add or-<model> to models.yml (model-manager), then:
ailang eval-suite --models or-<model> --benchmarks <smoke set> --langs ailang,python --trials 3 --output /tmp/screen_<model>
```
If it doesn't clear the bar vs your current local models, stop here — no pull.

### 3. Pull to the rig + confirm it fits for real
```bash
ollama pull <ollama-tag>
.claude/skills/local-ollama-eval/scripts/check_ollama_model.sh <ollama-tag>     # loads + sanity
.claude/skills/local-model-onboarding/scripts/check_model_fit.sh <ollama-tag>   # REAL resident VRAM verdict
```
Must come back GREEN/YELLOW. RED → pick a smaller quant (mxfp8 → nvfp4/Q4) or a
lower-active variant.

### 4. Register the cross-harness pair in models.yml
Always register **two** entries — the same local model on **opencode** and on
**pi** — so the OS/Local leaderboard gets a real harness comparison. They must
share a `model_family` so the Explorer pairs them.

```yaml
  opencode-<id>:
    api_name: "<ollama-tag>"
    provider: "ollama"
    agent_cli: "opencode"
    agent_model_name: "<ollama-tag>"
    model_family: "<family>"          # SAME on both entries → cross-harness pairing
    env_var: ""
    budgets: { hard_timeout_secs: 600 }   # local hang protection (see below)
    pricing: { input_per_1k: 0.0, output_per_1k: 0.0 }

  pi-<id>:
    api_name: "<ollama-tag>"
    provider: "ollama"
    agent_cli: "pi"
    agent_model_name: "ollama/<ollama-tag>"   # pi prefixes provider
    model_family: "<family>"
    env_var: ""
    budgets: { hard_timeout_secs: 600 }
    pricing: { input_per_1k: 0.0, output_per_1k: 0.0 }
```

**pi prerequisite:** pi has no built-in ollama provider — it needs the model
listed in `~/.pi/agent/models.json` (canonical copy:
`tools/setup/pi-ollama-models.json`). Add the new tag's `id` there too.

**hard_timeout_secs: 600** — local models hang on some hard benchmarks; the
2400s default burns 40 min per hang. 600s is the standard local backstop.

Rebuild so the embedded registry updates: `make quick-install` (the launchd jobs
use the installed binary). Verify routing:
```bash
ailang eval-suite --agent --models opencode-<id>,pi-<id> --benchmarks fizzbuzz --langs ailang --dry-run
```

### 5. Smoke-validate both harnesses (under the rig lock)
```bash
source tools/launchd/rig-lock.sh && rig_lock_acquire wait
ailang eval-suite --agent --models opencode-<id>,pi-<id> \
  --benchmarks fizzbuzz,cli_args --langs ailang,python --trials 1 --parallel 1 --output /tmp/onboard_<id>
```
Confirm: passes, `cost 0`, `harness` = opencode/pi, resident VRAM under budget
(`ollama ps`). The lock releases on shell exit.

### 6. Add to the continuous rotation
Append the pair to the filler's model list — it round-robins benchmarks for all
listed models, serially at p=1:
```bash
# tools/launchd/os-rotation-filler.sh
OS_FILLER_MODELS="opencode-qwen3-5-35b-a3b-mxfp8,pi-qwen3-5-35b-a3b-mxfp8,opencode-<id>,pi-<id>"
```
The launchd plist execs the repo script directly — next fire picks it up, no
reinstall. Mind cycle wall-time: each added model multiplies runs/cycle; the rig
lock prevents overlap so cycles just lengthen.

### 7. Record the decision
Note the screen result, fit verdict, and rotation status in the model's
models.yml `notes:` and (if it's a keeper) the OS leaderboard story. Commit
models.yml + the filler change + `~/.pi/agent/models.json` mirror together.

## Gotchas
- **Never `--parallel >1`** on the rig (the whole point of this skill). Bandwidth.
- **Two entries or it's not comparable** — opencode + pi with a shared family.
- **pi needs the ollama provider block** in `~/.pi/agent/models.json`.
- **600s hard timeout** for local models, not the 2400s cloud default.
- **Blackout 04:00–07:00** — the filler skips it (nightly window). Don't schedule
  heavy onboarding probes there.
- **`make quick-install` after models.yml edits** — the registry is embedded; the
  launchd jobs run the installed binary, not the source tree.
- **OpenRouter screen is a quality filter only** — a model that scores well hosted
  may differ on-device due to quant; always re-validate after pull (step 5), and
  the opencode-vs-pi-vs-OR comparison in the Explorer surfaces that delta.

## Related
- `model-manager` — models.yml mechanics, API access, pricing (cloud + OR).
- `local-ollama-eval` — running/monitoring evals on the rig, OTEL observability.
- `tools/launchd/os-rotation-filler.sh` — the continuous rotation.
- `docs/docs/guides/evaluation/local-ollama.md` — human-facing rig reference.
