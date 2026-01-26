import { describe, it, expect } from 'vitest';
import {
  formatTime,
  formatTimestamp,
  formatDuration,
  formatDurationMs,
  formatCost,
  formatTokens,
  truncateId,
  formatRelativeTime,
  formatDurationMsOpt,
  formatCostOpt,
  formatTokensOpt,
} from './formatters';

describe('formatTime', () => {
  it('formats timestamp to HH:MM', () => {
    const date = new Date('2024-01-15T14:30:00');
    expect(formatTime(date.getTime())).toMatch(/14:30|2:30/); // Depends on locale
  });
});

describe('formatDuration', () => {
  it('formats seconds', () => {
    expect(formatDuration(30)).toBe('30s');
  });

  it('formats minutes and seconds', () => {
    expect(formatDuration(90)).toBe('1m 30s');
  });

  it('formats hours and minutes', () => {
    expect(formatDuration(3700)).toBe('1h 1m');
  });

  it('handles negative values', () => {
    expect(formatDuration(-1)).toBe('Unknown');
  });
});

describe('formatDurationMs', () => {
  it('formats milliseconds', () => {
    expect(formatDurationMs(500)).toBe('500ms');
  });

  it('formats seconds', () => {
    expect(formatDurationMs(2500)).toBe('2.5s');
  });

  it('formats minutes', () => {
    expect(formatDurationMs(120000)).toBe('2.0min');
  });

  it('formats hours', () => {
    expect(formatDurationMs(7200000)).toBe('2.0h');
  });

  it('handles zero', () => {
    expect(formatDurationMs(0)).toBe('0ms');
  });

  it('handles negative values', () => {
    expect(formatDurationMs(-1)).toBe('Unknown');
  });
});

describe('formatDurationMsOpt', () => {
  it('returns fallback for undefined', () => {
    expect(formatDurationMsOpt(undefined)).toBe('');
    expect(formatDurationMsOpt(undefined, '-')).toBe('-');
  });

  it('returns fallback for null', () => {
    expect(formatDurationMsOpt(null)).toBe('');
  });

  it('returns fallback for zero', () => {
    expect(formatDurationMsOpt(0)).toBe('');
  });

  it('formats valid milliseconds', () => {
    expect(formatDurationMsOpt(500)).toBe('500ms');
    expect(formatDurationMsOpt(2500)).toBe('2.5s');
  });
});

describe('formatCost', () => {
  it('formats zero cost', () => {
    expect(formatCost(0)).toBe('$0.00');
  });

  it('formats small costs with precision', () => {
    expect(formatCost(0.0012)).toBe('$0.0012');
  });

  it('formats normal costs', () => {
    expect(formatCost(1.50)).toBe('$1.50');
  });
});

describe('formatCostOpt', () => {
  it('returns fallback for undefined/null/zero', () => {
    expect(formatCostOpt(undefined)).toBe('');
    expect(formatCostOpt(null)).toBe('');
    expect(formatCostOpt(0)).toBe('');
    expect(formatCostOpt(undefined, '-')).toBe('-');
  });

  it('formats valid costs', () => {
    expect(formatCostOpt(0.0012)).toBe('$0.0012');
    expect(formatCostOpt(1.50)).toBe('$1.50');
  });
});

describe('formatTokens', () => {
  it('formats small numbers', () => {
    expect(formatTokens(500)).toBe('500');
  });

  it('formats thousands', () => {
    expect(formatTokens(1500)).toBe('1.5K');
  });

  it('formats millions', () => {
    expect(formatTokens(1500000)).toBe('1.50M');
  });
});

describe('formatTokensOpt', () => {
  it('returns fallback for undefined/null/zero', () => {
    expect(formatTokensOpt(undefined)).toBe('');
    expect(formatTokensOpt(null)).toBe('');
    expect(formatTokensOpt(0)).toBe('');
  });

  it('formats valid token counts', () => {
    expect(formatTokensOpt(500)).toBe('500');
    expect(formatTokensOpt(1500)).toBe('1.5K');
  });
});

describe('truncateId', () => {
  it('returns short IDs unchanged', () => {
    expect(truncateId('abc123')).toBe('abc123');
  });

  it('truncates long IDs', () => {
    expect(truncateId('very-long-id-string-here')).toBe('very-lon...');
  });

  it('respects custom maxLength', () => {
    expect(truncateId('abcdefghij', 5)).toBe('abcdefgh...');
  });
});
