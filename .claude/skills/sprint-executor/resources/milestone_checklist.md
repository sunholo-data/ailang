# Milestone Execution Checklist

## Pre-Implementation
- [ ] Mark milestone as `in_progress` in TodoWrite
- [ ] Review milestone goals and acceptance criteria
- [ ] Identify files to create/modify
- [ ] Verify estimated LOC matches plan

## Implementation
- [ ] Write implementation code
- [ ] Follow design patterns from sprint plan
- [ ] Add inline comments for complex logic
- [ ] Keep functions small and focused (<50 lines)

## Testing
- [ ] Create/update test files (*_test.go)
- [ ] Comprehensive coverage (all acceptance criteria)
- [ ] Include edge cases and error conditions
- [ ] Test both success and failure paths

## Quality Verification
- [ ] Run tests: `make test` - MUST PASS
- [ ] Run linting: `make lint` - MUST PASS
- [ ] Check formatting: `make fmt-check` or run `make fmt`
- [ ] If any fail, fix immediately before proceeding

## Documentation
- [ ] Update CHANGELOG.md with milestone completion:
  - What was implemented
  - LOC counts (implementation + tests)
  - Key design decisions
  - Files modified/created
- [ ] Update sprint plan with completion status (✅)
- [ ] Add metrics (actual LOC vs estimated, time spent)

## Example Files (REQUIRED for new language features)
- [ ] Create example file in `examples/runnable/<feature>.ail`
- [ ] Add entry to `examples/manifest.json`:
  - `path`: relative path from examples/
  - `status`: "working"
  - `tags`: relevant tags (e.g., "records", "types")
  - `description`: brief description of what example demonstrates
- [ ] Update statistics in manifest (increment total, working counts)
- [ ] Verify searchable: `ailang examples search "<feature>"`
- [ ] Verify viewable: `ailang examples show <name>`

## Sprint JSON Update (CRITICAL)
**⚠️ The checkpoint script will remind you, but DON'T SKIP THIS!**
- [ ] Update `.ailang/state/sprints/sprint_<id>.json`:
  - Set `passes: true` (or `false` if failing)
  - Set `completed: "<ISO timestamp>"`
  - Add `notes: "<what was done>"`
- [ ] If sprint is fully complete, change `status: "completed"`
- File path shown in checkpoint output

## Pause for Breath
- [ ] Show summary of what was completed
- [ ] Show current sprint progress (X of Y milestones done)
- [ ] Show velocity (LOC/day vs planned)
- [ ] Ask user: "Ready to continue to next milestone?"
- [ ] If user says "pause" or "stop", save state and exit gracefully

## Commit (After Quality Checks Pass)
- [ ] Stage files: `git add <files>`
- [ ] Commit with descriptive message
- [ ] Include milestone name and key changes
- [ ] Push if appropriate

## Milestone Complete
- [ ] Mark milestone as `completed` in TodoWrite
- [ ] Move to next milestone or finalize sprint
