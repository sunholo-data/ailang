# Sprint Progress JSON Schema

This document defines the JSON format for sprint progress tracking, following the [Anthropic long-running agent patterns](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents).

## Purpose

The JSON progress file enables **multi-session continuity** by:
- Providing structured, machine-readable progress state
- Following the "constrained modification" pattern (only `passes` field changes)
- Enabling session resumption across multiple Claude Code sessions
- Supporting velocity tracking and estimation accuracy

## File Location

Sprint progress files are stored in `.ailang/state/` directory:

```
.ailang/state/sprints/sprint_<id>.json
```

**Examples:**
- `.ailang/state/sprints/sprint_M-S1.json` (for sprint M-S1)
- `.ailang/state/sprints/sprint_M-POLY-A.json` (for sprint M-POLY-A)

## Schema Definition

### Root Object

```json
{
  "sprint_id": "string",
  "created": "ISO 8601 timestamp",
  "estimated_duration_days": "number",
  "correlation_id": "string",
  "design_doc": "string (path)",
  "markdown_plan": "string (path)",
  "features": [...],
  "velocity": {...},
  "last_session": "ISO 8601 timestamp",
  "last_checkpoint": "string | null",
  "status": "not_started | in_progress | paused | completed"
}
```

### Features Array

Each feature in the sprint:

```json
{
  "id": "string",
  "description": "string",
  "estimated_loc": "number",
  "actual_loc": "number | null",
  "dependencies": ["string"],
  "acceptance_criteria": ["string"],
  "passes": null | true | false,
  "started": "ISO 8601 timestamp | null",
  "completed": "ISO 8601 timestamp | null",
  "notes": "string | null"
}
```

**⚠️ CRITICAL: Constrained Modification Pattern**

According to the Anthropic article, only the `passes` field should be modified during execution:
- ✅ **Allowed**: Change `passes` from `null` → `true` or `false`
- ✅ **Allowed**: Update `actual_loc`, `completed`, `notes` (progress tracking)
- ❌ **Forbidden**: Change `description`, `acceptance_criteria` (prevents accidental requirement changes)
- ❌ **Forbidden**: Remove features from array (prevents losing work)
- ❌ **Forbidden**: Add new features mid-sprint (add to backlog instead)

### Velocity Object

Tracks planned vs actual velocity:

```json
{
  "target_loc_per_day": "number",
  "actual_loc_per_day": "number",
  "target_milestones_per_week": "number",
  "actual_milestones_per_week": "number",
  "estimated_total_loc": "number",
  "actual_total_loc": "number",
  "estimated_days": "number",
  "actual_days": "number | null"
}
```

## Complete Example

### Initial State (Created by sprint-planner)

```json
{
  "sprint_id": "M-S1",
  "created": "2025-01-27T10:00:00Z",
  "estimated_duration_days": 7,
  "correlation_id": "sprint_M-S1",
  "design_doc": "design_docs/planned/v0_4_0/m-s1-parser-improvements.md",
  "markdown_plan": "design_docs/planned/v0_4_0/m-s1-sprint-plan.md",
  "features": [
    {
      "id": "M-S1.1",
      "description": "Parser foundation - Add helper functions",
      "estimated_loc": 200,
      "actual_loc": null,
      "dependencies": [],
      "acceptance_criteria": [
        "Can parse basic type expressions",
        "Test coverage > 80%",
        "No parser errors on example files"
      ],
      "passes": null,
      "started": null,
      "completed": null,
      "notes": null
    },
    {
      "id": "M-S1.2",
      "description": "Type integration - Connect parser to type checker",
      "estimated_loc": 150,
      "actual_loc": null,
      "dependencies": ["M-S1.1"],
      "acceptance_criteria": [
        "Types resolve correctly",
        "Type errors show source location"
      ],
      "passes": null,
      "started": null,
      "completed": null,
      "notes": null
    },
    {
      "id": "M-S1.3",
      "description": "Testing - Add comprehensive parser tests",
      "estimated_loc": 300,
      "actual_loc": null,
      "dependencies": ["M-S1.1", "M-S1.2"],
      "acceptance_criteria": [
        "Test coverage > 90%",
        "All edge cases covered",
        "Golden files updated"
      ],
      "passes": null,
      "started": null,
      "completed": null,
      "notes": null
    }
  ],
  "velocity": {
    "target_loc_per_day": 200,
    "actual_loc_per_day": 0,
    "target_milestones_per_week": 5,
    "actual_milestones_per_week": 0,
    "estimated_total_loc": 650,
    "actual_total_loc": 0,
    "estimated_days": 7,
    "actual_days": null
  },
  "last_session": "2025-01-27T10:00:00Z",
  "last_checkpoint": null,
  "status": "not_started"
}
```

### In Progress (Updated by sprint-executor - Session 1)

```json
{
  "sprint_id": "M-S1",
  "created": "2025-01-27T10:00:00Z",
  "estimated_duration_days": 7,
  "correlation_id": "sprint_M-S1",
  "design_doc": "design_docs/planned/v0_4_0/m-s1-parser-improvements.md",
  "markdown_plan": "design_docs/planned/v0_4_0/m-s1-sprint-plan.md",
  "features": [
    {
      "id": "M-S1.1",
      "description": "Parser foundation - Add helper functions",
      "estimated_loc": 200,
      "actual_loc": 214,
      "dependencies": [],
      "acceptance_criteria": [
        "Can parse basic type expressions",
        "Test coverage > 80%",
        "No parser errors on example files"
      ],
      "passes": true,
      "started": "2025-01-27T10:30:00Z",
      "completed": "2025-01-27T14:30:00Z",
      "notes": "Added test helpers, took 20% longer than estimated due to edge cases"
    },
    {
      "id": "M-S1.2",
      "description": "Type integration - Connect parser to type checker",
      "estimated_loc": 150,
      "actual_loc": 0,
      "dependencies": ["M-S1.1"],
      "acceptance_criteria": [
        "Types resolve correctly",
        "Type errors show source location"
      ],
      "passes": null,
      "started": "2025-01-27T14:45:00Z",
      "completed": null,
      "notes": "In progress - working on type resolution"
    },
    {
      "id": "M-S1.3",
      "description": "Testing - Add comprehensive parser tests",
      "estimated_loc": 300,
      "actual_loc": null,
      "dependencies": ["M-S1.1", "M-S1.2"],
      "acceptance_criteria": [
        "Test coverage > 90%",
        "All edge cases covered",
        "Golden files updated"
      ],
      "passes": null,
      "started": null,
      "completed": null,
      "notes": null
    }
  ],
  "velocity": {
    "target_loc_per_day": 200,
    "actual_loc_per_day": 53.5,
    "target_milestones_per_week": 5,
    "actual_milestones_per_week": 1.75,
    "estimated_total_loc": 650,
    "actual_total_loc": 214,
    "estimated_days": 7,
    "actual_days": 1
  },
  "last_session": "2025-01-27T14:30:00Z",
  "last_checkpoint": "M-S1.1 complete - tests pass, linting clean",
  "status": "in_progress"
}
```

### Paused (User requested pause after Session 1)

```json
{
  "sprint_id": "M-S1",
  ...
  "features": [
    {
      "id": "M-S1.1",
      "passes": true,
      "completed": "2025-01-27T14:30:00Z",
      ...
    },
    {
      "id": "M-S1.2",
      "passes": null,
      "started": "2025-01-27T14:45:00Z",
      "completed": null,
      "notes": "In progress - paused mid-implementation. Next: wire up type resolution logic"
    },
    ...
  ],
  "last_session": "2025-01-27T16:00:00Z",
  "last_checkpoint": "M-S1.1 complete, M-S1.2 50% complete",
  "status": "paused"
}
```

### Completed (All features done)

```json
{
  "sprint_id": "M-S1",
  "created": "2025-01-27T10:00:00Z",
  "estimated_duration_days": 7,
  "correlation_id": "sprint_M-S1",
  "design_doc": "design_docs/implemented/v0_4_0/m-s1-parser-improvements.md",
  "markdown_plan": "design_docs/implemented/v0_4_0/m-s1-sprint-plan.md",
  "features": [
    {
      "id": "M-S1.1",
      "passes": true,
      "actual_loc": 214,
      "completed": "2025-01-27T14:30:00Z",
      ...
    },
    {
      "id": "M-S1.2",
      "passes": true,
      "actual_loc": 163,
      "completed": "2025-01-28T10:15:00Z",
      ...
    },
    {
      "id": "M-S1.3",
      "passes": true,
      "actual_loc": 287,
      "completed": "2025-01-28T16:00:00Z",
      ...
    }
  ],
  "velocity": {
    "target_loc_per_day": 200,
    "actual_loc_per_day": 186,
    "target_milestones_per_week": 5,
    "actual_milestones_per_week": 5.25,
    "estimated_total_loc": 650,
    "actual_total_loc": 664,
    "estimated_days": 7,
    "actual_days": 3.5
  },
  "last_session": "2025-01-28T16:00:00Z",
  "last_checkpoint": "All milestones complete - sprint done!",
  "status": "completed"
}
```

## Usage Workflow

### sprint-planner Creates Initial File

```bash
# In sprint-planner skill, after creating markdown plan:
.claude/skills/sprint-planner/scripts/create_sprint_json.sh \
  "M-S1" \
  "design_docs/planned/v0_4_0/m-s1-sprint-plan.md"

# Creates: .ailang/state/sprints/sprint_M-S1.json
# Sends handoff message with correlation_id to sprint-executor
```

### sprint-executor Reads and Updates File

```bash
# Session 1: Start sprint
.claude/skills/sprint-executor/scripts/session_start.sh "M-S1"
# Reads .ailang/state/sprints/sprint_M-S1.json
# Prints "Here's where we left off" summary

# During milestone completion:
# - Update feature.passes to true/false
# - Update feature.actual_loc
# - Update velocity metrics
# - Update last_session timestamp
# - Update last_checkpoint description

# Session 2: Resume sprint (next day)
.claude/skills/sprint-executor/scripts/session_start.sh "M-S1"
# Reads JSON, sees M-S1.1 complete, M-S1.2 in progress
# Continues from where we left off
```

## Why JSON Instead of Markdown?

From the Anthropic article:

> **Using structured JSON prevents accidental modifications better than markdown.**
>
> The feature list constrains agents to mark only the `passes` field, with strong instructions preventing removal or editing of test descriptions.

**Benefits:**
1. **Machine-readable**: Easy to parse and validate
2. **Constrained updates**: Clear which fields can/can't change
3. **Atomic updates**: jq or Go can update specific fields safely
4. **Validation**: Can validate schema before/after updates
5. **History tracking**: Git shows exactly what changed
6. **Multi-tool support**: Any tool can read/write JSON

**Markdown limitations:**
- Easy to accidentally delete lines
- Parsing is fragile (format changes break parsers)
- Hard to enforce field constraints
- Difficult to validate programmatically

## Validation

### Required Fields

- `sprint_id` - Must be unique, match design doc
- `created` - ISO 8601 timestamp
- `features[]` - Must have at least one feature
- `features[].id` - Must be unique within sprint
- `features[].passes` - Must be null, true, or false
- `velocity` - All fields must be present
- `status` - Must be valid enum value

### Validation Script

```bash
# Validate JSON structure
jq -e . .ailang/state/sprints/sprint_M-S1.json >/dev/null

# Check required fields
jq -e '.sprint_id, .created, .features, .velocity, .status' \
  .ailang/state/sprints/sprint_M-S1.json >/dev/null

# Check passes field only has valid values
jq -e '[.features[].passes] | all(. == null or . == true or . == false)' \
  .ailang/state/sprints/sprint_M-S1.json
```

## Migration from Markdown

**Old workflow:**
1. sprint-planner creates markdown plan
2. sprint-executor marks milestones with ✅ in markdown
3. Hard to parse, easy to corrupt

**New workflow:**
1. sprint-planner creates both markdown (human-readable) and JSON (machine-readable)
2. sprint-executor updates JSON (authoritative), markdown (informational)
3. JSON is source of truth for state, markdown is for documentation

**Backwards compatibility:**
- Old sprint plans (markdown only) still work
- New sprints (JSON + markdown) get improved resumption support
- No migration needed for old sprints
