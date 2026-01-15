/**
 * CliCommandHint - Shows the equivalent CLI command for the current data view
 *
 * Displays a copyable `ailang dashboard` command that users can run to
 * verify the same data shown in the UI.
 */
import React, { useState, useCallback } from 'react';
import type { ControlPlaneFilters } from '../types';
import styles from '../ControlPlane.module.css';

export type CommandType = 'inbox' | 'spans' | 'stats' | 'traces' | 'hierarchy' | 'health';

export interface CliCommandHintProps {
  /** Type of dashboard command (inbox, spans, stats, traces, hierarchy, health) */
  commandType: CommandType;
  /** Current filters to include in the command */
  filters?: ControlPlaneFilters;
  /** Optional limit parameter */
  limit?: number;
  /** Optional task ID for traces */
  taskId?: string;
  /** Optional trace ID */
  traceId?: string;
  /** Whether to show in compact mode (single line) */
  compact?: boolean;
}

/**
 * Build the CLI command string from filters
 */
function buildCliCommand(
  commandType: CommandType,
  filters?: ControlPlaneFilters,
  limit?: number,
  taskId?: string,
  traceId?: string
): string {
  const parts: string[] = ['ailang', 'dashboard', commandType];

  // Add filters
  if (filters) {
    if (filters.provider) {
      parts.push(`--provider ${filters.provider}`);
    }
    if (filters.model) {
      parts.push(`--model ${filters.model}`);
    }
    if (filters.source_type) {
      parts.push(`--source ${filters.source_type}`);
    }
    if (filters.workspace) {
      parts.push(`--workspace ${filters.workspace}`);
    }
    if (filters.status && filters.status !== 'all') {
      parts.push(`--status ${filters.status}`);
    }
    if (filters.start_date) {
      parts.push(`--start ${filters.start_date}`);
    }
    if (filters.end_date) {
      parts.push(`--end ${filters.end_date}`);
    }
  }

  // Add optional parameters
  if (limit) {
    parts.push(`--limit ${limit}`);
  }
  if (taskId) {
    parts.push(`--task-id ${taskId}`);
  }
  if (traceId) {
    parts.push(`--trace-id ${traceId}`);
  }

  return parts.join(' ');
}

export const CliCommandHint: React.FC<CliCommandHintProps> = ({
  commandType,
  filters,
  limit,
  taskId,
  traceId,
  compact = false,
}) => {
  const [copied, setCopied] = useState(false);
  const [expanded, setExpanded] = useState(false);

  const command = buildCliCommand(commandType, filters, limit, taskId, traceId);

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(command);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy command:', err);
    }
  }, [command]);

  const handleToggleExpand = useCallback(() => {
    setExpanded(prev => !prev);
  }, []);

  return (
    <div className={`${styles.cliHint} ${compact ? styles.cliHintCompact : ''}`}>
      <span className={styles.cliHintLabel}>CLI:</span>
      <code
        className={`${styles.cliHintCommand} ${expanded ? styles.cliHintExpanded : ''}`}
        onClick={handleToggleExpand}
        title={expanded ? "Click to collapse" : "Click to expand full command"}
        style={{ cursor: 'pointer' }}
      >
        {command}
      </code>
      <button
        className={styles.cliHintCopy}
        onClick={handleCopy}
        title="Copy to clipboard"
      >
        {copied ? '✓' : '⎘'}
      </button>
    </div>
  );
};

export default CliCommandHint;
