# Local Ollama Eval — Troubleshooting

Symptom → cause → fix table for the common issues seen during the M-EVAL-LOCAL-OLLAMA investigation (May 2026).

## Tests / Eval-suite

### "Tests pass" but eval-suite reports immediate failure with `dur=0s`

| Cause | Fix |
|---|---|
| Benchmark spec `timeout:` field is cloud-tuned (90–120s) | Already fixed: M-EVAL-LOCAL-OLLAMA budget-precedence patch makes model `budgets.hard_timeout_secs` take precedence. Check the model has `budgets.hard_timeout_secs: 2400` set |
| `agent-timeout` CLI flag is too tight | Use `-agent-timeout 2400` (40 min) for local models |

### "opencode produced no output within 8m0s (prefill timeout)"

| Cause | Fix |
|---|---|
| TTFT (time-to-first-token) budget too tight under `-parallel N` | Bump `opencode-gemma4-26b.ttft_timeout` in `models.yml`. p=2: 600s; p=4: 900s+ |

### "opencode idle for 3m0s mid-generation (no output)"

| Cause | Fix |
|---|---|
| Local thinking model has long internal reasoning phase with no streamed output | Bump `opencode-gemma4-26b.generation_timeout` (per-session idle ceiling). 1200s is the verified default |

### "non-agentic result: 1 turns, 0 tool calls"

| Cause | Fix |
|---|---|
| Model produced complete answer in one turn without using tools — opencode rejects as "not agentic" | No clean workaround. Re-run usually helps. Some models (gemma4 at low concurrency) one-shot more often. Higher `-parallel` reduces per-token throughput which sometimes encourages multi-turn iteration |

### Same benchmark passes once, fails next time (variance)

| Cause | Fix |
|---|---|
| Stochastic agent execution — same model produces different output across runs | This is fundamental, not a bug. For trustworthy assessment use N≥3 trials. Use `eval-analyzer` skill to look for patterns across multiple runs |

## Observability / Monitoring

### `observatory.db` stays at 0 spans after running eval

| Cause | Fix |
|---|---|
| `OTEL_EXPORTER_OTLP_ENDPOINT` env var not set | `export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:1957` in the shell that runs eval-suite |
| `ailang server` not running on port 1957 | `make services-start` or load the launchd plist |
| Server logs show `FOREIGN KEY constraint failed` for span inserts | Should not happen post-M-EVAL-LOCAL-OBSERVABILITY M1 fix. If it does, rebuild: `make quick-install && make services-stop && make services-start` |

### `ailang chains live` shows "(no spans yet)" for all stages

| Cause | Fix |
|---|---|
| Spans land but lack `chain_id`/`stage_id` resource attrs | Should not happen post-M-EVAL-LOCAL-OBSERVABILITY-FOLLOWUP. If it does, ensure your `ailang` binary is post-2026-05-22; rebuild with `make quick-install` |
| `chains live` SQL fails to parse TIMESTAMP | Pre-FOLLOWUP-M2 bug. Fixed in chains_live.go `lastSpanForStage`. Re-pull and rebuild |
| Spans really aren't being emitted (server not receiving) | Tail `~/.ailang/logs/server.log` for incoming OTLP requests |

### `ailang chains diagnose` flags "No session ID recorded"

| Cause | Fix |
|---|---|
| Eval-suite chain stages don't have session_id captured yet | Known limitation. Doesn't affect functionality — chains_view + chains_live both work without it. Deferred polish |

## Process / System

### `ollama runner` at <1% CPU for >5 min while session "should be running"

| Cause | Fix |
|---|---|
| Model genuinely stuck or has crashed | `pkill -f "ollama runner"`. Ollama will reload on next request. Check `~/.ollama/logs/` for crash dumps |
| Model unloaded due to inactivity (`OLLAMA_KEEP_ALIVE` expired) | `launchctl setenv OLLAMA_KEEP_ALIVE 24h` for longer retention |

### High wall time per benchmark (>20 min for fizzbuzz)

| Cause | Fix |
|---|---|
| Running too many `-parallel` slots | Reduce to `-parallel 2` (default) |
| Different model loaded — `OLLAMA_MAX_LOADED_MODELS=1` forced unload+reload | Ensure rotation runs only ONE model per time slot (avoid concurrent multi-model invocations) |
| Other workloads competing for GPU | `ps -eo pid,pcpu,command | grep -E "ollama|opencode"` to check |

### Memory pressure / system slowdown

| Cause | Fix |
|---|---|
| Too many loaded models or per-instance KV caches | `curl -s http://localhost:11434/api/ps` to see loaded models. `MAX_LOADED_MODELS=1` is the safe default |
| Disk fills up with eval_results/ | `du -sh eval_results/rotation/` periodically. Archive old days or add a retention cron |

## Configuration

### "ailang not found" on a fresh shell

| Cause | Fix |
|---|---|
| `$HOME/go/bin` not on PATH | Add to `.zshrc`: `export PATH="$HOME/go/bin:$PATH"` |

### "make eval-smoke" complains about `--agent mode requires explicit --benchmarks list`

| Cause | Fix |
|---|---|
| Safety guard — agent mode is expensive, must pass explicit list | Use `run_smoke.sh` from this skill, OR pass `EXTRA='-agent -benchmarks bench1,bench2 ...'` |

### `golangci-lint not found` when running `make lint`

| Cause | Fix |
|---|---|
| Lint tool not installed | `make install-lint` — installs the pinned version |

## When in doubt

1. Check the most recent server log: `tail -50 ~/.ailang/logs/server.log`
2. Check Ollama status: `curl -s http://localhost:11434/api/ps`
3. Check chain state: `ailang chains list --since 1h`
4. Run setup checker: `.claude/skills/local-ollama-eval/scripts/verify_setup.sh`
5. Read [docs/docs/guides/evaluation/local-ollama.md](../../../../docs/docs/guides/evaluation/local-ollama.md) — the canonical user-facing guide
