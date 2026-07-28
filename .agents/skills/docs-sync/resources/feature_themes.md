# AILANG Feature Themes

Features should be grouped by theme, not by version. Users want to understand "how do I use effects?" not "what changed in v0.5.3?"

## Current Feature Themes (Implemented)

### 1. Core Language
**Theme**: Pure functional programming fundamentals

| Feature | Since | Design Doc | Website Page |
|---------|-------|------------|--------------|
| Lambda calculus | v0.1.0 | - | `/reference/language-syntax` |
| Pattern matching | v0.1.0 | - | `/reference/language-syntax` |
| Algebraic Data Types | v0.1.0 | - | `/reference/language-syntax` |
| Records | v0.1.0 | - | `/reference/language-syntax` |
| Type inference (HM) | v0.1.0 | - | `/reference/language-syntax` |
| Block expressions | v0.3.0 | - | `/reference/language-syntax` |
| Anonymous functions | v0.4.5 | - | `/reference/language-syntax` |

**Expected page**: `/docs/reference/language-syntax.md` (comprehensive)

---

### 2. Type System
**Theme**: Static types, inference, and safety

| Feature | Since | Design Doc | Website Page |
|---------|-------|------------|--------------|
| Hindley-Milner inference | v0.1.0 | - | `/reference/types` |
| Row polymorphism | v0.3.0 | - | `/reference/types` |
| Type classes (Num, Eq, Ord, Show) | v0.3.0 | - | `/reference/types` |
| Type annotations | v0.3.0 | - | `/reference/types` |

**Expected page**: `/docs/reference/types.md` (MISSING - merge into language-syntax)

---

### 3. Effect System
**Theme**: Capability-based side effects

| Feature | Since | Design Doc | Website Page |
|---------|-------|------------|--------------|
| Effect declarations | v0.2.0 | `v0_2_0/` | `/reference/effects` |
| IO effect | v0.2.0 | - | `/reference/effects` |
| FS effect | v0.2.0 | - | `/reference/effects` |
| Net effect | v0.3.0 | - | `/reference/effects` |
| Clock effect | v0.3.0 | - | `/reference/effects` |
| Env effect | v0.4.0 | `v0_4_0/` | `/reference/effects` |
| Capability checking | v0.2.0 | - | `/reference/effects` |

**Page**: `/docs/reference/effects.md` ✅ CREATED

---

### 4. Module System
**Theme**: Code organization and imports

| Feature | Since | Design Doc | Website Page |
|---------|-------|------------|--------------|
| Module declarations | v0.2.0 | - | `/reference/modules` |
| Import/export | v0.2.0 | - | `/reference/modules` |
| Import aliasing | v0.5.1 | `v0_5_0/m-dx7-import-alias.md` | `/reference/modules` |
| Symbol aliasing | v0.5.1 | `v0_5_0/m-dx7-import-alias.md` | `/reference/modules` |
| Relaxed module matching | v0.5.2 | `v0_5_2/m-dx11-relaxed-module-matching.md` | `/reference/modules` |
| Standard library | v0.3.0 | - | `/reference/modules` |

**Page**: `/docs/reference/modules.md` ✅ CREATED

---

### 5. Go Codegen
**Theme**: Compiling AILANG to Go

| Feature | Since | Design Doc | Website Page |
|---------|-------|------------|--------------|
| Basic Go codegen | v0.5.0 | `v0_5_0/` | `/guides/go-codegen` |
| Multi-file compilation | v0.5.2 | - | MISSING |
| Named ADT constructor fields | v0.5.3 | - | MISSING |
| Typed ADT slices | v0.5.3 | - | MISSING |
| Typed function signatures | v0.5.5 | - | MISSING |
| RecordUpdate support | v0.5.1 | - | MISSING |
| Array codegen | v0.5.6 | - | MISSING |
| Effect handler interfaces | v0.5.2 | - | MISSING |

**Expected page**: `/docs/guides/go-codegen.md` (NEEDS EXPANSION)

---

### 6. Arrays
**Theme**: Fixed-size collections (new in v0.5.6)

| Feature | Since | Design Doc | Website Page |
|---------|-------|------------|--------------|
| Array type (`Array[T]`) | v0.5.6 | `v0_5_6/` | `/reference/arrays` |
| Array literals (`#[...]`) | v0.5.6 | - | `/reference/arrays` |
| Array runtime functions | v0.5.6 | - | `/reference/arrays` |

**Page**: `/docs/reference/arrays.md` ✅ CREATED

---

### 7. Testing
**Theme**: Verifying AILANG code

| Feature | Since | Design Doc | Website Page |
|---------|-------|------------|--------------|
| Inline tests | v0.4.5 | `v0_4_5/` | `/guides/testing` |
| `ailang test` command | v0.4.5 | - | `/guides/testing` |
| Property-based testing | v0.4.2 | `v0_4_2/m-testing-property-based-testing.md` | MISSING |

**Expected page**: `/docs/guides/testing.md` (EXISTS - update)

---

### 8. AI Integration
**Theme**: Using AILANG with AI models

| Feature | Since | Design Doc | Website Page |
|---------|-------|------------|--------------|
| Teaching prompts | v0.3.23 | - | `/prompts/` |
| M-EVAL benchmarks | v0.3.10 | - | `/evaluation/` |
| Agent messaging | v0.5.6 | `v0_5_6/` | `/guides/agent-messaging` |

**Expected pages**: Multiple in `/docs/guides/` and `/docs/prompts/`

---

### 9. Developer Experience
**Theme**: Tools for development

| Feature | Since | Design Doc | Website Page |
|---------|-------|------------|--------------|
| REPL | v0.2.0 | - | `/reference/repl-commands` |
| Debug flags | v0.3.0 | - | `/guides/debugging` |
| Error messages | v0.3.0 | - | `/guides/debugging` |
| `ailang builtins` | v0.5.1 | - | MISSING |

**Expected pages**: `/docs/guides/development.md`, `/docs/reference/repl-commands.md`

---

## Roadmap Themes (Planned - NOT Implemented)

### R1. Execution Profiles (v0.6.0)
**Status**: PLANNED
**Design Doc**: `design_docs/planned/v0_6_0/execution-profiles.md`

| Feature | Status | Design Doc |
|---------|--------|------------|
| SimProfile | Planned | Yes |
| ServiceProfile | Planned | Yes |
| CliProfile | Planned | Yes |
| `--profile` flag | Planned | Yes |

**Page**: `/docs/roadmap/execution-profiles.md` ✅ CREATED

---

### R2. Deterministic Tooling (v0.7.0)
**Status**: PLANNED
**Design Doc**: Various in `planned/`

| Feature | Status | Design Doc |
|---------|--------|------------|
| `ailang normalize` | Planned | No |
| `ailang suggest-imports` | Planned | No |
| `ailang apply` | Planned | No |
| Total recursion (`fold`, `unfold`) | Planned | No |

**Page**: `/docs/roadmap/deterministic-tooling.md` ✅ CREATED

---

### R3. Shared Semantic State (v0.6.0)
**Status**: PLANNED
**Design Doc**: `design_docs/planned/v0_6_0/semantic-caching.md`

| Feature | Status | Design Doc |
|---------|--------|------------|
| `SharedMem` effect | Planned | Yes |
| `sem_frame` type | Planned | Yes |
| `AI.embed` effect | Planned | Yes |
| CAS operations | Planned | Yes |

**Page**: `/docs/roadmap/shared-semantic-state.md` ✅ CREATED

---

## Website Structure (IMPLEMENTED)

```
docs/
├── Getting Started
│   ├── intro.mdx (landing page)
│   ├── getting-started.md (installation)
│   └── editor-setup.md
│
├── Language Reference ✅ UPDATED
│   ├── language-syntax.md
│   ├── effects.md ✅ NEW
│   ├── modules.md ✅ NEW
│   ├── arrays.md ✅ NEW
│   ├── implementation-status.md
│   ├── repl-commands.md
│   ├── no-loops.md
│   └── limitations.md
│
├── Guides
│   ├── go-interop.md (Go codegen)
│   ├── testing.md
│   ├── debugging.md
│   └── development.md
│
├── AI & Agents
│   ├── ai-prompt-guide.mdx
│   ├── agent-integration.mdx
│   └── agent-messaging.md
│
├── Evaluation & Testing
│   └── (existing pages)
│
├── Benchmarks
│   └── performance.md
│
├── Roadmap ✅ NEW SECTION
│   ├── index.md (overview) ✅ NEW
│   ├── execution-profiles.mdx ✅ MOVED
│   ├── deterministic-tooling.md ✅ NEW
│   └── shared-semantic-state.mdx ✅ MOVED
│
└── Vision
    ├── vision.mdx
    └── why-ailang.mdx
```

---

## Theme Mapping Rules

1. **One theme = one comprehensive page** (not many small pages)
2. **New features augment existing themes** (don't create new pages per feature)
3. **Roadmap section for planned features** (not mixed with current)
4. **Design docs = ultimate source of truth** - Website pages MUST link to them:
   - Planned features: Link to `design_docs/planned/vX_Y_Z/feature.md` on GitHub
   - Implemented features: Can link to `design_docs/implemented/vX_Y_Z/feature.md`
   - Design doc folder determines target version automatically
5. **Examples must be runnable** (use raw-loader from examples/)
6. **Feature lifecycle tracked by folder moves**: `planned/` → `implemented/`

### GitHub Link Format

```markdown
**Design Document**: [feature-name.md](https://github.com/sunholo-data/ailang/blob/main/design_docs/planned/v0_6_0/feature-name.md)
```

Validation: `.claude/skills/docs-sync/scripts/derive_roadmap_versions.sh --check`
