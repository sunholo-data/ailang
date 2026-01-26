import { describe, it, expect } from 'vitest';
import {
  COLORS,
  getNodeTypeColor,
  getStatusColor,
  getTaskNodeColor,
  getSpanNodeColor,
  getNodeIcon,
  getSpanNodeIcon,
  getProviderIcon,
  getApprovalIcon,
  getApprovalColor,
  getApprovalBadge,
  getAgentStatusIcon,
} from './nodeStyles';

describe('getNodeTypeColor', () => {
  it('returns correct colors for known node types', () => {
    expect(getNodeTypeColor('message')).toBe(COLORS.message);
    expect(getNodeTypeColor('exec')).toBe(COLORS.exec);
    expect(getNodeTypeColor('turn')).toBe(COLORS.turn);
    expect(getNodeTypeColor('tool_use')).toBe(COLORS.toolUse);
    expect(getNodeTypeColor('approval')).toBe(COLORS.approval);
  });

  it('returns gray for unknown types', () => {
    expect(getNodeTypeColor('unknown')).toBe(COLORS.gray);
  });
});

describe('getStatusColor', () => {
  it('returns green for completed/done', () => {
    expect(getStatusColor('completed')).toBe(COLORS.completed);
    expect(getStatusColor('done')).toBe(COLORS.completed);
  });

  it('returns blue for busy/running', () => {
    expect(getStatusColor('busy')).toBe(COLORS.busy);
    expect(getStatusColor('running')).toBe(COLORS.busy);
  });

  it('returns red for error/failed', () => {
    expect(getStatusColor('error')).toBe(COLORS.error);
    expect(getStatusColor('failed')).toBe(COLORS.error);
  });

  it('returns amber for pending', () => {
    expect(getStatusColor('pending')).toBe(COLORS.pending);
    expect(getStatusColor('pending_approval')).toBe(COLORS.pending);
  });

  it('returns gray for unknown status', () => {
    expect(getStatusColor('unknown')).toBe(COLORS.gray);
  });
});

describe('getTaskNodeColor', () => {
  it('prioritizes approval status when pending', () => {
    expect(getTaskNodeColor('completed', 'pending')).toBe(COLORS.pending);
  });

  it('prioritizes approval status when rejected', () => {
    expect(getTaskNodeColor('completed', 'rejected')).toBe(COLORS.error);
  });

  it('uses task status when approval is approved or undefined', () => {
    expect(getTaskNodeColor('completed', 'approved')).toBe(COLORS.completed);
    expect(getTaskNodeColor('busy', undefined)).toBe(COLORS.busy);
  });
});

describe('getSpanNodeColor', () => {
  it('returns correct colors for span node types', () => {
    expect(getSpanNodeColor('coordinator')).toBe(COLORS.coordinator);
    expect(getSpanNodeColor('executor')).toBe(COLORS.executor);
    expect(getSpanNodeColor('tool')).toBe(COLORS.tool);
  });

  it('returns gray for unknown types', () => {
    expect(getSpanNodeColor(undefined)).toBe(COLORS.gray);
    expect(getSpanNodeColor('unknown')).toBe(COLORS.gray);
  });
});

describe('getNodeIcon', () => {
  it('returns correct icons for node types', () => {
    expect(getNodeIcon('message')).toBe('✉');
    expect(getNodeIcon('exec')).toBe('⚡');
    expect(getNodeIcon('turn')).toBe('↻');
    expect(getNodeIcon('tool_use')).toBe('⚙');
    expect(getNodeIcon('approval')).toBe('👤');
  });

  it('returns bullet for unknown types', () => {
    expect(getNodeIcon('unknown')).toBe('●');
  });
});

describe('getSpanNodeIcon', () => {
  it('returns correct icons for span node types', () => {
    expect(getSpanNodeIcon('coordinator')).toBe('\u2B21'); // Hexagon
    expect(getSpanNodeIcon('executor')).toBe('\u25CF'); // Circle
    expect(getSpanNodeIcon('turn')).toBe('\u25C9'); // Fish eye
    expect(getSpanNodeIcon('tool')).toBe('\u2699'); // Gear
  });

  it('returns bullet for unknown types', () => {
    expect(getSpanNodeIcon(undefined)).toBe('\u2022');
    expect(getSpanNodeIcon('unknown')).toBe('\u2022');
  });
});

describe('getProviderIcon', () => {
  it('returns orange circle for Claude', () => {
    expect(getProviderIcon('claude')).toBe('\u{1F7E0}');
    expect(getProviderIcon('claude-code')).toBe('\u{1F7E0}');
  });

  it('returns blue circle for Gemini', () => {
    expect(getProviderIcon('gemini')).toBe('\u{1F535}');
    expect(getProviderIcon('gemini-cli')).toBe('\u{1F535}');
  });

  it('returns purple circle for Ollama', () => {
    expect(getProviderIcon('ollama')).toBe('\u{1F7E3}');
  });

  it('returns green circle for script', () => {
    expect(getProviderIcon('script')).toBe('\u{1F7E2}');
  });

  it('returns empty string for unknown providers', () => {
    expect(getProviderIcon(undefined)).toBe('');
    expect(getProviderIcon('unknown')).toBe('');
  });
});

describe('getApprovalIcon', () => {
  it('returns hourglass for pending', () => {
    expect(getApprovalIcon('pending')).toBe('\u23F3');
  });

  it('returns check for approved', () => {
    expect(getApprovalIcon('approved')).toBe('\u2713');
  });

  it('returns X for rejected', () => {
    expect(getApprovalIcon('rejected')).toBe('\u2717');
  });

  it('returns empty string for unknown', () => {
    expect(getApprovalIcon(undefined)).toBe('');
  });
});

describe('getApprovalColor', () => {
  it('returns amber for pending', () => {
    expect(getApprovalColor('pending')).toBe(COLORS.pending);
  });

  it('returns green for approved', () => {
    expect(getApprovalColor('approved')).toBe(COLORS.completed);
  });

  it('returns red for rejected', () => {
    expect(getApprovalColor('rejected')).toBe(COLORS.error);
  });

  it('returns gray for unknown', () => {
    expect(getApprovalColor(undefined)).toBe(COLORS.gray);
  });
});

describe('getApprovalBadge', () => {
  it('returns combined icon and color for pending', () => {
    const badge = getApprovalBadge('pending');
    expect(badge.icon).toBe('\u23F3');
    expect(badge.color).toBe(COLORS.pending);
  });

  it('returns combined icon and color for approved', () => {
    const badge = getApprovalBadge('approved');
    expect(badge.icon).toBe('\u2713');
    expect(badge.color).toBe(COLORS.completed);
  });
});

describe('getAgentStatusIcon', () => {
  it('returns correct icons for agent statuses', () => {
    expect(getAgentStatusIcon('idle')).toBe('○');
    expect(getAgentStatusIcon('busy')).toBe('●');
    expect(getAgentStatusIcon('blocked')).toBe('◐');
    expect(getAgentStatusIcon('error')).toBe('✕');
  });
});
