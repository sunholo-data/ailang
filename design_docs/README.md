# AILANG Design Documentation

## Structure

```
design_docs/
├── implemented/           # Features that have been built
│   └── v3_2/             # v3.2 AI-First Features (September 2024)
│       └── V3_2_IMPLEMENTATION_REPORT.md
├── planned/              # Future features and designs
└── archive/              # Deprecated or superseded designs
```

## Document Organization

### Implemented Features
When a feature is completed, its implementation report should be moved to `implemented/` with:
- Version number folder (e.g., `v3_2/`)
- Implementation report with what was built
- Links to code locations
- Test coverage metrics
- Known limitations

### Planned Features
Active design documents for features not yet built:
- Feature specifications
- API designs
- Architecture proposals
- RFC-style documents

### Archive
Old designs that have been superseded or abandoned:
- Include reason for archival
- Date of archival
- Link to replacement if applicable

## Version History

### v3.2 - AI-First Features (September 2024)
- Schema registry with versioning
- Error JSON encoder with taxonomy
- Test reporter with structured output
- Effects inspector
- Golden test framework
- [Full Report](implemented/v3_2/V3_2_IMPLEMENTATION_REPORT.md)

### v2.3 - Type Classes (September 2024)
- Type class resolution with dictionary-passing
- Interactive REPL with history
- Defaulting for numeric types

## Guidelines

1. **Before Implementation**: Create design doc in `planned/`
2. **After Implementation**: Create report in `implemented/`
3. **Version Numbering**: Use semantic versioning (vMAJOR.MINOR.PATCH)
4. **Always Update**: CHANGELOG.md and README.md when features ship