# Model Manager Skill

**Created**: November 18, 2025
**Purpose**: Simplify the process of testing and adding new AI models to the AILANG eval suite

## What This Skill Does

This skill automates the complex workflow of:
1. Testing API access to new models
2. Finding correct API model names and pricing
3. Updating models.yml configuration
4. Running test benchmarks to verify end-to-end

## Files Created

```
.claude/skills/model-manager/
├── SKILL.md                           # Main skill documentation (291 lines)
├── scripts/
│   ├── test_model_access.sh           # Test API access (OpenAI, Anthropic, Google)
│   ├── verify_vertex_model.sh         # Check Gemini model availability in Vertex AI
│   ├── update_models_yml.sh           # Safely update models.yml configuration
│   ├── run_test_benchmark.sh          # Run test benchmark to verify model works
│   └── find_model_info.sh             # Helper for web search guidance
└── resources/
    ├── provider_endpoints.md          # API endpoint reference for all providers
    └── pricing_guide.md               # How to find and convert pricing

Total: 5 executable scripts, 2 resources, ~700 lines
```

## Quick Usage

### Test a New Model

```bash
# Test OpenAI model
.claude/skills/model-manager/scripts/test_model_access.sh openai gpt-5.1

# Test Anthropic model
.claude/skills/model-manager/scripts/test_model_access.sh anthropic claude-sonnet-4-5-20250929

# Test Google Gemini model (Vertex AI)
.claude/skills/model-manager/scripts/test_model_access.sh google gemini-2.5-pro
```

### Add Model to models.yml

```bash
.claude/skills/model-manager/scripts/update_models_yml.sh \
  gpt5-1 \
  "gpt-5.1" \
  openai \
  0.00125 \
  0.01 \
  "GPT-5.1 with adaptive reasoning"
```

### Run Test Benchmark

```bash
.claude/skills/model-manager/scripts/run_test_benchmark.sh gpt5-1
```

### Check Vertex AI Availability

```bash
.claude/skills/model-manager/scripts/verify_vertex_model.sh gemini-3-pro-preview-11-2025
```

## Automatic Invocation

Just ask Claude:
- "Can we add GPT-5.1 to the eval suite?"
- "Test if Gemini 3 Pro is available"
- "What's the pricing for the new model?"
- "Update models.yml with the new model"

Claude will automatically invoke this skill and use the scripts.

## Validation Status

✅ Skill validated with skill-builder
✅ All 5 scripts executable
✅ SKILL.md is 291 lines (within 300 line target)
✅ Progressive disclosure (resources loaded on demand)
✅ Tested with GPT-5.1 (working)

## Benefits Over Manual Process

**Before** (manual process, ~30-45 minutes):
1. Write curl commands to test API access
2. Search web for pricing and model names
3. Manually edit models.yml (risky)
4. Run custom benchmark commands
5. Debug errors and fix issues

**After** (with skill, ~5-10 minutes):
1. Run one script to test access
2. Run one script to update models.yml
3. Run one script to test benchmark
4. Done!

**Time savings**: ~70% reduction in time
**Error reduction**: Automated validation, backup creation, YAML syntax checking
**Knowledge capture**: All provider-specific details in resources

## Next Steps

Use this skill to add:
- ✅ GPT-5.1 and GPT-5.1 Instant (tested, ready)
- ⏳ Gemini 3 Pro (not yet available in Vertex AI)
- 🔮 Future models as they're released

## Related Files

- Primary config: `internal/eval_harness/models.yml`
- Example usage: `NEW_MODELS_STATUS.md` (created during initial testing)
- Draft config: `NEW_MODELS_YAML_DRAFT.yml` (can be used as reference)

## Testing This Skill

From the AILANG project root, try asking Claude:

```
"Can you test if we have access to GPT-5.1?"
```

Claude should automatically:
1. Invoke the model-manager skill
2. Run test_model_access.sh script
3. Report results
4. Suggest next steps
