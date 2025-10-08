#!/usr/bin/env bash
# Automated fix implementation loop
# Usage: ./tools/eval_auto_improve.sh [--benchmark <id>] [--apply]

set -euo pipefail

# Parse arguments
BENCHMARK=""
APPLY=false
DRY_RUN=true

while [[ $# -gt 0 ]]; do
    case $1 in
        --benchmark)
            BENCHMARK="$2"
            shift 2
            ;;
        --apply)
            APPLY=true
            DRY_RUN=false
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--benchmark <id>] [--apply]"
            exit 1
            ;;
    esac
done

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  M-EVAL-LOOP: Automated Fix Implementation${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}⚠️  DRY-RUN MODE: No changes will be made${NC}"
    echo -e "${YELLOW}   Use --apply to actually implement fixes${NC}"
    echo ""
fi

# Step 1: Check for baseline
if [ ! -d "eval_results/baselines" ] || [ -z "$(ls -A eval_results/baselines 2>/dev/null)" ]; then
    echo -e "${YELLOW}⚠️  No baseline found. Running eval to create baseline...${NC}"
    make eval-baseline
fi

# Step 2: Run eval if no recent results
if [ -z "$(find eval_results -name '*.json' -mmin -60 2>/dev/null)" ]; then
    echo -e "${BLUE}📊 Step 1: Running benchmarks...${NC}"
    if [ -n "$BENCHMARK" ]; then
        make eval BENCH="$BENCHMARK"
    else
        make eval-suite-repair
    fi
else
    echo -e "${GREEN}✓ Using existing eval results (less than 1 hour old)${NC}"
fi

# Step 3: Analyze failures and generate design docs
echo ""
echo -e "${BLUE}📋 Step 2: Analyzing failures...${NC}"
make eval-analyze

# Find most recent design doc
DESIGN_DOC=$(ls -t design_docs/planned/EVAL_ANALYSIS_*.md 2>/dev/null | head -1)

if [ -z "$DESIGN_DOC" ]; then
    echo -e "${GREEN}✓ No new failures found - all benchmarks passing!${NC}"
    exit 0
fi

echo -e "${GREEN}✓ Design doc generated: $DESIGN_DOC${NC}"

# Step 4: Prepare for AI agent implementation
echo ""
echo -e "${BLUE}🤖 Step 3: Preparing AI agent task...${NC}"

# Extract key info from design doc
DOC_TITLE=$(head -1 "$DESIGN_DOC" | sed 's/^# //')
echo -e "   Issue: ${DOC_TITLE}"
echo -e "   Design doc: $DESIGN_DOC"
echo ""

if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}[DRY-RUN] Would create AI agent task and invoke implementation${NC}"
    echo -e "${YELLOW}[DRY-RUN] Use --apply to actually run the AI agent${NC}"
    echo ""
    echo -e "${BLUE}Design doc preview:${NC}"
    head -50 "$DESIGN_DOC"
    echo ""
    echo -e "${YELLOW}In --apply mode, this would:${NC}"
    echo -e "  1. Create task file for general-purpose agent"
    echo -e "  2. Agent reads design doc and implements fix"
    echo -e "  3. Agent runs tests to verify"
    echo -e "  4. Agent re-runs affected benchmarks"
    echo -e "  5. Agent reports results"
    exit 0
fi

# Create detailed task for the AI agent
AGENT_TASK_FILE=".eval_auto_improve_task.md"
cat > "$AGENT_TASK_FILE" <<EOF
# M-EVAL-LOOP: Automated Fix Implementation

## Overview
You are implementing a fix for an AILANG language issue identified through automated evaluation.

## Design Document
**Location**: $DESIGN_DOC

Please read this design document carefully. It contains:
- Problem description (what's failing)
- Root cause analysis
- Proposed solution
- Files to modify

## Your Task

### Step 1: Read and Understand
1. Read the design document at: $DESIGN_DOC
2. Identify the files that need to be modified
3. Understand the proposed solution

### Step 2: Implement the Fix
1. Make minimal, focused changes to implement the solution
2. Follow AILANG coding standards (see CLAUDE.md)
3. Add comments explaining non-obvious changes
4. DO NOT modify unrelated code

### Step 3: Verify with Tests
1. Run: \`make test\` to verify existing tests still pass
2. If tests fail, analyze why and adjust your implementation
3. If you can't fix tests, explain the issue and suggest alternatives

### Step 4: Validate with Benchmarks
1. Identify which benchmark(s) this fix addresses
2. Re-run those benchmarks to verify they now pass
3. Compare before/after success rates

### Step 5: Report Results
Provide a summary including:
- ✅ Files modified (with line counts)
- ✅ Brief explanation of changes
- ✅ Test results (\`make test\` output)
- ✅ Benchmark validation results
- ✅ Any issues encountered

## Safety Guardrails
- ⚠️ Make minimal changes only
- ⚠️ Run tests after each significant change
- ⚠️ If tests break, revert and try a different approach
- ⚠️ DO NOT commit changes - human review required first
- ⚠️ If uncertain about a change, ask for clarification

## Success Criteria
- [ ] Tests pass (\`make test\`)
- [ ] Affected benchmarks now pass
- [ ] No regressions introduced
- [ ] Code follows AILANG standards
- [ ] Changes are minimal and focused

## Context Files
- Project guidelines: [CLAUDE.md](CLAUDE.md)
- Design doc: $DESIGN_DOC
- Recent eval results: eval_results/*.json

---
**Note**: This is an automated task generated by M-EVAL-LOOP Milestone 4.
The fix will be reviewed by a human before committing.
EOF

echo -e "${GREEN}✓ Agent task created: $AGENT_TASK_FILE${NC}"
echo ""
echo -e "${BLUE}📤 Task file ready for AI agent${NC}"
echo ""
echo -e "${YELLOW}This task file should be passed to:${NC}"
echo -e "  • Claude Code Task agent (general-purpose)"
echo -e "  • Or any AI coding assistant with file access"
echo ""
echo -e "${YELLOW}The agent will:${NC}"
echo -e "  1. Read design doc: $DESIGN_DOC"
echo -e "  2. Implement the fix"
echo -e "  3. Run tests"
echo -e "  4. Validate with benchmarks"
echo -e "  5. Report results"
echo ""
echo -e "${GREEN}✓ Auto-improve setup complete${NC}"
echo ""
echo -e "To execute, pass this task file to your AI coding agent."
echo ""
