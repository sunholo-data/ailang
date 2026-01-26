/**
 * Tests for idService utility functions
 */

import { describe, it, expect } from 'vitest';
import {
  isTaskId,
  isEvalId,
  isUuidId,
  isTraceId,
  isSpanId,
  normalizeId,
  extractShortId,
  formatIdForDisplay,
  extractTaskId,
  extractParentTaskId,
  extractSessionId,
  extractAgentId,
  extractApprovalStatus,
  extractApprovalId,
  extractCoordinatorContext,
  isCoordinatorTask,
  collectTaskIds,
  collectSpanTypes,
} from './idService';
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
// ID Format Detection Tests
// ============================================================================

describe('isTaskId', () => {
  it('returns true for task- prefix', () => {
    expect(isTaskId('task-12345678')).toBe(true);
    expect(isTaskId('task-abcd')).toBe(true);
  });

  it('returns false for non-task IDs', () => {
    expect(isTaskId('eval-12345678')).toBe(false);
    expect(isTaskId('12345678-1234-1234-1234-123456789012')).toBe(false);
    expect(isTaskId('random-string')).toBe(false);
  });
});

describe('isEvalId', () => {
  it('returns true for eval- prefix', () => {
    expect(isEvalId('eval-1768073642020901000')).toBe(true);
    expect(isEvalId('eval-abcd')).toBe(true);
  });

  it('returns false for non-eval IDs', () => {
    expect(isEvalId('task-12345678')).toBe(false);
    expect(isEvalId('evaluation-123')).toBe(false);
  });
});

describe('isUuidId', () => {
  it('returns true for valid UUIDs', () => {
    expect(isUuidId('12345678-1234-1234-1234-123456789012')).toBe(true);
    expect(isUuidId('ABCDEFAB-ABCD-ABCD-ABCD-ABCDEFABCDEF')).toBe(true);
    expect(isUuidId('abcdefab-abcd-abcd-abcd-abcdefabcdef')).toBe(true);
  });

  it('returns false for invalid UUIDs', () => {
    expect(isUuidId('task-12345678')).toBe(false);
    expect(isUuidId('12345678123412341234123456789012')).toBe(false); // no dashes
    expect(isUuidId('12345678-1234-1234-1234-12345678901')).toBe(false); // too short
  });
});

describe('isTraceId', () => {
  it('returns true for 32-char hex strings', () => {
    expect(isTraceId('0ebf5e64bb654fcc1d19256b59f05ae3')).toBe(true);
    expect(isTraceId('ABCDEF1234567890ABCDEF1234567890')).toBe(true);
  });

  it('returns false for non-trace IDs', () => {
    expect(isTraceId('0ebf5e64bb654fcc')).toBe(false); // too short
    expect(isTraceId('0ebf5e64bb654fcc1d19256b59f05ae30')).toBe(false); // too long
    expect(isTraceId('0ebf5e64bb654fcc1d19256b59f05agz')).toBe(false); // invalid chars
  });
});

describe('isSpanId', () => {
  it('returns true for 16-char hex strings', () => {
    expect(isSpanId('0f9632b58df815e4')).toBe(true);
    expect(isSpanId('ABCDEF1234567890')).toBe(true);
  });

  it('returns false for non-span IDs', () => {
    expect(isSpanId('0f9632b58df815e')).toBe(false); // too short
    expect(isSpanId('0f9632b58df815e40')).toBe(false); // too long
  });
});

// ============================================================================
// ID Normalization Tests
// ============================================================================

describe('normalizeId', () => {
  it('strips task- prefix', () => {
    expect(normalizeId('task-12345678')).toBe('12345678');
  });

  it('strips eval- prefix', () => {
    expect(normalizeId('eval-1768073642')).toBe('1768073642');
  });

  it('returns ID unchanged if no prefix', () => {
    expect(normalizeId('12345678-1234-1234-1234-123456789012')).toBe('12345678-1234-1234-1234-123456789012');
    expect(normalizeId('random-string')).toBe('random-string');
  });
});

describe('extractShortId', () => {
  it('extracts first 8 characters by default', () => {
    expect(extractShortId('12345678-1234-1234-1234-123456789012')).toBe('12345678');
  });

  it('extracts custom length', () => {
    expect(extractShortId('12345678901234567890', 4)).toBe('1234');
  });

  it('normalizes before extracting', () => {
    expect(extractShortId('task-12345678', 8)).toBe('12345678');
  });

  it('returns full ID if shorter than length', () => {
    expect(extractShortId('abc', 8)).toBe('abc');
  });
});

describe('formatIdForDisplay', () => {
  it('truncates with ellipsis if too long', () => {
    expect(formatIdForDisplay('12345678901234567890', 12)).toBe('123456789012...');
  });

  it('returns unchanged if within limit', () => {
    expect(formatIdForDisplay('short', 12)).toBe('short');
  });
});

// ============================================================================
// ID Extraction from Attributes Tests
// ============================================================================

describe('extractTaskId', () => {
  it('extracts task.id', () => {
    expect(extractTaskId({ 'task.id': 'task-12345678' })).toBe('task-12345678');
  });

  it('extracts task_id', () => {
    expect(extractTaskId({ task_id: 'task-12345678' })).toBe('task-12345678');
  });

  it('extracts ailang.task_id', () => {
    expect(extractTaskId({ 'ailang.task_id': 'task-12345678' })).toBe('task-12345678');
  });

  it('prefers task.id over others', () => {
    expect(
      extractTaskId({
        'task.id': 'primary',
        task_id: 'secondary',
      })
    ).toBe('primary');
  });

  it('returns undefined for missing', () => {
    expect(extractTaskId({})).toBeUndefined();
    expect(extractTaskId(undefined)).toBeUndefined();
  });
});

describe('extractParentTaskId', () => {
  it('extracts task.parent_id', () => {
    expect(extractParentTaskId({ 'task.parent_id': 'task-parent' })).toBe('task-parent');
  });

  it('extracts parent_task_id', () => {
    expect(extractParentTaskId({ parent_task_id: 'task-parent' })).toBe('task-parent');
  });

  it('extracts ailang.parent_task_id', () => {
    expect(extractParentTaskId({ 'ailang.parent_task_id': 'task-parent' })).toBe('task-parent');
  });

  it('returns undefined for missing', () => {
    expect(extractParentTaskId({})).toBeUndefined();
  });
});

describe('extractSessionId', () => {
  it('extracts session.id', () => {
    expect(extractSessionId({ 'session.id': 'session-uuid' })).toBe('session-uuid');
  });

  it('extracts session_id', () => {
    expect(extractSessionId({ session_id: 'session-uuid' })).toBe('session-uuid');
  });

  it('returns undefined for missing', () => {
    expect(extractSessionId({})).toBeUndefined();
  });
});

describe('extractAgentId', () => {
  it('extracts agent.id', () => {
    expect(extractAgentId({ 'agent.id': 'design-doc-creator' })).toBe('design-doc-creator');
  });

  it('extracts agent_id', () => {
    expect(extractAgentId({ agent_id: 'design-doc-creator' })).toBe('design-doc-creator');
  });

  it('returns undefined for missing', () => {
    expect(extractAgentId({})).toBeUndefined();
  });
});

describe('extractApprovalStatus', () => {
  it('extracts approval.status', () => {
    expect(extractApprovalStatus({ 'approval.status': 'pending' })).toBe('pending');
    expect(extractApprovalStatus({ 'approval.status': 'approved' })).toBe('approved');
    expect(extractApprovalStatus({ 'approval.status': 'rejected' })).toBe('rejected');
    expect(extractApprovalStatus({ 'approval.status': 'none' })).toBe('none');
  });

  it('extracts approval_status', () => {
    expect(extractApprovalStatus({ approval_status: 'pending' })).toBe('pending');
  });

  it('returns undefined for invalid status', () => {
    expect(extractApprovalStatus({ 'approval.status': 'invalid' })).toBeUndefined();
  });

  it('returns undefined for missing', () => {
    expect(extractApprovalStatus({})).toBeUndefined();
  });
});

describe('extractApprovalId', () => {
  it('extracts approval.id', () => {
    expect(extractApprovalId({ 'approval.id': 'apr-123' })).toBe('apr-123');
  });

  it('extracts approval_id', () => {
    expect(extractApprovalId({ approval_id: 'apr-123' })).toBe('apr-123');
  });

  it('returns undefined for missing', () => {
    expect(extractApprovalId({})).toBeUndefined();
  });
});

// ============================================================================
// Coordinator Context Tests
// ============================================================================

describe('extractCoordinatorContext', () => {
  it('extracts all coordinator context fields', () => {
    const span = makeSpan({
      attributes: {
        'task.id': 'task-12345678',
        'task.parent_id': 'task-parent',
        'agent.id': 'design-doc-creator',
        'approval.status': 'pending',
        'approval.id': 'apr-123',
      },
    });
    const context = extractCoordinatorContext(span);
    expect(context.taskId).toBe('task-12345678');
    expect(context.parentTaskId).toBe('task-parent');
    expect(context.agentId).toBe('design-doc-creator');
    expect(context.approvalStatus).toBe('pending');
    expect(context.approvalId).toBe('apr-123');
  });

  it('handles missing attributes', () => {
    const span = makeSpan({});
    const context = extractCoordinatorContext(span);
    expect(context.taskId).toBeUndefined();
    expect(context.parentTaskId).toBeUndefined();
    expect(context.agentId).toBeUndefined();
    expect(context.approvalStatus).toBeUndefined();
    expect(context.approvalId).toBeUndefined();
  });
});

describe('isCoordinatorTask', () => {
  it('returns true for coordinator.task.execute', () => {
    const span = makeSpan({ name: 'coordinator.task.execute' });
    expect(isCoordinatorTask(span)).toBe(true);
  });

  it('returns false for other spans', () => {
    expect(isCoordinatorTask(makeSpan({ name: 'claude.execute' }))).toBe(false);
    expect(isCoordinatorTask(makeSpan({ name: 'exec.turn' }))).toBe(false);
  });
});

// ============================================================================
// Collection Functions Tests
// ============================================================================

describe('collectTaskIds', () => {
  it('collects task IDs from spans', () => {
    const spans: Span[] = [
      makeSpan({ attributes: { 'task.id': 'task-11111111' } }),
      makeSpan({ attributes: { 'task.id': 'task-22222222' } }),
    ];
    const taskIds = collectTaskIds(spans);
    expect(taskIds).toContain('task-11111111');
    expect(taskIds).toContain('task-22222222');
    expect(taskIds).toHaveLength(2);
  });

  it('collects task IDs from nested children', () => {
    const spans: Span[] = [
      makeSpan({
        attributes: { 'task.id': 'task-parent' },
        children: [
          makeSpan({ attributes: { 'task.id': 'task-child1' } }),
          makeSpan({
            attributes: { 'task.id': 'task-child2' },
            children: [makeSpan({ attributes: { 'task.id': 'task-grandchild' } })],
          }),
        ],
      }),
    ];
    const taskIds = collectTaskIds(spans);
    expect(taskIds).toContain('task-parent');
    expect(taskIds).toContain('task-child1');
    expect(taskIds).toContain('task-child2');
    expect(taskIds).toContain('task-grandchild');
    expect(taskIds).toHaveLength(4);
  });

  it('only collects task- prefixed IDs', () => {
    const spans: Span[] = [
      makeSpan({ attributes: { 'task.id': 'task-12345678' } }),
      makeSpan({ attributes: { 'task.id': 'eval-12345678' } }),
      makeSpan({ attributes: { 'task.id': 'uuid-12345678' } }),
    ];
    const taskIds = collectTaskIds(spans);
    expect(taskIds).toContain('task-12345678');
    expect(taskIds).not.toContain('eval-12345678');
    expect(taskIds).not.toContain('uuid-12345678');
    expect(taskIds).toHaveLength(1);
  });

  it('deduplicates IDs', () => {
    const spans: Span[] = [
      makeSpan({ attributes: { 'task.id': 'task-same' } }),
      makeSpan({ attributes: { 'task.id': 'task-same' } }),
      makeSpan({ attributes: { 'task.id': 'task-same' } }),
    ];
    const taskIds = collectTaskIds(spans);
    expect(taskIds).toHaveLength(1);
    expect(taskIds[0]).toBe('task-same');
  });

  it('returns empty array for no task IDs', () => {
    const spans: Span[] = [makeSpan({}), makeSpan({})];
    expect(collectTaskIds(spans)).toHaveLength(0);
  });
});

describe('collectSpanTypes', () => {
  it('collects unique span types', () => {
    const spans: Span[] = [
      makeSpan({ name: 'coordinator.task.execute' }),
      makeSpan({ name: 'claude.execute' }),
      makeSpan({ name: 'exec.turn' }),
    ];
    const types = collectSpanTypes(spans);
    expect(types).toContain('coordinator.task.execute');
    expect(types).toContain('claude.execute');
    expect(types).toContain('exec.turn');
    expect(types).toHaveLength(3);
  });

  it('collects from nested children', () => {
    const spans: Span[] = [
      makeSpan({
        name: 'parent',
        children: [makeSpan({ name: 'child1' }), makeSpan({ name: 'child2' })],
      }),
    ];
    const types = collectSpanTypes(spans);
    expect(types).toContain('parent');
    expect(types).toContain('child1');
    expect(types).toContain('child2');
  });

  it('deduplicates types', () => {
    const spans: Span[] = [
      makeSpan({ name: 'same.type' }),
      makeSpan({ name: 'same.type' }),
      makeSpan({ name: 'same.type' }),
    ];
    const types = collectSpanTypes(spans);
    expect(types).toHaveLength(1);
  });

  it('returns sorted types', () => {
    const spans: Span[] = [makeSpan({ name: 'zebra' }), makeSpan({ name: 'apple' }), makeSpan({ name: 'mango' })];
    const types = collectSpanTypes(spans);
    expect(types).toEqual(['apple', 'mango', 'zebra']);
  });
});
