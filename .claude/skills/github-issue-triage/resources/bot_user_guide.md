# Bot User Guide: sunholo-voight-kampff

This guide explains how to set up and use the `sunholo-voight-kampff` bot account for automated GitHub issue management.

## Overview

**Bot account:** https://github.com/sunholo-voight-kampff

Using a bot account for automated actions provides:
- **Clear audit trail** - All automated actions attributed to bot
- **Separate rate limits** - Won't impact personal account limits
- **Revocable access** - Easy to disable without affecting personal auth
- **CI/CD friendly** - Same credentials work in automation pipelines

## Setup Instructions

### 1. Generate Personal Access Token (PAT)

1. Log in to GitHub as `sunholo-voight-kampff`
2. Go to **Settings** → **Developer settings** → **Personal access tokens** → **Fine-grained tokens**
3. Click **Generate new token**
4. Configure token:
   - **Name:** `ailang-issue-triage`
   - **Expiration:** 90 days (or as needed)
   - **Repository access:** `sunholo-data/ailang` only
   - **Permissions:**
     - Issues: Read and write
     - Metadata: Read-only
5. Click **Generate token** and save it securely

### 2. Add Bot to gh CLI

```bash
# Option A: Interactive login (paste token when prompted)
gh auth login --hostname github.com --git-protocol https

# Option B: Token file (more scriptable)
echo "ghp_xxxxx" > /tmp/bot-token.txt
gh auth login --with-token < /tmp/bot-token.txt
rm /tmp/bot-token.txt
```

### 3. Verify Multiple Accounts

```bash
gh auth status
# Should show both accounts:
# github.com
#   ✓ Logged in to github.com account MarkEdmondson1234 (keyring)
#   - Active account: true
#
#   ✓ Logged in to github.com account sunholo-voight-kampff (keyring)
#   - Active account: false
```

### 4. Switch to Bot Account

```bash
gh auth switch --user sunholo-voight-kampff
```

### 5. Update Config

Edit `~/.ailang/config.yaml`:

```yaml
github:
  expected_user: sunholo-voight-kampff
  default_repo: sunholo-data/ailang
  triage:
    stale_days: 30
    auto_close_implemented: false
```

## Usage Patterns

### For Manual Triage (Personal Account)

When you want issues closed under your name:

```bash
# Ensure personal account is active
gh auth switch --user MarkEdmondson1234

# Run triage
.claude/skills/github-issue-triage/scripts/triage_report.sh
```

### For Automated/Bot Triage

When you want issues closed by the bot:

```bash
# Switch to bot
gh auth switch --user sunholo-voight-kampff

# Run triage with auto-close
.claude/skills/github-issue-triage/scripts/find_closable.sh --close
```

### For CI/CD Pipelines

```yaml
# GitHub Actions example
jobs:
  triage:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup gh CLI
        run: |
          gh auth login --with-token <<< "${{ secrets.BOT_PAT }}"

      - name: Run triage
        run: |
          .claude/skills/github-issue-triage/scripts/triage_report.sh
```

## Switching Between Accounts

```bash
# Check current account
gh auth status

# Switch to personal
gh auth switch --user MarkEdmondson1234

# Switch to bot
gh auth switch --user sunholo-voight-kampff

# Logout of specific account (if needed)
gh auth logout --user sunholo-voight-kampff
```

## Security Considerations

### Token Storage

- gh CLI stores tokens in the system keyring
- Never commit tokens to version control
- Use environment variables in CI/CD

### Token Rotation

- Rotate tokens every 90 days
- Delete old tokens after rotation
- Monitor token usage in GitHub settings

### Access Scope

- Use fine-grained tokens with minimal permissions
- Only grant access to required repositories
- Review and revoke unused tokens

## Troubleshooting

### "Wrong user" error

```
ERROR: GitHub account mismatch!
  Active:   MarkEdmondson1234
  Expected: sunholo-voight-kampff
```

**Fix:**
```bash
gh auth switch --user sunholo-voight-kampff
```

### "Not authenticated" error

```
ERROR: Not authenticated to GitHub
```

**Fix:**
```bash
gh auth login
# Or re-login as bot:
gh auth login --hostname github.com
```

### Bot not in account list

```bash
gh auth status
# Only shows personal account, not bot
```

**Fix:** Re-authenticate the bot:
```bash
gh auth login --with-token < /path/to/bot-token.txt
```

## Best Practices

1. **Use bot for automated actions** - Keeps personal history clean
2. **Use personal for manual work** - Proper attribution
3. **Always verify before close** - Check `--dry-run` first
4. **Rotate tokens regularly** - Every 90 days
5. **Monitor activity** - Check bot's contribution history

## Integration with Claude Code

Claude Code sessions can use either account. The `check_auth.sh` script will:
1. Verify `gh` is authenticated
2. Check active account matches config
3. Fail with instructions if mismatch

This prevents accidentally using the wrong account for operations.
