---
tier: internal
reviewed: 2026-11-26
reviewBy: 2027-02-26
---

# Setting Up the Coordinator for External Projects

Configure the AILANG coordinator to manage autonomous agents, workflow chains, and GitHub sync across external repositories.

## Prerequisites & Compatibility

1. AILANG installed (`ailang --version`) and `~/.ailang/config.yaml` created.
2. Target repository contains skills in `.claude/skills/<skill-name>/SKILL.md` (or external `invoke.template_file` prompts).
3. Claude Code (or configured `provider`) authenticated.

| Repo default branch | `merge_branch` setting |
|---|---|
| `dev` | Omit (default) |
| `main` / `master` | `merge_branch: main` or `master` (**required** or worktree creation fails) |

## Step 1: Agent Configuration (`~/.ailang/config.yaml`)

```yaml
coordinator:
  agents:
    - id: my-project-design-doc
      label: "My Project Design Doc"
      inbox: my-project-design-doc
      workspace: /absolute/path/to/my-project
      merge_branch: main
      capabilities: [research, docs]
      provider: claude
      trigger_on_complete: [my-project-sprint-planner]
      auto_approve_handoffs: false
      auto_merge: false
      session_continuity: true
      max_concurrent_tasks: 1
      invoke:
        type: skill
        name: design-doc-creator
      output_markers: ["DESIGN_DOC_PATH:"]
      artifact_patterns: ["design_docs/**/*.md"]
```

| Key Field | Required | Notes |
|---|---|---|
| `id` / `inbox` / `workspace` | Yes | `workspace` must be an **absolute path** to a git repo |
| `merge_branch` | If not `dev` | Base branch for worktrees and PR merges |
| `invoke.type` | Yes | `skill`, `script`, or `prompt` (`invoke.template` or `invoke.template_file` with `{{.TaskID}}`, `{{.GithubIssue}}`, `{{.Content}}`, `{{.Stage}}`, `{{.OutputMarkers}}`) |
| `trigger_on_complete` | No | Agent IDs chained after approval |
| `auth_mode` | No | `"oauth"` (default) or `"apikey"` with `credit_account` |
| `inputs_allow` | Cloud inputs | Exact `owner/repo` list (no wildcards) for `--inputs-file` / `AILANG_TASK_INPUTS` |

<details>
<summary>Cloud input grants, generic prompt templates, and Stapledons Voyage multi-skill example</summary>

**Cloud input grants (`inputs_allow`)**: Add `inputs_allow: [sunholo-data/daneel-memory]` to the trusted cloud registry entry, keep `.incoming/` guidance in `AGENTS.md`, and verify the staged checklist in [`examples/task_inputs`](https://github.com/sunholo-data/ailang/tree/dev/examples/task_inputs).

**Generic templates**: Copy `design-doc.md`, `sprint-planner.md`, or `sprint-executor.md` from [`templates/`](https://github.com/sunholo-data/ailang/tree/dev/templates) to `~/.ailang/templates/` or `.claude/templates/` for `invoke.template_file`.

```yaml
# Multi-agent external repo example (Stapledons Voyage)
- id: stapledon-design-doc
  inbox: stapledon-design-doc
  workspace: /Users/mark/dev/sunholo/stapledons_voyage
  merge_branch: main
  trigger_on_complete: [stapledon-sprint-planner]
  invoke: { type: skill, name: design-doc-creator }
```

</details>

## Step 2: Multi-Repo GitHub Sync (Optional)

```yaml
coordinator:
  github_sync:
    repos:
      - repo: owner/primary-repo
        enabled: true
        interval_secs: 300
        target_inbox: design-doc-creator
        label_routing:
          - label_prefix: "coordinator:bug"
            target: design-doc-creator
      - repo: owner/external-repo
        enabled: true
        interval_secs: 300
        target_inbox: external-design-doc
        label_routing:
          - label_prefix: "feature"
            target: external-design-doc
```

## Step 3: Start & Verify

```bash
ailang coordinator start
ailang coordinator status
ailang messages send my-project-design-doc "Create design doc for auth" --title "Feature: Auth" --from "test"
ailang coordinator pending
```

Each task runs in an isolated git worktree at `~/.ailang/state/worktrees/<agent-id>/<task-id>/`.

## Troubleshooting

| Symptom | Fix |
|---|---|
| Agent ignores messages | Verify `ailang messages list --inbox <inbox>`, `ailang coordinator status`, and matching `id:` in `~/.ailang/config.yaml` |
| Skill not found | Ensure `<workspace>/.claude/skills/<skill-name>/SKILL.md` exists (with optional `scripts/` subdir) |
| `fatal: not a valid object name: 'dev'` | Set `merge_branch: main` (or your repo's default branch) on every agent for that repo |
| GitHub sync failing | Check `gh auth status` and `owner/repo` spelling |

## Related Documentation

- [Coordinator Guide](./coordinator.md) · [Telemetry & Tracing](./telemetry.md) · [Agent Messaging](./agent-messaging.md) · [Collaboration Hub](./collaboration-hub.md) · [Debugging Guide](./debugging.md)
