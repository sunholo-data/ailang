---
sidebar_position: 2
---

# Deterministic Tooling

:::caution PLANNED FOR v0.7.0
This feature is **not yet implemented**. This page describes the planned deterministic tooling system for AILANG.

**Design Document**: [deterministic-tooling.md](https://github.com/sunholo-data/ailang/blob/main/design_docs/planned/v0_7_0/deterministic-tooling.md)
:::

## Overview

Deterministic tooling provides AI-friendly commands for code transformation that produce consistent, predictable results.

## Planned Commands

### Canonical Normalization

```bash
ailang normalize file.ail
```

Rewrites code to canonical form:
- Consistent whitespace and formatting
- Standardized import ordering
- Deterministic AST serialization

### Import Suggestion

```bash
ailang suggest-imports file.ail
```

Analyzes undefined identifiers and suggests imports:
- Scans stdlib for matching exports
- Checks project modules
- Outputs machine-readable suggestions

### Deterministic Edits

```bash
ailang apply plan.json file.ail
```

Applies structured edit plans:
- JSON-defined transformations
- Atomic all-or-nothing application
- Reproducible across runs

### Training Data Export

```bash
ailang run --emit-trace jsonl file.ail
```

Exports execution traces for AI training:
- Step-by-step evaluation
- Type information at each step
- Effect tracking

## Current Status

**What works today:**
- Basic formatting via `go fmt` on generated Go code
- Manual import management
- Standard execution (no trace export)

**What's planned:**
- Native AILANG formatter
- Intelligent import suggestions
- Structured edit application
- JSONL trace export for training

## Design Goals

1. **Reproducibility**: Same input always produces same output
2. **Machine-Readable**: JSON/JSONL formats for AI consumption
3. **Composability**: Tools can be chained in pipelines
4. **Minimal Dependencies**: Pure Go implementation

## Use Cases

- **AI Self-Training**: Export execution traces to improve code generation
- **CI/CD**: Enforce canonical formatting in pipelines
- **Refactoring**: Apply large-scale changes via edit plans
- **Code Review**: Normalize before diff to reduce noise
