# M-PROCESS-SUBCMD: Subcommand Allowlists for Process Effect

**Status**: Planned
**Target**: v0.9.0
**Priority**: P1 (High — capability-secure AI agents need this)
**Estimated**: 4 hours (Phase 1: CLI) + future (Phase 2: Type system)
**Dependencies**: M-PROCESS (implemented)

## Problem Statement

The `--process-allowlist` flag currently operates at the command level only:

```bash
ailang run --caps Process --process-allowlist "git,gh,ailang,echo,date" ...
```

This grants access to ALL subcommands of `git`, `gh`, and `ailang`. For AI agents that need to run `git status` but must never run `git push --force`, there is no runtime-level filtering. The demos team building the speak voice agent had to write ~100 lines of userland AILANG to manually filter subcommands:

```ailang
-- Current workaround: ~100 lines of boilerplate per agent
pure func isGitSubcmdSafe(args: [string]) -> bool =
  match args {
    [] => false,
    subcmd :: _ => match subcmd {
      "status" => true,  "log" => true,  "diff" => true,
      "branch" => true,  "add" => true,  "commit" => true,
      _ => false
    }
  }

pure func isGhSubcmdSafe(args: [string]) -> bool = ...  -- another 30 lines
pure func isAilangSubcmdSafe(args: [string]) -> bool = ... -- another 30 lines
```

This is exactly what the effect system should handle — security policy should live in runtime configuration, not userland string matching.

## Goals

1. Move subcommand filtering from userland to runtime (eliminate ~100 LOC boilerplate per agent)
2. Make Process capabilities auditable at the CLI invocation level
3. Maintain backward compatibility with existing `--process-allowlist` syntax
4. Keep the security model consistent with Net effect's domain allowlist pattern

## Solution Design

### Phase 1: CLI Subcommand Syntax (This Feature)

Extend `--process-allowlist` to support `command:subcommand` syntax:

```bash
# Allow specific commands (existing behavior, unchanged)
--process-allowlist "echo,date,ls"

# Allow specific subcommands
--process-allowlist "git:status,git:log,git:diff,git:branch,git:add,git:commit"

# Mix command-level and subcommand-level
--process-allowlist "echo,date,git:status,git:log,git:diff"

# Deny-list approach (new flag)
--process-denylist "git:push,git:reset,git:clean,gh:pr:merge,gh:issue:close"
```

**Semantics:**
- `echo` — allow `echo` with any arguments (command-level, existing behavior)
- `git:status` — allow `git` only when first argument is `status`
- `git:*` — allow `git` with any subcommand (explicit wildcard, same as bare `git`)
- Denylist is checked AFTER allowlist (deny takes precedence)
- Subcommand matching checks `args[0]` only (not deeper nesting like `gh pr merge`)
- For multi-level commands like `gh pr list`, use colon notation: `gh:pr:list` checks `args[0] == "pr"` AND `args[1] == "list"`

**Error messages:**
```
Err(NotAllowed("git push"))  -- subcommand "push" not in allowlist for "git"
Err(NotAllowed("git"))       -- command "git" not in allowlist at all
```

### Phase 2: Type System Annotations (Future)

Longer-term, expose allowlists in effect annotations for static verification:

```ailang
-- Allow specific commands in the type signature
func safeGitOps() -> () ! {Process @allow=[git:status, git:log, git:diff]} { ... }

-- Deny-list approach
func restrictedExec() -> () ! {Process @allow=[git] @deny=[git:push, git:reset]} { ... }
```

This would enable:
- Static verification of command access at the function signature level
- Z3 could prove a function only calls allowed commands
- Capability attenuation (callee gets subset of caller's Process capability)

**Phase 2 is a separate design doc** — it requires parser changes, type elaboration, and budget system extensions.

## Architecture

### Data Model Changes

**`internal/effects/process_context.go`:**

```go
type ProcessContext struct {
    Timeout      time.Duration
    MaxOutput    int64
    Allowlist    map[string]string   // Existing: command → resolved path
    HasAllowlist bool
    // NEW:
    SubcommandAllowlist map[string][]string  // command → allowed subcommands (nil = all)
    Denylist            map[string][]string  // command → denied subcommands
    HasDenylist         bool
}
```

### Allowlist Resolution

The subcommand check happens in `processExec()` AFTER command-level allowlist check passes:

```go
func (pc *ProcessContext) IsSubcommandAllowed(cmd string, args []string) (bool, string) {
    // 1. Check denylist first (deny takes precedence)
    if pc.HasDenylist {
        if denied := pc.matchDenylist(cmd, args); denied != "" {
            return false, denied
        }
    }

    // 2. Check subcommand allowlist
    if subs, hasSubFilter := pc.SubcommandAllowlist[cmd]; hasSubFilter {
        if len(args) == 0 {
            return false, cmd + " (no subcommand)"
        }
        subcmd := buildSubcmdKey(args)
        for _, allowed := range subs {
            if allowed == "*" || allowed == subcmd {
                return true, ""
            }
        }
        return false, cmd + " " + args[0]
    }

    // 3. No subcommand filter for this command — allow all subcommands
    return true, ""
}
```

### CLI Flag Parsing

**`cmd/ailang/run_helpers.go`:**

Parse the `--process-allowlist` value, splitting on `:` to detect subcommands:

```go
func parseProcessAllowlist(raw string) (commands map[string]bool, subcmds map[string][]string) {
    for _, entry := range strings.Split(raw, ",") {
        entry = strings.TrimSpace(entry)
        if strings.Contains(entry, ":") {
            parts := strings.SplitN(entry, ":", 2)
            cmd := parts[0]
            sub := parts[1]
            subcmds[cmd] = append(subcmds[cmd], sub)
            commands[cmd] = true  // also add to command-level allowlist
        } else {
            commands[entry] = true
        }
    }
    return
}
```

### New CLI Flag

```
--process-denylist <list>   Denied command:subcommand patterns (comma-separated)
```

## Files to Modify

| File | Change |
|------|--------|
| `internal/effects/process_context.go` | Add SubcommandAllowlist, Denylist fields |
| `internal/effects/process.go` | Add subcommand check after command-level check |
| `cmd/ailang/main.go` | Add `--process-denylist` flag |
| `cmd/ailang/run_helpers.go` | Parse `:` syntax in allowlist, setup denylist |
| `internal/effects/process_test.go` | Add subcommand filtering tests |

## Testing Strategy

**Unit tests:**
- `git:status` allowed, `exec("git", ["status"])` → Ok
- `git:status` allowed, `exec("git", ["push"])` → Err(NotAllowed)
- `git:push` denied, `exec("git", ["push"])` → Err(NotAllowed)
- `echo` allowed (no subcommand filter), `exec("echo", ["anything"])` → Ok
- Multi-level: `gh:pr:list` allowed, `exec("gh", ["pr", "list"])` → Ok
- Multi-level: `gh:pr:list` allowed, `exec("gh", ["pr", "merge"])` → Err(NotAllowed)
- Denylist precedence: allow `git`, deny `git:push` → push blocked
- Empty args with subcommand filter → Err(NotAllowed)
- Backward compatibility: bare `--process-allowlist "echo,git"` still works

**Integration test:**
- End-to-end AILANG program using `exec("git", ["status"])` with subcommand allowlist

## Context from Feature Request

**Source**: Message `3a07d6fe` from `demos` (speak voice agent)
**Use case**: Streaming/Gemini Live agent that uses `exec()` for:
- `git status`, `git log`, `git diff` (read-only git)
- `gh pr list`, `gh issue list` (read-only GitHub)
- `ailang messages`, `ailang chains` (read-only AILANG CLI)
- `date`, `ls`, `echo` (basic utilities)

**Must block**: `git push`, `git reset --hard`, `git clean`, `gh pr merge`, `gh issue close`

## Non-Goals

- Shell expansion (never — security risk)
- Regex patterns for subcommand matching (too complex, audit-unfriendly)
- Per-subcommand path resolution (only command-level resolution needed)
- Type-system `@allow`/`@deny` annotations (Phase 2, separate doc)

---

**Document created**: 2026-02-18
