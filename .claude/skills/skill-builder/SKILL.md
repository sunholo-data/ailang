---
name: Skill Builder
description: Create new Anthropic Agent Skills with proper structure, validation, and best practices. Use when user asks to create a new skill, add skill functionality, or wants to build a custom workflow.
---

# Skill Builder

Create new Anthropic Agent Skills following the October 2025 specification.

## Quick Start

**Most common usage:**
```bash
# User says: "Create a skill for managing database migrations"
# This skill will:
# 1. Run create_skill.sh to scaffold the structure
# 2. Guide you through customizing SKILL.md
# 3. Help create scripts and resources
# 4. Validate the final skill
```

## When to Use This Skill

Invoke this skill when:
- User asks to "create a skill" or "make a new skill"
- User wants to add custom workflow automation
- User says "I need a skill for [specific task]"
- User wants to build reusable capabilities

## Available Scripts

### `scripts/create_skill.sh <skill-name> <description>`
Create a new skill with proper directory structure and template files.

**Usage:**
```bash
.claude/skills/skill-builder/scripts/create_skill.sh my-skill "Does something useful. Use when user asks for X."
```

**What it creates:**
- `skill-name/SKILL.md` with YAML frontmatter
- `skill-name/scripts/example_script.sh` (executable)
- `skill-name/resources/reference.md`

**Output:**
```
Creating new skill: my-skill
Description: Does something useful. Use when user asks for X.

✓ Created skill structure:
  .claude/skills/my-skill/
  ├── SKILL.md
  ├── scripts/
  │   └── example_script.sh
  └── resources/
      └── reference.md

Next steps:
  1. Edit .claude/skills/my-skill/SKILL.md to customize
  2. Implement scripts in .claude/skills/my-skill/scripts/
  3. Add resources to .claude/skills/my-skill/resources/
  4. Update .claude/skills/README.md to list the new skill
  5. Test the skill by asking Claude to use it
```

### `scripts/validate_skill.sh <skill-name>`
Validate that a skill follows Anthropic Agent Skills specification.

**Usage:**
```bash
.claude/skills/skill-builder/scripts/validate_skill.sh my-skill
```

**What it checks:**
1. Skill directory exists
2. SKILL.md exists
3. YAML frontmatter present (name + description)
4. Scripts are executable
5. Resources exist (if any)
6. SKILL.md has required sections
7. File sizes follow progressive disclosure (≤300 lines)

**Output:**
```
Validating skill: my-skill

1/7 Checking skill directory...
  ✓ Directory exists: .claude/skills/my-skill

2/7 Checking SKILL.md...
  ✓ SKILL.md exists

3/7 Checking YAML frontmatter...
  ✓ YAML frontmatter present
  ✓ 'name' field present
  ✓ 'description' field present

4/7 Checking scripts...
  ✓ Found 1 script(s)
  ✓ example_script.sh is executable

5/7 Checking resources...
  ✓ Found 1 resource file(s)

6/7 Checking SKILL.md structure...
  ✓ 'Quick Start' section present
  ✓ 'When to Use' section present
  ✓ 'Workflow' section present

7/7 Checking file sizes (progressive disclosure)...
  ✓ SKILL.md is 245 lines (target: ≤300)

✓ Skill 'my-skill' is valid!
Ready to use.
```

## Workflow

### 1. Gather Requirements

**Ask user:**
- What should the skill do?
- What are the key triggers/use cases?
- Does it need automation (scripts)?
- Does it need detailed references (resources)?

### 2. Create Skill Structure

**Use the create script:**
```bash
.claude/skills/skill-builder/scripts/create_skill.sh <skill-name> "<description>"
```

**Naming conventions:**
- Use lowercase with hyphens (e.g., `database-manager`)
- Keep names concise and descriptive
- Avoid generic names like `helper` or `utility`

**Description requirements:**
- Brief (max 1024 chars)
- Include what it does
- **Must include when to use it** (triggers)
- Example: "Manages database migrations. Use when user asks to create migration, run migrations, or rollback database."

### 3. Customize SKILL.md

**Edit the generated SKILL.md:**

**Update frontmatter:**
- Ensure `name` is in Title Case
- Ensure `description` includes clear triggers

**Customize sections:**
- **Quick Start**: Show most common usage pattern
- **When to Use**: List clear invocation triggers
- **Workflow**: Step-by-step instructions
- **Available Scripts**: Describe what each script does
- **Resources**: Link to detailed references

**Keep it concise:**
- Target: ≤300 lines for always-loaded content
- Move detailed info to resources/
- Focus on workflow and when to use

### 4. Implement Scripts (Optional)

**If skill needs automation:**

**Create script in `scripts/` directory:**
```bash
touch .claude/skills/my-skill/scripts/my_script.sh
chmod +x .claude/skills/my-skill/scripts/my_script.sh
```

**Script best practices:**
- Start with `#!/usr/bin/env bash`
- Use `set -euo pipefail`
- Clear output with ✓, ✗, ⚠ symbols
- Exit 0 for success, non-zero for failure
- Provide helpful error messages
- Log details to /tmp for debugging

**See:** [resources/skill_template.md](resources/skill_template.md) for script template

### 5. Add Resources (Optional)

**If skill needs detailed reference:**

**Create resource in `resources/` directory:**
```bash
touch .claude/skills/my-skill/resources/reference.md
```

**Resource types:**
- Templates (e.g., config templates, plan templates)
- Quick references (e.g., syntax guides, checklists)
- Detailed guides (e.g., comprehensive how-tos)
- Examples (e.g., sample outputs, use cases)

**Keep resources focused:**
- One resource per topic
- Use clear section headings
- Include concrete examples
- Link back from SKILL.md

### 6. Validate Skill

**Run validation:**
```bash
.claude/skills/skill-builder/scripts/validate_skill.sh my-skill
```

**Fix any issues:**
- Missing YAML frontmatter → Add to SKILL.md
- Non-executable scripts → `chmod +x script.sh`
- SKILL.md too long → Move content to resources/
- Missing sections → Add required sections

### 7. Test Scripts

**Test each script independently:**
```bash
.claude/skills/my-skill/scripts/my_script.sh test_arg
echo $?  # Should be 0 for success

# Test with various inputs
.claude/skills/my-skill/scripts/my_script.sh ""       # Empty
.claude/skills/my-skill/scripts/my_script.sh "valid"  # Valid
.claude/skills/my-skill/scripts/my_script.sh "error"  # Error case
```

### 8. Update README

**Add skill to `.claude/skills/README.md`:**

Under appropriate category:
```markdown
**[my-skill/](my-skill/)** - Brief description
- Key capability 1
- Key capability 2
- **Scripts**: script1.sh, script2.sh
- **Resources**: reference.md
```

### 9. Test Skill in Practice

**Ask Claude to use the skill:**
```
User: "I need to manage database migrations"
→ Claude should invoke my-skill

Verify:
- Skill was invoked correctly
- Workflow makes sense
- Scripts execute properly
- Resources load when needed
```

### 10. Iterate

**Based on testing:**
- Refine triggers in description
- Clarify workflow steps
- Improve script output
- Add missing resources
- Optimize for progressive disclosure

## Resources

### Skill Template
See [resources/skill_template.md](resources/skill_template.md) for:
- YAML frontmatter template
- SKILL.md structure template
- Script template
- Resource template
- Validation checklist

## Best Practices

### 1. Clear Triggers

**Good description (includes triggers):**
```yaml
description: Manages database migrations. Use when user asks to create migration, run migrations, or rollback database.
```

**Bad description (no triggers):**
```yaml
description: Handles database stuff.
```

### 2. Progressive Disclosure

**Always-loaded (SKILL.md):**
- Overview and when to use
- Workflow steps
- Script descriptions
- Links to resources

**On-demand (resources/):**
- Detailed examples
- Complete references
- Templates
- Comprehensive guides

**Never-loaded (scripts/):**
- Automation
- Validation
- Report generation

### 3. Script Naming

**Good names:**
- `validate_prerequisites.sh` - Clear what it does
- `update_dashboard.sh` - Action-oriented
- `extract_metrics.sh` - Specific purpose

**Bad names:**
- `script.sh` - Too vague
- `helper.sh` - Not descriptive
- `utils.sh` - Generic

### 4. Documentation

**Document in SKILL.md:**
- What scripts do (not how they work)
- When to use resources
- Prerequisites and assumptions

**Document in scripts:**
- Header comment explaining purpose
- Usage examples in help text
- Error messages explaining how to fix

**Document in resources:**
- Detailed how-tos
- Complete examples
- Background information

### 5. Testing

**Test at each stage:**
- After creating structure → validate_skill.sh
- After implementing scripts → test with various inputs
- After customizing → ask Claude to use it
- After updates → re-validate and re-test

## Common Patterns

### Pattern 1: Automation Skill

**Purpose**: Automate complex workflows

**Structure:**
- SKILL.md: Workflow overview
- scripts/: Multiple automation scripts
- resources/: Manual steps that can't be automated

**Example**: post-release skill

### Pattern 2: Reference Skill

**Purpose**: Provide knowledge/guidance

**Structure:**
- SKILL.md: Overview and when to use
- scripts/: Validation/checking (optional)
- resources/: Detailed guides, references

**Example**: use-ailang skill

### Pattern 3: Planning Skill

**Purpose**: Create plans/documents

**Structure:**
- SKILL.md: Planning workflow
- scripts/: Analysis helpers (optional)
- resources/: Templates

**Example**: sprint-planner skill

## Progressive Disclosure

This skill loads information progressively:

1. **Always loaded**: This SKILL.md file (YAML frontmatter + workflow)
2. **Execute as needed**: Scripts in `scripts/` (create, validate)
3. **Load on demand**: `resources/skill_template.md` (detailed templates)

## Notes

- Skills are discoverable via YAML frontmatter
- Scripts run without loading into context (saves tokens)
- Resources load only when needed (progressive disclosure)
- Follow Anthropic Agent Skills specification (October 2025)
- See [.claude/skills/SKILLS_GUIDE.md](../SKILLS_GUIDE.md) for comprehensive guide
