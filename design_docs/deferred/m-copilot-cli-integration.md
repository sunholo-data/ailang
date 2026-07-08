# M-COPILOT-CLI: GitHub Copilot CLI Integration

**Status:** Planned
**Target:** v0.6.4
**Priority:** P1 (High)
**Estimated:** 2 weeks
**Dependencies:** M-EXEC-CLAUDE, M-EXEC-GEMINI (existing executor infrastructure)
**Created:** 2026-01-11

---

## Problem Statement

GitHub Copilot CLI is an AI-powered terminal tool that integrates with GitHub's ecosystem (issues, PRs, repositories) to provide in-terminal code editing, execution, and exploration capabilities. While AILANG already has executor infrastructure (`internal/executor/`) for both Claude Code CLI and Gemini CLI, there is no standardized way to integrate other AI-powered development tools like Copilot CLI.

Currently, developers must:
1. Use separate tools (Claude Code, Gemini CLI, Copilot CLI) with different interfaces
2. Lose context when switching between tools
3. Manually manage credentials for each tool
4. Write custom integrations for tool orchestration

**Key metrics:**
- Development velocity: GitHub Copilot users report 35-50% faster task completion
- Terminal integration: Copilot CLI is the fastest way to interact with GitHub context
- Current gap: No unified interface across AI-powered development tools in AILANG

---

## Goals

**Primary Goal:** Create a unified integration layer for GitHub Copilot CLI that enables AILANG users to leverage Copilot's GitHub-native context awareness and terminal-first UX while maintaining consistency with existing executor architecture.

**Success Metrics:**
1. ✅ Copilot CLI executor implemented and tested (achieves feature parity with Claude Code and Gemini executors)
2. ✅ MCP (Model Context Protocol) server support for extensibility
3. ✅ Unified credentials management across all three executors
4. ✅ Documentation and examples showing Copilot CLI integration
5. ✅ Coordinator support for delegating tasks to Copilot CLI agent

---

## Solution Design

### Overview

Extend AILANG's executor framework to support GitHub Copilot CLI alongside Claude Code and Gemini CLI. The integration will:
- Implement a `CopilotExecutor` following the existing `executor.Executor` interface
- Leverage Copilot's GitHub context awareness (issues, PRs, commits)
- Provide MCP server extensibility for custom tool integration
- Integrate with the coordinator's task delegation system
- Maintain unified credential management through `~/.ailang/config.yaml`

### Architecture

#### 1. Copilot Executor (`internal/executor/copilot/`)

New package structure:
```
internal/executor/copilot/
├── executor.go           # Main executor implementation (~250 LOC)
├── copilot_types.go      # Request/response types (~150 LOC)
├── mcp_server.go         # MCP server integration (~200 LOC)
├── github_context.go     # GitHub context awareness (~150 LOC)
├── credentials.go        # Copilot-specific auth (~100 LOC)
└── executor_test.go      # Comprehensive tests (~300 LOC)
```

#### 2. Copilot Request/Response Model

**CopilotRequest:**
```go
type CopilotRequest struct {
    Directive string
    Workspace string

    // GitHub context (auto-detected or explicit)
    Owner      string   // GitHub repo owner
    Repo       string   // GitHub repo name
    Issues     []int    // Related issue IDs
    PRs        []int    // Related PR numbers

    // Execution parameters
    Timeout    time.Duration
    MCPServers []string // MCP server endpoints

    // Copilot-specific options
    RawContext bool     // Pass raw GitHub objects vs summaries
}
```

**CopilotResponse:**
```go
type CopilotResponse struct {
    Status    string
    Output    string

    // GitHub artifacts created
    CommitSHA string
    PRNumber  int
    Issues    []int

    // Context aware insights
    GitHubContext GitHubContextResult
    MCPExecution  MCPResult

    // Diagnostics
    ExecutionTime time.Duration
    TokensUsed    int
}
```

#### 3. GitHub Context Awareness

Implement automatic detection of GitHub context from workspace:
```go
type GitHubContextDetector struct {
    workspace string
}

func (gcd *GitHubContextDetector) DetectContext(ctx context.Context) (*GitHubContext, error) {
    // Detect from .git/config, gh CLI, $GITHUB_* env vars
    return &GitHubContext{
        Owner: "sunholo-data",
        Repo:  "ailang",
        Issues: [...]int{42, 43, 44},  // Open issues for PR
        Branch: "feature/copilot-cli",
        RemoteURL: "https://github.com/sunholo-data/ailang.git",
    }, nil
}
```

#### 4. MCP (Model Context Protocol) Support

MCP enables extensibility for custom tools and data sources:
```go
type MCPServer struct {
    Name     string                                      // "github", "jira", "slack"
    Endpoint string                                      // "stdio", "http://...", "sse://..."
    Tools    []MCPTool                                   // Available tools
}

type MCPTool struct {
    Name        string
    Description string
    InputSchema map[string]interface{}  // JSON Schema
}

// Copilot uses MCP to call tools
func (ce *CopilotExecutor) CallMCPTool(ctx context.Context, server, tool string, args map[string]interface{}) (interface{}, error) {
    // Send request to MCP server
    // Parse response
    // Return results
}
```

#### 5. Unified Credentials Management

Extend `~/.ailang/config.yaml`:
```yaml
executors:
  claude:
    type: claude-code
    installed: true
    workspace: /path/to/workspace

  gemini:
    type: gemini-cli
    installed: true
    workspace: /path/to/workspace

  copilot:
    type: github-copilot-cli
    installed: true
    workspace: /path/to/workspace

    # Copilot-specific config
    github:
      auth: gh-cli  # Use gh CLI auth, or "api-token"
      org_context: sunholo-data

    mcp_servers:
      - name: github
        endpoint: stdio
        config:
          token: $GITHUB_TOKEN  # Env var reference

      - name: jira
        endpoint: http://localhost:3000
        auth:
          username: ${JIRA_USER}
          password: ${JIRA_PASS}
```

### Implementation Plan

#### Phase 1: Core Executor Implementation (3-4 days)

**Task 1.1: CopilotExecutor Scaffold** (1 day)
- [ ] Create `internal/executor/copilot/` package
- [ ] Implement `executor.Executor` interface
- [ ] Add basic request/response types
- [ ] Create CLI invocation wrapper
- [ ] Wire into global executor factory

**Task 1.2: GitHub Context Detection** (1 day)
- [ ] Detect from `.git/config` (local repo)
- [ ] Detect from `gh` CLI configuration
- [ ] Parse `$GITHUB_*` environment variables
- [ ] Cache context for performance (LRU, 1h TTL)
- [ ] Add unit tests for all detection paths

**Task 1.3: Basic Execution Flow** (1 day)
- [ ] Copilot CLI invocation subprocess management
- [ ] JSON request/response marshaling
- [ ] Stream handling (stdout/stderr/stdin)
- [ ] Error parsing and reporting
- [ ] Timeout and signal handling

**Task 1.4: Integration Tests** (0.5 days)
- [ ] Mock Copilot CLI for testing (no real subprocess)
- [ ] Test request marshaling
- [ ] Test response parsing
- [ ] Test error scenarios

#### Phase 2: MCP Server Support (2-3 days)

**Task 2.1: MCP Protocol Implementation** (1.5 days)
- [ ] Implement MCP client protocol (`stdio`, `sse`)
- [ ] Tool discovery and introspection
- [ ] Tool invocation with parameter validation
- [ ] Response handling (text, resources, tool_results)
- [ ] Error handling per MCP spec

**Task 2.2: Built-in MCP Servers** (0.5 days)
- [ ] GitHub API MCP server integration
- [ ] Environment variable substitution in server config
- [ ] Server lifecycle management (start/stop/reconnect)

**Task 2.3: MCP Tests** (1 day)
- [ ] Mock MCP servers for testing
- [ ] Tool discovery tests
- [ ] Invocation tests with various parameter types
- [ ] Error recovery tests

#### Phase 3: Coordinator Integration (2-3 days)

**Task 3.1: Copilot Agent Configuration** (0.5 days)
- [ ] Add `copilot` agent template to `~/.ailang/config.yaml`
- [ ] Document configuration options
- [ ] Implement credential loading for Copilot

**Task 3.2: Task Routing & Delegation** (1 day)
- [ ] Extend task classification to recognize Copilot-suitable tasks
- [ ] Implement task routing logic
  - GitHub context available → prefer Copilot
  - Terminal interaction preferred → prefer Copilot
  - File-heavy editing → prefer Claude Code
- [ ] Add routing logic to `internal/coordinator/analyzer.go`

**Task 3.3: Copilot-Specific Workflows** (1 day)
- [ ] Leverage GitHub issue/PR context in prompts
- [ ] Auto-create PRs for changes
- [ ] Link commits to related issues
- [ ] Copilot CLI handoff messaging

**Task 3.4: Integration Tests** (0.5 days)
- [ ] Test coordinator task delegation to Copilot
- [ ] Test GitHub context propagation
- [ ] Test workflow integration

#### Phase 4: Documentation & Polish (1.5-2 days)

**Task 4.1: User Documentation** (1 day)
- [ ] Setup guide (install Copilot CLI, auth with gh)
- [ ] Configuration guide (MCP servers, GitHub context)
- [ ] Examples (common workflows)
- [ ] Troubleshooting guide

**Task 4.2: Developer Documentation** (0.5 days)
- [ ] Architecture overview
- [ ] Executor interface specification
- [ ] MCP implementation guide
- [ ] Adding new MCP servers

**Task 4.3: Examples** (0.5 days)
- [ ] `examples/copilot_github_integration.ail`
- [ ] `examples/copilot_with_mcp.ail`
- [ ] Coordinator workflow examples

#### Phase 5: Testing & Refinement (1-2 days)

**Task 5.1: Comprehensive Testing** (1 day)
- [ ] Integration tests with real Copilot CLI (if available)
- [ ] Error scenario testing
- [ ] Performance benchmarks
- [ ] Coverage analysis

**Task 5.2: Polish & Optimization** (0.5 days)
- [ ] Code review feedback
- [ ] Performance optimizations
- [ ] Edge case handling

---

## Technical Considerations

### 1. GitHub Context Privacy & Security

**Concern:** Sending full GitHub context (PRs, issues, commits) to Copilot CLI might expose sensitive information.

**Solution:**
- Make context detail level configurable (raw vs. summary vs. none)
- Allow blocklisting sensitive issue labels (`security`, `internal`)
- Sanitize file paths and code snippets before passing to Copilot
- Log context sharing for audit trails

### 2. MCP Server Reliability

**Concern:** External MCP servers may be slow, timeout, or fail.

**Solution:**
- Implement timeout handling per MCP call (default 30s)
- Fallback to cached results if available
- Graceful degradation (proceed without MCP if not critical)
- Retry logic with exponential backoff
- Circuit breaker for repeatedly failing servers

### 3. Credential Management

**Concern:** Managing credentials for multiple executors and MCP servers.

**Solution:**
- Use `gh` CLI auth for GitHub (already authenticated)
- Support environment variable interpolation in config
- Secret masking in logs/UI
- Per-agent credential scoping

### 4. Task Classification Accuracy

**Concern:** Routing tasks to correct executor (Copilot vs Claude vs Gemini).

**Solution:**
- Use semantic classification with `eval-analyzer` patterns
- User can explicitly specify preferred executor
- Fallback to Claude Code if no preference
- Track executor success rates to inform routing

### 5. Copilot CLI Installation & Availability

**Concern:** Copilot CLI may not be installed or may be unavailable in some regions.

**Solution:**
- Add `ailang doctor copilot` check command
- Provide installation instructions in error messages
- Gracefully disable Copilot executor if not available
- Support headless mode without terminal interaction

---

## Examples

### Basic Task Delegation to Copilot

```bash
# Explicit Copilot routing
ailang exec copilot "Fix the type inference bug in PR #123"

# Implicit routing (coordinator detects GitHub context)
ailang exec "Fix the type inference bug in PR #123"
# → Coordinator routes to Copilot because PR context is available
```

### Using MCP Servers with Copilot

```bash
# Configure JIRA MCP server
ailang config exec copilot mcp-add \
  --name jira \
  --endpoint http://localhost:3000 \
  --auth-user $JIRA_USER \
  --auth-pass $JIRA_PASS

# Use Copilot with JIRA context
ailang exec copilot "Implement JIRA-1234 as described in the issue"
# → Copilot uses MCP to fetch issue details and link implementation
```

### Coordinator Workflow with Copilot

```yaml
# In ~/.ailang/config.yaml
coordinator:
  agents:
    - id: copilot-github
      label: "Copilot (GitHub Context)"
      inbox: copilot-github
      workspace: /path/to/ailang
      invoke:
        type: executor
        executor_type: copilot
      trigger_on_complete: []
      auto_approve_handoffs: false

  github_sync:
    enabled: true
    interval_secs: 300
    target_inbox: copilot-github
```

Send GitHub issue to Copilot:
```bash
ailang messages send copilot-github "Implement semantic caching for embeddings" \
  --title "Feature: M-CACHE" \
  --github  # Creates GitHub issue + message
```

---

## Files to Modify/Create

### New Files

| File | LOC | Purpose |
|------|-----|---------|
| `internal/executor/copilot/executor.go` | 250 | Main CopilotExecutor implementation |
| `internal/executor/copilot/copilot_types.go` | 150 | Request/response types |
| `internal/executor/copilot/mcp_server.go` | 200 | MCP protocol implementation |
| `internal/executor/copilot/github_context.go` | 150 | GitHub context detection |
| `internal/executor/copilot/credentials.go` | 100 | Auth & credential management |
| `internal/executor/copilot/executor_test.go` | 300 | Unit + integration tests |
| `docs/guides/executor-copilot.md` | 200 | Setup & usage guide |
| `examples/copilot_github_integration.ail` | 50 | Basic integration example |
| `examples/copilot_with_mcp.ail` | 80 | MCP server example |

**Total New Code:** ~1,480 LOC

### Modified Files

| File | Changes | Impact |
|------|---------|--------|
| `internal/executor/factory.go` | Register CopilotExecutor | Minor +20 LOC |
| `internal/executor/executor.go` | Document interface | Minor +15 LOC |
| `internal/coordinator/analyzer.go` | Task routing logic | +50 LOC |
| `~/.ailang/config.yaml` template | Add copilot section | +30 LOC |
| `cmd/ailang/doctor.go` | Add copilot diagnostics | +30 LOC |
| `CHANGELOG.md` | Feature announcement | +20 LOC |
| `README.md` | Add Copilot CLI to capabilities | +10 LOC |

**Total Modified Code:** ~175 LOC

---

## Success Criteria

- [ ] CopilotExecutor fully implements `executor.Executor` interface
- [ ] GitHub context auto-detection works from `.git`, `gh`, environment
- [ ] MCP server protocol implemented with `stdio` and `sse` transports
- [ ] Coordinator can route tasks to Copilot CLI agent
- [ ] All tests passing (unit + integration)
- [ ] Documentation complete with setup guide and examples
- [ ] `ailang doctor copilot` validates installation
- [ ] Performance benchmarks show <100ms overhead vs. direct Copilot CLI
- [ ] Security review complete (no credential leaks, safe context sharing)
- [ ] Example files verified to work with current implementation

---

## Timeline

| Week | Phase | Milestones |
|------|-------|-----------|
| **W1** | Phase 1 | Core executor scaffold, GitHub context detection |
| **W1** | Phase 2 | MCP server support, basic tool invocation |
| **W2** | Phase 3 | Coordinator integration, task routing |
| **W2** | Phase 4 & 5 | Documentation, testing, polish |

**Critical Path:** Phase 1 → Phase 2 → Phase 3 (sequential)
**Parallel:** Documentation, examples during Phases 1-3

---

## Related Documents

- [M-EXEC-CLAUDE: Claude Code CLI Executor](../implemented/v0_6_0/m-exec-claude-sprint-plan.md) - Executor foundation
- [M-EXEC-GEMINI: Gemini CLI Executor](../implemented/v0_6_1/m-exec-gemini-sprint-plan.md) - Alternative executor pattern
- [M-COORDINATOR: Task Delegation Framework](../implemented/v0_6_2/m-coordinator-sprint-plan.md) - Agent coordination
- [Executor Architecture Overview](../../docs/docs/guides/executor.md) - Interface specification

---

## Risks & Mitigation

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|-----------|
| Copilot CLI API changes | Medium | High | Version pinning, API monitoring |
| MCP server unreliability | Medium | Medium | Timeout handling, fallback paths |
| GitHub credential leaks | Low | Critical | Audit logging, secret masking |
| Task routing errors | Medium | Low | User override option, monitoring |
| Installation issues in CI/CD | High | Low | Docker image with pre-installed CLI |

---

## Open Questions

1. **MCP Server Defaults:** Should we include MCP servers in the distribution, or require manual setup?
   - **Answer (pending):** Start with GitHub (built-in), users can add others

2. **GitHub Context Privacy:** Should we ask for confirmation before sending large PRs/issues to Copilot?
   - **Answer (pending):** Yes, with `raw_context: false` as default (summary mode)

3. **Executor Preference Signaling:** How should users specify they prefer Copilot for a task?
   - **Answer (pending):** Support 3 levels: explicit exec target, agent config, automatic routing

4. **Copilot CLI Version Support:** Which versions of Copilot CLI should we support?
   - **Answer (pending):** Latest stable + one previous version; check in `ailang doctor copilot`

---

## Notes

- This design builds on existing executor infrastructure (M-EXEC-CLAUDE, M-EXEC-GEMINI)
- Follows established patterns for credential management and error handling
- MCP support enables extensibility beyond GitHub context
- Security and privacy are paramount—detailed audit logging required
- Documentation should emphasize GitHub context benefits over Claude/Gemini

---

**Design Doc Version:** 1.0
**Last Updated:** 2026-01-11
**Status:** Ready for Sprint Planning
