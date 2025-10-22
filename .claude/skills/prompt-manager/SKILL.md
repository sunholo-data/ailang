---
name: Prompt Manager
description: Manage AILANG teaching prompts. Use when user asks to create new prompt version, update prompt, fix prompt documentation, or verify prompt accuracy against implementation.
---

# Prompt Manager

Manage AILANG teaching prompt versions with automated accuracy verification and version control.

## Quick Start

**Most common usage:**
```bash
# User says: "Create a new prompt version fixing the httpRequest documentation"
# This skill will:
# 1. Create new prompt version from base
# 2. Guide you through edits
# 3. Verify prompt accuracy against implementation
# 4. Update versions.json with new hash
```

## When to Use This Skill

Invoke this skill when:
- User asks to "create new prompt" or "update prompt"
- User wants to "fix prompt documentation" or "add feature to prompt"
- After implementing new language features (update prompt to match)
- Before running eval baselines (verify prompt accuracy first!)
- User mentions "prompt-code mismatch" or "false limitation"

## Available Scripts

### `scripts/create_prompt_version.sh`

Create a new prompt version based on an existing version.

**Usage:**
```bash
.claude/skills/prompt-manager/scripts/create_prompt_version.sh <new_version> <base_version> "<description>"
```

**Example:**
```bash
.claude/skills/prompt-manager/scripts/create_prompt_version.sh v0.3.17 v0.3.16 "Fix httpRequest documentation and remove false HTTP headers limitation"
```

**What it does:**
1. Validates version format (vX.Y.Z)
2. Checks base version exists
3. Copies base prompt to new file
4. Computes SHA256 hash
5. Adds entry to `prompts/versions.json`
6. Sets new version as active

**Output:**
```
Creating new prompt version: v0.3.17
  Base: v0.3.16 (prompts/v0.3.16.md)
  New:  prompts/v0.3.17.md
  Description: Fix httpRequest documentation

✓ Copied prompts/v0.3.16.md → prompts/v0.3.17.md
✓ Computed hash: f6e1e7a39e35aa7a7116d11f8fb9d01b1be7862b0723a1ed3d8ed8b76030501d
✓ Added v0.3.17 to prompts/versions.json
✓ Set as active version

Next steps:
  1. Edit prompts/v0.3.17.md to make your changes
  2. Run .claude/skills/eval-analyzer/scripts/verify_prompt_accuracy.sh v0.3.17
  3. Update hash: .claude/skills/prompt-manager/scripts/update_hash.sh v0.3.17
  4. Test with: ailang repl (check if changes work)
  5. Commit: git add prompts/v0.3.17.md prompts/versions.json
```

### `scripts/update_hash.sh`

Update SHA256 hash in versions.json after editing a prompt file.

**Usage:**
```bash
.claude/skills/prompt-manager/scripts/update_hash.sh <version>
```

**Example:**
```bash
.claude/skills/prompt-manager/scripts/update_hash.sh v0.3.17
```

**What it does:**
1. Reads prompt file path from versions.json
2. Computes new SHA256 hash
3. Updates hash in versions.json
4. Reports old vs new hash

**Output:**
```
Updating hash for v0.3.17
  File: prompts/v0.3.17.md
  Old hash: f6e1e7a39e35aa7a7116d11f8fb9d01b1be7862b0723a1ed3d8ed8b76030501d
  New hash: a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0e1f2
✓ Updated hash in prompts/versions.json
```

## Workflow

### 1. Create New Prompt Version

Use when adding features or fixing bugs in the teaching prompt.

```bash
# Create from current active version
.claude/skills/prompt-manager/scripts/create_prompt_version.sh v0.3.17 v0.3.16 "Your changes description"
```

**Tips:**
- Use semantic versioning (vMAJOR.MINOR.PATCH)
- Increment PATCH for bug fixes (v0.3.16 → v0.3.17)
- Increment MINOR for new features (v0.3.17 → v0.4.0)
- Keep description concise but specific

### 2. Edit the Prompt File

Make your changes to the new prompt file:

```bash
# Open in editor
code prompts/v0.3.17.md  # or vim, etc.
```

**Common edits:**
- Remove false limitations (e.g., "NO custom HTTP headers")
- Add documentation for new features
- Update examples with working code
- Fix incorrect syntax examples
- Add new imports to checklist

### 3. Verify Prompt Accuracy

**Critical step** - catches prompt-code mismatches:

```bash
.claude/skills/eval-analyzer/scripts/verify_prompt_accuracy.sh v0.3.17
```

**What it checks:**
- False limitations (prompt says NO but feature exists)
- Undocumented features (feature exists but not in prompt)
- Prompt-implementation mismatches

**Example output:**
```
=== Prompt Accuracy Check for v0.3.17 ===
Prompt file: prompts/v0.3.17.md

--- Checking for False Limitations ---
✓ No false HTTP headers limitation found

--- Checking for Undocumented Features ---
✓ httpRequest() exists in stdlib
✓ httpRequest() documented in prompt

--- Summary ---
✓ No prompt accuracy issues found!
```

**If issues found:**
- Edit prompt to fix false limitations
- Add documentation for undocumented features
- Re-run verification until clean

### 4. Update Hash

After making edits, update the hash:

```bash
.claude/skills/prompt-manager/scripts/update_hash.sh v0.3.17
```

**Why:** Hash integrity ensures prompt content matches metadata.

### 5. Test Changes

Test that the prompt works correctly:

```bash
# Quick syntax test in REPL
ailang repl
> :type httpRequest  # Check if documented features work

# Test with example code
ailang run --caps Net,IO examples/api_call.ail

# Test with eval benchmarks (RECOMMENDED)
# First: Quick test with one model (~2-3 min)
ailang eval-suite --models gpt5-mini --output eval_results/test_v0.3.17_quick

# If quick test passes: Run with all dev models (~7-8 min)
ailang eval-suite --output eval_results/test_v0.3.17_dev

# Compare results
ailang eval-compare eval_results/baselines/0.3.16 eval_results/test_v0.3.17_dev
```

**Why test with eval-suite:**
- Catches regressions in prompt quality
- Validates that documented features work in practice
- Shows if success rates improve (e.g., fixing httpRequest should improve api_call_json benchmark)
- Quick feedback loop before committing

### 6. Commit Changes

```bash
git add prompts/v0.3.17.md prompts/versions.json
git commit -m "feat: Add v0.3.17 prompt with httpRequest documentation

- Remove false 'NO custom HTTP headers' limitation
- Add httpRequest() documentation with examples
- Update import checklist with httpRequest
- Fix prompt-code mismatch found by verify_prompt_accuracy.sh"
```

## Common Tasks

### Fix False Limitation

**Scenario:** Feature exists but prompt says it doesn't

**Steps:**
1. Create new prompt version
2. Search for false limitation text
3. Remove limitation from list
4. Add feature to capabilities section
5. Add usage examples
6. Verify with `verify_prompt_accuracy.sh`
7. Update hash and commit

**Example:**
```bash
# Step 1
.claude/skills/prompt-manager/scripts/create_prompt_version.sh v0.3.17 v0.3.16 "Remove false HTTP headers limitation"

# Step 2-5: Edit prompts/v0.3.17.md
# Remove: "⚠️ NO custom HTTP headers"
# Add: "✅ HTTP headers - httpRequest() with custom headers (since v0.3.9)"
# Add examples from v0.3.9 prompt

# Step 6
.claude/skills/eval-analyzer/scripts/verify_prompt_accuracy.sh v0.3.17

# Step 7
.claude/skills/prompt-manager/scripts/update_hash.sh v0.3.17
git add prompts/v0.3.17.md prompts/versions.json
git commit -m "fix: Remove false HTTP headers limitation from prompt"
```

### Add New Feature Documentation

**Scenario:** Implemented new builtin/feature, need to document in prompt

**Steps:**
1. Create new prompt version
2. Add to capabilities list
3. Add to import checklist (if applicable)
4. Add usage examples
5. Add to quick reference section
6. Verify and commit

### Update Active Version

**Scenario:** Switch back to older prompt for testing

```bash
# Manually edit prompts/versions.json
jq '.active = "v0.3.16"' prompts/versions.json > tmp.json && mv tmp.json prompts/versions.json
```

## Resources

### Prompt Structure Guide

See [`resources/prompt_structure.md`](resources/prompt_structure.md) for:
- Required sections for teaching prompts
- Best practices for examples
- How to document limitations
- Common pitfalls to avoid

## Progressive Disclosure

This skill loads information progressively:

1. **Always loaded**: This SKILL.md file (workflow + scripts)
2. **Execute as needed**: `create_prompt_version.sh`, `update_hash.sh`
3. **Load on demand**: `resources/prompt_structure.md` (detailed guide)

## Integration

Works with:
- **eval-analyzer** skill - `verify_prompt_accuracy.sh` catches prompt bugs
- **post-release** skill - Run baselines after prompt changes
- `prompts/versions.json` - Version registry with hashes
- `ailang eval-suite` - Test prompt effectiveness

## Notes

- Always verify prompt accuracy before running eval baselines
- Use `verify_prompt_accuracy.sh` to catch documentation bugs
- Keep prompt file sizes reasonable (<2000 lines)
- Include concrete examples, not just descriptions
- Test prompts with REPL before committing
- Hash must be updated after any edits
