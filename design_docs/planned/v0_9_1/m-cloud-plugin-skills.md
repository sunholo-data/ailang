# M-CLOUD-PLUGIN-SKILLS: Cross-Project Skill Distribution via Plugin System

**Status**: Implemented (code changes complete, skill migration pending)
**Priority**: High
**Version**: v0.9.1
**Triggered by**: Cloud agent coordinator needing access to user-level skills

## Problem Statement

Skills are stored in three separate locations with no sync mechanism:

| Location | Count | Scope | Available in Cloud? |
|----------|-------|-------|-------------------|
| `.claude/skills/` (AILANG repo) | 27 | AILANG project only | Yes (git clone) |
| `~/.claude/skills/` (sunholo-data/skills.git) | 4 | Cross-project, user-level | **No** |
| `ailang_bootstrap` plugin (sunholo-data/ailang_bootstrap) | 3 | External AILANG users | **No** (not installed in cloud) |

**The gap**: Cloud Run Jobs clone the project repo (getting project-level skills) but have no access to user-level skills (`~/.claude/skills/`) or the bootstrap plugin. Agents that need cross-project skills (e.g., `website-builder`, `skill-builder`) fail silently.

**Locally**, the developer has all three sources available. But adding skills requires updating multiple repos, and there's no guarantee cloud stays in sync.

### Current Architecture

```
LAPTOP                                    CLOUD RUN JOB
───────                                   ──────────────
~/.claude/skills/        ← git clone      (doesn't exist)
  ailang-feedback/         of sunholo-
  skill-builder/           data/skills
  website-builder/
  release/

~/dev/sunholo/ailang/                     /workspace/{taskID}/
  .claude/skills/        ← git clone →     .claude/skills/     ✅
    sprint-executor/       of sunholo-       sprint-executor/
    parser-developer/      data/ailang       parser-developer/
    ...27 skills                             ...27 skills

~/.claude/plugins/cache/                  (doesn't exist)
  claude-code-plugins/
    frontend-design/
```

## Solution: Consolidate into ailang_bootstrap Plugin

### Design

Use the existing `sunholo-data/ailang_bootstrap` repo as the **single plugin** for all cross-project AILANG skills. Claude Code's plugin system handles distribution, namespacing, and discovery.

**Key insight**: `ailang_bootstrap` is already a Claude Code plugin (`name: "ailang"`) with 3 skills, commands, and an MCP server. Extending it with the 4 user-level skills makes it the canonical source for all non-project-specific AILANG capabilities.

### Proposed Architecture

```
LAPTOP                                    CLOUD RUN JOB
───────                                   ──────────────
~/.claude/plugins/cache/                  /plugins/{taskID}/ailang_bootstrap/
  ailang/                ← marketplace      (git cloned at job start)
    skills/                install
      ailang/                             Skills available via
      ailang-debug/                       --plugin-dir flag
      ailang-inbox/
      skill-builder/       ← NEW
      website-builder/     ← NEW
      ailang-feedback/     ← NEW (merge with ailang-inbox)

~/dev/sunholo/ailang/                     /workspace/{taskID}/
  .claude/skills/        ← git clone →     .claude/skills/     ✅
    sprint-executor/                         sprint-executor/
    ...27 skills                             ...27 skills
```

### Skill Categorization

**Stays in `.claude/skills/` (project-level)** — AILANG compiler/toolchain specific:
- sprint-executor, sprint-planner, design-doc-creator, parser-developer,
  builtin-developer, eval-analyzer, eval-gap-finder, benchmark-manager,
  codebase-organizer, collaboration-hub, coordinator-helper, design-spec-auditor,
  docs-sync, github-issue-triage, model-manager, perf-reviewer, post-release,
  prompt-manager, release-manager, test-coverage-guardian, trace-debugger,
  use-ailang, cloud-setup, headless-runner, gemini-cli-helper, cli-doc-maintainer,
  agent-inbox

**Moves to ailang_bootstrap plugin** — cross-project:
- `skill-builder` → `ailang:skill-builder` (useful in any project)
- `website-builder` → `ailang:website-builder` (used by website-builder agent)
- `ailang-feedback` → merge into existing `ailang:ailang-inbox` or keep as `ailang:ailang-feedback`

**Stays user-level only** (`~/.claude/skills/`):
- `release` — Python project release tool, not AILANG-specific

### Naming: Plugin Prefix

All plugin skills are namespaced as `ailang:<skill-name>`:

| Current (user-level) | Plugin (new) |
|----------------------|-------------|
| `/website-builder` | `/ailang:website-builder` |
| `/skill-builder` | `/ailang:skill-builder` |
| `/ailang-feedback` | `/ailang:ailang-feedback` |

External AILANG users already see:
- `/ailang:ailang` (write AILANG code)
- `/ailang:ailang-debug` (debug errors)
- `/ailang:ailang-inbox` (agent messaging)

## Implementation

### Phase 1: Update ailang_bootstrap Repo (Pending — separate workstream)

**Repo**: `sunholo-data/ailang_bootstrap`

1. Copy `skill-builder` from `~/.claude/skills/skill-builder/` into `skills/skill-builder/`
2. Copy `website-builder` from `~/.claude/skills/website-builder/` into `skills/website-builder/`
3. Merge `ailang-feedback` into `skills/ailang-inbox/` (or add as `skills/ailang-feedback/`)
4. Bump version in `.claude-plugin/plugin.json` to `0.9.1`
5. Tag release

### Phase 2: Coordinator Code Changes (✅ COMPLETE)

All code changes are implemented and tested. Two execution paths exist:

#### Local Mode: AgentConfig → executor.Task → Claude CLI

```
AgentConfig.PluginDirs → ExecuteOptions.PluginDirs → executor.Task.PluginDirs
                                                    → claude --plugin-dir /path/to/plugin
```

**Plus** per-agent third-party plugins:

```
AgentConfig.Plugins → ExecuteOptions.Plugins → executor.Task.Plugins
                                              → claude plugin marketplace add X
                                              → claude plugin install Y
```

#### Cloud Mode: CoordinatorConfig → DispatchParams → AILANG_PLUGIN_REPO → git clone

```
CoordinatorConfig.PluginRepo → DispatchParams.PluginRepo
                              → AILANG_PLUGIN_REPO env var
                              → Cloud Run Job clones plugin repo
                              → claude --plugin-dir /plugins/{taskID}/ailang_bootstrap
```

**Plus** AGENTS.md injection:

```
Plugin dir has AGENTS.md → copied into workspace (if repo doesn't have one)
```

#### Files Modified

| File | Change | Status |
|------|--------|--------|
| `internal/executor/executor.go` | Added `PluginDirs []string` and `Plugins *PluginsConfig` to Task struct; added `PluginsConfig` type | ✅ |
| `internal/executor/claude/claude.go` | Pass `--plugin-dir` flags; `installPlugins()` method for marketplace/install | ✅ |
| `internal/coordinator/agent_registry.go` | Added `PluginDirs` field and `Plugins *PluginsConfig` to AgentConfig; added `PluginsConfig` struct | ✅ |
| `internal/coordinator/agent_config.go` | Added `PluginRepo` to CoordinatorConfig | ✅ |
| `internal/coordinator/provider.go` | Added `PluginDirs` and `Plugins` to ExecuteOptions | ✅ |
| `internal/coordinator/provider_executor.go` | Wire PluginDirs + Plugins through to executor.Task; `convertPluginsConfig()` helper | ✅ |
| `internal/coordinator/daemon_tasks_exec.go` | Wire PluginDirs + Plugins from AgentConfig; pass PluginRepo in cloud dispatch | ✅ |
| `internal/coordinator/cloud_dispatcher.go` | Added `PluginRepo` to DispatchParams | ✅ |
| `internal/dispatch/cloudrun/dispatcher.go` | Pass `AILANG_PLUGIN_REPO` env var override (conditional) | ✅ |
| `internal/dispatch/cloudrun/dispatcher_test.go` | 2 new tests: with/without PluginRepo | ✅ |
| `cmd/ailang/coordinator_cloud.go` | Clone plugin repo (Step 0), pass `--plugin-dir`, inject AGENTS.md | ✅ |
| `.claude/settings.json` | Added `Skill(ailang:*)` permission | ✅ |
| `docker/Dockerfile.agent` | Pre-clone plugin repo with configurable ARG | ✅ |

### Phase 3: Config Changes

#### 3a. Agent config (`~/.ailang/config.yaml`)

```yaml
coordinator:
  # Plugin repo for cloud agents (cloned at job start)
  plugin_repo: https://github.com/sunholo-data/ailang_bootstrap.git

  agents:
    - id: website-builder
      invoke:
        type: skill
        name: "ailang:website-builder"  # plugin-namespaced
      plugin_dirs:
        - /path/to/local/ailang_bootstrap  # local development
      plugins:
        marketplaces:
          - anthropics/claude-code
        install:
          - frontend-design@anthropics-claude-code
      # ...
```

#### 3b. Permissions (`.claude/settings.json`)

```json
{
  "permissions": {
    "allow": [
      "Skill(ailang:*)"
    ]
  }
}
```

#### 3c. Dockerfile (`docker/Dockerfile.agent`)

```dockerfile
# Pre-clone plugin for faster cold starts (configurable)
ARG AILANG_PLUGIN_REPO=https://github.com/sunholo-data/ailang_bootstrap.git
RUN git clone --depth 1 ${AILANG_PLUGIN_REPO} /plugins/ailang_bootstrap 2>/dev/null || true
```

#### 3d. Terraform — Add env var to Cloud Run Job

```hcl
env {
  name  = "AILANG_PLUGIN_REPO"
  value = "https://github.com/sunholo-data/ailang_bootstrap.git"
}
```

### Phase 4: Local Migration (Pending — separate workstream)

1. Install ailang_bootstrap as plugin locally:
   ```bash
   claude plugin marketplace add sunholo-data/ailang_bootstrap
   claude plugin install ailang@sunholo-data
   ```
   Or for development: `claude --plugin-dir ~/path/to/ailang_bootstrap`

2. Remove migrated skills from `~/.claude/skills/`:
   ```bash
   rm -rf ~/.claude/skills/{skill-builder,website-builder,ailang-feedback}
   ```

3. Keep `~/.claude/skills/release/` (not AILANG-specific)

4. Update any local scripts/aliases that reference old skill names

## Migration Path

### Backward Compatibility

During migration, both old and new skill names should work:

1. **Coordinator**: If `invoke.name` has no `:`, try both local and plugin skill
2. **Grace period**: Keep user-level skills alongside plugin for 1 release cycle
3. **v0.9.2**: Remove user-level copies, plugin-only

### Testing

1. **Local**: Verify `/ailang:website-builder` triggers correctly from coordinator
2. **Cloud**: Verify Cloud Run Job clones plugin and executor can invoke skills
3. **Permissions**: Verify `Skill(ailang:*)` grants access to all plugin skills
4. **Fallback**: Verify agent still works if plugin clone fails (graceful degradation)

## Risks

| Risk | Mitigation |
|------|-----------|
| Plugin clone adds latency to cloud jobs | Pre-clone in Docker image; `--depth 1` |
| Plugin version drift | Pin to git tag in AILANG_PLUGIN_REPO URL |
| Skill name change breaks existing configs | Grace period with both names |
| MCP server in plugin may conflict | MCP server is optional, only starts if configured |
| Plugin install commands may not exist in all Claude versions | Best-effort with error logging |

## Success Criteria

- [x] `PluginDirs` and `PluginsConfig` wired through coordinator → executor
- [x] Cloud Run dispatcher passes `AILANG_PLUGIN_REPO` env var
- [x] Cloud job executor clones plugin and passes `--plugin-dir`
- [x] AGENTS.md injected from plugin into workspace
- [x] Per-agent third-party plugin installation (marketplace + install)
- [x] Dockerfile pre-clones plugin for cold start optimization
- [x] Tests pass (dispatcher_test.go: with/without PluginRepo)
- [ ] `website-builder` agent works in cloud with plugin-provided skill (needs Phase 1)
- [ ] Same skill works locally via installed plugin (needs Phase 4)
- [ ] Single repo (`ailang_bootstrap`) is source of truth for cross-project skills (needs Phase 1)

## File Impact (Final)

| File | Change |
|------|--------|
| `internal/executor/executor.go` | `PluginDirs`, `Plugins *PluginsConfig`, `PluginsConfig` type |
| `internal/executor/claude/claude.go` | `--plugin-dir` flags, `installPlugins()` for marketplace/install |
| `internal/coordinator/agent_registry.go` | `PluginDirs`, `Plugins *PluginsConfig`, `PluginsConfig` struct |
| `internal/coordinator/agent_config.go` | `PluginRepo` on CoordinatorConfig |
| `internal/coordinator/provider.go` | `PluginDirs`, `Plugins` on ExecuteOptions |
| `internal/coordinator/provider_executor.go` | Wire through, `convertPluginsConfig()` |
| `internal/coordinator/daemon_tasks_exec.go` | Wire from AgentConfig, cloud dispatch PluginRepo |
| `internal/coordinator/cloud_dispatcher.go` | `PluginRepo` on DispatchParams |
| `internal/dispatch/cloudrun/dispatcher.go` | `AILANG_PLUGIN_REPO` env var override |
| `internal/dispatch/cloudrun/dispatcher_test.go` | 2 new tests |
| `cmd/ailang/coordinator_cloud.go` | Plugin clone, `--plugin-dir`, AGENTS.md injection |
| `.claude/settings.json` | `Skill(ailang:*)` permission |
| `docker/Dockerfile.agent` | Pre-clone with configurable ARG |
