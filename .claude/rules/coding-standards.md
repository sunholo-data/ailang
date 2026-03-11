# Coding Standards

## Testing Policy

ALWAYS remove out-of-date tests. No backward compatibility. When architecture changes, delete old tests and write new ones.

## Linting & "Unused" Code Warnings

**NEVER delete functions just because linter says "unused".**

The Import System Disaster (Sept 2025): Linter said functions were unused because calls were renamed/commented out. Functions were blindly deleted. Result: working import system completely broken.

**Rules:**
1. Understand WHY they're unused — check git history, search for commented-out references
2. If renaming calls, rename definitions too
3. Test between each change (`make test` after rename, after commenting, after deleting)
4. Special care for parser/module/import code — run `make test-imports` and `make verify-examples`

## Code Organization

File size targets: 200-500 (sweet spot), 500-800 (acceptable), 1200+ (MUST split).
Use `codebase-organizer` skill for refactoring. `make check-file-sizes` enforces in CI.

## Documentation Updates

Every change requires:
1. **CHANGELOG.md** — Semantic versioning, grouped by category
2. **README.md** — Update status/capabilities if public-facing
3. **Design docs** — Before: `design_docs/planned/`, After: `design_docs/implemented/vX_Y/`
4. **Example files** — Every language feature needs `examples/feature_name.ail`
5. **Website examples** — Import from `examples/` using raw-loader, never embed code inline
