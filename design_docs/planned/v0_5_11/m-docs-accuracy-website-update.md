# M-DOCS-ACCURACY: Website Documentation Accuracy Update

**Status**: COMPLETED (2025-12-16)
**CLI Example Verification**: Added `make verify-cli-examples` and `examples/cli_examples.txt`
**Priority**: HIGH
**Estimated Effort**: 4-6 hours
**Target**: v0.5.12

## Problem Statement

The documentation website at https://ailang.sunholo.com has several accuracy issues:

1. **Broken Benchmark Page**: Fetching from `/ailang/benchmarks/latest.json` but baseUrl is `/`
2. **Old Domain References**: 50+ instances of `sunholo-data.github.io/ailang/` URLs
3. **Outdated Prompts**: Website shows v0.4.4 but current version is v0.5.11
4. **Stale Feature Status**: Implementation status doesn't reflect v0.5.11 features

## Root Cause Analysis

When DNS changed from GitHub Pages (`sunholo-data.github.io/ailang/`) to custom domain (`ailang.sunholo.com`), the `baseUrl` in docusaurus.config.js was correctly changed from `/ailang/` to `/`, but:

1. Hardcoded paths in React components weren't updated
2. Static asset script paths in config weren't updated
3. Internal link references throughout codebase weren't updated
4. Prompts sync workflow hasn't run since v0.4.4

## Detailed Fix Plan

### Phase 1: Fix Broken Benchmark & REPL Pages (CRITICAL)

**Files to update:**

| File | Line | Old Path | New Path |
|------|------|----------|----------|
| `docs/src/components/BenchmarkDashboard/index.jsx` | 21 | `/ailang/benchmarks/latest.json` | `/benchmarks/latest.json` |
| `docs/src/components/ModelRadarComparison/index.jsx` | ~19 | `/ailang/benchmarks/latest.json` | `/benchmarks/latest.json` |
| `docs/src/components/BenchmarkMini.jsx` | ~line | `/ailang/benchmarks/latest.json` | `/benchmarks/latest.json` |
| `docs/src/components/AilangRepl.jsx` | ~line | `/ailang/wasm/ailang.wasm` | `/wasm/ailang.wasm` |
| `docs/docusaurus.config.js` | 43 | `/ailang/wasm/wasm_exec.js` | `/wasm/wasm_exec.js` |
| `docs/docusaurus.config.js` | 47 | `/ailang/js/ailang-repl.js` | `/js/ailang-repl.js` |

**Verification:**
```bash
# After changes, test locally
cd docs && npm run start
# Visit http://localhost:3000/docs/benchmarks/performance
# Verify benchmark data loads
# Visit http://localhost:3000/docs/playground
# Verify REPL loads
```

### Phase 2: Update Domain References (HIGH)

**Files with old domain references to update:**

#### Docusaurus Config
- `docs/docusaurus.config.js` lines 186, 243: Change `https://sunholo-data.github.io/ailang/llms.txt` to `https://ailang.sunholo.com/llms.txt`

#### Documentation Files
- `README.md` - 8 instances of old domain
- `docs/README.md` - Live site reference
- `llms.txt` - Multiple self-references
- `docs/llms.txt` - Mirror file
- `docs/docs/intro.mdx` - llms.txt reference
- `docs/docs/guides/agent-integration.mdx` - Multiple references
- `docs/docs/guides/claude-code-integration.mdx` - Documentation reference

#### Go Source Code (Help URLs in Error Messages)
- `internal/parser/parser_file.go` line 134
- `internal/parser/parser_error.go` lines 120, 139
- `internal/parser/parser_decl.go` lines 104, 120
- `internal/pipeline/validate_coretypeinfo.go` line 350
- `internal/messaging/config.go` line 177

#### Test Files
- `internal/parser/cli_integration_test.go` - Update expected URLs in assertions

#### Skills Documentation
- `.claude/skills/sprint-planner/SKILL.md`
- `.claude/skills/use-ailang/SKILL.md`
- `.claude/skills/sprint-executor/SKILL.md`
- `.claude/skills/sprint-executor/resources/dx_improvement_patterns.md`

**Note**: CHANGELOG.md historical entries should be LEFT AS-IS (historical accuracy)

### Phase 3: Update Prompts Documentation (MEDIUM)

**Current State:**
- Website shows: v0.4.4 (from October 2024)
- Current version: v0.5.11 (December 2025)

**Tasks:**

1. **Create new prompt version for v0.5.11**
   ```bash
   # Check if prompt exists
   ls prompts/v0.5.*.md

   # If not, create from v0.4.4 with updates:
   # - Add SharedMem, SharedIndex effects
   # - Add bytes type and operations
   # - Add SimHash/Hamming distance
   # - Add JSON accessor functions
   # - Add sem_frame type documentation
   ```

2. **Update prompts/versions.json**
   - Add v0.5.11 entry with `latest` and `production` tags
   - Remove `latest` tag from v0.4.4

3. **Run sync script**
   ```bash
   make sync-prompts
   # This copies production/latest prompts to docs/docs/prompts/
   ```

4. **Update docs/docs/prompts/index.md**
   - Change "v0.4.x" references to "v0.5.x"
   - Update feature list to include v0.5.11 features

### Phase 4: Verify Implementation Status Against Design Docs (MEDIUM)

**Use design_docs/implemented/ as source of truth**

Key v0.5.11 features that should be documented on website:

| Feature | Design Doc | Website Location | Status |
|---------|------------|------------------|--------|
| SharedMem Effect | v0_5_11/dx-15-sharedmem-effect.md | guides/semantic-caching-how-to.mdx | CHECK |
| SharedIndex Effect | v0_5_11/dx-16-sharedindex-effect.md | guides/semantic-search.md | CHECK |
| Neural Embeddings | v0_5_11/dx-17-neural-embeddings.md | guides/semantic-search.md | CHECK |
| JSON Accessors | v0_5_11/m-json-accessors.md | reference/builtins.md | CHECK |
| bytes Type | v0_5_11/bytes-type-support.md | reference/types.md | CHECK |
| SimHash | v0_5_11/simhash-support.md | guides/semantic-caching-how-to.mdx | CHECK |

**Verification script:**
```bash
# For each implemented feature, verify website mentions it
grep -r "SharedMem" docs/docs/
grep -r "SharedIndex" docs/docs/
grep -r "simhash" docs/docs/
grep -r "_bytes_" docs/docs/
```

### Phase 5: Build and Deploy Verification

```bash
# 1. Build locally and test
cd docs
npm run build
npm run serve
# Test all key pages manually

# 2. Verify no broken links
# Docusaurus will fail build on broken links (onBrokenLinks: 'throw')

# 3. Commit and push
git add -A
git commit -m "fix(docs): Update website for ailang.sunholo.com domain

- Fix hardcoded /ailang/ paths in benchmark components
- Fix script paths in docusaurus.config.js
- Update llms.txt references to new domain
- Update Go source help URLs to new domain
- Update prompts to v0.5.11"

# 4. Monitor GitHub Actions deployment
gh run watch
```

## Implementation Order

1. **Phase 1** - Fix critical benchmark/REPL paths (blocks website usability)
2. **Phase 2** - Update domain references (consistency)
3. **Phase 3** - Update prompts (accuracy)
4. **Phase 4** - Verify feature documentation (completeness)
5. **Phase 5** - Build, test, deploy

## Acceptance Criteria

- [ ] Benchmark page loads data successfully at https://ailang.sunholo.com/docs/benchmarks/performance
- [ ] REPL/Playground works at https://ailang.sunholo.com/docs/playground
- [ ] No references to `sunholo-data.github.io/ailang/` in documentation
- [ ] llms.txt navbar/footer links point to new domain
- [ ] Prompts page shows v0.5.11 as current version
- [ ] All v0.5.11 features mentioned in relevant docs
- [ ] `make docs-build` succeeds with no broken link errors
- [ ] GitHub Actions deployment completes successfully

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Broken links after domain change | Medium | High | Docusaurus throws on broken links |
| WASM not loading on new paths | Low | High | Test playground locally first |
| Search indexing issues | Low | Medium | Resubmit sitemap after deploy |

## Files Changed Summary

**Critical (must fix):**
- `docs/src/components/BenchmarkDashboard/index.jsx`
- `docs/src/components/ModelRadarComparison/index.jsx`
- `docs/src/components/BenchmarkMini.jsx`
- `docs/src/components/AilangRepl.jsx`
- `docs/docusaurus.config.js`

**Domain references:**
- `README.md`
- `docs/README.md`
- `llms.txt`
- `docs/llms.txt`
- `docs/docs/intro.mdx`
- `docs/docs/guides/agent-integration.mdx`
- `docs/docs/guides/claude-code-integration.mdx`
- `internal/parser/parser_*.go` (3 files)
- `internal/pipeline/validate_coretypeinfo.go`
- `internal/messaging/config.go`
- `.claude/skills/*/SKILL.md` (3 files)

**Prompts:**
- `prompts/versions.json`
- `prompts/v0.5.11.md` (new)
- `docs/docs/prompts/index.md`
- `docs/docs/prompts/v0.5.11.md` (synced)
