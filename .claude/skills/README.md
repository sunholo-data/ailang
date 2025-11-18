# AILANG Skills

This directory contains Anthropic Agent Skills for AILANG development, following the official October 2025 specification.

## What are Agent Skills?

**Agent Skills** are specialized capabilities that combine:
- **Structured instructions** (markdown with YAML frontmatter)
- **Progressive disclosure** (load information only as needed)
- **Executable scripts** (bash/python/go for automation)
- **Resource files** (loaded on demand)

Skills differ from Agents in that they provide focused, reusable workflows rather than autonomous multi-step reasoning.

## Available Skills

### Core Development

**[use-ailang/](use-ailang/)** - Write correct AILANG code
- Quick reference for AILANG v0.3.12+ syntax
- Common patterns (recursion, pattern matching, effects)
- What works vs what doesn't
- Running and testing AILANG programs
- **Scripts**: check_version.sh, validate_code.sh
- **Resources**: syntax_quick_ref.md, common_patterns.md
- **Public-facing**: Can be shared with the community

**[builtin-developer/](builtin-developer/)** - Add new AILANG builtin functions
- Modern registry system with M-DX1 (67% faster development)
- Type Builder DSL and hermetic testing with MockEffContext
- Automatic wiring to runtime/link
- Complete validation and inspection tools
- **Scripts**: validate_builtins.sh, check_builtin_health.sh
- **Resources**: type_builder_examples.md, testing_patterns.md
- **Time savings**: 2.5h instead of 7.5h per builtin

**[parser-developer/](parser-developer/)** - Master AILANG parser development
- Critical token positioning conventions (saves 30% development time)
- AST type quick reference and common patterns
- API discovery with make doc (80% faster)
- Debug mode tracing (DEBUG_PARSER=1)
- **Scripts**: trace_parser.sh, check_ast_types.sh, find_api.sh
- **Resources**: token_positioning.md, ast_quick_reference.md, common_patterns.md, api_discovery.md

**[design-doc-creator/](design-doc-creator/)** - Create AILANG design documents
- Creates design docs in correct format and location
- Handles both planned/ and implemented/ workflow
- Comprehensive templates with best practices
- Automated status updates and versioning
- **Scripts**: create_planned_doc.sh, move_to_implemented.sh
- **Resources**: design_doc_structure.md

### Release Management

**[release-manager/](release-manager/)** - Create new releases
- Pre-release verification (tests, linting, file sizes)
- Updates documentation (README, CHANGELOG, version numbers)
- Creates git tags and pushes to remote
- Monitors CI/CD and verifies release artifacts
- **Scripts**: pre_release_checks.sh, post_release_checks.sh
- **Resources**: release_checklist.md

**[post-release/](post-release/)** - Post-release tasks
- Runs evaluation baseline with all models
- Updates website benchmark dashboard with history preservation
- Extracts metrics for CHANGELOG
- Moves design docs to implemented/
- **Scripts**: run_eval_baseline.sh, update_dashboard.sh, extract_changelog_metrics.sh
- **Resources**: post_release_checklist.md

### Sprint Planning

**[sprint-planner/](sprint-planner/)** - Create comprehensive sprint plans
- Analyzes design docs and current implementation status
- Calculates velocity from recent work
- Proposes realistic milestones with LOC estimates
- Creates actionable day-by-day breakdowns
- **Scripts**: analyze_velocity.sh
- **Resources**: sprint_plan_template.md

**[sprint-executor/](sprint-executor/)** - Execute approved sprint plans
- Test-driven development (tests must pass)
- Continuous linting enforcement
- Progressive documentation updates (CHANGELOG.md)
- TodoWrite integration for progress tracking
- Pause points after each milestone
- **Scripts**: validate_prerequisites.sh, milestone_checkpoint.sh
- **Resources**: milestone_checklist.md

### Automation & Integration

**[model-manager/](model-manager/)** - Test, validate, and add new AI models to eval suite
- Test API access to new models (OpenAI, Anthropic, Google)
- Find correct API model names and pricing
- Update models.yml configuration safely
- Run test benchmarks to verify end-to-end
- Vertex AI compatibility checking for Gemini models
- **Scripts**: test_model_access.sh, verify_vertex_model.sh, update_models_yml.sh, run_test_benchmark.sh, find_model_info.sh
- **Resources**: provider_endpoints.md, pricing_guide.md
- **Use for**: Adding GPT-5.1, Gemini 3 Pro, and other new models to benchmarks

**[headless-runner/](headless-runner/)** - Run Claude Code in headless/programmatic mode
- Execute Claude from scripts, CI/CD pipelines, and autonomous agents
- Full access to project configuration (skills, agents, commands)
- Multi-turn conversations with session management
- AILANG agent messaging integration (task claiming, handoffs, error handling)
- Comprehensive workflow patterns (single agent, pipeline, load balancing, retry/DLQ)
- **Scripts**: test_headless.sh, run_with_retry.sh
- **Resources**: cli_reference.md, examples.md, agent_workflows.md, troubleshooting.md
- **Use for**: Building autonomous agents, CI/CD integration, scheduled workflows

**[agent-inbox/](agent-inbox/)** - Check and process AILANG agent messages
- Check inbox for messages from autonomous agents
- Acknowledge messages to claim tasks
- Send results for agent-to-agent handoffs
- Message format reference and workflow patterns
- **Resources**: Integrated with headless-runner for full automation

## How Skills Work

### Discovery (YAML Frontmatter)

Each skill has a `SKILL.md` file with YAML frontmatter:

```yaml
---
name: AILANG Code Writing
description: Write and run AILANG code with correct syntax. Use when user asks to write AILANG programs, fix AILANG syntax errors, or run AILANG code.
---
```

The `name` and `description` are loaded into Claude's system prompt at startup, allowing automatic skill discovery and invocation.

### Progressive Disclosure

Skills load information in stages:

1. **Always loaded**: SKILL.md with YAML frontmatter + overview (200-300 lines)
2. **Execute as needed**: Scripts in `scripts/` directory (run without loading into context)
3. **Load on demand**: Resources in `resources/` directory (loaded when detailed reference needed)

**Example - use-ailang skill:**
- **Always loaded**: SKILL.md (288 lines) - Basic overview, when to use, script descriptions
- **On demand**: resources/syntax_quick_ref.md (131 lines) - Detailed syntax rules
- **On demand**: resources/common_patterns.md (218 lines) - Examples and patterns
- **Execute**: scripts/check_version.sh - Runs without loading into context
- **Execute**: scripts/validate_code.sh - Runs without loading into context

**Context savings**: 42% reduction (498 → 288 lines always-loaded) with full reference available when needed.

### Executable Scripts

Scripts provide automation without consuming context tokens:

```bash
# Script executes, only output is shown to Claude
.claude/skills/release-manager/scripts/pre_release_checks.sh
# Output: "✓ All tests pass\n✓ Linting passes\n✓ File sizes OK"
# The script code itself never loads into context!
```

**Benefits:**
- No token cost for script code
- Faster execution (native bash vs AI generation)
- Deterministic results
- Reusable across sessions

## Skill Directory Structure

```
.claude/skills/
├── skill-name/
│   ├── SKILL.md                    # Main file with YAML frontmatter (required)
│   ├── scripts/                    # Executable automation (optional)
│   │   ├── script1.sh
│   │   └── script2.sh
│   └── resources/                  # Reference files loaded on demand (optional)
│       ├── reference.md
│       └── template.md
```

### Required Components

**SKILL.md** - Main skill file:
```markdown
---
name: Skill Name
description: Brief description with when to use (max 1024 chars)
---

# Skill Name

## Quick Start
[1-2 sentence overview + most common usage]

## When to Use This Skill
[Clear triggers for when Claude should invoke]

## Available Scripts
[List of scripts with one-line descriptions]

## Workflow
[Detailed instructions - loaded when skill is active]

## Resources
[Reference to additional files - loaded on demand]
```

### Optional Components

**scripts/** - Executable automation:
- Bash scripts (`.sh`) for system operations
- Python scripts (`.py`) for data processing
- Any executable that can run from command line
- Must be marked executable (`chmod +x`)

**resources/** - Reference files:
- Templates (e.g., `sprint_plan_template.md`)
- Quick references (e.g., `syntax_quick_ref.md`)
- Checklists (e.g., `release_checklist.md`)
- Any markdown file loaded on demand

## Using Skills

### As a User

Skills are invoked automatically by Claude when appropriate for the task:

```
User: "Ready to release v0.3.14"
→ Claude uses release-manager skill

User: "Update benchmarks for the release"
→ Claude uses post-release skill

User: "Help me write an AILANG function"
→ Claude uses use-ailang skill

User: "Plan the next sprint"
→ Claude uses sprint-planner skill

User: "Execute the sprint plan"
→ Claude uses sprint-executor skill
```

**Just describe what you want - Claude will invoke the right skill.**

### As a Developer

To create a new skill:

1. **Create directory structure:**
   ```bash
   mkdir -p .claude/skills/my-skill/scripts
   mkdir -p .claude/skills/my-skill/resources
   ```

2. **Create SKILL.md with frontmatter:**
   ```markdown
   ---
   name: My Skill Name
   description: What it does and when to use it
   ---

   # My Skill Name
   [Rest of skill documentation]
   ```

3. **Add scripts (optional):**
   ```bash
   # Create executable scripts
   touch .claude/skills/my-skill/scripts/my_script.sh
   chmod +x .claude/skills/my-skill/scripts/my_script.sh
   ```

4. **Add resources (optional):**
   ```bash
   # Create reference files
   touch .claude/skills/my-skill/resources/reference.md
   ```

5. **Update this README:**
   - Add skill to "Available Skills" section
   - Document what it does and what it includes

See [SKILLS_GUIDE.md](SKILLS_GUIDE.md) for detailed skill development guide.

## Skills vs Agents vs Commands

| Feature | Skills | Agents | Commands |
|---------|--------|--------|----------|
| **Purpose** | Reusable workflows | Autonomous reasoning | Explicit invocation |
| **Invocation** | Auto (based on context) | Auto or explicit | Manual (`/command`) |
| **Structure** | SKILL.md + scripts + resources | Markdown prompt | Markdown prompt |
| **Progressive disclosure** | ✅ Yes | ❌ No | ❌ No |
| **Code execution** | ✅ Yes (scripts) | ✅ Yes (via tools) | ✅ Yes (via tools) |
| **Tool access** | ✅ Full access | ✅ Full access | ✅ Full access |
| **Use case** | Focused workflows | Complex multi-step tasks | User-triggered actions |

**When to use each:**
- **Use Skills** for well-defined, reusable workflows (releases, writing AILANG code)
- **Use Agents** for complex reasoning (eval-orchestrator, codebase-organizer)
- **Use Commands** when user wants explicit control (deprecated, prefer skills)

## Migration from Slash Commands

Old slash commands in `.claude/commands/` have been superseded by skills:

| Old Command | New Skill | Status |
|-------------|-----------|--------|
| `/release` | `release-manager` | ✅ Migrated |
| `/post-release` | `post-release` | ✅ Migrated |
| `/plan-sprint` | `sprint-planner` | ✅ Migrated |
| `/sprint` | `sprint-executor` | ✅ Migrated |
| N/A | `use-ailang` | ✅ New (public) |

**Old commands still exist but are deprecated.** They will be removed in a future cleanup.

**Migration benefits:**
- Auto-invocation (no need to remember command names)
- Progressive disclosure (better context management)
- Executable scripts (faster, more reliable)
- Shareable with community (especially use-ailang)

## Public Distribution

The **use-ailang** skill is designed for public distribution:
- Includes installation instructions
- Provides complete syntax reference
- Shows how to run AILANG programs
- Can be shared on GitHub, docs site, etc.

To share publicly:
1. Copy `use-ailang/` directory to docs site or release notes
2. Ensure it's up-to-date with latest AILANG version
3. Link to it from README.md or getting started guide

## Metrics

**Context savings from progressive disclosure:**
- use-ailang: 42% reduction (498 → 288 lines always-loaded)
- release-manager: Similar savings (checklist loaded on demand)
- post-release: Scripts handle automation (no code in context)
- sprint-planner: Template loaded on demand
- sprint-executor: Checklist loaded on demand

**Total skills**: 13 (project-specific)
**Total scripts**: 20 executable automation scripts
**Total resources**: 12+ reference files
**Total lines**: ~4,500 lines of structured, reusable content

**Note**: skill-builder is available globally at `~/.claude/skills/skill-builder/` and is not included in project skills count

## Best Practices

1. **Keep SKILL.md concise** - Overview only, details in resources
2. **Use scripts for automation** - Don't make Claude generate code that can be scripted
3. **Progressive disclosure** - Only load what's needed
4. **Clear triggers** - Description should make it obvious when to use
5. **Test scripts** - Ensure they work before committing
6. **Document assumptions** - Make prerequisites clear
7. **Version control** - Skills evolve with AILANG, keep them updated

## References

- **Anthropic Agent Skills announcement**: October 16, 2025
- **Specification**: Skills are directories with SKILL.md (YAML frontmatter required)
- **Examples**: anthropics/skills GitHub repository
- **Documentation**: https://docs.claude.com/en/docs/agents-and-tools/agent-skills/
