/**
 * Shared formatter utilities for the UI
 * Consolidated from ConversationView, Monitor, and ApprovalQueue
 */

/**
 * Format a timestamp to a short time string (HH:MM)
 */
export const formatTime = (timestamp: number | string): string => {
  const date = new Date(timestamp);
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
};

/**
 * Format a timestamp to a medium date/time string (Mon DD, HH:MM)
 */
export const formatTimestamp = (timestamp: number | string): string => {
  const date = new Date(timestamp);
  return date.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
};

/**
 * Format a duration in seconds to a human-readable string
 */
export const formatDuration = (seconds: number): string => {
  if (seconds < 0) return 'Unknown';
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}m ${secs}s`;
  }
  const hours = Math.floor(seconds / 3600);
  const mins = Math.floor((seconds % 3600) / 60);
  return `${hours}h ${mins}m`;
};

/**
 * Format a duration in milliseconds to an auto-scaled human-readable string
 * Used for chart metrics and analytics displays
 */
export const formatDurationMs = (ms: number): string => {
  if (ms === 0) return '0ms';
  if (ms < 0) return 'Unknown';
  if (ms < 1000) return `${Math.round(ms)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  if (ms < 3600000) return `${(ms / 60000).toFixed(1)}min`;
  return `${(ms / 3600000).toFixed(1)}h`;
};

/**
 * Format a cost value to a currency string
 */
export const formatCost = (cost: number): string => {
  if (cost === 0) return '$0.00';
  if (cost < 0.01) return `$${cost.toFixed(4)}`;
  return `$${cost.toFixed(2)}`;
};

/**
 * Format token counts with K/M suffixes for large numbers
 */
export const formatTokens = (tokens: number): string => {
  if (tokens < 1000) return tokens.toString();
  if (tokens < 1000000) return `${(tokens / 1000).toFixed(1)}K`;
  return `${(tokens / 1000000).toFixed(2)}M`;
};

/**
 * Truncate an ID for display purposes
 */
export const truncateId = (id: string, maxLength: number = 12): string => {
  if (id.length > maxLength) {
    return `${id.slice(0, 8)}...`;
  }
  return id;
};

/**
 * Format a relative time (e.g., "2m ago", "1h ago")
 */
export const formatRelativeTime = (timestamp: number | string): string => {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);

  if (diffSec < 60) return 'just now';
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`;
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`;
  return `${Math.floor(diffSec / 86400)}d ago`;
};
