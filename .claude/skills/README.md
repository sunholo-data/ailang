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

**[cli-doc-maintainer/](cli-doc-maintainer/)** - Maintain CLI help as source of truth
- Audits commands in main.go against help.go
- Validates environment variables are documented
- Suggests improvements for discoverability
- Ensures CLI help stays synchronized with implementation
- **Scripts**: audit_commands.sh, audit_env_vars.sh, suggest_improvements.sh
- **Resources**: best_practices.md, help_template.md
- **Critical for**: AI discoverability in external repositories

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

**[docs-sync/](docs-sync/)** - Sync documentation website with codebase
- Audits design docs (planned vs implemented)
- Validates version constants against git tags
- Checks example files work
- Generates sync status reports
- Tracks feature themes and creates new ones as needed
- **Scripts**: audit_design_docs.sh, check_versions.sh, check_examples.sh, generate_report.sh
- **Resources**: feature_themes.md, landing_page_checklist.md
- **Use for**: Post-release website updates, accuracy audits, finding stale docs

### Sprint Planning

**[mission-control/](mission-control/)** - Run one outer-loop iteration of a long-running mission
- Observes mission state (default: design_docs/v1-mission.md), picks top backlog item
- Reality-checks doc status against git/code before working
- Routes through design-doc-creator → sprint-planner → sprint-executor → sprint-evaluator with the mission's model routing policy
- Records an append-only log entry (routing evidence + ruled-out ledger), then runs the retro (skill/process/backlog lanes)
- Fired nightly by the dev.ailang.mission-control launchd job (tools/launchd/)

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

### Performance & Quality

**[perf-reviewer/](perf-reviewer/)** - Review code for performance issues and run benchmarks
- Cross-language benchmarks: AILANG interpreted vs Python vs AILANG compiled to Go
- Performance principles guide (Abseil-inspired + Go patterns)
- Phase timing profiler for AILANG compilation
- Memory layout, batch operations, algorithmic complexity checks
- **Scripts**: benchmark.sh, profile_ailang.sh
- **Resources**: principles.md, go_patterns.md
- **Key finding**: AILANG interpreter ~5x slower than Python for recursive workloads; compilation provides 10-50x speedup

**[test-coverage-guardian/](test-coverage-guardian/)** - Analyze test coverage, identify gaps, improve test quality
- Coverage analysis and gap detection
- Dead code identification
- Test robustness improvements
- **Use for**: Pre-release quality checks, identifying untested code

**[design-spec-auditor/](design-spec-auditor/)** - Verify code implementation matches design specifications
- Compare design docs with actual code
- Identify architectural deviations
- Post-implementation verification
- **Use for**: After implementing features, during code reviews, refactoring

**[codebase-organizer/](codebase-organizer/)** - Monitor and refactor large files into AI-friendly modules
- File size monitoring (target: <500 lines)
- Safe refactoring with test verification
- **Use for**: When files exceed thresholds, improving maintainability

### Environment Setup

**[cloud-setup/](cloud-setup/)** - Set up cloud/mobile Claude Code environments
- Full automated setup script for Go, make, gh
- Verification script to check environment health
- DNS fix, Go module proxy workarounds
- Comprehensive troubleshooting guide
- **Scripts**: setup.sh, verify.sh
- **Resources**: troubleshooting.md
- **Use for**: New cloud sessions, mobile Claude Code, missing tools

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

**[local-model-onboarding/](local-model-onboarding/)** - Onboard a new on-device model to the Mac Studio eval rig
- Hardware size/shape feasibility gate (128 GB M4 Max, MoE small-active, p=1 bandwidth rule)
- OpenRouter quality pre-screen before spending rig time
- ollama pull + real resident-VRAM verdict
- opencode + pi cross-harness registration in models.yml (shared model_family)
- Wire into the continuous OS rotation (os-rotation-filler)
- **Scripts**: check_model_fit.sh (sizing verdict, estimate or measure)
- **Use for**: "is this model the right size for the rig?", trying a new local coding model
- **Complements**: model-manager (models.yml mechanics) + local-ollama-eval (running on the rig)

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

**Total skills**: 16 (project-specific)
**Total scripts**: 28 executable automation scripts
**Total resources**: 17+ reference files
**Total lines**: ~5,500 lines of structured, reusable content

**Note**: skill-builder is available globally at `~/.claude/skills/skill-builder/` and is not included in project skills count

## Long-Running Agent Patterns (NEW)

**Our skills now implement patterns from [Anthropic's long-running agent article](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents)!**

### Multi-Session Continuity

Skills can now span multiple Claude Code sessions with no loss of context:

**Pattern**: Initializer + Coding Agent
- **sprint-planner** (Initializer): Creates JSON progress file + infrastructure
- **sprint-executor** (Coding Agent): Resumes work from JSON across sessions

### Key Features Implemented

#### 1. Session Startup Routine
Every continuing session starts with context check:
- `.claude/skills/sprint-executor/scripts/session_start.sh`
- Checks pwd, reads JSON progress, reviews git log, runs tests
- Prints "Here's where we left off" summary

#### 2. Structured Progress Tracking (JSON)
Machine-readable state files replace markdown-only tracking:
- `.ailang/state/sprints/sprint_<id>.json` - Sprint progress
- `.ailang/state/release_<version>.json` - Release progress
- Follows "constrained modification" pattern (only specific fields change)

#### 3. Correlation IDs
Messages linked across agent handoffs:
- `correlation_id: "sprint_M-S1"` - Track entire workflow
- `reply_to: "<message_id>"` - Link conversation threads
- Filter messages by workflow: `grep "Correlation: sprint_M-S1"`

#### 4. End-to-End Testing
Test as users would, not just unit tests:
- `.claude/skills/sprint-executor/scripts/acceptance_test.sh`
- Tests: parser (run examples), builtin (REPL), e2e (full pipeline)

### Workflow Diagram

```
Session 1:
  design-doc-creator → sprint-planner (creates JSON)
    └─ JSON: .ailang/state/sprints/sprint_M-S1.json
    └─ Message: correlation_id="sprint_M-S1"

Session 2:
  sprint-executor → session_start.sh (reads JSON)
    └─ Implements M-S1.1, M-S1.2
    └─ Updates JSON (passes=true/false)
    └─ Message: milestone_complete, correlation_id="sprint_M-S1"

Session 3:
  sprint-executor → session_start.sh (resumes from JSON)
    └─ Continues M-S1.3, M-S1.4
    └─ Updates JSON
    └─ Message: sprint_complete, correlation_id="sprint_M-S1"
```

### Documentation

- Sprint JSON Schema: `.claude/skills/sprint-executor/resources/json_progress_schema.md`
- Message Format: `.claude/skills/agent-inbox/resources/message_format.md`
- Release State: `.claude/skills/release-manager/resources/release_state_schema.md`

## Best Practices

1. **Keep SKILL.md concise** - Overview only, details in resources
2. **Use scripts for automation** - Don't make Claude generate code that can be scripted
3. **Progressive disclosure** - Only load what's needed
4. **Clear triggers** - Description should make it obvious when to use
5. **Test scripts** - Ensure they work before committing
6. **Document assumptions** - Make prerequisites clear
7. **Version control** - Skills evolve with AILANG, keep them updated
8. **NEW: Session resumption** - Always provide session_start scripts for long-running work
9. **NEW: JSON state files** - Use structured state for machine-readable progress
10. **NEW: Correlation IDs** - Link messages across agent handoffs

## References

- **Anthropic Agent Skills announcement**: October 16, 2025
- **Specification**: Skills are directories with SKILL.md (YAML frontmatter required)
- **Examples**: anthropics/skills GitHub repository
- **Documentation**: https://docs.claude.com/en/docs/agents-and-tools/agent-skills/
