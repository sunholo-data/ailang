/**
 * ExecHierarchyTree - Tree view for the 4-level exec hierarchy
 * Shows: Messages -> Execs -> Turns -> Tool Uses
 */
import React, { useState } from 'react';
import type { HierarchyNode, NodeStatus } from './types';
import styles from './ExecHierarchy.module.css';

// Format duration in human-readable format
function formatDuration(ms: number | undefined): string {
  if (!ms) return '';
  if (ms < 1000) return `${ms.toFixed(0)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

// Get status class for styling
function getStatusClass(status: NodeStatus): string {
  switch (status) {
    case 'completed':
      return styles.statusCompleted;
    case 'busy':
      return styles.statusBusy;
    case 'error':
      return styles.statusError;
    case 'pending':
      return styles.statusPending;
    default:
      return styles.statusIdle;
  }
}

// Get icon for node type
function getNodeIcon(type: HierarchyNode['type']): string {
  switch (type) {
    case 'message':
      return '✉'; // Envelope
    case 'exec':
      return '⚡'; // Lightning bolt
    case 'turn':
      return '↻'; // Clockwise arrow
    case 'tool_use':
      return '⚙'; // Gear
    case 'approval':
      return '👤'; // Person (human decision)
    default:
      return '●'; // Circle
  }
}

// Get provider-specific class for exec nodes
function getProviderClass(provider?: string): string {
  switch (provider?.toLowerCase()) {
    case 'claude':
      return styles.providerClaude;
    case 'gemini':
      return styles.providerGemini;
    case 'ollama':
      return styles.providerOllama;
    default:
      return '';
  }
}

interface TreeNodeProps {
  node: HierarchyNode;
  depth: number;
  selectedId?: string | null;
  onNodeClick?: (node: HierarchyNode) => void;
}

const TreeNode: React.FC<TreeNodeProps> = ({ node, depth, selectedId, onNodeClick }) => {
  const [expanded, setExpanded] = useState(depth < 2);
  const hasChildren = node.children && node.children.length > 0;
  const isSelected = selectedId === node.id;

  const handleClick = () => {
    if (hasChildren) {
      setExpanded(!expanded);
    }
    onNodeClick?.(node);
  };

  // Extract provider for exec nodes
  const provider = (node as { provider?: string }).provider;

  return (
    <div className={styles.treeNode}>
      <div
        className={`${styles.treeNodeHeader} ${isSelected ? styles.treeNodeSelected : ''}`}
        style={{ paddingLeft: `${depth * 20 + 8}px` }}
        onClick={handleClick}
      >
        {/* Expand/collapse indicator */}
        <span className={styles.expandIcon}>
          {hasChildren ? (expanded ? '▼' : '▶') : '•'}
        </span>

        {/* Node icon */}
        <span className={`${styles.nodeIcon} ${getProviderClass(provider)}`}>
          {getNodeIcon(node.type)}
        </span>

        {/* Label */}
        <span className={styles.nodeLabel}>{node.label}</span>

        {/* Status indicator */}
        <span className={`${styles.nodeStatus} ${getStatusClass(node.status)}`}>
          {node.status}
        </span>

        {/* Duration */}
        {node.durationMs && (
          <span className={styles.nodeDuration}>
            {formatDuration(node.durationMs)}
          </span>
        )}

        {/* Child count badge */}
        {hasChildren && (
          <span className={styles.childCount}>
            {node.children!.length}
          </span>
        )}
      </div>

      {/* Children */}
      {expanded && hasChildren && (
        <div className={styles.treeNodeChildren}>
          {node.children!.map((child) => (
            <TreeNode
              key={child.id}
              node={child}
              depth={depth + 1}
              selectedId={selectedId}
              onNodeClick={onNodeClick}
            />
          ))}
        </div>
      )}
    </div>
  );
};

export interface ExecHierarchyTreeProps {
  nodes: HierarchyNode[];
  selectedNodeId?: string | null;
  onNodeClick?: (node: HierarchyNode) => void;
  loading?: boolean;
  error?: string | null;
}

export const ExecHierarchyTree: React.FC<ExecHierarchyTreeProps> = ({
  nodes,
  selectedNodeId,
  onNodeClick,
  loading,
  error,
}) => {
  if (loading) {
    return (
      <div className={styles.treeLoading}>
        <div className={styles.loadingSpinner} />
        <span>Loading hierarchy...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className={styles.treeError}>
        <span className={styles.errorIcon}>⚠</span>
        <span>{error}</span>
      </div>
    );
  }

  if (!nodes || nodes.length === 0) {
    return (
      <div className={styles.treeEmpty}>
        <span className={styles.emptyIcon}>⬡</span>
        <div className={styles.emptyTitle}>No Executions Yet</div>
        <div className={styles.emptyText}>
          Execution nodes will appear here as ailang commands run.
        </div>
      </div>
    );
  }

  return (
    <div className={styles.treeContainer}>
      {nodes.map((node) => (
        <TreeNode
          key={node.id}
          node={node}
          depth={0}
          selectedId={selectedNodeId}
          onNodeClick={onNodeClick}
        />
      ))}
    </div>
  );
};

export default ExecHierarchyTree;
