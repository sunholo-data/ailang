/**
 * Tests for smartLabel utility functions
 */

import { describe, it, expect } from 'vitest';
import {
  getNodeType,
  getSemanticType,
  getNodeStatus,
  getNodeIcon,
  getSmartLabel,
  getSmartLabelResult,
  extractMetrics,
  getTurnNumber,
  formatDuration,
} from './smartLabel';
import type { Span } from '../components/ExecHierarchy/types';

// ============================================================================
// Test Helpers
// ============================================================================

function makeSpan(overrides: Partial<Span> = {}): Span {
  return {
    id: 'test-span-id',
    name: 'test.span',
    startMs: Date.now(),
    durationMs: 1000,
    ...overrides,
  };
}

// ============================================================================
// getNodeType Tests
// ============================================================================

describe('getNodeType', () => {
  it('returns approval for approval.decision', () => {
    expect(getNodeType('approval.decision')).toBe('approval');
  });

  it('returns approval for human.approval', () => {
    expect(getNodeType('human.approval')).toBe('approval');
  });

  it('returns approval for human.feedback', () => {
    expect(getNodeType('human.feedback')).toBe('approval');
  });

  it('returns exec for claude_code.session', () => {
    expect(getNodeType('claude_code.session')).toBe('exec');
  });

  it('returns turn for api_request', () => {
    expect(getNodeType('api_request')).toBe('turn');
  });

  it('returns tool_use for claude_code.tool.*', () => {
    expect(getNodeType('claude_code.tool.Read')).toBe('tool_use');
    expect(getNodeType('claude_code.tool.Bash')).toBe('tool_use');
    expect(getNodeType('claude_code.tool.Write')).toBe('tool_use');
  });

  it('returns exec for coordinator.task.execute', () => {
    expect(getNodeType('coordinator.task.execute')).toBe('exec');
  });

  it('returns exec for claude.execute and gemini.execute', () => {
    expect(getNodeType('claude.execute')).toBe('exec');
    expect(getNodeType('gemini.execute')).toBe('exec');
  });

  it('returns turn for exec.turn* spans', () => {
    expect(getNodeType('exec.turn')).toBe('turn');
    expect(getNodeType('exec.turn.1')).toBe('turn');
    expect(getNodeType('some.turn.other')).toBe('turn');
  });

  it('returns tool_use for tool spans', () => {
    expect(getNodeType('exec.tool_use')).toBe('tool_use');
    expect(getNodeType('some.tool.call')).toBe('tool_use');
  });

  it('returns message for eval.event.* spans', () => {
    expect(getNodeType('eval.event.task_complete')).toBe('message');
    expect(getNodeType('eval.event.error')).toBe('message');
  });

  it('returns message for message spans', () => {
    expect(getNodeType('messages.send')).toBe('message');
    expect(getNodeType('msg.receive')).toBe('message');
  });

  it('returns exec for ailang/compile/eval spans', () => {
    expect(getNodeType('ailang.run')).toBe('exec');
    expect(getNodeType('compile.parse')).toBe('exec');
    expect(getNodeType('eval.suite')).toBe('exec');
  });
});

// ============================================================================
// getSemanticType Tests
// ============================================================================

describe('getSemanticType', () => {
  it('returns coordinator for coordinator.task.execute', () => {
    expect(getSemanticType('coordinator.task.execute')).toBe('coordinator');
  });

  it('returns executor for claude.execute and gemini.execute', () => {
    expect(getSemanticType('claude.execute')).toBe('executor');
    expect(getSemanticType('gemini.execute')).toBe('executor');
  });

  it('returns ailang for ailang.*, compile.*, eval.*', () => {
    expect(getSemanticType('ailang.run')).toBe('ailang');
    expect(getSemanticType('compile.typecheck')).toBe('ailang');
    expect(getSemanticType('eval.benchmark')).toBe('ailang');
  });

  it('falls back to getNodeType for other spans', () => {
    expect(getSemanticType('api_request')).toBe('turn');
    expect(getSemanticType('messages.send')).toBe('message');
  });
});

// ============================================================================
// getNodeStatus Tests
// ============================================================================

describe('getNodeStatus', () => {
  it('returns error for error status', () => {
    expect(getNodeStatus(makeSpan({ status: 'error' }))).toBe('error');
  });

  it('returns completed for ok status', () => {
    expect(getNodeStatus(makeSpan({ status: 'ok' }))).toBe('completed');
  });

  it('returns busy for zero duration (in-progress)', () => {
    expect(getNodeStatus(makeSpan({ durationMs: 0 }))).toBe('busy');
  });

  it('returns completed for non-zero duration without explicit status', () => {
    expect(getNodeStatus(makeSpan({ durationMs: 1000 }))).toBe('completed');
  });
});

// ============================================================================
// getNodeIcon Tests
// ============================================================================

describe('getNodeIcon', () => {
  it('returns correct icons for each type', () => {
    expect(getNodeIcon('message')).toBe('✉');
    expect(getNodeIcon('exec')).toBe('⚡');
    expect(getNodeIcon('turn')).toBe('↻');
    expect(getNodeIcon('tool_use')).toBe('⚙');
    expect(getNodeIcon('approval')).toBe('👤');
  });

  it('returns default icon for unknown type', () => {
    // @ts-expect-error testing unknown type
    expect(getNodeIcon('unknown')).toBe('●');
  });
});

// ============================================================================
// getSmartLabel Tests
// ============================================================================

describe('getSmartLabel', () => {
  it('returns display_name if available', () => {
    const span = makeSpan({
      name: 'claude_code.tool.Read',
      display_name: 'Read: config.yaml',
    });
    expect(getSmartLabel(span)).toBe('Read: config.yaml');
  });

  it('formats coordinator task with title', () => {
    const span = makeSpan({
      name: 'coordinator.task.execute',
      attributes: { 'task.title': 'Fix the parser bug' },
    });
    expect(getSmartLabel(span)).toBe('Fix the parser bug');
  });

  it('truncates long coordinator task titles', () => {
    const longTitle = 'A'.repeat(50);
    const span = makeSpan({
      name: 'coordinator.task.execute',
      attributes: { 'task.title': longTitle },
    });
    expect(getSmartLabel(span)).toBe('A'.repeat(40) + '...');
  });

  it('formats claude.execute with directive', () => {
    const span = makeSpan({
      name: 'claude.execute',
      attributes: { directive: 'Run the tests' },
    });
    expect(getSmartLabel(span)).toBe('Claude: Run the tests');
  });

  it('formats gemini.execute with directive', () => {
    const span = makeSpan({
      name: 'gemini.execute',
      attributes: { 'task.directive': 'Write documentation' },
    });
    expect(getSmartLabel(span)).toBe('Gemini: Write documentation');
  });

  it('formats turn with turn number', () => {
    const span = makeSpan({
      name: 'exec.turn',
      attributes: { 'turn.number': '3' },
    });
    expect(getSmartLabel(span)).toBe('Turn 3');
  });

  it('formats exec.tool_use with tool name and input', () => {
    const span = makeSpan({
      name: 'exec.tool_use',
      attributes: {
        'tool.name': 'Bash',
        'tool.input': 'npm run test',
      },
    });
    expect(getSmartLabel(span)).toBe('Bash: npm run test');
  });

  it('formats api_request with model and cost', () => {
    const span = makeSpan({
      name: 'api_request',
      attributes: {
        model: 'claude-sonnet-4-20251101',
        cost_usd: '0.05',
      },
    });
    const label = getSmartLabel(span);
    expect(label).toContain('Turn');
    expect(label).toContain('sonnet-4');
    expect(label).toContain('$0.05');
  });

  it('formats claude_code.tool.Read with file path', () => {
    const span = makeSpan({
      name: 'claude_code.tool.Read',
      attributes: {
        file_path: '/path/to/file.ts',
      },
    });
    expect(getSmartLabel(span)).toBe('Read: file.ts');
  });

  it('formats claude_code.tool.Bash with command', () => {
    const span = makeSpan({
      name: 'claude_code.tool.Bash',
      attributes: {
        command: 'npm run build',
      },
    });
    expect(getSmartLabel(span)).toBe('Bash: npm run build');
  });

  it('formats claude_code.tool.Grep with pattern', () => {
    const span = makeSpan({
      name: 'claude_code.tool.Grep',
      attributes: {
        pattern: 'function.*test',
      },
    });
    expect(getSmartLabel(span)).toBe('Grep: function.*test');
  });

  it('formats ailang spans', () => {
    const span = makeSpan({ name: 'ailang.run.check' });
    expect(getSmartLabel(span)).toBe('run → check');
  });

  it('formats compile spans', () => {
    const span = makeSpan({ name: 'compile.parse' });
    expect(getSmartLabel(span)).toBe('Compile: parse');
  });

  it('formats eval spans', () => {
    const span = makeSpan({ name: 'eval.benchmark' });
    expect(getSmartLabel(span)).toBe('Eval: benchmark');
  });

  it('formats approval.decision approve', () => {
    const span = makeSpan({
      name: 'approval.decision',
      attributes: {
        'approval.action': 'approve',
        'approval.by': 'mark',
        'approval.channel': 'cli',
      },
    });
    expect(getSmartLabel(span)).toBe('✓ Approved by mark via cli');
  });

  it('formats approval.decision reject', () => {
    const span = makeSpan({
      name: 'approval.decision',
      attributes: {
        'approval.action': 'reject',
        'rejected.by': 'mark',
      },
    });
    expect(getSmartLabel(span)).toBe('✗ Rejected by mark');
  });

  it('formats messages.send with inbox and category', () => {
    const span = makeSpan({
      name: 'messages.send',
      attributes: {
        'message.to_inbox': 'design-doc-creator',
        'message.category': 'feature',
      },
    });
    expect(getSmartLabel(span)).toBe('Send → design-doc-creator (feature)');
  });

  it('cleans up default span names', () => {
    const span = makeSpan({ name: 'some.span.name_with_underscores' });
    expect(getSmartLabel(span)).toBe('some span name with underscores');
  });
});

// ============================================================================
// getSmartLabelResult Tests
// ============================================================================

describe('getSmartLabelResult', () => {
  it('returns label, icon, and metadata', () => {
    const span = makeSpan({
      name: 'coordinator.task.execute',
      attributes: {
        'task.title': 'Test Task',
        'task.id': 'task-12345678',
        'session.id': 'session-uuid',
      },
    });
    const result = getSmartLabelResult(span);
    expect(result.label).toBe('Test Task');
    expect(result.icon).toBe('⚡');
    expect(result.metadata?.taskId).toBe('task-12345678');
    expect(result.metadata?.sessionId).toBe('session-uuid');
  });
});

// ============================================================================
// extractMetrics Tests
// ============================================================================

describe('extractMetrics', () => {
  it('extracts cost_usd', () => {
    const span = makeSpan({
      attributes: { cost_usd: '0.05' },
    });
    expect(extractMetrics(span).cost).toBe(0.05);
  });

  it('extracts tokens_in and tokens_out', () => {
    const span = makeSpan({
      attributes: {
        tokens_in: '1000',
        tokens_out: '500',
      },
    });
    const metrics = extractMetrics(span);
    expect(metrics.tokensIn).toBe(1000);
    expect(metrics.tokensOut).toBe(500);
  });

  it('extracts gen_ai attribute format', () => {
    const span = makeSpan({
      attributes: {
        'gen_ai.usage.prompt_tokens': '2000',
        'gen_ai.usage.completion_tokens': '800',
      },
    });
    const metrics = extractMetrics(span);
    expect(metrics.tokensIn).toBe(2000);
    expect(metrics.tokensOut).toBe(800);
  });

  it('detects claude provider', () => {
    const span = makeSpan({ name: 'claude.execute' });
    expect(extractMetrics(span).provider).toBe('claude');
  });

  it('detects gemini provider', () => {
    const span = makeSpan({ name: 'gemini.execute' });
    expect(extractMetrics(span).provider).toBe('gemini');
  });

  it('detects ollama provider', () => {
    const span = makeSpan({ name: 'ollama.generate' });
    expect(extractMetrics(span).provider).toBe('ollama');
  });

  it('extracts provider from attributes', () => {
    const span = makeSpan({
      name: 'api.request',
      attributes: { provider: 'openai' },
    });
    expect(extractMetrics(span).provider).toBe('openai');
  });

  it('returns undefined for missing metrics', () => {
    const span = makeSpan({});
    const metrics = extractMetrics(span);
    expect(metrics.cost).toBeUndefined();
    expect(metrics.tokensIn).toBeUndefined();
    expect(metrics.tokensOut).toBeUndefined();
  });
});

// ============================================================================
// getTurnNumber Tests
// ============================================================================

describe('getTurnNumber', () => {
  it('extracts from turn.number attribute', () => {
    const span = makeSpan({
      attributes: { 'turn.number': '5' },
    });
    expect(getTurnNumber(span)).toBe(5);
  });

  it('extracts from exec.turn attribute', () => {
    const span = makeSpan({
      attributes: { 'exec.turn': '3' },
    });
    expect(getTurnNumber(span)).toBe(3);
  });

  it('extracts from turn_number attribute', () => {
    const span = makeSpan({
      attributes: { turn_number: '7' },
    });
    expect(getTurnNumber(span)).toBe(7);
  });

  it('falls back to sibling index', () => {
    const span = makeSpan({});
    expect(getTurnNumber(span, 4)).toBe(5); // 1-based
  });

  it('returns undefined when no info available', () => {
    const span = makeSpan({});
    expect(getTurnNumber(span)).toBeUndefined();
  });
});

// ============================================================================
// formatDuration Tests
// ============================================================================

describe('formatDuration', () => {
  it('returns - for undefined', () => {
    expect(formatDuration(undefined)).toBe('-');
  });

  it('returns - for 0', () => {
    expect(formatDuration(0)).toBe('-');
  });

  it('formats milliseconds', () => {
    expect(formatDuration(500)).toBe('500ms');
  });

  it('formats seconds', () => {
    expect(formatDuration(5000)).toBe('5.0s');
  });

  it('formats minutes', () => {
    expect(formatDuration(120000)).toBe('2.0m');
  });
});
