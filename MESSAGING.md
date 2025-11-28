# AILANG Cross-Project Messaging

Send feedback, bug reports, and improvement requests from your project to AILANG core.

## Quick Start

```bash
# Send a message to AILANG developers
ailang agent send --to-user --from "your-project-name" '{
  "type": "improvement_request",
  "message": "Your feedback here"
}'
```

## Message Types

| Type | Use For |
|------|---------|
| `improvement_request` | Feature suggestions |
| `bug_report` | Something broken |
| `compatibility_issue` | Version/platform problems |
| `performance_issue` | Slow/resource problems |
| `documentation_request` | Missing/unclear docs |

## Examples

### Bug Report
```bash
ailang agent send --to-user --from "my-project" '{
  "type": "bug_report",
  "priority": "high",
  "title": "Parser fails on multi-line strings",
  "example_code": "let s = \"line1\\nline2\"",
  "actual_behavior": "Parser error at line -1",
  "ailang_version": "0.4.7"
}'
```

### Feature Request
```bash
ailang agent send --to-user --from "my-project" '{
  "type": "improvement_request",
  "priority": "medium",
  "title": "Better error messages for type mismatches",
  "description": "When types dont match, show what was expected vs actual",
  "project_context": "Building a game engine with AILANG scripting"
}'
```

### Compatibility Issue
```bash
ailang agent send --to-user --from "ci-system" '{
  "type": "compatibility_issue",
  "priority": "critical",
  "title": "Crashes on ARM64 Linux",
  "os": "Ubuntu 22.04 ARM64",
  "ailang_version": "0.4.7"
}'
```

## Full Message Schema

```json
{
  "type": "bug_report|improvement_request|...",
  "priority": "low|medium|high|critical",
  "title": "Short summary",
  "description": "Detailed explanation",
  "example_code": "Code that shows the issue",
  "expected_behavior": "What should happen",
  "actual_behavior": "What actually happens",
  "ailang_version": "0.4.7",
  "os": "macOS 14.2",
  "project_context": "What you're building",
  "workaround": "If you found one"
}
```

## Storage

Messages stored in: `~/.ailang/state/messages/inbox/user/_unread/`

This is a global location - messages persist across all projects.

## How It Works

1. You send a message with `ailang agent send --to-user`
2. Message stored in global inbox
3. AILANG developer's SessionStart hook detects new messages
4. Developer reviews and takes action
5. Message acknowledged with `ailang agent ack`

## Check Message Status

```bash
ailang agent inbox user              # See all messages
ailang agent inbox --unread-only user  # Just unread
```

## Full Documentation

See: https://sunholo-data.github.io/ailang/docs/guides/cross-project-messaging
