/**
 * ExecHierarchy - Main container for the 4-level exec hierarchy view
 * Provides toggle between tree and graph views
 * Shows: Messages -> Execs -> Turns -> Tool Uses
 *
 * Uses the SAME spans data as TraceWaterfall for consistency.
 */
import React, { useState, useCallback, useMemo, useEffect, useRef } from 'react';
import type { HierarchyNode, ViewMode, ExecHierarchyProps, Span, NodeStatus } from './types';
import { ExecHierarchyTree } from './ExecHierarchyTree';
import { ExecHierarchyGraph } from './ExecHierarchyGraph';
import styles from './ExecHierarchy.module.css';

// Popover max dimensions
const POPOVER_MAX_WIDTH = 420;
const POPOVER_MAX_HEIGHT = 500;
const POPOVER_MARGIN = 16;

// Determine node type from span name
function getNodeType(name: string): HierarchyNode['type'] {
  const lower = name.toLowerCase();
  if (lower.includes('message') || lower.includes('msg')) return 'message';
  if (lower.includes('turn')) return 'turn';
  if (lower.includes('tool') || lower.includes('bash') || lower.includes('read') || lower.includes('write') || lower.includes('edit')) return 'tool_use';
  // Default to exec for exec.*, compile.*, etc.
  return 'exec';
}

// Convert span status to node status
function getNodeStatus(span: Span): NodeStatus {
  if (span.status === 'error') return 'error';
  if (span.status === 'ok') return 'completed';
  // Check if still running (no end time, or endMs === startMs for in-progress)
  if (span.durationMs === 0) return 'busy';
  return 'completed';
}

// Extract turn number from span attributes or sibling index
function getTurnNumber(span: Span, siblingIndex?: number): number | undefined {
  // Try to get from span attributes
  const fromAttr = span.attributes?.['turn.number']
    || span.attributes?.['exec.turn']
    || span.attributes?.['turn_number'];

  if (fromAttr) {
    const num = parseInt(String(fromAttr), 10);
    if (!isNaN(num)) return num;
  }

  // Fall back to sibling index (1-based)
  if (siblingIndex !== undefined) return siblingIndex + 1;

  return undefined;
}

// Transform Span[] to HierarchyNode[] (recursive)
function spanToHierarchyNode(span: Span, siblingIndex?: number): HierarchyNode {
  const nodeType = getNodeType(span.name);
  const isTurn = nodeType === 'turn';
  const turnNumber = isTurn ? getTurnNumber(span, siblingIndex) : undefined;

  // Generate label with turn number if applicable
  const label = isTurn && turnNumber
    ? `Turn ${turnNumber}`
    : span.name;

  return {
    id: span.id,
    type: nodeType,
    label,
    status: getNodeStatus(span),
    durationMs: span.durationMs,
    turnNumber,
    _span: span,  // Preserve original span for popover
    children: span.children?.map((child, idx) => spanToHierarchyNode(child, idx)),
  };
}

// Transform spans to hierarchy nodes
function transformSpansToNodes(spans: Span[]): HierarchyNode[] {
  return spans.map((span, idx) => spanToHierarchyNode(span, idx));
}

// Format duration for display
function formatDuration(ms?: number): string {
  if (!ms) return '-';
  if (ms < 1000) return `${ms.toFixed(0)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

// Get icon for node type
function getNodeIcon(type: HierarchyNode['type']): string {
  switch (type) {
    case 'message': return '✉';
    case 'exec': return '⚡';
    case 'turn': return '↻';
    case 'tool_use': return '⚙';
    default: return '●';
  }
}

export const ExecHierarchy: React.FC<ExecHierarchyProps> = ({
  isExpanded,
  onToggleExpand,
  onNodeClick,
  selectedNodeId,
  isEmpty: propsIsEmpty,
  spans,
  loading,
}) => {
  // Default to graph view (ReactFlow)
  const [viewMode, setViewMode] = useState<ViewMode>('graph');

  // Popover state
  const [popoverNode, setPopoverNode] = useState<HierarchyNode | null>(null);
  const [popoverPos, setPopoverPos] = useState({ x: 0, y: 0 });
  const [attributesExpanded, setAttributesExpanded] = useState(false);
  const popoverRef = useRef<HTMLDivElement>(null);

  // Transform spans to hierarchy nodes (same data as TraceWaterfall)
  const nodes = useMemo(() => {
    if (!spans || spans.length === 0) return [];
    return transformSpansToNodes(spans);
  }, [spans]);

  const isEmpty = propsIsEmpty ?? nodes.length === 0;

  // Calculate viewport-aware position for popover
  const calculatePopoverPosition = useCallback((clickX: number, clickY: number) => {
    const viewportWidth = window.innerWidth;
    const viewportHeight = window.innerHeight;

    let x = clickX + 10;
    let y = clickY + 10;

    // Keep popover within right edge
    if (x + POPOVER_MAX_WIDTH + POPOVER_MARGIN > viewportWidth) {
      x = clickX - POPOVER_MAX_WIDTH - 10;
    }
    // Keep within left edge
    if (x < POPOVER_MARGIN) {
      x = POPOVER_MARGIN;
    }

    // Keep popover within bottom edge
    if (y + POPOVER_MAX_HEIGHT + POPOVER_MARGIN > viewportHeight) {
      y = viewportHeight - POPOVER_MAX_HEIGHT - POPOVER_MARGIN;
    }
    // Keep within top edge
    if (y < POPOVER_MARGIN) {
      y = POPOVER_MARGIN;
    }

    return { x, y };
  }, []);

  // Handle node click - show popover
  const handleNodeClick = useCallback(
    (node: HierarchyNode, event?: React.MouseEvent) => {
      onNodeClick?.(node);

      // Toggle popover - close if clicking same node
      if (popoverNode?.id === node.id) {
        setPopoverNode(null);
      } else {
        setPopoverNode(node);
        setAttributesExpanded(false); // Reset attributes collapsed state
        // Position with viewport awareness
        if (event) {
          const pos = calculatePopoverPosition(event.clientX, event.clientY);
          setPopoverPos(pos);
        } else {
          // Fallback to center of viewport
          setPopoverPos({ x: window.innerWidth / 2 - POPOVER_MAX_WIDTH / 2, y: 100 });
        }
      }
    },
    [onNodeClick, popoverNode, calculatePopoverPosition]
  );

  // Close popover when clicking outside
  const handleContainerClick = useCallback((e: React.MouseEvent) => {
    // Only close if clicking directly on container (not on children)
    if (e.target === e.currentTarget) {
      setPopoverNode(null);
    }
  }, []);

  return (
    <div className={`${styles.container} ${isExpanded ? styles.containerExpanded : ''}`}>
      {/* Header */}
      <div className={styles.header}>
        <div className={styles.headerTitle}>
          <span className={styles.headerIcon}>◎</span>
          Exec Hierarchy
          {!isEmpty && nodes.length > 0 && (
            <span className={styles.headerBadge}>{nodes.length}</span>
          )}
        </div>

        <div className={styles.headerControls}>
          {/* View Toggle */}
          <div className={styles.viewToggle}>
            <button
              className={`${styles.viewToggleBtn} ${viewMode === 'tree' ? styles.viewToggleBtnActive : ''}`}
              onClick={() => setViewMode('tree')}
              title="Tree View"
            >
              ≡
            </button>
            <button
              className={`${styles.viewToggleBtn} ${viewMode === 'graph' ? styles.viewToggleBtnActive : ''}`}
              onClick={() => setViewMode('graph')}
              title="Graph View"
            >
              ⬡
            </button>
          </div>

          {/* Expand Button */}
          <button
            className={styles.expandBtn}
            onClick={onToggleExpand}
            title={isExpanded ? 'Collapse' : 'Expand'}
          >
            {isExpanded ? '⤢' : '⤡'}
          </button>
        </div>
      </div>

      {/* Viewport */}
      <div
        className={`${styles.viewport} ${isExpanded ? styles.viewportExpanded : ''}`}
        onClick={handleContainerClick}
      >
        {viewMode === 'tree' ? (
          <ExecHierarchyTree
            nodes={nodes}
            selectedNodeId={selectedNodeId}
            onNodeClick={handleNodeClick}
            loading={loading}
            error={null}
          />
        ) : (
          <ExecHierarchyGraph
            nodes={nodes}
            selectedNodeId={selectedNodeId}
            onNodeClick={handleNodeClick}
            loading={loading}
            error={null}
            isExpanded={isExpanded}
          />
        )}
      </div>

      {/* Node Popover */}
      {popoverNode && (
        <div
          ref={popoverRef}
          className={styles.nodePopover}
          style={{ left: popoverPos.x, top: popoverPos.y }}
        >
          <div className={styles.popoverHeader}>
            <span className={styles.popoverIcon}>{getNodeIcon(popoverNode.type)}</span>
            <span className={styles.popoverTitle}>{popoverNode.label}</span>
            <button
              className={styles.popoverClose}
              onClick={() => setPopoverNode(null)}
              title="Close"
            >
              ×
            </button>
          </div>
          <div className={styles.popoverBody}>
            {/* Summary Grid */}
            <div className={styles.popoverSummary}>
              <div className={styles.popoverSummaryItem}>
                <span className={styles.popoverSummaryLabel}>Type</span>
                <span className={styles.popoverSummaryValue}>{popoverNode.type}</span>
              </div>
              <div className={styles.popoverSummaryItem}>
                <span className={styles.popoverSummaryLabel}>Status</span>
                <span className={`${styles.popoverSummaryValue} ${styles[`status${popoverNode.status.charAt(0).toUpperCase() + popoverNode.status.slice(1)}`]}`}>
                  {popoverNode.status}
                </span>
              </div>
              <div className={styles.popoverSummaryItem}>
                <span className={styles.popoverSummaryLabel}>Duration</span>
                <span className={styles.popoverSummaryValue}>{formatDuration(popoverNode.durationMs)}</span>
              </div>
              {popoverNode.turnNumber && (
                <div className={styles.popoverSummaryItem}>
                  <span className={styles.popoverSummaryLabel}>Turn</span>
                  <span className={styles.popoverSummaryValue}>{popoverNode.turnNumber}</span>
                </div>
              )}
            </div>

            {popoverNode._span && (
              <>
                {/* Span ID */}
                <div className={styles.popoverSection}>
                  <div className={styles.popoverSectionTitle}>Span ID</div>
                  <div className={styles.popoverIdValue}>{popoverNode._span.id}</div>
                </div>

                {/* Attributes - Collapsible */}
                {popoverNode._span.attributes && Object.keys(popoverNode._span.attributes).length > 0 && (
                  <div className={styles.popoverSection}>
                    <button
                      className={styles.popoverSectionToggle}
                      onClick={() => setAttributesExpanded(!attributesExpanded)}
                    >
                      <span className={styles.popoverToggleIcon}>
                        {attributesExpanded ? '▼' : '▶'}
                      </span>
                      <span className={styles.popoverSectionTitle}>
                        Attributes ({Object.keys(popoverNode._span.attributes).length})
                      </span>
                    </button>
                    {attributesExpanded && (
                      <div className={styles.popoverAttributesList}>
                        {Object.entries(popoverNode._span.attributes).map(([key, value]) => (
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
          </div>
        </div>
      )}
    </div>
  );
};

export default ExecHierarchy;
