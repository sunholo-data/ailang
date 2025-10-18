# AILANG Design Documentation

## Structure

```
design_docs/
├── implemented/           # Features that have been built
│   ├── v0_0_3/           # Initial design docs (September 2024)
│   ├── v0_0_4/           # Early development
│   ├── v3_2/             # v3.2 AI-First Features
│   ├── v0_2_0/           # Module execution & effects (October 2024)
│   ├── v0_3/             # v0.3.x features (generic)
│   ├── v0_3_0/           # Eval harness foundation
│   ├── v0_3_5/           # AI usability improvements
│   ├── v0_3_6/           # Incremental improvements
│   ├── v0_3_10/          # Builtin registry (M-DX1), dashboard reliability
│   ├── v0_3_12/          # show() function recovery
│   └── [summaries]       # Top-level implementation summaries
├── planned/              # Future features and designs
│   ├── v0_4_0/           # Features planned for v0.4.0
│   └── [unversioned]     # Features without clear version target
└── archived/             # Obsolete/superseded designs
    └── 2025-10/          # Archived analysis reports and old roadmaps
```

## Document Organization

### Implemented Features
When a feature is completed, its implementation report should be moved to `implemented/` with:
- **Version number folder** (e.g., `v0_3_12/`) - Use semantic versioning
- Implementation report with what was built
- Links to code locations
- Test coverage metrics
- Known limitations

**Version folder naming convention:**
- Use underscores: `v0_3_12` not `v0.3.12`
- Match CHANGELOG.md version tags
- Group related minor/patch versions when appropriate (e.g., `v0_3/` for generic v0.3.x docs)

### Planned Features
Active design documents for features not yet built:
- Feature specifications
- API designs
- Architecture proposals
- RFC-style documents

**Version organization:**
- Create version-specific folders when target version is known (e.g., `planned/v0_4_0/`)
- Keep unversioned plans in `planned/` root until version is assigned

### Archive
Old designs that have been superseded or abandoned:
- Include reason for archival
- Date of archival (use YYYY-MM folders for time-based organization)
- Link to replacement if applicable

## Recent Versions

### v0.3.12 - Recovery Release (October 2024)
- Restored `show()` builtin function lost in v0.3.10 migration
- 51% of AILANG benchmarks recovered
- [CHANGELOG](../CHANGELOG.md#v0312)

### v0.3.11 - Critical Row Unification Fix (October 2024)
- Fixed row unification regression
- Effect propagation fixes
- REPL builtin environment fix
- [CHANGELOG](../CHANGELOG.md#v0311)

### v0.3.10 - Developer Experience (October 2024)
- M-DX1: Modern builtin development system
- Central builtin registry with validation
- Type Builder DSL
- Dashboard reliability improvements (M-DASH)
- [CHANGELOG](../CHANGELOG.md#v0310)

### v0.3.0 - v0.3.6 - Type System & Eval (October 2024)
- M-EVAL-LOOP: AI self-improvement framework
- Row polymorphism
- Pattern matching improvements
- Numeric coercion
- [CHANGELOG](../CHANGELOG.md)

### v0.2.0 - Module Execution & Effects (October 2024)
- Module file execution (M-R1)
- Algebraic effects runtime (M-R2)
- Pattern matching (M-R3, M-R5b)
- IO, FS, Clock, Net effects (M-R6)
- [CHANGELOG](../CHANGELOG.md#v020)

### v0.0.3 - v0.0.4 - Initial Development (September 2024)
- Core language design
- Parser foundation
- Type inference basics
- [CHANGELOG](../CHANGELOG.md)

### v3.2 - AI-First Features (September 2024)
- Schema registry with versioning
- Error JSON encoder with taxonomy
- Test reporter with structured output
- Effects inspector
- Golden test framework
- [Full Report](implemented/v3_2/V3_2_IMPLEMENTATION_REPORT.md)

## Guidelines

1. **Before Implementation**: Create design doc in `planned/`
   - Use version folder if target version is known
   - Keep in root if version undecided

2. **After Implementation**: Create report in `implemented/`
   - Move to version-specific folder
   - Update version history in this README

3. **Version Numbering**: Use semantic versioning (vMAJOR.MINOR.PATCH)
   - Underscores in folder names: `v0_3_12`
   - Match CHANGELOG.md tags exactly

4. **Always Update**: CHANGELOG.md and README.md when features ship

5. **Archiving**: Move superseded/obsolete docs to `archived/YYYY-MM/`
   - Include archival reason in commit message
   - Keep for historical reference

## Migration Notes

**October 2024**: Reorganized from date-based folders (20250926-20251016) to version-based structure. Date folders removed, all documents now organized by implementation status and version.
