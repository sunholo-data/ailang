# Prompt Manager Skill - README

## Purpose

Optimize AILANG teaching prompts for **maximum conciseness and accuracy** while maintaining AI code generation effectiveness.

## Key Changes (October 2025)

### Before
- Skill focused on version management and workflow
- No emphasis on prompt optimization
- No automated analysis tools
- Verbose documentation

### After
- **Primary mission:** Optimize prompts for token efficiency
- **Target:** <4000 words per prompt (currently 4358 words)
- **Automated analysis:** identify optimization opportunities
- **Progressive disclosure:** Detailed guides in resources/

## What This Skill Does Now

### 1. Analyzes Prompts for Bloat
```bash
.claude/skills/prompt-manager/scripts/analyze_prompt_size.sh prompts/v0.3.16.md
```

**Identifies:**
- Word count vs target (<4000)
- Code block proliferation (60 blocks → target: 5-10)
- Missing tables (0 → target: 10+)
- Section-by-section word counts
- Optimization opportunities

### 2. Guides Optimization Strategy

**Key techniques taught:**
- Replace prose with tables (-67% tokens)
- Consolidate scattered examples (-56% tokens)
- Link to external docs (-76% tokens)
- Create quick reference section
- Remove redundancy

### 3. Validates Optimizations

**Ensures:**
- Token reduction: 30-50%
- Eval success rate maintained
- All external links resolve
- Examples still work

## File Structure

```
.claude/skills/prompt-manager/
├── skill.md                              # Main skill (concise, optimization-focused)
├── scripts/
│   ├── create_prompt_version.sh          # Version management
│   ├── update_hash.sh                    # Hash integrity
│   └── analyze_prompt_size.sh            # NEW: Optimization analysis
└── resources/
    ├── prompt_optimization.md            # Detailed optimization strategies
    └── workflow_guide.md                 # Detailed workflow examples
```

## Progressive Disclosure

1. **skill.md** (always loaded) - Concise workflow + optimization principles
2. **prompt_optimization.md** (load when optimizing) - Detailed strategies, before/after examples
3. **workflow_guide.md** (load when needed) - Detailed workflows, troubleshooting
4. **Scripts** (execute as needed) - Automation tools

## Current Prompt Analysis (v0.3.16)

```
Total words: 4358 (target: <4000) - 8% over
Total lines: 1214 (target: <200) - 507% over
Code blocks: 60 (target: 5-10) - 500% over
Tables: 0 (target: 10+) - missing

Optimization potential: ~50% token reduction
```

## Optimization Roadmap

### Phase 1: Quick Wins (v0.3.17)
- [ ] Convert builtin docs to tables (~1000 tokens saved)
- [ ] Consolidate 60 code blocks to 8-10 comprehensive examples (~500 tokens)
- [ ] Add quick reference section at top
- [ ] Target: 3500 words (-20%)

### Phase 2: Deep Optimization (v0.3.18)
- [ ] Move type system details to docs/guides/types.md (~800 tokens)
- [ ] Move module system details to docs/guides/modules.md (~500 tokens)
- [ ] Create effect system guide, link from prompt (~400 tokens)
- [ ] Target: 2500 words (-43%)

### Phase 3: Maximum Density (v0.4.0)
- [ ] Reference `ailang builtins list` instead of duplicating
- [ ] Single comprehensive example covering all features
- [ ] Quick reference + external links structure
- [ ] Target: <2000 words (-54%)

## Success Metrics

**Prompt profile v0.3.16 → v0.4.0:**
- Words: 4358 → <2000 (-54%)
- Lines: 1214 → <200 (-84%)
- Code blocks: 60 → 8-10 (-83%)
- Tables: 0 → 10+ (new)
- External links: minimal → 10+ (new)

**Validation:**
- Eval success rate: maintained or improved
- AI response time: improved (less token processing)
- Developer experience: faster to read and understand

## Integration

Works with:
- **eval-analyzer:** Verify prompt accuracy before optimization
- **post-release:** Run baselines after optimization to validate
- **ailang builtins:** Reference instead of duplicating builtin docs
- **docs/guides/:** External docs for detailed explanations

## Usage Example

```bash
# Analyze current prompt
.claude/skills/prompt-manager/scripts/analyze_prompt_size.sh prompts/v0.3.16.md

# Create optimized version
.claude/skills/prompt-manager/scripts/create_prompt_version.sh v0.3.17 v0.3.16 "Optimize for conciseness (-30% tokens)"

# Apply optimization strategies (refer to resources/prompt_optimization.md)
# - Convert builtin docs to tables
# - Consolidate 60 examples to 8 comprehensive ones
# - Link type/module system details to docs/

# Validate
.claude/skills/prompt-manager/scripts/analyze_prompt_size.sh prompts/v0.3.17.md
# Should show: <3500 words, 10+ tables, 8-10 code blocks

# Verify accuracy
.claude/skills/eval-analyzer/scripts/verify_prompt_accuracy.sh v0.3.17

# Update hash
.claude/skills/prompt-manager/scripts/update_hash.sh v0.3.17

# Test effectiveness
ailang eval-suite --models gpt5-mini --output eval_results/test_v0.3.17
ailang eval-compare eval_results/baselines/v0.3.16 eval_results/test_v0.3.17

# Commit if success rate maintained
git add prompts/v0.3.17.md prompts/versions.json
git commit -m "feat: Optimize v0.3.17 prompt for conciseness (-30% tokens)"
```

## Anti-Patterns (What This Skill Prevents)

❌ Verbose builtin documentation (use tables + reference `ailang builtins list`)
❌ 60 scattered examples (consolidate to 8-10 comprehensive)
❌ Explaining "why" in prompt (move to design docs)
❌ Historical context (move to CHANGELOG.md)
❌ Implementation details (link to code)
❌ Apologetic limitations (be direct)

## References

- Optimization strategies: [resources/prompt_optimization.md](resources/prompt_optimization.md)
- Detailed workflows: [resources/workflow_guide.md](resources/workflow_guide.md)
- Main skill: [skill.md](skill.md)
