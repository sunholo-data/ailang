### Fixed — eval: a typed executor kill is no longer banked as api_error

When an executor failed, the harness passed only its error string to the categoriser and dropped
the structured `FinishReason`. pi reports a token WORK-gate kill as `thrash_aborted`, but those
rows were banked as `api_error` ("cause unknown"): 7 os-rotation rows by 2026-10-02. Failed and
diagnostic rows now carry `FinishReason` (`withProvenance`), and the agent-eval error path reads
it ahead of the string.

### Fixed — motoko executor refuses a profile the checkout does not have

motoko resolves an unknown `MOTOKO_CONFIG` profile to defaults silently. The `ollama_fmt`,
`ollama_docs` and `ollama_dp7` rows named profiles that `~/dev/mk-main` no longer carries, so a
new trial on them would have banked defaults under the treatment's name. The executor now
refuses before starting when `<repo>/.motoko/config/<profile>/config.json` is missing.

### Changed — only the single GPU serializes agent runs

`eval-suite --agent` forced `--parallel 1` on any motoko row because every motoko run used to
bind port 8080. Since 2026-09-28 each run gets its own `ENV_PORT`. Re-measured 2026-10-02: six
concurrent cloud motoko trials (`motoko-or-deepseek-v4-flash`) scored 6/6 with no collision
($0.06). Ollama-cloud motoko rows now keep their `--parallel`; local-GPU rows still clamp to 1.
