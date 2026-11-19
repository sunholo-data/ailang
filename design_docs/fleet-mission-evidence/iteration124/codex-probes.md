# Iteration 124 — codex probes (controller)

Host rig, 2026-11-19 ~08:48Z. Prompt: 'Reply with exactly the word OK and nothing else.' (-s read-only, --skip-git-repo-check, stdin </dev/null).

Two earlier runs were instrument failures, not verdicts: run 1 blocked reading stdin (rc 142, alarm); run 3 under zsh did not word-split, so --model was empty ('No models are available').

## Run 2 — PATH codex 0.169.0, default home (~/.codex)
### gpt-6.1-sol
```
Reading additional input from stdin...
OpenAI Codex v0.169.0
--------
workdir: /private/tmp/fleet_i124_probe
model: gpt-6.1-sol
provider: openai
approval: never
sandbox: read-only
reasoning effort: high
reasoning summaries: none
session id: 01a1f2fe-09f9-7d10-94be-351f8c852353
--------
user
Reply with exactly the word OK and nothing else.
codex
OK
6.116
OK
```
### gpt-6.2-sol
```
Reading additional input from stdin...
OpenAI Codex v0.169.0
--------
workdir: /private/tmp/fleet_i124_probe
model: gpt-6.2-sol
provider: openai
approval: never
sandbox: read-only
reasoning effort: high
reasoning summaries: none
session id: 01a1f2fe-1d38-7e30-9ed8-dfd9cc56960a
--------
user
Reply with exactly the word OK and nothing else.
codex
OK
6.000
OK
```
## Run 4 — CODEX_HOME=~/.codex-missions (the mission account), version isolation
### probe4_NEW_gpt-6.2-sol.out
```
OK
tokens used
112
OK
```
### probe4_OLD_gpt-6.1-sol.out
```
OK
tokens used
OK
```
### probe4_OLD_gpt-6.2-sol.out
```
ERROR: {"type":"error","status":400,"error":{"type":"invalid_request_error","message":"The 'gpt-6.2-sol' model is not supported when using Codex with a ChatGPT account.","code":"model_not_found"}}
ERROR: {"type":"error","status":400,"error":{"type":"invalid_request_error","message":"The 'gpt-6.2-sol' model is not supported when using Codex with a ChatGPT account.","code":"model_not_found"}}
```
