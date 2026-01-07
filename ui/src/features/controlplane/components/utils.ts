/**
 * Shared utility functions for Control Plane components
 */
import type { Agent } from './types';

// Get trust level from score
export const getTrustLevel = (score: number): string => {
  if (score >= 85) return 'auto';
  if (score >= 60) return 'low-risk';
  if (score >= 25) return 'review';
  return 'manual';
};

// Get status icon for agent
export const getStatusIcon = (status: Agent['status']): string => {
  switch (status) {
    case 'idle': return '○';
    case 'busy': return '●';
    case 'blocked': return '◐';
    case 'error': return '✕';
  }
};

// Format duration in human-readable format
export const formatDuration = (ms: number): string => {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
};

// Default trust capabilities
export const defaultTrustCapabilities = [
  { name: 'Read Files', score: 75, icon: '◉' },
  { name: 'Write Docs', score: 75, icon: '◎' },
  { name: 'Write Code', score: 50, icon: '⬡' },
  { name: 'Run Tests', score: 75, icon: '▣' },
  { name: 'Git Commit', score: 50, icon: '◈' },
  { name: 'Git Push', score: 25, icon: '◇' },
  { name: 'Release', score: 0, icon: '⬢' },
];
