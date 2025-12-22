# M-GITHUB-USER-OVERRIDE: GitHub Account Override Flag

**Status:** IMPLEMENTED
**Version:** v0.6.2
**Date:** 2025-12-22

## Problem

When using `ailang messages` with GitHub sync (`--github` flag), the CLI validates that the active `gh` account matches the `expected_user` configured in `~/.ailang/config.yaml`. If there's a mismatch:

1. **Error was not prominent enough** - shown as a yellow warning, easily missed
2. **No override option** - users had to either switch accounts or change config
3. **Bot account friction** - developers using both personal and bot accounts had no quick workaround

## Solution

### 1. Added `--github-user` Flag

New flag on all GitHub-enabled commands:
- `ailang messages send --github-user USERNAME`
- `ailang messages import-github --github-user USERNAME`
- `ailang messages reply --github-user USERNAME`

When set, bypasses `expected_user` validation if the specified user matches the active `gh` account.

### 2. Improved Error Display

Account mismatch errors now:
- Display in **RED** (ERROR:) instead of yellow warning
- Show all three fix options clearly:
  1. Switch account: `gh auth switch --user EXPECTED`
  2. Override: `--github-user ACTIVE`
  3. Update config

### 3. Sentinel Error Type

Added `messaging.ErrAccountMismatch` sentinel error for programmatic detection:

```go
if errors.Is(err, messaging.ErrAccountMismatch) {
    // Handle account mismatch specifically
}
```

## Files Changed

| File | Change |
|------|--------|
| `internal/messaging/github.go` | Added `ErrAccountMismatch`, `SetOverrideUser()`, updated `ValidateUser()` |
| `cmd/ailang/messages_send.go` | Added `--github-user` flag, improved error display |
| `cmd/ailang/messages_github.go` | Added `--github-user` flag to import-github |

## Usage Examples

```bash
# Normal use - fails if wrong account active
ailang messages send user "bug report" --github

# Override to use active account
ailang messages send user "bug report" --github --github-user MarkEdmondson1234

# Import with override
ailang messages import-github --github-user MarkEdmondson1234
```

## Error Output (Before vs After)

**Before:**
```
⚠ GitHub sync failed: GitHub account mismatch!
  Expected: sunholo-voight-kampff
  Active:   MarkEdmondson1234
...
```

**After:**
```
ERROR: GitHub account mismatch
  Expected: sunholo-voight-kampff (from config)
  Active:   MarkEdmondson1234

Fix with one of:
  1. Switch account:  gh auth switch --user sunholo-voight-kampff
  2. Override:        --github-user MarkEdmondson1234
  3. Update config:   Set expected_user in ~/.ailang/config.yaml

  Message saved locally but GitHub sync BLOCKED.
```

## Testing

All existing tests pass. The changes were validated with:
- Account mismatch detection works correctly
- `--github-user` override bypasses validation when users match
- Error messages display prominently
- Override works for send, import-github, and reply commands

## Related

- Created as part of `github-issue-triage` skill development
- Complements bot user setup in `.claude/skills/github-issue-triage/resources/bot_user_guide.md`
