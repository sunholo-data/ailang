# microrag Frontend Templates

Hook templates for wiring the `ailang micro-rag` engine into different coding
harnesses. All frontends shell into the same engine (`ailang micro-rag context`
and `ailang micro-rag lint-builtin`); per-harness directories only differ in:

- Session id env var fallback (`CLAUDE_SESSION_ID` / `GEMINI_SESSION_ID` /
  `CODEX_SESSION_ID`)
- Harness-specific hook registration file format (Claude: `settings.json`,
  Gemini: `.gemini/settings.json`, Codex: `.codex/hooks.json`)

The stdin JSON schema (`tool_name` + `tool_input.file_path` + content fields)
is identical across Claude Code, Gemini CLI, and Codex CLI, and the output
envelope (`hookSpecificOutput.additionalContext`) is accepted by all three.

## Layout

```
frontends/
  claude-code/    # symlink/placeholder — canonical shim lives at ~/.ailang/hooks/
  gemini/
    microrag_context.sh      # PreToolUse hook for Edit|Write|Read|MultiEdit
    microrag_lint.sh         # PostToolUse hook for Edit|Write|MultiEdit on *.ail
    settings.json.example    # Drop-in .gemini/settings.json
  codex/
    microrag_context.sh
    microrag_lint.sh
    hooks.json.example       # Drop-in .codex/hooks.json
```

## Install (Gemini CLI)

```bash
# 1. Install shims
cp .claude/skills/microrag/frontends/gemini/*.sh ~/.ailang/hooks/
chmod +x ~/.ailang/hooks/microrag_*.sh

# 2. Register in project settings
mkdir -p .gemini
cp .claude/skills/microrag/frontends/gemini/settings.json.example .gemini/settings.json

# 3. Enable
export AILANG_MICRORAG_ENABLED=1
```

## Install (Codex CLI)

```bash
# 1. Install shims (reuse Claude Code shims if already installed — schema is identical)
cp .claude/skills/microrag/frontends/codex/*.sh ~/.ailang/hooks/
chmod +x ~/.ailang/hooks/microrag_*.sh

# 2. Register in project settings
mkdir -p .codex
cp .claude/skills/microrag/frontends/codex/hooks.json.example .codex/hooks.json

# 3. Enable
export AILANG_MICRORAG_ENABLED=1
```

## Graceful Degradation

All shims exit `0` silently when any of:

- `AILANG_MICRORAG_ENABLED=0` (master switch)
- `ailang` or `jq` binary not on PATH
- Tool name / file path is unsupported (non-.ail for lint, binary asset for context)
- Engine returns `no_route` / empty `injection_text`
- Engine exceeds 3s timeout (via `gtimeout`/`timeout`)

This guarantees hooks never break a coding session.

## Transcript Marker

The engine embeds `🧠 μRAG` in every `injection_text` it produces, so grepping
any harness transcript for that emoji string finds every microrag injection
regardless of which frontend produced it.
