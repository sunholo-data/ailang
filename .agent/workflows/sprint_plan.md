---
description: Plan AILANG development sprints based on velocity and design docs.
---

# AILANG Sprint Planning Workflow

This workflow helps plan development sprints by analyzing recent velocity and mapping it to design document requirements.

## Steps

1.  **Analyze Velocity**
    Calculate the recent development velocity to establish a baseline for realistic planning.
    ```bash
    .claude/skills/sprint-planner/scripts/analyze_velocity.sh
    ```
    Note the "Average LOC/day" and "Typical milestone duration".

2.  **Review Design Document**
    Read the relevant design document in `design_docs/planned/`.
    Identify:
    - Completed milestones (✅)
    - Remaining milestones (❌, ⏳)
    - Dependencies

3.  **Check Implementation Status**
    Compare the design doc with reality:
    - Check `CHANGELOG.md` for recent features.
    - Check `examples/` for working/broken examples.
    - Check `make test-coverage-badge` for coverage.

4.  **Draft Sprint Plan**
    Create a new plan using the template structure:
    - **Goal**: One sentence summary.
    - **Milestones**: Breakdown of work with estimated LOC.
    - **Timeline**: Day-by-day or weekly schedule.
    - **Risks**: What could go wrong?

5.  **Create Plan Document**
    Save the plan as `design_docs/YYYYMMDD/M-[MILESTONE].md`.
    ```bash
    # Example
    # git add design_docs/20251118/M-P2.md
    ```

## Best Practices
- **Be Conservative**: Use actual velocity, not optimistic guesses.
- **Prioritize Examples**: Every new feature MUST have a corresponding example file in `examples/`.
- **Include Tests**: Plan for 30-50% of LOC to be tests.
