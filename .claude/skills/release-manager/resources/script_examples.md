# Release Manager Script Examples

Detailed output examples for release-manager scripts.

## collect_closable_issues.sh Output

### Default Output
```
Syncing GitHub issues via ailang messages...
  ✓ Synced issues from GitHub
Scanning commits since v0.5.8 for issue references...
  Found 0 issue(s) referenced in commits
Scanning CHANGELOG.md for issue references...
  Found 0 issue(s) referenced in CHANGELOG
Scanning design docs for issue references...
  Found 3 issue(s) referenced in design docs
Querying ailang messages for GitHub-linked issues...
  Found 7 issue(s) tracked in ailang messages
Matching issues against CHANGELOG keywords...
  Found 5 issue(s) potentially related to CHANGELOG entries

============================================
Issues to close for v0.5.9
============================================
  #23: [cli] Bug: Record update with nested record fails
       Sources: design_doc,ailang_message,keyword_match
       Labels: bug, ailang-message

  #25: [cli] Bug: Record list type inference
       Sources: ailang_message,keyword_match
       Labels: bug, ailang-message

To close these issues, run:
  ./scripts/collect_closable_issues.sh 0.5.9 --close
```

### JSON Output (--json)
```json
[
  {
    "number": 23,
    "title": "[cli] Bug: Record update with nested record fails",
    "sources": "design_doc,ailang_message,keyword_match",
    "labels": "bug,ailang-message",
    "url": "https://github.com/sunholo-data/ailang/issues/23"
  }
]
```

## close_issues_with_references.sh Output

### Generated Comment Format
```markdown
Fixed in [v0.5.10](https://github.com/sunholo-data/ailang/releases/tag/v0.5.10) - Description (M-FEATURE-CODE).

Additional details from CHANGELOG...

See: [Design Doc](link)

Release commit: [`abc1234`](https://github.com/sunholo-data/ailang/commit/abc1234)
```

## broadcast_release.sh Output

### Console Output
```
Extracting changelog for v0.4.5...
Broadcasting release notification...

Release notification broadcast for v0.4.5
Projects can check their inbox with: ailang agent inbox --unread-only user

Changelog excerpt:
---
## [v0.4.5] - 2025-11-30

### Added
- Import aliasing support
- CLI arguments feature

### Fixed
- Parser recovery improvements
...
---
```

### Message Format Sent
```json
{
  "type": "release_notification",
  "title": "AILANG v0.4.5 Released",
  "version": "v0.4.5",
  "description": "## [v0.4.5] - 2025-11-30\n\n### Added\n...",
  "priority": "high",
  "release_url": "https://github.com/sunholo-data/ailang/releases/tag/v0.4.5",
  "closed_issues": [
    {"number": 23, "title": "Bug: Record update fails", "url": "..."}
  ]
}
```

## post_release_checks.sh Output

```
Verifying release v0.3.14...

1/4 Checking git tag...
  ✓ Tag v0.3.14 exists

2/4 Checking GitHub release...
  ✓ GitHub release v0.3.14 exists

3/4 Checking release binaries...
  ✓ ailang-darwin-amd64.tar.gz
  ✓ ailang-darwin-arm64.tar.gz
  ✓ ailang-linux-amd64.tar.gz
  ✓ ailang-windows-amd64.zip
  ✓ All platform binaries present

4/4 Checking CI status...
  ✓ Latest CI run passed

✓ Release v0.3.14 verified successfully!
URL: https://github.com/sunholo-data/ailang/releases/tag/v0.3.14
```

## check_implemented_docs.sh Output

```
Checking implemented design docs for v0.5.10...

  Found 18 design doc(s) in design_docs/implemented/v0_5_10
  Referenced in CHANGELOG: 12
    ✓ m-unified-ai-providers
    ✓ m-string-conversion
    ...

  ⚠ NOT in CHANGELOG: 2
    ✗ m-codegen-nested-record-type
    ✗ m-fix-float-operator-dispatch

  Action: Add entries for these features before releasing.
```
