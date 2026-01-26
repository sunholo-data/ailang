/**
 * ExecHierarchyPopover - Detailed popover for hierarchy nodes
 * Shows span details, metrics, attributes, tool I/O, and CLI hints
 *
 * Extracted from ExecHierarchy.tsx (PR 3 - M-DASHBOARD-SIMPLIFICATION)
 */

import React, { useState, useRef } from 'react';
import type { HierarchyNode, Span } from './types';
import { getNodeIcon, formatDuration } from '../../utils/smartLabel';
import styles from './ExecHierarchy.module.css';

// ============================================================================
// Types
// ============================================================================

export interface ExecHierarchyPopoverProps {
  /** The node to display details for */
  node: HierarchyNode;
  /** Position on screen */
  position: { x: number; y: number };
  /** Close handler */
  onClose: () => void;
  /** Hidden span types (for filter toggle button) */
  hiddenSpanTypes?: Set<string>;
  /** Toggle span type visibility */
  onToggleSpanType?: (spanType: string) => void;
}

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Generate CLI command for a span
 */
function getCliCommand(span: Span | undefined): string {
  if (!span) return '';

  // For spans with trace_id, show trace view command
  if ((span as any).trace_id) {
    return `ailang trace view ${(span as any).trace_id}`;
  }

  // For spans with task_id attribute, show filtered spans
  const taskId = span.attributes?.['task.id'] || span.attributes?.['task_id'] || span.attributes?.['ailang.task_id'];
  if (taskId) {
    return `ailang dashboard spans --task-id ${taskId} --enriched --json`;
  }

  // For spans with session.id, show session tools
  const sessionId = span.attributes?.['session.id'];
  if (sessionId) {
    return `ailang dashboard tools ${sessionId} --json`;
  }

  // Fallback: generic dashboard spans query
  return `ailang dashboard spans --enriched --limit 10 --json`;
}

/**
 * Generate shortened CLI command for display
 */
function getCliCommandDisplay(span: Span | undefined): string {
  if (!span) return '';

  if ((span as any).trace_id) {
    return `ailang trace view ${(span as any).trace_id}`;
  }

  const taskId = span.attributes?.['task.id'] || span.attributes?.['task_id'] || span.attributes?.['ailang.task_id'];
  if (taskId) {
    return `ailang dashboard spans --task-id ${taskId.substring(0, 8)}...`;
  }

  const sessionId = span.attributes?.['session.id'];
  if (sessionId) {
    return `ailang dashboard tools ${sessionId.substring(0, 8)}...`;
  }

  return `ailang dashboard spans --enriched --limit 10`;
}

/**
 * Safely parse and format JSON
 */
function formatJsonValue(value: unknown): string {
  try {
    const parsed = typeof value === 'string' ? JSON.parse(value) : value;
    return JSON.stringify(parsed, null, 2);
  } catch {
    return String(value);
  }
}

// ============================================================================
// Component
// ============================================================================

export const ExecHierarchyPopover: React.FC<ExecHierarchyPopoverProps> = ({
  node,
  position,
  onClose,
  hiddenSpanTypes,
  onToggleSpanType,
}) => {
  const popoverRef = useRef<HTMLDivElement>(null);

  // Collapsible section states
  const [attributesExpanded, setAttributesExpanded] = useState(false);
  const [metricsExpanded, setMetricsExpanded] = useState(false);
  const [toolDetailsExpanded, setToolDetailsExpanded] = useState(false);
  const [chatContextExpanded, setChatContextExpanded] = useState(true); // Default expanded for chat

  const span = node._span;

  return (
    <div
      ref={popoverRef}
      className={styles.nodePopover}
      style={{ left: position.x, top: position.y }}
    >
      {/* Header */}
      <div className={styles.popoverHeader}>
        <span className={styles.popoverIcon}>{getNodeIcon(node.type)}</span>
        <span className={styles.popoverTitle}>{node.label}</span>
        <div className={styles.popoverHeaderActions}>
          {/* Filter toggle button */}
          {span && onToggleSpanType && (
            <button
              className={`${styles.popoverFilterBtn} ${hiddenSpanTypes?.has(span.name) ? styles.popoverFilterActive : ''}`}
              onClick={() => onToggleSpanType(span.name)}
              title={hiddenSpanTypes?.has(span.name) ? 'Show this span type' : 'Hide this span type'}
            >
              {hiddenSpanTypes?.has(span.name) ? '◯' : '◉'}
            </button>
          )}
          <button
            className={styles.popoverClose}
            onClick={onClose}
            title="Close"
          >
            ×
          </button>
        </div>
      </div>

      <div className={styles.popoverBody}>
        {/* Hero Row - Duration + Status */}
        <div className={styles.popoverHero}>
          <div className={styles.popoverDuration}>
            <span className={styles.popoverDurationIcon}>⏱</span>
            <span className={styles.popoverDurationValue}>{formatDuration(node.durationMs)}</span>
          </div>
          <span className={`${styles.popoverStatusBadge} ${styles[`status${node.status.charAt(0).toUpperCase() + node.status.slice(1)}`]}`}>
            {node.status === 'ok' || node.status === 'complete' ? '✓' : node.status === 'error' ? '✕' : '◎'}
            {' '}{node.status}
          </span>
        </div>

        {/* Quick Info Row */}
        <div className={styles.popoverQuickInfo}>
          <span className={styles.popoverQuickItem}>
            <span className={styles.popoverQuickLabel}>Type</span>
            <span className={styles.popoverQuickValue}>{node.type}</span>
          </span>
          {node.provider && (
            <span className={styles.popoverQuickItem}>
              <span className={styles.popoverQuickLabel}>Provider</span>
              <span className={styles.popoverQuickValue}>{node.provider}</span>
            </span>
          )}
          {node.turnNumber && (
            <span className={styles.popoverQuickItem}>
              <span className={styles.popoverQuickLabel}>Turn</span>
              <span className={styles.popoverQuickValue}>{node.turnNumber}</span>
            </span>
          )}
        </div>

        {/* Metrics Section - Collapsible */}
        {(node.cost || node.tokensIn || node.tokensOut) && (
          <div className={styles.popoverSection}>
            <button
              className={styles.popoverSectionToggle}
              onClick={() => setMetricsExpanded(!metricsExpanded)}
            >
              <span className={styles.popoverToggleIcon}>
                {metricsExpanded ? '▼' : '▶'}
              </span>
              <span className={styles.popoverSectionTitle}>
                Metrics
                {node.cost && node.cost > 0 && (
                  <span className={styles.popoverSectionBadge}>
                    ${node.cost < 0.01 ? node.cost.toFixed(4) : node.cost.toFixed(2)}
                  </span>
                )}
              </span>
            </button>
            {metricsExpanded && (
              <div className={styles.popoverMetricsGrid}>
                {node.cost !== undefined && node.cost > 0 && (
                  <div className={styles.popoverMetricItem}>
                    <span className={styles.popoverMetricLabel}>Cost</span>
                    <span className={styles.popoverMetricValue}>
                      ${node.cost < 0.01 ? node.cost.toFixed(4) : node.cost.toFixed(3)}
                    </span>
                  </div>
                )}
                {node.tokensIn !== undefined && node.tokensIn > 0 && (
                  <div className={styles.popoverMetricItem}>
                    <span className={styles.popoverMetricLabel}>Input</span>
                    <span className={styles.popoverMetricValue}>
                      {node.tokensIn >= 1000 ? `${(node.tokensIn / 1000).toFixed(1)}K` : node.tokensIn} tokens
                    </span>
                  </div>
                )}
                {node.tokensOut !== undefined && node.tokensOut > 0 && (
                  <div className={styles.popoverMetricItem}>
                    <span className={styles.popoverMetricLabel}>Output</span>
                    <span className={styles.popoverMetricValue}>
                      {node.tokensOut >= 1000 ? `${(node.tokensOut / 1000).toFixed(1)}K` : node.tokensOut} tokens
                    </span>
                  </div>
                )}
              </div>
            )}
          </div>
        )}

        {/* Agent ID */}
        {node.agentId && (
          <div className={styles.popoverCompactInfo}>
            <span className={styles.popoverCompactLabel}>Agent</span>
            <span className={styles.popoverCompactValue}>{node.agentId}</span>
          </div>
        )}

        {/* Coordinator Context */}
        {(node.taskId || node.parentTaskId) && (
          <div className={styles.popoverSection}>
            <div className={styles.popoverSectionTitle}>Task Context</div>
            <div className={styles.popoverInfoList}>
              {node.taskId && (
                <div className={styles.popoverInfoRow}>
                  <span className={styles.popoverInfoLabel}>Task ID</span>
                  <span className={styles.popoverInfoValue} style={{ fontFamily: 'var(--font-mono)' }}>
                    {node.taskId.substring(0, 16)}...
                  </span>
                </div>
              )}
              {node.parentTaskId && (
                <div className={styles.popoverInfoRow}>
                  <span className={styles.popoverInfoLabel}>Parent Task</span>
                  <span className={styles.popoverInfoValue} style={{ fontFamily: 'var(--font-mono)' }}>
                    {node.parentTaskId.substring(0, 16)}...
                  </span>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Approval Status */}
        {node.approvalStatus && node.approvalStatus !== 'none' && (
          <div className={styles.popoverSection}>
            <div className={styles.popoverSectionTitle}>Approval</div>
            <div className={styles.popoverApproval}>
              <span className={`${styles.popoverApprovalStatus} ${styles[`approval${node.approvalStatus.charAt(0).toUpperCase() + node.approvalStatus.slice(1)}`]}`}>
                {node.approvalStatus === 'pending' ? '⏳' : node.approvalStatus === 'approved' ? '✓' : '✗'}
                {' '}{node.approvalStatus}
              </span>
            </div>
          </div>
        )}

        {/* Chat Context - Embedded conversation content */}
        {span?.chat_context && (
          <div className={styles.popoverSection}>
            <button
              className={styles.popoverSectionToggle}
              onClick={() => setChatContextExpanded(!chatContextExpanded)}
            >
              <span className={styles.popoverToggleIcon}>
                {chatContextExpanded ? '▼' : '▶'}
              </span>
              <span className={styles.popoverSectionTitle}>
                💬 Chat Context
                {span.chat_context.turn_number && (
                  <span className={styles.popoverSectionBadge}>
                    Turn {span.chat_context.turn_number}
                  </span>
                )}
              </span>
            </button>
            {chatContextExpanded && (
              <div className={styles.chatContextContent}>
                {/* User Prompt */}
                {span.chat_context.user_prompt && (
                  <div className={`${styles.chatContextMessage} ${styles.userMessage}`}>
                    <div className={styles.chatContextRole}>👤 User</div>
                    <div className={styles.chatContextText}>
                      {span.chat_context.user_prompt}
                    </div>
                  </div>
                )}
                {/* Assistant Response */}
                {span.chat_context.assistant_response && (
                  <div className={`${styles.chatContextMessage} ${styles.assistantMessage}`}>
                    <div className={styles.chatContextRole}>🤖 Assistant</div>
                    <div className={styles.chatContextText}>
                      {span.chat_context.assistant_response}
                    </div>
                  </div>
                )}
                {/* Thinking indicator */}
                {span.chat_context.has_thinking && (
                  <div className={styles.chatContextThinking}>
                    💭 Includes thinking blocks
                  </div>
                )}
                {/* View full conversation link */}
                {span.chat_context.full_chat_url && (
                  <div className={styles.chatContextLink}>
                    <a
                      href={span.chat_context.full_chat_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      onClick={(e) => {
                        e.preventDefault();
                        // Copy URL to clipboard for now (could be expanded to open in panel)
                        navigator.clipboard.writeText(window.location.origin + span.chat_context!.full_chat_url!);
                      }}
                      title="Click to copy API URL for full conversation"
                    >
                      📋 Copy full chat URL
                    </a>
                  </div>
                )}
              </div>
            )}
          </div>
        )}

        {/* Span-specific sections */}
        {span && (
          <>
            {/* Tool Details Section - Collapsible */}
            <div className={styles.popoverSection}>
              <button
                className={styles.popoverSectionToggle}
                onClick={() => setToolDetailsExpanded(!toolDetailsExpanded)}
              >
                <span className={styles.popoverToggleIcon}>
                  {toolDetailsExpanded ? '▼' : '▶'}
                </span>
                <span className={styles.popoverSectionTitle}>
                  Tool Details
                  <span className={styles.popoverSectionBadge}>
                    {span.name}
                  </span>
                </span>
              </button>
              {toolDetailsExpanded && (
                <div className={styles.popoverInfoList}>
                  {/* Display Name (enriched) */}
                  <div className={styles.popoverInfoRow}>
                    <span className={styles.popoverInfoLabel}>Display Name</span>
                    <span className={styles.popoverInfoValue}>
                      {span.display_name || node.label}
                    </span>
                  </div>
                  {/* Raw Span Name */}
                  <div className={styles.popoverInfoRow}>
                    <span className={styles.popoverInfoLabel}>Span Type</span>
                    <span className={styles.popoverInfoValue} style={{ fontFamily: 'var(--font-mono)', fontSize: '11px' }}>
                      {span.name}
                    </span>
                  </div>
                  {/* Timestamp */}
                  <div className={styles.popoverInfoRow}>
                    <span className={styles.popoverInfoLabel}>Start Time</span>
                    <span className={styles.popoverInfoValue}>
                      {span.startMs ? new Date(span.startMs).toLocaleString() : '-'}
                    </span>
                  </div>
                  {/* Tool Input */}
                  {(span as any).tool_input && (
                    <div className={styles.popoverNestedSection}>
                      <div className={styles.popoverSectionTitle}>
                        Tool Input
                        {(span as any).tool_success !== undefined && (
                          <span className={(span as any).tool_success ? styles.popoverToolSuccess : styles.popoverToolError}>
                            {(span as any).tool_success ? ' ✓' : ' ✗'}
                          </span>
                        )}
                      </div>
                      <pre className={styles.popoverCodeBlock}>
                        {formatJsonValue((span as any).tool_input)}
                      </pre>
                    </div>
                  )}
                  {/* Tool Response */}
                  {(span as any).tool_response && (
                    <div className={styles.popoverNestedSection}>
                      <div className={styles.popoverSectionTitle}>Tool Response</div>
                      <pre className={styles.popoverCodeBlock}>
                        {formatJsonValue((span as any).tool_response)}
                      </pre>
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* Span ID */}
            <div className={styles.popoverSection}>
              <div className={styles.popoverSectionTitle}>Span ID</div>
              <div
                className={styles.popoverIdValue}
                onClick={() => navigator.clipboard.writeText(span.id)}
                title="Click to copy"
                style={{ cursor: 'pointer' }}
              >
                {span.id}
                <span className={styles.popoverCopyHint}>📋</span>
              </div>
            </div>

            {/* CLI Command Hint */}
            <div className={styles.popoverCliHint}>
              <div className={styles.popoverSectionTitle}>CLI Command</div>
              <div
                className={styles.popoverCliCommand}
                onClick={() => navigator.clipboard.writeText(getCliCommand(span))}
                title="Click to copy"
              >
                <code>{getCliCommandDisplay(span)}</code>
                <span className={styles.popoverCopyHint}>📋</span>
              </div>
            </div>

            {/* Attributes - Collapsible */}
            {span.attributes && Object.keys(span.attributes).length > 0 && (
              <div className={styles.popoverSection}>
                <button
                  className={styles.popoverSectionToggle}
                  onClick={() => setAttributesExpanded(!attributesExpanded)}
                >
                  <span className={styles.popoverToggleIcon}>
                    {attributesExpanded ? '▼' : '▶'}
                  </span>
                  <span className={styles.popoverSectionTitle}>
                    Attributes ({Object.keys(span.attributes).length})
                  </span>
                </button>
                {attributesExpanded && (
                  <div className={styles.popoverAttributesList}>
                    {Object.entries(span.attributes).map(([key, value]) => (
                      <div key={key} className={styles.popoverAttrRow}>
                        <span className={styles.popoverAttrKey}>{key}</span>
                        <span className={styles.popoverAttrValue}>
                          {String(value).length > 100
                            ? `${String(value).substring(0, 100)}...`
                            : String(value)}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </>
        )}

        {/* Custom Attributes Section - for nodes without _span */}
        {!span && node.attributes && Object.keys(node.attributes).length > 0 && (
          <div className={styles.popoverSection}>
            <button
              className={styles.popoverSectionToggle}
              onClick={() => setAttributesExpanded(!attributesExpanded)}
            >
              <span className={styles.popoverToggleIcon}>
                {attributesExpanded ? '▼' : '▶'}
              </span>
              <span className={styles.popoverSectionTitle}>
                Details ({Object.keys(node.attributes).length})
              </span>
            </button>
            {attributesExpanded && (
              <div className={styles.popoverAttributesList}>
                {Object.entries(node.attributes).map(([key, value]) => (
                  <div key={key} className={styles.popoverAttrRow}>
                    <span className={styles.popoverAttrKey}>{key}</span>
                    <span
                      className={styles.popoverAttrValue}
                      style={{ whiteSpace: 'pre-wrap' }}
                    >
                      {String(value).length > 500
                        ? `${String(value).substring(0, 500)}...`
                        : String(value)}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

export default ExecHierarchyPopover;
