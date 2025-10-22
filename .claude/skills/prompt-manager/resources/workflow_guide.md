# Prompt Manager - Detailed Workflow Guide

This resource file contains detailed examples, troubleshooting, and best practices. See `skill.md` for the main workflow.

## Detailed Workflows

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

## Script Output Examples

### create_prompt_version.sh

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

### update_hash.sh

```
Updating hash for v0.3.17
  File: prompts/v0.3.17.md
  Old hash: f6e1e7a39e35aa7a7116d11f8fb9d01b1be7862b0723a1ed3d8ed8b76030501d
  New hash: a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0e1f2
✓ Updated hash in prompts/versions.json
```

### verify_prompt_accuracy.sh

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

## Testing Guide

### Quick REPL Test
```bash
ailang repl
> :type httpRequest  # Check if documented features work
```

### Example Code Test
```bash
ailang run --caps Net,IO examples/api_call.ail
```

### Eval Benchmark Testing

**Recommended workflow:**

```bash
# First: Quick test with one model (~2-3 min)
ailang eval-suite --models gpt5-mini --output eval_results/test_v0.3.17_quick

# If quick test passes: Run with all dev models (~7-8 min)
ailang eval-suite --output eval_results/test_v0.3.17_dev

# Compare results
ailang eval-compare eval_results/baselines/v0.3.16 eval_results/test_v0.3.17_dev
```

**Why test with eval-suite:**
- Catches regressions in prompt quality
- Validates that documented features work in practice
- Shows if success rates improve (e.g., fixing httpRequest should improve api_call_json benchmark)
- Quick feedback loop before committing

## Versioning Best Practices

### Semantic Versioning

- **PATCH** (v0.3.16 → v0.3.17): Bug fixes, typo corrections, clarifications
- **MINOR** (v0.3.17 → v0.4.0): New features documented, major structural changes
- **MAJOR** (v0.4.0 → v1.0.0): Breaking changes in prompt structure

### Description Guidelines

**Good descriptions:**
- "Fix httpRequest documentation and remove false HTTP headers limitation"
- "Add higher-order functions and list operations"
- "Update import syntax and add module examples"

**Bad descriptions:**
- "Update prompt" (too vague)
- "Various fixes" (not specific)
- "WIP" (never commit WIP prompts)

## Common Edits

### Remove False Limitations
Search for patterns like:
- "⚠️ NO custom HTTP headers"
- "❌ Cannot ..."
- "Not supported:"

Replace with:
- "✅ HTTP headers - httpRequest() with custom headers (since vX.Y.Z)"
- Add examples
- Update quick reference

### Add New Features
1. Find appropriate section (e.g., "Available Builtins")
2. Add feature with signature and description
3. Add 2-3 working examples
4. Add to import checklist if it's a stdlib function
5. Add to quick reference at bottom

### Fix Incorrect Examples
1. Test example code in REPL first
2. Fix syntax errors
3. Update expected output
4. Verify with `ailang run` if using effects

## Integration with Other Skills

### With eval-analyzer
- Use `verify_prompt_accuracy.sh` to catch prompt bugs
- Run after creating new prompt version
- Fix issues before running eval baselines

### With post-release
- Run baselines after prompt changes to validate improvements
- Update dashboard if success rates improve significantly

### With release-manager
- New prompt versions often accompany new releases
- Update prompt before release, not after
- Include prompt version in release notes

## Troubleshooting

### Hash Mismatch Error
**Problem:** Git shows changes to versions.json even after running update_hash.sh

**Solution:** You edited the prompt file after running update_hash.sh. Run it again:
```bash
.claude/skills/prompt-manager/scripts/update_hash.sh v0.3.17
```

### Verification Script Fails
**Problem:** verify_prompt_accuracy.sh reports false limitations

**Solution:**
1. Read the error message carefully
2. Search for the reported text in the prompt file
3. Remove or update the limitation
4. Run update_hash.sh
5. Re-run verification

### Examples Don't Work
**Problem:** Documented examples fail in REPL

**Solution:**
1. Test the example yourself in REPL
2. Check for syntax errors or missing imports
3. Verify the feature actually works (check implementation)
4. Update example with working code
5. Add version note (e.g., "since v0.3.9")

## Prompt Structure Guidelines

### Required Sections
1. **Quick Reference** - Syntax cheat sheet at top
2. **Available Builtins** - Complete list with types
3. **Limitations** - Current restrictions (accurate!)
4. **Examples** - Working code snippets
5. **Import Checklist** - Common mistakes to avoid

### Best Practices
- Use concrete examples, not just descriptions
- Include expected output for examples
- Mark version when features were added
- Use ✅ for supported, ⚠️ for partial, ❌ for unsupported
- Keep limitation list accurate (remove false ones!)
- Test all examples before committing

### Common Pitfalls to Avoid
- Saying "NO X" when X actually exists (verify first!)
- Outdated examples from old syntax
- Missing version notes for new features
- Vague descriptions without code examples
- Copy-paste errors from other sections
