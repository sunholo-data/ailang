---
title: OS-model leaderboard
sidebar_position: 1
---

# OS-model smoke leaderboard

Per-release pass-rate matrices for the **open-source / locally-hosted Ollama models** that
AILANG's eval rig runs continuously on dedicated hardware.

## Why this section exists

Pay-per-token cloud APIs make N-trial evaluation expensive — you usually see "one shot,
one model, one benchmark." Running on a 128 GB Apple-silicon rig with [opencode + local
Ollama](/docs/guides/evaluation/local-ollama) lets the AILANG project publish:

- **N≥3 trials per (model, benchmark)** so variance is visible, not hidden behind a single roll.
- **Trend deltas across releases** — same benchmark, same model, last release vs. this release.
- **Cross-model comparison at fixed snapshot points** — gemma4-26b vs. qwen3-coder-30b
  vs. devstral on the same prompts and rubric.

The numbers feed the language-design feedback loop: a benchmark that fails persistently across
N trials becomes a candidate for a stdlib/prompt/syntax fix, which is then re-measured against
the same (model, benchmark) at the next release.

## How a release page is generated

Each release page is auto-built by `ailang eval-publish`:

```bash
ailang eval-publish v0.23.0 \
  --rotation eval_results/rotation/2026-05-23 \
  --prev     eval_results/rotation/2026-04-15 \
  --prev-tag v0.22.0
```

The command walks the rotation directory for `summary.json` files (one per rotation slot,
produced by `make eval-smoke ... --trials N`), merges per `(benchmark, model, lang)`, and emits
the per-release page with optional trend-delta section vs. the previous release.

See [the evaluation guide](/docs/guides/evaluation/local-ollama) for how the rotation directories
get populated.

## Releases

Browse the per-release leaderboards from the sidebar to see how each AILANG release performs
across the open-model rotation.
