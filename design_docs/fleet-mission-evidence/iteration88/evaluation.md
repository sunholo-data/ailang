# Iteration 88 independent evaluation (sprint-evaluator, Sonnet 5.5) — PASS 86/100

| Row | Pts | Max |
|---|---|---|
| 1 Correctness | 31 | 35 |
| 2 Test quality | 22 | 25 |
| 3 Honesty | 16 | 20 |
| 4 Scope/hygiene | 9 | 10 |
| 5 Fitness for Mark | 8 | 10 |
| Total | 86 | 100 |

## Evidence (all under env -i, synthetic HOME, fake values; worktree restored: git status --porcelain empty)
- Suite 57/57 on /bin/bash 3.2 + /bin/zsh 5.9; shellcheck clean; make check-changelog ok.
- apply, both shells: pi-role keeps only OLLAMA; strips parsed OPENAI/CRLF/tab-indented/double-space `export  X`/`FOO=$(..)`/backtick keys; sentinel files never created (no eval); CRLF line `CRLFK=1\r` parsed as CRLFK; unknown lane ('a b', '*', '') -> strip all parsed + rc 64; readonly var -> rc 78 + floor still cleared others; empty inventory -> rc 0 kept=0 stripped=0; chmod 000 inventory -> rc 78; helper-name key (_mc_scope_name) cleared; works under `set -eu`; HOME unset -> bash rc 78 (zsh falls back to passwd HOME, a harmless shell difference).
- Host line (extracted, 1 physical line) in synthetic .zshenv between a fake secrets source and `unset ANTHROPIC_*`, real ZDOTDIR startup: `zsh -c`, `zsh -lc`, `zsh -i -c`: no marker -> everything stays, no file access; pi-role + good root -> BODY runs, only OLLAMA kept (OPENAI/GOOGLE/NEWKEY gone); pi-role + missing root, and bogus lane -> FAIL-CLOSED message, rc 78, BODY never printed. `exit 78` in a sourced .zshenv does stop the body under zsh. No values printed anywhere.

## Own mutations (15, on scratch copies; none from executor bank)
KILLED (9): pi-role OPENROUTER->OPENAI; unset without subshell probe (zsh unset-failure); floor drops GOOGLE_API_KEY; awk drops leading-whitespace; helper-name list drops _mc_scope_nkept (12 fails); host drops inventory fallback loop; unknown lane returns 0; names-status check disabled; host `exit 78`->`exit 0`.
SURVIVED (6), all analysed as equivalent/no-op: host fallback drops ANTHROPIC from the seven (inventory loop also covers it, and the shell exits anyway); host leaves `_ailang_scope_key` set (shell exits); bridge drops `|| return 78` on lib source (undefined function -> rc 127 still propagates); bridge drops MISSION_DRIVER_ROOT -n check (empty root -> /tools/... unreadable -> 78); awk `sub(/[= ].*/)` (regex needs name immediately before `=`); a `; :` no-op I added. No genuine survivor.
Suite hygiene: runs children in env dict with fresh HOME/ZDOTDIR/CODEX_HOME/BASH_ENV, writes only under mktemp; missing /bin/zsh -> bad() (fails, not passes).

## Findings
BLOCKING: none.
NON-BLOCKING:
1. Doc says "Every lane strips ... GOOGLE_API_KEY, GEMINI_AGENTS_API_KEY and ANTHROPIC_API_KEY". Library strips a name only if it appears in secrets.env (or on failure floor). Probe: ANTHROPIC_API_KEY exported but absent from inventory stays SET after apply (both shells). Covered in production only by the later driver/.zshenv unset and the secrets.env being the source; wording overclaims. Command: /tmp/j88/probe.sh.
2. Host line sources the bridge with `2>/dev/null`, so the documented names-only `lane= kept= stripped=` audit line is invisible on the zsh path; doc "reports one names-only stderr line" is true of the library, not of the host install.
3. Suite exercises the host line via `. startup.sh` inside `zsh -c`, not a real ZDOTDIR/.zshenv startup nor `-lc`/`-i`; I did that manually and it holds, but the suite does not pin it.
4. Cite drift: mission-control.sh:154 is the comment about the setup-token; the consuming `if` is :157. Other spot-checks (mission_quota_cmd.go:58,75; provider-quota.ts:35; call.go:89 / 120-128 region; claude_reviewer.go:69; gate-3-route.md:343; lane-probe.sh:153-156; billing guard 163-171) match at base. `~/.zshenv:14` claim is controller-measured and I did not (by safety rule) read it.
5. The host line's fallback clearing is dead code in effect (shell exits 78 immediately); harmless belt-and-braces.
Scope: 6 files, all fleet-writable; changelog valid; no claim that M2/M3 exist ("NOT YET BUILT"); bash 3.2 portable (ran under /bin/bash).

## Fitness for Mark
Good: allowlist table with evidence refs, host line exact and tested, risks list (retained keys, re-sourcing, native subagents inherit, mutable driver root), clear "M1 changes no production lane". Missing for the ruling: a before/after exposure table per lane (what a codex/claude tool shell sees today vs with A), cost/risk of option B stated beside it, and what Mark must verify on first install (rollback = delete one line).

## Strongest objection
The control's guarantee is conditional on three things that are not built or verifiable from this diff: the launch wiring setting the marker, MISSION_DRIVER_ROOT being pinned/immutable, and the host line preceding any re-source. The doc discloses all three, but "Every lane strips X" in the allowlist text reads stronger than the name-in-inventory semantics actually deliver.
