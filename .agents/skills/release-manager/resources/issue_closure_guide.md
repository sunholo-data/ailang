# Issue Closure Guide

This guide explains how to properly close GitHub issues when making a release, with appropriate references to commits, release URLs, and changelog entries.

## Overview

When releasing a new version, issues should be closed with informative comments that include:
1. **Release version and URL** - Link to the GitHub release
2. **Relevant fix reference** - Which feature/fix in the CHANGELOG addresses the issue
3. **Commit hash** (optional) - The specific commit that fixed the issue

## Workflow

### Step 1: Identify Fixed Issues

Compare CHANGELOG entries against open GitHub issues:

```bash
# List open issues
gh issue list --repo sunholo-data/ailang --state open --json number,title,body

# Read CHANGELOG for the version
# Look for: M-* feature codes, "Fixed" sections, issue references (#123)
```

### Step 2: Match Issues to CHANGELOG Entries

For each CHANGELOG entry, identify which issues it addresses:

| CHANGELOG Entry | Likely Issues |
|-----------------|---------------|
| `### Fixed - Option Type Assertions (M-CODEGEN-OPTION-TYPE-ASSERT)` | Issues mentioning "Option", "type assertion" |
| `### Fixed - Record Literals (M-TYPENAME-NESTED-PROPAGATION)` | Issues mentioning "record", "struct", "map" |
| `### Fixed - Tuple Pattern Matching` | Issues mentioning "tuple", "pattern" |

### Step 3: Get Release and Commit Info

```bash
# Get release URL
gh release view vX.X.X --json url

# Get commit hash for the release tag
git rev-parse vX.X.X

# Find specific commits for a feature
git log --oneline --all --grep="M-FEATURE-CODE"
```

### Step 4: Close Issues with Proper References

Use this format for closing comments:

```markdown
Fixed in [vX.X.X](https://github.com/sunholo-data/ailang/releases/tag/vX.X.X) - Brief description of the fix (M-FEATURE-CODE).

[Optional: More detail about what was fixed]

Commit: [`abc1234`](https://github.com/sunholo-data/ailang/commit/abc1234567890)
```

### Example Closing Comments

**Good - with release link and feature code:**
```markdown
Fixed in [v0.5.10](https://github.com/sunholo-data/ailang/releases/tag/v0.5.10) - TApp/TCon unification now handles `Option[a]` type parameters correctly across modules.
```

**Better - with commit reference:**
```markdown
Fixed in [v0.5.10](https://github.com/sunholo-data/ailang/releases/tag/v0.5.10) - Added `floatToStr` and `intToStr` functions in std/string module (M-STRING-CONVERT).

Usage:
\`\`\`ailang
import std/string (floatToStr)
let velocity = floatToStr(15.0) ++ "% c"
\`\`\`

Commit: [`3ed805c`](https://github.com/sunholo-data/ailang/commit/3ed805cd)
```

**Best - with design doc link:**
```markdown
Fixed in [v0.5.10](https://github.com/sunholo-data/ailang/releases/tag/v0.5.10) - Record literals now compile to struct instantiation instead of `map[string]interface{}` (M-TYPENAME-NESTED-PROPAGATION).

See: [Design Doc](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/v0_5_10/m-typename-nested-propagation.md)
```

## Common Patterns

### Bug Fixes in Codegen
```markdown
Fixed in [vX.X.X](https://github.com/sunholo-data/ailang/releases/tag/vX.X.X) - [Describe what was wrong] now [describe correct behavior] (M-FEATURE-CODE).
```

### New Features/Functions
```markdown
Added in [vX.X.X](https://github.com/sunholo-data/ailang/releases/tag/vX.X.X) - New `functionName` function for [purpose].

Usage:
\`\`\`ailang
import module (functionName)
let result = functionName(args)
\`\`\`
```

### Type System Fixes
```markdown
Fixed in [vX.X.X](https://github.com/sunholo-data/ailang/releases/tag/vX.X.X) - [Type unification/inference] now handles [specific case] correctly.
```

## Automated Closure

The `collect_closable_issues.sh` script can help, but for best results:

1. Run the script to identify issues: `.claude/skills/release-manager/scripts/collect_closable_issues.sh X.X.X`
2. Review suggested issues manually
3. For each issue, find the matching CHANGELOG entry
4. Close with a customized comment using `gh issue close`

```bash
# Close with custom comment
gh issue close 123 --comment "Fixed in [v0.5.10](https://github.com/sunholo-data/ailang/releases/tag/v0.5.10) - Description here."
```

## Checklist

Before closing an issue, verify:

- [ ] The fix is actually in the released version (check CHANGELOG)
- [ ] The closing comment includes the release URL
- [ ] The description matches what the issue was about
- [ ] Any relevant usage examples are included (for features)
- [ ] The feature code (M-*) is referenced if applicable

## GitHub Auto-Close

Alternatively, include issue references in commit messages to auto-close:

```bash
git commit -m "Fix type assertion for Option fields

Fixes #40, Fixes #41

M-CODEGEN-OPTION-TYPE-ASSERT"
```

GitHub automatically closes issues when commits with "Fixes #X" are merged to the default branch.

## Tips

1. **Batch similar issues** - If multiple issues report the same bug, close them all with the same comment
2. **Link related issues** - If issues are duplicates, mention "See also #X"
3. **Be specific** - Generic "fixed" comments don't help users understand what changed
4. **Include usage** - For new features, show how to use them
5. **Reference design docs** - If there's a design doc, link to it
