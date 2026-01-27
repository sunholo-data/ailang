/**
 * Smart label extraction utilities for exec hierarchy spans
 * Extracts human-readable labels from span names and attributes
 *
 * Consolidated from ExecHierarchy.tsx (~240 lines extracted)
 */

import type { Span, HierarchyNode, NodeStatus } from '../components/ExecHierarchy/types';

// Extended node types for enhanced visualization
export type ExtendedNodeType = HierarchyNode['type'] | 'coordinator' | 'executor' | 'ailang';

// ============================================================================
// Node Type Detection
// ============================================================================

/**
 * Determine node type from span name with semantic patterns
 */
export function getNodeType(name: string): HierarchyNode['type'] {
  // Approval spans (M-DASHBOARD-APPROVAL-INTEGRATION)
  if (name === 'approval.decision' || name === 'human.approval' || name === 'human.feedback') return 'approval';
  // Claude Code session (virtual root)
  if (name === 'claude_code.session') return 'exec';
  // Claude Code api_request (turn in a session)
  if (name === 'api_request') return 'turn';
  // Claude Code tool calls
  if (name.startsWith('claude_code.tool.')) return 'tool_use';
  // Coordinator task execution
  if (name === 'coordinator.task.execute') return 'exec';
  // Executor spans (Claude/Gemini)
  if (name === 'claude.execute' || name === 'gemini.execute') return 'exec';
  // Turn spans
  if (name.startsWith('exec.turn') || name.includes('.turn')) return 'turn';
  // Tool use spans
  if (name === 'exec.tool_use' || name.includes('tool')) return 'tool_use';
  // Eval event spans (Milestone 12: spans created alongside inbox messages)
  if (name.startsWith('eval.event.')) return 'message';
  // Message spans
  if (name.includes('message') || name.includes('msg')) return 'message';
  // Default to exec for ailang.*, compile.*, etc.
  return 'exec';
}

/**
 * Get semantic type for enhanced display (used for labels/icons)
 */
export function getSemanticType(name: string): ExtendedNodeType {
  if (name === 'coordinator.task.execute') return 'coordinator';
  if (name === 'claude.execute' || name === 'gemini.execute') return 'executor';
  if (name.startsWith('ailang.') || name.startsWith('compile.') || name.startsWith('eval.')) return 'ailang';
  return getNodeType(name);
}

/**
 * Convert span status to node status
 */
export function getNodeStatus(span: Span): NodeStatus {
  if (span.status === 'error') return 'error';
  if (span.status === 'ok') return 'completed';
  // Check if still running (no end time, or endMs === startMs for in-progress)
  if (span.durationMs === 0) return 'busy';
  return 'completed';
}

/**
 * Get icon for node type
 */
export function getNodeIcon(type: HierarchyNode['type']): string {
  switch (type) {
    case 'message': return '✉';
    case 'exec': return '⚡';
    case 'turn': return '↻';
    case 'tool_use': return '⚙';
    case 'approval': return '👤';
    default: return '●';
  }
}

// ============================================================================
// Smart Label Extraction
// ============================================================================

/**
 * Smart label result with multiple display components
 */
export interface SmartLabelResult {
  /** Primary display label */
  label: string;
  /** Optional subtitle for additional context */
  subtitle?: string;
  /** Icon character for the span type */
  icon: string;
  /** Metadata extracted from attributes */
  metadata?: Record<string, string>;
}

/**
 * Extract smart label from span name and attributes
 * Handles many span types: coordinator, Claude, Gemini, turns, tools, etc.
 */
export function getSmartLabel(span: Span): string {
  // Prefer backend-enriched display_name if available (from /api/observatory/spans/enriched)
  // This includes tool metadata like file paths, commands, patterns from Claude Code hooks
  // Exception: api_request spans with chat_context use chat preview instead (Phase 5)
  if (span.display_name && !(span.name === 'api_request' && span.chat_context)) {
    return span.display_name;
  }

  const name = span.name;
  const attrs = span.attributes || {};

  // Claude Code session: show session summary
  if (name === 'claude_code.session') {
    // Session has aggregated metrics
    const cost = (span as any).cost_usd || 0;
    const tokensIn = (span as any).tokens_in || 0;
    const tokensOut = (span as any).tokens_out || 0;
    const durationMs = span.durationMs || 0;
    const durationMins = Math.round(durationMs / 60000);
    const children = (span as any).children?.length || 0;
    return `Claude Code Session (${children} turns, $${cost.toFixed(2)}, ${durationMins}m)`;
  }

  // Approval decision spans (M-DASHBOARD-APPROVAL-INTEGRATION)
  if (name === 'approval.decision') {
    const action = attrs['approval.action'] || attrs['action'];
    const by = attrs['approval.by'] || attrs['approved.by'] || attrs['rejected.by'] || 'user';
    const channel = attrs['approval.channel'] || '';
    const channelSuffix = channel ? ` via ${channel}` : '';
    if (action === 'approve') {
      return `✓ Approved by ${by}${channelSuffix}`;
    } else if (action === 'reject') {
      return `✗ Rejected by ${by}${channelSuffix}`;
    }
    return `Approval Decision by ${by}`;
  }

  // Human approval spans
  if (name === 'human.approval') {
    const by = attrs['approved.by'] || 'user';
    return `✓ Approved by ${by}`;
  }

  // Human feedback spans
  if (name === 'human.feedback') {
    const action = attrs['feedback.action'] || '';
    const by = attrs['feedback.user'] || 'user';
    if (action === 'reject') {
      return `✗ Feedback from ${by}`;
    }
    return `Feedback from ${by}`;
  }

  // Coordinator task: use task.title or extract from directive
  if (name === 'coordinator.task.execute') {
    const title = attrs['task.title'] || attrs['directive'];
    if (title) {
      return title.length > 40 ? title.substring(0, 40) + '...' : title;
    }
    return 'Coordinator Task';
  }

  // Claude/Gemini executor: show provider + directive prefix
  if (name === 'claude.execute') {
    const directive = attrs['directive'] || attrs['task.directive'] || '';
    if (directive) {
      const prefix = directive.length > 35 ? directive.substring(0, 35) + '...' : directive;
      return `Claude: ${prefix}`;
    }
    return 'Claude Execute';
  }
  if (name === 'gemini.execute') {
    const directive = attrs['directive'] || attrs['task.directive'] || '';
    if (directive) {
      const prefix = directive.length > 35 ? directive.substring(0, 35) + '...' : directive;
      return `Gemini: ${prefix}`;
    }
    return 'Gemini Execute';
  }

  // Turn: use turn.number
  if (name.startsWith('exec.turn') || name.includes('.turn')) {
    const turnNum = attrs['turn.number'] || attrs['exec.turn'] || attrs['turn_number'];
    if (turnNum) return `Turn ${turnNum}`;
    return name.replace('exec.', '');
  }

  // Tool use: show tool name and brief input
  if (name === 'exec.tool_use') {
    const toolName = attrs['tool.name'] || attrs['tool_name'] || 'Tool';
    const input = attrs['tool.input'] || attrs['input'] || '';
    if (input && typeof input === 'string') {
      // Extract first meaningful part of input
      const brief = input.split('\n')[0].substring(0, 30);
      return `${toolName}: ${brief}${input.length > 30 ? '...' : ''}`;
    }
    return toolName;
  }

  // Eval event spans (Milestone 12): show the event title from attributes
  if (name.startsWith('eval.event.')) {
    const eventTitle = attrs['event.title'];
    if (eventTitle) {
      return eventTitle.length > 45 ? eventTitle.substring(0, 45) + '...' : eventTitle;
    }
    // Fallback: clean up the event type
    const eventType = name.replace('eval.event.', '');
    return `Eval Event: ${eventType.replace(/_/g, ' ')}`;
  }

  // Message send operations: show destination inbox and category
  if (name === 'messages.send') {
    const toInbox = attrs['message.to_inbox'] || '';
    const category = attrs['message.category'] || '';
    const fromAgent = attrs['message.from_agent'] || '';
    if (toInbox && category) {
      return `Send → ${toInbox} (${category})`;
    }
    if (toInbox) {
      return `Send → ${toInbox}`;
    }
    if (fromAgent) {
      return `Send from ${fromAgent}`;
    }
    return 'Send Message';
  }

  // Claude Code tool calls: extract tool name and context
  if (name.startsWith('claude_code.tool.')) {
    return getClaudeCodeToolLabel(name, attrs);
  }

  // API requests (Claude Code turns): show Turn N (model) $cost
  // Chat preview text is shown as subtitle via getSmartLabelResult()
  if (name === 'api_request') {
    const chatCtx = span.chat_context;
    const turnNum = chatCtx?.turn_number;
    const model = attrs['model'] || '';
    const cost = parseFloat(attrs['cost_usd'] || '0');
    let modelShort = model.replace('claude-', '').replace('-20251101', '').replace('-20251001', '');
    if (modelShort.length > 15) modelShort = modelShort.substring(0, 15);
    let label = turnNum ? `Turn ${turnNum}` : 'Turn';
    if (modelShort) label += ` (${modelShort})`;
    if (cost > 0) label += ` $${cost < 0.01 ? cost.toFixed(4) : cost.toFixed(2)}`;
    return label;
  }

  // AILANG operations: show clean operation name
  if (name.startsWith('ailang.')) {
    return name.replace('ailang.', '').replace(/\./g, ' → ');
  }
  if (name.startsWith('compile.')) {
    return 'Compile: ' + name.replace('compile.', '');
  }
  if (name.startsWith('eval.')) {
    return 'Eval: ' + name.replace('eval.', '');
  }

  // Other API requests: show model
  if (name.includes('generate')) {
    const model = attrs['model'] || attrs['gen_ai.request.model'] || '';
    if (model) return `API: ${model}`;
  }

  // Default: clean up the span name
  return name.replace(/\./g, ' ').replace(/_/g, ' ');
}

/**
 * Extract label for Claude Code tool spans
 * Handles: Read, Write, Edit, Bash, Grep, WebFetch, etc.
 */
function getClaudeCodeToolLabel(name: string, attrs: Record<string, string>): string {
  const toolName = name.replace('claude_code.tool.', '');

  // Try individual attributes first (Claude Code sends these directly)
  // File-based tools: Read, Write, Edit, Glob
  const filePath = attrs['file_path'] || attrs['path'];
  if (filePath && typeof filePath === 'string') {
    const fileName = filePath.split('/').pop() || filePath;
    return `${toolName}: ${fileName}`;
  }

  // Bash tool: show command or description
  const command = attrs['command'] || attrs['bash_command'];
  if (command && typeof command === 'string') {
    const brief = command.length > 35 ? command.substring(0, 35) + '...' : command;
    return `${toolName}: ${brief}`;
  }

  const description = attrs['description'];
  if (description && typeof description === 'string') {
    const brief = description.length > 35 ? description.substring(0, 35) + '...' : description;
    return `${toolName}: ${brief}`;
  }

  // Grep/Search tools: show pattern or query
  const pattern = attrs['pattern'] || attrs['query'] || attrs['search'];
  if (pattern && typeof pattern === 'string') {
    const brief = pattern.length > 30 ? pattern.substring(0, 30) + '...' : pattern;
    return `${toolName}: ${brief}`;
  }

  // WebFetch: show URL hostname
  const url = attrs['url'];
  if (url && typeof url === 'string') {
    try {
      const hostname = new URL(url).hostname;
      return `${toolName}: ${hostname}`;
    } catch {
      const brief = url.length > 30 ? url.substring(0, 30) + '...' : url;
      return `${toolName}: ${brief}`;
    }
  }

  // Edit tool: show what was changed
  const oldString = attrs['old_string'];
  if (oldString && typeof oldString === 'string') {
    const brief = oldString.split('\n')[0].substring(0, 25);
    return `${toolName}: "${brief}..."`;
  }

  // Fallback: try tool_parameters JSON (legacy support)
  const params = attrs['tool_parameters'] || '';
  if (params && typeof params === 'string') {
    try {
      const parsed = JSON.parse(params);
      if (parsed.file_path) {
        const path = parsed.file_path.split('/').pop() || parsed.file_path;
        return `${toolName}: ${path}`;
      }
      if (parsed.description) {
        const desc = parsed.description;
        return `${toolName}: ${desc.length > 35 ? desc.substring(0, 35) + '...' : desc}`;
      }
      if (parsed.bash_command || parsed.command) {
        const cmd = parsed.bash_command || parsed.command;
        return `${toolName}: ${cmd.length > 30 ? cmd.substring(0, 30) + '...' : cmd}`;
      }
    } catch {
      // Ignore JSON parse errors
    }
  }

  return toolName;
}

/**
 * Sanitize chat preview text for display as subtitle.
 * Strips XML-like tags (e.g. <ide_opened_file>, <system-reminder>),
 * trims whitespace, and truncates to maxLen chars.
 */
export function sanitizeChatPreview(text: string, maxLen = 80): string {
  // Strip XML-like tags
  let clean = text.replace(/<[^>]+>/g, '');
  // Collapse whitespace and newlines
  clean = clean.replace(/\s+/g, ' ').trim();
  if (!clean) return '';
  if (clean.length <= maxLen) return clean;
  return clean.substring(0, maxLen) + '...';
}

/**
 * Get full smart label result with all components.
 * For api_request (turn) spans, includes a sanitized chat preview as subtitle.
 */
export function getSmartLabelResult(span: Span): SmartLabelResult {
  const nodeType = getNodeType(span.name);
  const label = getSmartLabel(span);
  const icon = getNodeIcon(nodeType);

  // Extract metadata from attributes
  const attrs = span.attributes || {};
  const metadata: Record<string, string> = {};

  // Extract common metadata fields
  if (attrs['task.id'] || attrs['task_id']) {
    metadata.taskId = attrs['task.id'] || attrs['task_id'];
  }
  if (attrs['session.id']) {
    metadata.sessionId = attrs['session.id'];
  }
  if (attrs['model']) {
    metadata.model = attrs['model'];
  }
  if (attrs['provider']) {
    metadata.provider = attrs['provider'];
  }

  // For api_request spans, add chat preview as subtitle
  let subtitle: string | undefined;
  if (span.name === 'api_request' && span.chat_context) {
    const chatCtx = span.chat_context;
    if (chatCtx.user_prompt) {
      subtitle = sanitizeChatPreview(chatCtx.user_prompt);
    } else if (chatCtx.assistant_response) {
      subtitle = sanitizeChatPreview(chatCtx.assistant_response);
    }
  }

  return {
    label,
    subtitle,
    icon,
    metadata: Object.keys(metadata).length > 0 ? metadata : undefined,
  };
}

// ============================================================================
// Metrics Extraction
// ============================================================================

/**
 * Extract metrics from span attributes
 */
export function extractMetrics(span: Span): {
  cost?: number;
  tokensIn?: number;
  tokensOut?: number;
  provider?: string;
} {
  const attrs = span.attributes || {};

  // Try various attribute naming conventions
  const cost = parseFloat(attrs['cost_usd'] || attrs['cost'] || attrs['total_cost'] || '0') || undefined;
  const tokensIn = parseInt(attrs['tokens_in'] || attrs['input_tokens'] || attrs['gen_ai.usage.prompt_tokens'] || '0', 10) || undefined;
  const tokensOut = parseInt(attrs['tokens_out'] || attrs['output_tokens'] || attrs['gen_ai.usage.completion_tokens'] || '0', 10) || undefined;

  // Detect provider from span name or attributes
  let provider: string | undefined;
  if (span.name.includes('claude') || attrs['provider'] === 'claude') {
    provider = 'claude';
  } else if (span.name.includes('gemini') || attrs['provider'] === 'gemini') {
    provider = 'gemini';
  } else if (span.name.includes('ollama') || attrs['provider'] === 'ollama') {
    provider = 'ollama';
  } else if (attrs['provider']) {
    provider = attrs['provider'];
  }

  return { cost, tokensIn, tokensOut, provider };
}

/**
 * Extract turn number from span attributes or sibling index
 */
export function getTurnNumber(span: Span, siblingIndex?: number): number | undefined {
  // Try to get from span attributes
  const attrs = span.attributes || {};
  const fromAttr = attrs['turn.number'] || attrs['exec.turn'] || attrs['turn_number'];

  if (fromAttr) {
    const num = parseInt(String(fromAttr), 10);
    if (!isNaN(num)) return num;
  }

  // Fall back to sibling index (1-based)
  if (siblingIndex !== undefined) return siblingIndex + 1;

  return undefined;
}

// ============================================================================
// Duration Formatting
// ============================================================================

/**
 * Format duration for display (compact format)
 */
export function formatDuration(ms?: number): string {
  if (!ms) return '-';
  if (ms < 1000) return `${ms.toFixed(0)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}
