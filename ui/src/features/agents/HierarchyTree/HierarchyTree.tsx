import React, { useState, useEffect } from 'react';
import { HierarchyNode, HierarchyResponse, Selection, Badge } from '../../../types';
import { getAgentLabel, isHumanUser } from '../../../utils/displayNames';
import './HierarchyTree.css';

interface HierarchyTreeProps {
  selection: Selection;
  onSelect: (selection: Selection) => void;
  onRefresh?: () => void;
}

const API_BASE = '';

export const HierarchyTree: React.FC<HierarchyTreeProps> = ({ selection, onSelect, onRefresh }) => {
  const [hierarchy, setHierarchy] = useState<HierarchyResponse | null>(null);
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set(['all']));
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchHierarchy = async () => {
    try {
      const response = await fetch(`${API_BASE}/api/hierarchy`);
      if (!response.ok) throw new Error('Failed to fetch hierarchy');
      const data = await response.json();
      setHierarchy(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchHierarchy();
    // Refresh every 5 seconds
    const interval = setInterval(fetchHierarchy, 5000);
    return () => clearInterval(interval);
  }, []);

  const toggleNode = (nodeId: string) => {
    setExpandedNodes(prev => {
      const next = new Set(prev);
      if (next.has(nodeId)) {
        next.delete(nodeId);
      } else {
        next.add(nodeId);
      }
      return next;
    });
  };

  const handleNodeClick = (node: HierarchyNode) => {
    if (node.type === 'root') {
      onSelect({ type: 'overview' });
    } else if (node.type === 'agent') {
      onSelect({ type: 'agent', agentId: node.id });
    } else if (node.type === 'thread') {
      // Find the parent agent
      const agent = hierarchy?.root.children?.find(
        a => a.children?.some(t => t.id === node.id)
      );
      onSelect({ type: 'thread', agentId: agent?.id, threadId: node.id });
    }
  };

  const isSelected = (node: HierarchyNode): boolean => {
    if (node.type === 'root' && selection.type === 'overview') return true;
    if (node.type === 'agent' && selection.type === 'agent' && selection.agentId === node.id) return true;
    if (node.type === 'thread' && selection.threadId === node.id) return true;
    return false;
  };

  const renderBadges = (badges?: Badge[]) => {
    if (!badges || badges.length === 0) return null;
    return (
      <span className="badges">
        {badges.map((badge, i) => (
          <span key={i} className={`badge badge-${badge.type}`} title={`${badge.count} ${badge.type}`}>
            {badge.type === 'pending' && '\u23F3'}
            {badge.type === 'unread' && '\uD83D\uDCEC'}
            {badge.type === 'running' && '\u25B6\uFE0F'}
            {badge.count}
          </span>
        ))}
      </span>
    );
  };

  const renderStatusIndicator = (status?: string) => {
    if (!status) return null;
    const colors: Record<string, string> = {
      active: '#22c55e',
      pending: '#f59e0b',
      idle: '#6b7280',
    };
    return (
      <span
        className="status-indicator"
        style={{ backgroundColor: colors[status] || colors.idle }}
        title={status}
      />
    );
  };

  const renderNode = (node: HierarchyNode, depth: number = 0) => {
    const isExpanded = expandedNodes.has(node.id);
    const hasChildren = node.children && node.children.length > 0;
    const selected = isSelected(node);

    return (
      <div key={node.id} className="tree-node">
        <div
          className={`tree-node-content ${selected ? 'selected' : ''} ${node.type}`}
          style={{ paddingLeft: `${depth * 16 + 8}px` }}
          onClick={() => handleNodeClick(node)}
        >
          {hasChildren && (
            <span
              className={`expand-icon ${isExpanded ? 'expanded' : ''}`}
              onClick={(e) => {
                e.stopPropagation();
                toggleNode(node.id);
              }}
            >
              {isExpanded ? '\u25BC' : '\u25B6'}
            </span>
          )}
          {!hasChildren && <span className="expand-icon-placeholder" />}

          {node.type === 'agent' && renderStatusIndicator(node.status)}

          <span className="node-label">
            {node.type === 'agent' ? getAgentLabel(node.id) : node.label}
          </span>

          {renderBadges(node.badges)}
        </div>

        {hasChildren && isExpanded && (
          <div className="tree-children">
            {node.children!.map(child => renderNode(child, depth + 1))}
          </div>
        )}
      </div>
    );
  };

  if (loading && !hierarchy) {
    return <div className="hierarchy-tree loading">Loading...</div>;
  }

  if (error) {
    return (
      <div className="hierarchy-tree error">
        <p>Error: {error}</p>
        <button onClick={fetchHierarchy}>Retry</button>
      </div>
    );
  }

  return (
    <div className="hierarchy-tree">
      <div className="tree-header">
        <h3>Agents</h3>
        <button className="refresh-btn" onClick={() => { fetchHierarchy(); onRefresh?.(); }} title="Refresh">
          ↻
        </button>
      </div>

      <div className="tree-content">
        {hierarchy && renderNode(hierarchy.root)}
      </div>

      {hierarchy && (
        <div className="tree-footer">
          <div className="aggregate-stats">
            <span title="Total agents">{hierarchy.aggregate.total_agents} agents</span>
            <span title="Active">{hierarchy.aggregate.active_agents} active</span>
            {hierarchy.aggregate.pending_approvals > 0 && (
              <span className="pending" title="Pending approvals">
                {hierarchy.aggregate.pending_approvals} pending
              </span>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default HierarchyTree;
