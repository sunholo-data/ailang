# Sprint Plan: M-DASHBOARD-APPROVAL-INTEGRATION

## Summary
Implement multi-channel coordinator approval workflow integration with Dashboard, CLI, and GitHub as equal sources, including event queue filter fixes, GitHub comment harvesting, and approval telemetry.

**Duration:** 5 days
**Dependencies:** M-COORD-UI-APPROVALS (v0.6.3) - Complete
**Risk Level:** Medium (UI + backend coordination, external API integration)

## Current Status Analysis

### Completed Recently
- M-COORD-STABLE: Coordinator daemon with SQLite storage
- M-COORD-FEEDBACK-LOOP: CLI approve/reject with 3 iterations
- M-COORD-UI-APPROVALS: CoordinatorApprovalPanel, DiffViewer components
- ApprovalWatcher: GitHub label polling mechanism

### Velocity
- Recent average: ~150 LOC/day (based on coordinator work)
- Estimated capacity: 860 LOC for this sprint

### Remaining from Design Doc
- Event queue filter fix: ~60 LOC
- GitHub comment harvesting: ~200 LOC
- Approval workflow UI: ~180 LOC
- Bidirectional GitHub comments: ~120 LOC
- Approval telemetry spans: ~150 LOC
- Test data & documentation: ~150 LOC

## Proposed Milestones

### Milestone 1: Event Queue Filter Fix
**Goal:** Fix broken filter buttons in MessageQueue component
**Estimated:** 60 LOC implementation
**Duration:** 0.5 days

**Tasks:**
- Day 1 (AM): Investigate current filter state management in MessageQueue.tsx
- Day 1 (AM): Fix activeFilters state to properly filter displayed events
- Day 1 (AM): Add "All" button to reset filters
- Day 1 (AM): Test filter combinations (task_start, task_end, handoff, human_feedback)

**Acceptance Criteria:**
- [ ] Filter buttons toggle correctly (visual feedback)
- [ ] Events list updates when filters change
- [ ] "All" button shows all event types
- [ ] Filter state persists during session
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- React state management complexity - Mitigation: Use React DevTools to debug

### Milestone 2: GitHub Comment Harvesting
**Goal:** Fetch and incorporate human comments from GitHub when labels are detected
**Estimated:** 200 LOC implementation + tests
**Duration:** 1 day

**Tasks:**
- Day 1 (PM): Create `internal/coordinator/github_comments.go` with IssueComment struct
- Day 1 (PM): Implement `GetRecentHumanComments()` in GitHubPoster
- Day 1 (PM): Add bot filtering logic (exclude github-actions[bot], dependabot, etc.)
- Day 2 (AM): Integrate comment harvesting into ApprovalWatcher.handleEvent()
- Day 2 (AM): Write unit tests for comment parsing and bot filtering

**Acceptance Criteria:**
- [ ] IssueComment struct with ID, Body, Author, CreatedAt, IsBot fields
- [ ] GetRecentHumanComments fetches comments since last poll
- [ ] Bot comments filtered out (configurable bot patterns)
- [ ] Comments appended to approval decision feedback
- [ ] 80%+ test coverage for new code
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- GitHub API rate limits - Mitigation: Use conditional requests, cache results
- Edge cases with comment timestamps - Mitigation: Use 5-minute buffer window

### Milestone 3: Approval Workflow UI Enhancements
**Goal:** Add feedback field and iteration badge to ApprovalPanel
**Estimated:** 180 LOC implementation
**Duration:** 1 day

**Tasks:**
- Day 2 (PM): Add FeedbackInput component with textarea and character counter
- Day 2 (PM): Create IterationBadge.tsx component ("Iteration 2/3" display)
- Day 3 (AM): Add iteration status to pending approvals list
- Day 3 (AM): Wire feedback field to reject endpoint with feedback parameter
- Day 3 (AM): Add visual feedback for approve/reject actions (loading states)

**Acceptance Criteria:**
- [ ] FeedbackInput shows in reject modal with 1000 char limit
- [ ] IterationBadge shows "Iteration N/3" for retriggered tasks
- [ ] Iteration 3 shows warning "Final attempt"
- [ ] Feedback sent with rejection via API
- [ ] Loading states during approve/reject operations
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- UI/UX complexity - Mitigation: Follow existing component patterns

### Milestone 4: Bidirectional GitHub Comments
**Goal:** Post feedback comments TO GitHub and display harvested comments in UI
**Estimated:** 120 LOC implementation
**Duration:** 0.75 days

**Tasks:**
- Day 3 (PM): Implement `PostFeedback(issueNum, feedback, iteration)` in GitHubPoster
- Day 3 (PM): Call PostFeedback from Coordinator.RejectTask when GitHub issue linked
- Day 3 (PM): Create GitHubFeedback.tsx component to display harvested comments
- Day 4 (AM): Add GitHub comment source indicator in UI

**Acceptance Criteria:**
- [ ] Feedback posted to linked GitHub issue on reject
- [ ] Comment includes iteration number and agent context
- [ ] Harvested GitHub comments displayed in approval detail view
- [ ] Source indicator shows "via GitHub", "via CLI", "via Dashboard"
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- GitHub permissions - Mitigation: Check write access before posting

### Milestone 5: Approval Telemetry Spans
**Goal:** Add OTEL spans for approval decisions to ExecHierarchy
**Estimated:** 150 LOC implementation
**Duration:** 0.75 days

**Tasks:**
- Day 4 (AM): Add `approval.decision` span in handlers_coordinator.go
- Day 4 (PM): Include decision_type, channel, iteration, feedback_length attributes
- Day 4 (PM): Link spans to parent task via task_id
- Day 4 (PM): Add ApprovalSpan type to ExecHierarchy for dashboard display

**Acceptance Criteria:**
- [ ] `approval.decision` span created on approve/reject
- [ ] Span attributes include channel source (dashboard/cli/github)
- [ ] Spans visible in trace timeline
- [ ] ExecHierarchy includes approval spans in task view
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- OTEL integration complexity - Mitigation: Follow existing span patterns in codebase

### Milestone 6: Test Data & Documentation
**Goal:** Add test fixtures and update documentation
**Estimated:** 150 LOC (tests + docs)
**Duration:** 1 day

**Tasks:**
- Day 5 (AM): Create test fixtures for multi-iteration approval scenarios
- Day 5 (AM): Add integration tests for GitHub → Dashboard → CLI flow
- Day 5 (PM): Update coordinator.md with multi-channel workflow
- Day 5 (PM): Add screenshots to collaboration-hub.md
- Day 5 (PM): Update CLAUDE.md with new debug flags

**Acceptance Criteria:**
- [ ] Test fixtures for 1, 2, 3 iteration scenarios
- [ ] Integration test for cross-channel approval
- [ ] coordinator.md updated with multi-channel section
- [ ] collaboration-hub.md has approval workflow screenshots
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- Screenshot workflow - Mitigation: Use existing screenshot patterns

## Success Metrics
- Test coverage: >80% for new code
- Event queue filters: Working for all 4 event types
- GitHub integration: Comments harvested and posted
- Documentation: All 3 guides updated
- All tests passing
- All linting passing

## Dependencies
- GitHub API access (already configured via `gh`)
- OTEL exporter (already configured)
- Dashboard server running (make services-start)

## Open Questions
- None - design doc addresses all requirements

## Notes
- Multi-channel architecture: Dashboard, CLI, and GitHub are equal sources
- GitHub comment harvesting uses existing ApprovalWatcher polling mechanism
- Bot filtering is configurable to handle different GitHub bot naming patterns
- Feedback loop limited to 3 iterations (existing limit in human_interaction.go)
