#!/usr/bin/env bash
# Validate that a skill follows Anthropic Agent Skills specification

set -euo pipefail

if [[ $# -eq 0 ]]; then
    echo "Usage: $0 <skill-name>" >&2
    echo "Example: $0 my-skill" >&2
    exit 1
fi

SKILL_NAME="$1"
SKILL_DIR=".claude/skills/$SKILL_NAME"

echo "Validating skill: $SKILL_NAME"
echo

FAILURES=0

# 1. Check directory exists
echo "1/7 Checking skill directory..."
if [[ -d "$SKILL_DIR" ]]; then
    echo "  ✓ Directory exists: $SKILL_DIR"
else
    echo "  ✗ Directory not found: $SKILL_DIR"
    exit 1
fi
echo

# 2. Check SKILL.md exists
echo "2/7 Checking SKILL.md..."
if [[ -f "$SKILL_DIR/SKILL.md" ]]; then
    echo "  ✓ SKILL.md exists"
else
    echo "  ✗ SKILL.md not found"
    FAILURES=$((FAILURES + 1))
fi
echo

# 3. Check YAML frontmatter
echo "3/7 Checking YAML frontmatter..."
if head -1 "$SKILL_DIR/SKILL.md" | grep -q '^---$'; then
    echo "  ✓ YAML frontmatter present"

    # Check for required fields
    if grep -q '^name:' "$SKILL_DIR/SKILL.md"; then
        echo "  ✓ 'name' field present"
    else
        echo "  ✗ 'name' field missing"
        FAILURES=$((FAILURES + 1))
    fi

    if grep -q '^description:' "$SKILL_DIR/SKILL.md"; then
        echo "  ✓ 'description' field present"
    else
        echo "  ✗ 'description' field missing"
        FAILURES=$((FAILURES + 1))
    fi
else
    echo "  ✗ YAML frontmatter missing (must start with '---')"
    FAILURES=$((FAILURES + 1))
fi
echo

# 4. Check scripts are executable
echo "4/7 Checking scripts..."
if [[ -d "$SKILL_DIR/scripts" ]]; then
    SCRIPT_COUNT=$(find "$SKILL_DIR/scripts" -type f -name "*.sh" | wc -l | tr -d ' ')
    if [[ $SCRIPT_COUNT -gt 0 ]]; then
        echo "  ✓ Found $SCRIPT_COUNT script(s)"

        NON_EXECUTABLE=0
        while IFS= read -r script; do
            if [[ -x "$script" ]]; then
                echo "  ✓ $(basename "$script") is executable"
            else
                echo "  ✗ $(basename "$script") is not executable (run: chmod +x $script)"
                NON_EXECUTABLE=$((NON_EXECUTABLE + 1))
            fi
        done < <(find "$SKILL_DIR/scripts" -type f -name "*.sh")

        if [[ $NON_EXECUTABLE -gt 0 ]]; then
            FAILURES=$((FAILURES + 1))
        fi
    else
        echo "  ⚠ No scripts found (optional)"
    fi
else
    echo "  ⚠ scripts/ directory not found (optional)"
fi
echo

# 5. Check resources exist
echo "5/7 Checking resources..."
if [[ -d "$SKILL_DIR/resources" ]]; then
    RESOURCE_COUNT=$(find "$SKILL_DIR/resources" -type f -name "*.md" | wc -l | tr -d ' ')
    if [[ $RESOURCE_COUNT -gt 0 ]]; then
        echo "  ✓ Found $RESOURCE_COUNT resource file(s)"
    else
        echo "  ⚠ No resources found (optional)"
    fi
else
    echo "  ⚠ resources/ directory not found (optional)"
fi
echo

# 6. Check SKILL.md structure
echo "6/7 Checking SKILL.md structure..."
REQUIRED_SECTIONS=("Quick Start" "When to Use" "Workflow")
for section in "${REQUIRED_SECTIONS[@]}"; do
    if grep -q "## $section" "$SKILL_DIR/SKILL.md"; then
        echo "  ✓ '$section' section present"
    else
        echo "  ⚠ '$section' section recommended"
    fi
done
echo

# 7. Check file sizes (progressive disclosure)
echo "7/7 Checking file sizes (progressive disclosure)..."
SKILL_MD_LINES=$(wc -l < "$SKILL_DIR/SKILL.md" | tr -d ' ')
if [[ $SKILL_MD_LINES -le 300 ]]; then
    echo "  ✓ SKILL.md is $SKILL_MD_LINES lines (target: ≤300)"
elif [[ $SKILL_MD_LINES -le 500 ]]; then
    echo "  ⚠ SKILL.md is $SKILL_MD_LINES lines (consider moving content to resources/)"
else
    echo "  ✗ SKILL.md is $SKILL_MD_LINES lines (should be ≤300, move content to resources/)"
    FAILURES=$((FAILURES + 1))
fi
echo

# Summary
if [[ $FAILURES -eq 0 ]]; then
    echo "✓ Skill '$SKILL_NAME' is valid!"
    echo "Ready to use."
    exit 0
else
    echo "✗ Skill '$SKILL_NAME' has $FAILURES issue(s)"
    echo "Fix issues before using."
    exit 1
fi
