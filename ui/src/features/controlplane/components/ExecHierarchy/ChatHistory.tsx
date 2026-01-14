/**
 * ChatHistory - Shows conversation history for selected task/span
 *
 * Supports two data sources:
 * 1. Direct Claude Code sessions (OTEL spans) - uses hierarchy children (turns/tool_uses)
 * 2. Coordinator tasks - fetches events from /api/coordinator/tasks/{taskId}/events
 */
import React, { useState, useEffect, useMemo } from 'react';
import type { HierarchyNode, Span } from './types';
import styles from './ExecHierarchy.module.css';

export interface ChatHistoryProps {
  /** All hierarchy nodes */
  nodes: HierarchyNode[];
  /** Currently selected node ID */
  selectedNodeId?: string | null;
  /** Callback when a node is clicked */
  onNodeClick?: (node: HierarchyNode) => void;
  /** Loading state */
  loading?: boolean;
  /** All spans (for finding selected span) */
  spans?: Span[];
}

interface TaskEvent {
  id: string;
  task_id: string;
  stream_type: string;
  turn_num: number;
  text: string;
  tool_name?: string;
  tool_input?: string;
  tool_output?: string;
  error_msg?: string;
  tokens_in?: number;
  tokens_out?: number;
  cost?: number;
  created_at: string;
}

interface Turn {
  turnNumber: number;
  events: TaskEvent[];
  startTime?: string;
}

// Simplified turn structure for hierarchy-based display
interface HierarchyTurn {
  turnNumber: number;
  node: HierarchyNode;
  toolUses: HierarchyNode[];
}

/**
 * Consolidate consecutive text events into single message blocks.
 * Streaming text comes as many small chunks - we merge them for display.
 */
function consolidateTextEvents(events: TaskEvent[]): TaskEvent[] {
  const result: TaskEvent[] = [];
  let currentTextBlock: TaskEvent | null = null;

  for (const event of events) {
    if (event.stream_type === 'text' && event.text) {
      if (currentTextBlock) {
        // Append to existing text block
        currentTextBlock = {
          ...currentTextBlock,
          text: currentTextBlock.text + event.text,
        };
      } else {
        // Start new text block
        currentTextBlock = { ...event };
      }
    } else {
      // Non-text event: flush any pending text block
      if (currentTextBlock) {
        result.push(currentTextBlock);
        currentTextBlock = null;
      }
      result.push(event);
    }
  }

  // Flush final text block
  if (currentTextBlock) {
    result.push(currentTextBlock);
  }

  return result;
}

/**
 * Group events by turn number (for coordinator task events)
 */
function groupEventsByTurn(events: TaskEvent[]): Turn[] {
  const turnMap = new Map<number, TaskEvent[]>();

  for (const event of events) {
    const turnNum = event.turn_num || 0;
    if (!turnMap.has(turnNum)) {
      turnMap.set(turnNum, []);
    }
    turnMap.get(turnNum)!.push(event);
  }

  const turns: Turn[] = [];
  const sortedKeys = Array.from(turnMap.keys()).sort((a, b) => a - b);

  for (const turnNum of sortedKeys) {
    const turnEvents = turnMap.get(turnNum)!;
    // Sort by time, then consolidate consecutive text events
    const sortedEvents = turnEvents.sort((a, b) =>
      new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
    );
    const consolidatedEvents = consolidateTextEvents(sortedEvents);

    turns.push({
      turnNumber: turnNum,
      events: consolidatedEvents,
      startTime: turnEvents[0]?.created_at,
    });
  }

  return turns;
}

/**
 * Extract turns and tool uses from hierarchy node children (for OTEL spans)
 * Searches recursively since turns may be nested (e.g., coordinator → claude.execute → exec.turn)
 */
function extractHierarchyTurns(node: HierarchyNode): HierarchyTurn[] {
  const turns: HierarchyTurn[] = [];

  // Recursively collect all turn nodes from the hierarchy
  const collectTurns = (n: HierarchyNode) => {
    if (!n.children) return;

    for (const child of n.children) {
      if (child.type === 'turn') {
        const toolUses = child.children?.filter(c => c.type === 'tool_use') || [];
        turns.push({
          turnNumber: child.turnNumber || turns.length + 1,
          node: child,
          toolUses,
        });
      } else {
        // Recursively search in non-turn children (e.g., exec nodes like claude.execute)
        collectTurns(child);
      }
    }
  };

  collectTurns(node);

  // Sort turns by turnNumber
  turns.sort((a, b) => a.turnNumber - b.turnNumber);

  // If no explicit turns found, treat all tool_use children as a single implicit turn
  if (turns.length === 0 && node.children) {
    const collectToolUses = (n: HierarchyNode): HierarchyNode[] => {
      const tools: HierarchyNode[] = [];
      if (!n.children) return tools;
      for (const child of n.children) {
        if (child.type === 'tool_use') {
          tools.push(child);
        } else {
          tools.push(...collectToolUses(child));
        }
      }
      return tools;
    };

    const toolUses = collectToolUses(node);
    if (toolUses.length > 0) {
      turns.push({
        turnNumber: 1,
        node: node,
        toolUses,
      });
    }
  }

  return turns;
}

/**
 * Find node by ID in hierarchy
 */
function findNodeById(nodes: HierarchyNode[], id: string): HierarchyNode | null {
  for (const node of nodes) {
    if (node.id === id) return node;
    if (node.children) {
      const found = findNodeById(node.children, id);
      if (found) return found;
    }
  }
  return null;
}

/**
 * Find span by ID in spans array (recursive)
 */
function findSpanById(spans: Span[], id: string): Span | null {
  for (const span of spans) {
    if (span.id === id) return span;
    if (span.children) {
      const found = findSpanById(span.children, id);
      if (found) return found;
    }
  }
  return null;
}

/**
 * Format JSON for display
 */
function formatJson(input: string | undefined): React.ReactNode {
  if (!input) return null;

  try {
    const parsed = JSON.parse(input);
    const formatted = JSON.stringify(parsed, null, 2);
    return <pre className={styles.chatJsonContent}>{formatted}</pre>;
  } catch {
    return <pre className={styles.chatJsonContent}>{input}</pre>;
  }
}

/**
 * Format timestamp for display
 */
function formatTime(timestamp: string | number | undefined): string {
  if (!timestamp) return '—';
  try {
    if (typeof timestamp === 'number') {
      return new Date(timestamp).toLocaleTimeString();
    }
    return new Date(timestamp).toLocaleTimeString();
  } catch {
    return String(timestamp);
  }
}

/**
 * Format duration for display
 */
function formatDuration(ms: number | undefined): string {
  if (!ms) return '—';
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

export const ChatHistory: React.FC<ChatHistoryProps> = ({
  nodes,
  selectedNodeId,
  onNodeClick,
  loading,
  spans,
}) => {
  const [coordinatorEvents, setCoordinatorEvents] = useState<TaskEvent[]>([]);
  const [eventsLoading, setEventsLoading] = useState(false);
  const [eventsError, setEventsError] = useState<string | null>(null);
  const [expandedTools, setExpandedTools] = useState<Set<string>>(new Set());

  // Find selected node in hierarchy (by ID or use first root node)
  const selectedNode = useMemo(() => {
    if (!selectedNodeId) return null;

    // Try exact match first
    const exactMatch = findNodeById(nodes, selectedNodeId);
    if (exactMatch) return exactMatch;

    // If selectedNodeId looks like a task_id (e.g., "task-xxxx") but nodes use span_ids,
    // check if any node's taskId matches, or use the first root node as fallback
    const nodeWithTaskId = nodes.find(n =>
      n.taskId === selectedNodeId ||
      n._span?.attributes?.['task_id'] === selectedNodeId
    );
    if (nodeWithTaskId) return nodeWithTaskId;

    // Final fallback: use first root node if we have nodes but ID doesn't match
    // (This happens when selectedNodeId is a task_id but nodes have span_ids)
    if (nodes.length > 0) return nodes[0];

    return null;
  }, [nodes, selectedNodeId]);

  // Find selected span (for additional metadata)
  const selectedSpan = useMemo(() => {
    if (!selectedNodeId || !spans) return null;

    // Try exact match first
    const exactMatch = findSpanById(spans, selectedNodeId);
    if (exactMatch) return exactMatch;

    // If no match but we have spans, use first root span (same fallback logic as nodes)
    if (spans.length > 0) return spans[0];

    return null;
  }, [spans, selectedNodeId]);

  // Extract task ID (for coordinator tasks)
  // Priority: node.taskId > span.attributes (multiple keys) > selectedNodeId if it looks like a task_id
  const taskId = useMemo(() => {
    if (selectedNode?.taskId) return selectedNode.taskId;

    // Check span attributes - different executors use different attribute names!
    if (selectedSpan?.attributes) {
      const attrs = selectedSpan.attributes;
      // Coordinator uses "task.id" (DOT notation) - THIS WAS THE BUG!
      if (attrs['task.id']) return attrs['task.id'];
      // Gemini executor uses "exec.task_id"
      if (attrs['exec.task_id']) return attrs['exec.task_id'];
      // Fallback: underscore notation (legacy)
      if (attrs['task_id']) return attrs['task_id'];
    }

    // If selectedNodeId looks like a task_id, use it directly
    if (selectedNodeId && selectedNodeId.startsWith('task-')) return selectedNodeId;
    return undefined;
  }, [selectedNode, selectedSpan, selectedNodeId]);

  // Extract hierarchy turns (for OTEL spans)
  const hierarchyTurns = useMemo(() => {
    return selectedNode ? extractHierarchyTurns(selectedNode) : [];
  }, [selectedNode]);

  // Fetch coordinator events if we have a taskId
  // ALWAYS fetch for coordinator tasks - they have actual conversation text
  // OTEL spans only have tool metadata, not the assistant's text
  useEffect(() => {
    if (!taskId) {
      setCoordinatorEvents([]);
      return;
    }

    const fetchEvents = async () => {
      setEventsLoading(true);
      setEventsError(null);

      try {
        const response = await fetch(`/api/coordinator/tasks/${taskId}/events?limit=500`);
        if (!response.ok) {
          throw new Error(`Failed to fetch events: ${response.statusText}`);
        }
        const data = await response.json();
        setCoordinatorEvents(data.events || []);
      } catch (err) {
        setEventsError(err instanceof Error ? err.message : 'Failed to load events');
        setCoordinatorEvents([]);
      } finally {
        setEventsLoading(false);
      }
    };

    fetchEvents();
  }, [taskId]);

  // Group coordinator events by turn
  const coordinatorTurns = useMemo(() => {
    return groupEventsByTurn(coordinatorEvents);
  }, [coordinatorEvents]);

  // Determine if we have hierarchy data (OTEL spans) - used as fallback only
  const hasHierarchyData = hierarchyTurns.length > 0;

  // Toggle tool expansion
  const toggleTool = (id: string) => {
    setExpandedTools(prev => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  // Render loading state
  if (loading || eventsLoading) {
    return (
      <div className={styles.chatContainer}>
        <div className={styles.chatLoading}>Loading conversation...</div>
      </div>
    );
  }

  // Render no selection state
  if (!selectedNodeId) {
    return (
      <div className={styles.chatContainer}>
        <div className={styles.chatEmpty}>
          <div className={styles.chatEmptyIcon}>💬</div>
          <div className={styles.chatEmptyText}>Select a task to view conversation</div>
          <div className={styles.chatEmptyHint}>
            Click on a session in the tree or graph view
          </div>
        </div>
      </div>
    );
  }

  // Render error state (only if no fallback data available)
  if (eventsError && !hasHierarchyData) {
    return (
      <div className={styles.chatContainer}>
        <div className={styles.chatError}>
          <div className={styles.chatErrorIcon}>⚠️</div>
          <div className={styles.chatErrorText}>{eventsError}</div>
        </div>
      </div>
    );
  }

  // ========================================
  // RENDER: Coordinator task events (PRIORITY - has actual conversation text)
  // ========================================
  if (coordinatorTurns.length > 0) {
    return (
      <div className={styles.chatContainer}>
        {/* Header */}
        <div className={styles.chatHeader}>
          <span className={styles.chatHeaderIcon}>💬</span>
          <span className={styles.chatHeaderTitle}>
            {selectedNode?.label || 'Task Conversation'}
          </span>
          <span className={styles.chatHeaderTurnCount}>
            {coordinatorTurns.length} turn{coordinatorTurns.length !== 1 ? 's' : ''}
          </span>
        </div>

        {/* Turns from coordinator */}
        <div className={styles.chatTurns}>
          {coordinatorTurns.map(turn => (
            <div key={turn.turnNumber} className={styles.chatTurn}>
              <div className={styles.chatTurnHeader}>
                <span className={styles.chatTurnNumber}>Turn {turn.turnNumber}</span>
                <span className={styles.chatTurnTime}>{formatTime(turn.startTime)}</span>
              </div>

              <div className={styles.chatTurnEvents}>
                {turn.events.map(event => (
                  <div
                    key={event.id}
                    className={`${styles.chatEvent} ${styles[`chatEvent_${event.stream_type}`]}`}
                  >
                    {event.stream_type === 'text' && event.text && (
                      <div className={styles.chatTextEvent}>
                        <div className={styles.chatTextContent}>{event.text}</div>
                      </div>
                    )}

                    {event.stream_type === 'tool_use' && (
                      <div className={styles.chatToolEvent}>
                        <button
                          className={styles.chatToolHeader}
                          onClick={() => toggleTool(event.id)}
                        >
                          <span className={styles.chatToolIcon}>🔧</span>
                          <span className={styles.chatToolName}>{event.tool_name || 'Tool'}</span>
                          <span className={styles.chatToolExpand}>
                            {expandedTools.has(event.id) ? '▼' : '▶'}
                          </span>
                        </button>

                        {expandedTools.has(event.id) && (
                          <div className={styles.chatToolDetails}>
                            {event.tool_input && (
                              <div className={styles.chatToolSection}>
                                <div className={styles.chatToolSectionLabel}>Input</div>
                                {formatJson(event.tool_input)}
                              </div>
                            )}
                            {event.tool_output && (
                              <div className={styles.chatToolSection}>
                                <div className={styles.chatToolSectionLabel}>Output</div>
                                {formatJson(event.tool_output)}
                              </div>
                            )}
                          </div>
                        )}
                      </div>
                    )}

                    {event.stream_type === 'tool_result' && (
                      <div className={styles.chatToolResultEvent}>
                        <span className={styles.chatToolResultIcon}>✓</span>
                        <span className={styles.chatToolResultText}>
                          {event.tool_name ? `${event.tool_name} completed` : 'Tool completed'}
                        </span>
                      </div>
                    )}

                    {event.stream_type === 'error' && (
                      <div className={styles.chatErrorEvent}>
                        <span className={styles.chatErrorEventIcon}>❌</span>
                        <span className={styles.chatErrorEventText}>
                          {event.error_msg || event.text || 'Error occurred'}
                        </span>
                      </div>
                    )}

                    {event.stream_type === 'status' && (
                      <div className={styles.chatStatusEvent}>
                        <span className={styles.chatStatusEventIcon}>ℹ️</span>
                        <span className={styles.chatStatusEventText}>{event.text}</span>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>

        {/* CLI hint for coordinator task */}
        {taskId && (
          <div className={styles.chatCliHint}>
            <span className={styles.chatCliLabel}>CLI:</span>
            <code className={styles.chatCliCommand}>
              ailang coordinator logs {taskId} --limit 100
            </code>
            <button
              className={styles.chatCliCopy}
              onClick={() => {
                navigator.clipboard.writeText(`ailang coordinator logs ${taskId} --limit 100`);
              }}
              title="Copy full command"
            >
              ⎘
            </button>
          </div>
        )}
      </div>
    );
  }

  // ========================================
  // RENDER: OTEL hierarchy-based conversation (fallback for non-coordinator tasks)
  // ========================================
  if (hasHierarchyData) {
    return (
      <div className={styles.chatContainer}>
        {/* Header */}
        <div className={styles.chatHeader}>
          <span className={styles.chatHeaderIcon}>💬</span>
          <span className={styles.chatHeaderTitle}>
            {selectedNode?.label || 'Session Conversation'}
          </span>
          <span className={styles.chatHeaderTurnCount}>
            {hierarchyTurns.length} turn{hierarchyTurns.length !== 1 ? 's' : ''}
          </span>
        </div>

        {/* Turns from hierarchy */}
        <div className={styles.chatTurns}>
          {hierarchyTurns.map((turn, idx) => (
            <div key={turn.node.id || idx} className={styles.chatTurn}>
              {/* Turn header */}
              <div className={styles.chatTurnHeader}>
                <span className={styles.chatTurnNumber}>Turn {turn.turnNumber}</span>
                <span className={styles.chatTurnTime}>
                  {turn.node.startTime ? formatTime(turn.node.startTime) : '—'}
                </span>
                {turn.node.durationMs && (
                  <span className={styles.chatTurnDuration}>
                    {formatDuration(turn.node.durationMs)}
                  </span>
                )}
              </div>

              {/* Tool uses in this turn */}
              <div className={styles.chatTurnEvents}>
                {turn.toolUses.length === 0 ? (
                  <div className={styles.chatTextEvent}>
                    <div className={styles.chatTextContent}>
                      {turn.node.label || 'Processing...'}
                    </div>
                  </div>
                ) : (
                  turn.toolUses.map(tool => (
                    <div key={tool.id} className={styles.chatEvent}>
                      <div className={styles.chatToolEvent}>
                        <button
                          className={styles.chatToolHeader}
                          onClick={() => toggleTool(tool.id)}
                        >
                          <span className={styles.chatToolIcon}>🔧</span>
                          <span className={styles.chatToolName}>
                            {tool.label || tool._span?.name || 'Tool'}
                          </span>
                          {tool.durationMs && (
                            <span className={styles.chatToolDuration}>
                              {formatDuration(tool.durationMs)}
                            </span>
                          )}
                          <span className={`${styles.chatToolStatus} ${styles[`status_${tool.status}`]}`}>
                            {tool.status === 'completed' ? '✓' : tool.status === 'error' ? '✗' : '•'}
                          </span>
                          <span className={styles.chatToolExpand}>
                            {expandedTools.has(tool.id) ? '▼' : '▶'}
                          </span>
                        </button>

                        {expandedTools.has(tool.id) && tool._span?.attributes && (
                          <div className={styles.chatToolDetails}>
                            <div className={styles.chatToolSection}>
                              <div className={styles.chatToolSectionLabel}>Attributes</div>
                              <div className={styles.chatToolAttributes}>
                                {Object.entries(tool._span.attributes).map(([key, value]) => (
                                  <div key={key} className={styles.chatMetaRow}>
                                    <span className={styles.chatMetaLabel}>{key}</span>
                                    <span className={styles.chatMetaValue}>{value}</span>
                                  </div>
                                ))}
                              </div>
                            </div>
                          </div>
                        )}
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>
          ))}
        </div>

        {/* CLI hint for OTEL trace */}
        {selectedSpan && (
          <div className={styles.chatCliHint}>
            <span className={styles.chatCliLabel}>CLI:</span>
            <code className={styles.chatCliCommand}>
              ailang trace view {selectedSpan.id}
            </code>
            <button
              className={styles.chatCliCopy}
              onClick={() => {
                navigator.clipboard.writeText(`ailang trace view ${selectedSpan.id}`);
              }}
              title="Copy full command"
            >
              ⎘
            </button>
          </div>
        )}
      </div>
    );
  }

  // ========================================
  // RENDER: Fallback for spans without conversation data
  // ========================================
  return (
    <div className={styles.chatContainer}>
      <div className={styles.chatFallback}>
        <div className={styles.chatFallbackHeader}>
          <span className={styles.chatFallbackIcon}>📋</span>
          <span className={styles.chatFallbackTitle}>
            {selectedNode?.label || selectedSpan?.display_name || selectedSpan?.name || 'Span Details'}
          </span>
        </div>

        {selectedNode && (
          <div className={styles.chatFallbackMeta}>
            <div className={styles.chatMetaRow}>
              <span className={styles.chatMetaLabel}>Type</span>
              <span className={styles.chatMetaValue}>{selectedNode.type}</span>
            </div>
            <div className={styles.chatMetaRow}>
              <span className={styles.chatMetaLabel}>Status</span>
              <span className={`${styles.chatMetaValue} ${styles[`status_${selectedNode.status}`]}`}>
                {selectedNode.status}
              </span>
            </div>
            {selectedNode.durationMs !== undefined && (
              <div className={styles.chatMetaRow}>
                <span className={styles.chatMetaLabel}>Duration</span>
                <span className={styles.chatMetaValue}>{formatDuration(selectedNode.durationMs)}</span>
              </div>
            )}
            {selectedNode.startTime && (
              <div className={styles.chatMetaRow}>
                <span className={styles.chatMetaLabel}>Started</span>
                <span className={styles.chatMetaValue}>
                  {new Date(selectedNode.startTime).toLocaleString()}
                </span>
              </div>
            )}
            {selectedNode.cost !== undefined && (
              <div className={styles.chatMetaRow}>
                <span className={styles.chatMetaLabel}>Cost</span>
                <span className={styles.chatMetaValue}>${selectedNode.cost.toFixed(4)}</span>
              </div>
            )}
            {selectedNode.tokensIn !== undefined && (
              <div className={styles.chatMetaRow}>
                <span className={styles.chatMetaLabel}>Tokens In</span>
                <span className={styles.chatMetaValue}>{selectedNode.tokensIn.toLocaleString()}</span>
              </div>
            )}
            {selectedNode.tokensOut !== undefined && (
              <div className={styles.chatMetaRow}>
                <span className={styles.chatMetaLabel}>Tokens Out</span>
                <span className={styles.chatMetaValue}>{selectedNode.tokensOut.toLocaleString()}</span>
              </div>
            )}
          </div>
        )}

        {selectedSpan?.attributes && Object.keys(selectedSpan.attributes).length > 0 && (
          <div className={styles.chatFallbackAttributes}>
            <div className={styles.chatAttributesHeader}>Attributes</div>
            {Object.entries(selectedSpan.attributes).map(([key, value]) => (
              <div key={key} className={styles.chatMetaRow}>
                <span className={styles.chatMetaLabel}>{key}</span>
                <span className={styles.chatMetaValue}>{value}</span>
              </div>
            ))}
          </div>
        )}

        <div className={styles.chatFallbackHint}>
          No conversation data available for this span.
          {selectedNode?.children && selectedNode.children.length > 0 && (
            <span> This span has {selectedNode.children.length} child span(s) - try expanding in tree view.</span>
          )}
        </div>
      </div>
    </div>
  );
};

export default ChatHistory;
