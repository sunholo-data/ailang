# AILANG Skills Development Guide

This guide shows how to create and modify Anthropic Agent Skills for AILANG development.

## Table of Contents

- [Quick Start](#quick-start)
- [Skill Structure](#skill-structure)
- [YAML Frontmatter](#yaml-frontmatter)
- [Progressive Disclosure](#progressive-disclosure)
- [Creating Scripts](#creating-scripts)
- [Creating Resources](#creating-resources)
- [Testing Skills](#testing-skills)
- [Best Practices](#best-practices)
- [Examples](#examples)

## Quick Start

### Create a New Skill (5 Minutes)

```bash
# 1. Create directory structure
mkdir -p .claude/skills/my-skill/{scripts,resources}

# 2. Create SKILL.md with frontmatter
cat > .claude/skills/my-skill/SKILL.md << 'EOF'
---
name: My Skill Name
description: Brief description of what this skill does and when to use it.
---

# My Skill Name

Quick overview of the skill.

## Quick Start

Most common usage pattern.

## When to Use This Skill

Clear triggers for invocation.

## Workflow

Step-by-step instructions.
EOF

# 3. Make scripts executable (if any)
chmod +x .claude/skills/my-skill/scripts/*.sh

# 4. Update .claude/skills/README.md
# Add entry to "Available Skills" section

# 5. Test the skill
# Ask Claude to use the skill in context
```

## Skill Structure

### Directory Layout

```
.claude/skills/my-skill/
├── SKILL.md          # Main file with YAML frontmatter (REQUIRED)
├── scripts/          # Executable automation (OPTIONAL)
│   ├── script1.sh
│   └── script2.sh
└── resources/        # Reference files loaded on demand (OPTIONAL)
    ├── template.md
    └── reference.md
```

### Required Files

**SKILL.md** - Every skill MUST have this file with:
1. YAML frontmatter (name + description)
2. Quick Start section
3. When to Use section
4. Main workflow/instructions

### Optional Components

**scripts/** - Executable automation:
- Runs without loading into context (saves tokens)
- Provides deterministic, fast execution
- Can be bash, python, or any executable

**resources/** - Reference materials:
- Loaded only when needed (progressive disclosure)
- Templates, checklists, reference guides
- Markdown files for detailed documentation

## YAML Frontmatter

### Required Fields

Every SKILL.md MUST start with YAML frontmatter:

```yaml
---
name: Skill Name
description: Brief description with when to use (max 1024 chars)
---
```

### Field Specifications

**name** (required):
- Short, descriptive name
- Will appear in Claude's system prompt
- Should be unique across all skills
- Example: "AILANG Code Writing"

**description** (required):
- Brief description of what the skill does
- **MUST include when to use it**
- Max 1024 characters
- Loaded into Claude's system prompt for discovery
- Example: "Write and run AILANG code with correct syntax. Use when user asks to write AILANG programs, fix AILANG syntax errors, or run AILANG code."

### Good vs Bad Descriptions

**❌ BAD - Too vague:**
```yaml
description: Helps with AILANG development.
```

**✅ GOOD - Clear triggers:**
```yaml
description: Write and run AILANG code with correct syntax. Use when user asks to write AILANG programs, fix AILANG syntax errors, or run AILANG code.
```

**❌ BAD - Missing when to use:**
```yaml
description: Creates releases with version bumps and changelog updates.
```

**✅ GOOD - Includes triggers:**
```yaml
description: Create new AILANG releases with version bumps, changelog updates, git tags, and CI/CD verification. Use when user says "ready to release", "create release", or mentions version numbers.
```

## Progressive Disclosure

### The Three Levels

1. **Always Loaded** - SKILL.md core content (~200-300 lines)
   - YAML frontmatter
   - Quick start
   - When to use
   - Overview of available scripts/resources
   - Basic workflow

2. **Execute as Needed** - Scripts
   - Never loaded into context
   - Only execution output is shown
   - Deterministic, fast, reusable

3. **Load on Demand** - Resources
   - Loaded when detailed reference needed
   - Templates, checklists, detailed guides
   - Keeps always-loaded content minimal

### Example Breakdown

**use-ailang skill (before progressive disclosure):**
- Single file: 498 lines
- All content always loaded

**use-ailang skill (after progressive disclosure):**
- SKILL.md: 288 lines (always loaded) - 42% reduction
- resources/syntax_quick_ref.md: 131 lines (load on demand)
- resources/common_patterns.md: 218 lines (load on demand)
- scripts/check_version.sh: runs without loading
- scripts/validate_code.sh: runs without loading

**Result**: 42% context savings with more capability.

### Writing for Progressive Disclosure

**In SKILL.md (always loaded):**
- Keep it concise and high-level
- Link to resources for details
- Describe scripts but don't show their code
- Focus on workflow and when to use

**In resources/ (load on demand):**
- Detailed references
- Complete examples
- Templates and checklists
- Anything that's "nice to have" but not essential

**In scripts/ (execute, never load):**
- Automation that can be scripted
- Validation and checking
- Report generation
- Any deterministic operations

## Creating Scripts

### Script Basics

**Location**: `.claude/skills/my-skill/scripts/`

**Requirements:**
- Must be executable: `chmod +x script.sh`
- Should handle errors: `set -euo pipefail`
- Should provide clear output
- Exit codes: 0 for success, non-zero for failure

### Script Template

```bash
#!/usr/bin/env bash
# Brief description of what this script does

set -euo pipefail  # Fail on errors, undefined vars, pipe failures

# Parse arguments
if [[ $# -eq 0 ]]; then
    echo "Usage: $0 <arg1> [arg2]" >&2
    echo "Description of what this script does" >&2
    exit 1
fi

ARG1="$1"
ARG2="${2:-default}"  # Optional with default

# Main logic
echo "Running checks..."

FAILURES=0

# Check 1
echo "1/3 Checking something..."
if some_command; then
    echo "  ✓ Check 1 passed"
else
    echo "  ✗ Check 1 failed"
    FAILURES=$((FAILURES + 1))
fi

# Check 2
echo "2/3 Checking something else..."
if another_command; then
    echo "  ✓ Check 2 passed"
else
    echo "  ✗ Check 2 failed"
    FAILURES=$((FAILURES + 1))
fi

# Summary
if [[ $FAILURES -eq 0 ]]; then
    echo "✓ All checks passed!"
    exit 0
else
    echo "✗ $FAILURES check(s) failed"
    exit 1
fi
```

### Script Best Practices

1. **Clear output** - Use emoji (✓, ✗, ⚠) and formatting
2. **Actionable errors** - Tell user how to fix issues
3. **Exit codes** - 0 for success, non-zero for failure
4. **Logging** - Save detailed output to /tmp for debugging
5. **Idempotent** - Safe to run multiple times
6. **Fast** - Don't wait for user input, fail fast

### Example - Validation Script

```bash
#!/usr/bin/env bash
# Validate prerequisites before starting work

set -euo pipefail

echo "Validating prerequisites..."
FAILURES=0

# Test suite
if make test > /tmp/prereq_test.log 2>&1; then
    echo "  ✓ Tests pass"
else
    echo "  ✗ Tests fail - see /tmp/prereq_test.log"
    FAILURES=$((FAILURES + 1))
fi

# Linting
if make lint > /tmp/prereq_lint.log 2>&1; then
    echo "  ✓ Linting passes"
else
    echo "  ✗ Linting fails - see /tmp/prereq_lint.log"
    FAILURES=$((FAILURES + 1))
fi

[[ $FAILURES -eq 0 ]] && echo "✓ Ready!" && exit 0
echo "✗ Fix $FAILURES issue(s) first" && exit 1
```

## Creating Resources

### Resource Purpose

Resources are markdown files loaded on demand for detailed reference.

**Good uses:**
- Templates (e.g., sprint_plan_template.md)
- Quick references (e.g., syntax_quick_ref.md)
- Checklists (e.g., release_checklist.md)
- Detailed guides (e.g., common_patterns.md)

**Bad uses:**
- Duplication of SKILL.md content
- Information that's always needed (put in SKILL.md)
- Scripts (use scripts/ directory)

### Resource Template

```markdown
# Resource Title

Brief description of what this resource provides.

## Section 1

Content here.

## Section 2

More content.

## Examples

Concrete examples.
```

### Referencing Resources in SKILL.md

```markdown
## Resources

### Template Name
See [`resources/template.md`](resources/template.md) for detailed template structure.

### Reference Guide
See [`resources/reference.md`](resources/reference.md) for comprehensive reference.
```

## Testing Skills

### Manual Testing

1. **Create test scenario**: Describe task that should trigger skill
2. **Invoke**: Ask Claude to perform the task
3. **Verify**: Check that skill was invoked (look for skill-specific output)
4. **Validate**: Ensure workflow worked correctly

### Example Test

```
User: "Ready to release v0.3.14"

Expected:
- release-manager skill invoked
- Pre-release checks run
- Version updates proposed
- Git tag created
- CI/CD monitored

Actual:
[Verify each step occurred]
```

### Testing Scripts Directly

```bash
# Test script in isolation
.claude/skills/my-skill/scripts/my_script.sh test_arg

# Verify exit code
echo $?  # Should be 0 for success

# Test with various inputs
.claude/skills/my-skill/scripts/my_script.sh ""  # Empty
.claude/skills/my-skill/scripts/my_script.sh "valid"  # Valid
.claude/skills/my-skill/scripts/my_script.sh "invalid"  # Invalid
```

## Best Practices

### 1. Keep SKILL.md Concise

**Target**: 200-300 lines for always-loaded content

**Techniques:**
- Move details to resources/
- Use scripts for automation
- Link to external resources
- Focus on workflow, not reference

### 2. Write Clear Descriptions

**Description should answer:**
- What does this skill do?
- When should it be used?
- What are the key triggers?

**Good description formula:**
```
[Action verb] + [what it does] + "Use when" + [triggers]
```

### 3. Use Scripts for Automation

**When to script:**
- Validation/checking operations
- Report generation
- Data extraction
- Deterministic transformations

**When NOT to script:**
- Complex decision-making (let Claude handle it)
- Operations requiring context understanding
- Tasks that vary significantly each time

### 4. Progressive Disclosure

**Always loaded (SKILL.md):**
- Essential workflow
- When to use
- Overview of capabilities

**On demand (resources/):**
- Detailed examples
- Complete references
- Templates and checklists

**Never loaded (scripts/):**
- Automation
- Validation
- Report generation

### 5. Test Thoroughly

**Before committing:**
- Test scripts with various inputs
- Verify skill triggers correctly
- Check progressive disclosure works
- Validate resource links

### 6. Document Assumptions

**Make explicit:**
- Prerequisites
- Dependencies
- Environment requirements
- Expected state

### 7. Version with AILANG

**As AILANG evolves:**
- Update skills to match new features
- Keep syntax references current
- Update examples
- Maintain compatibility

## Examples

### Minimal Skill (No Scripts/Resources)

```yaml
---
name: Simple Task
description: Performs a simple task. Use when user asks for simple task.
---

# Simple Task

## Quick Start

Do the task in these steps.

## When to Use

Use when user says "do simple task".

## Workflow

1. Step 1
2. Step 2
3. Done
```

### Skill with Scripts

```yaml
---
name: Validation Task
description: Validates code quality. Use when user wants to check code quality.
---

# Validation Task

## Available Scripts

### `scripts/validate.sh`
Run validation checks.

## Workflow

1. Run validation script
2. Review results
3. Fix issues if any
```

### Skill with Resources

```yaml
---
name: Template Task
description: Creates documents from templates. Use when user needs to create document.
---

# Template Task

## Resources

### Document Template
See [`resources/template.md`](resources/template.md) for document structure.

## Workflow

1. Load template
2. Fill in details
3. Generate document
```

### Complete Skill (Scripts + Resources)

See `use-ailang/`, `release-manager/`, or `post-release/` for complete examples.

## Common Patterns

### Pattern 1: Validation Skill

**Purpose**: Check prerequisites before work

**Structure:**
- SKILL.md: Workflow
- scripts/validate.sh: Run checks
- resources/checklist.md: Manual verification items

**Example**: sprint-executor (validate_prerequisites.sh)

### Pattern 2: Automation Skill

**Purpose**: Automate complex workflows

**Structure:**
- SKILL.md: Overall workflow
- scripts/step1.sh: First automation step
- scripts/step2.sh: Second automation step
- resources/manual_steps.md: Steps that can't be automated

**Example**: post-release (run_eval_baseline.sh, update_dashboard.sh)

### Pattern 3: Reference Skill

**Purpose**: Provide knowledge/guidance

**Structure:**
- SKILL.md: Overview and when to use
- resources/quick_ref.md: Quick reference
- resources/detailed_guide.md: Comprehensive guide
- scripts/check_version.sh: Version checking

**Example**: use-ailang (syntax_quick_ref.md, common_patterns.md)

## Troubleshooting

### Skill Not Invoked

**Problem**: Claude doesn't use skill when expected

**Solutions:**
- Check description includes clear triggers
- Verify YAML frontmatter is valid
- Ensure description mentions when to use
- Test with explicit request

### Script Fails

**Problem**: Script exits with error

**Solutions:**
- Check script is executable (`chmod +x`)
- Verify bash shebang (`#!/usr/bin/env bash`)
- Test script in isolation
- Check error output in /tmp logs

### Resource Not Loading

**Problem**: Resource file not found

**Solutions:**
- Verify file path is correct
- Check markdown link syntax
- Ensure file exists in resources/
- Test link from SKILL.md

## References

- [Anthropic Agent Skills Documentation](https://docs.claude.com/en/docs/agents-and-tools/agent-skills/)
- [anthropics/skills GitHub Repository](https://github.com/anthropics/skills)
- [AILANG Skills README](.claude/skills/README.md)
- [Anthropic Skills Announcement](https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills)
