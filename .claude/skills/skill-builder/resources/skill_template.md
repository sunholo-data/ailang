# Skill Template

Use this as a starting point for creating new skills.

## YAML Frontmatter Template

```yaml
---
name: Skill Name
description: Brief description of what this skill does and when to use it (max 1024 chars). Use when user asks for [specific triggers].
---
```

## SKILL.md Structure Template

```markdown
---
name: Skill Name
description: Brief description with triggers.
---

# Skill Name

Brief description of what this skill does.

## Quick Start

**Most common usage:**
\`\`\`bash
# Example of typical usage pattern
# User says: "Do X"
# This skill will:
# 1. Step 1
# 2. Step 2
\`\`\`

## When to Use This Skill

Invoke this skill when:
- User asks for [action]
- User mentions [keyword]
- User wants to [goal]

## Available Scripts

### \`scripts/script_name.sh [args]\`
Description of what this script does.

**Usage:**
\`\`\`bash
.claude/skills/skill-name/scripts/script_name.sh arg1 arg2
\`\`\`

**Output:**
\`\`\`
Expected output format
\`\`\`

## Workflow

### 1. First Step

Description and instructions.

### 2. Second Step

Description and instructions.

### 3. Final Step

Description and instructions.

## Resources

### Resource Name
See [\`resources/resource.md\`](resources/resource.md) for detailed reference.

## Progressive Disclosure

This skill loads information progressively:

1. **Always loaded**: This SKILL.md file (YAML frontmatter + overview)
2. **Execute as needed**: Scripts in \`scripts/\` directory
3. **Load on demand**: Resources in \`resources/\` directory

## Notes

- Prerequisites
- Dependencies
- Important caveats
```

## Script Template

```bash
#!/usr/bin/env bash
# Brief description of what this script does

set -euo pipefail

# Parse arguments
if [[ $# -eq 0 ]]; then
    echo "Usage: $0 <arg1> [arg2]" >&2
    echo "Description of what this script does" >&2
    exit 1
fi

ARG1="$1"
ARG2="${2:-default}"

# Main logic
echo "Running script..."

FAILURES=0

# Step 1
echo "1/3 Doing first check..."
if some_command; then
    echo "  ✓ Check passed"
else
    echo "  ✗ Check failed"
    FAILURES=$((FAILURES + 1))
fi

# Summary
if [[ $FAILURES -eq 0 ]]; then
    echo "✓ All steps completed successfully!"
    exit 0
else
    echo "✗ $FAILURES step(s) failed"
    exit 1
fi
```

## Resource Template

```markdown
# Resource Title

Detailed reference information.

## Section 1

Content here.

## Section 2

More content.

## Examples

\`\`\`bash
# Example usage
\`\`\`
```

## Checklist for New Skills

- [ ] Created directory structure (skill-name/scripts/, skill-name/resources/)
- [ ] Created SKILL.md with YAML frontmatter
- [ ] Added 'name' field to frontmatter
- [ ] Added 'description' field with triggers to frontmatter
- [ ] Added "Quick Start" section
- [ ] Added "When to Use This Skill" section
- [ ] Added "Workflow" section
- [ ] Created scripts (if needed) and made them executable
- [ ] Created resources (if needed)
- [ ] Kept SKILL.md ≤300 lines (moved details to resources)
- [ ] Tested scripts in isolation
- [ ] Validated skill with validate_skill.sh
- [ ] Updated .claude/skills/README.md
- [ ] Tested skill by asking Claude to use it
