/**
 * Shared dagre layout utilities for ReactFlow graphs
 * Consolidated from ExecHierarchyGraph, TaskHierarchyGraph, and buildGraphFromSpans
 */

import dagre from 'dagre';
import type { Node, Edge } from '@xyflow/react';

// Default node dimensions (can be overridden)
export const DEFAULT_NODE_WIDTH = 200;
export const DEFAULT_NODE_HEIGHT = 80;

export interface DagreLayoutOptions {
  /** Layout direction: 'TB' = top-to-bottom, 'LR' = left-to-right */
  direction?: 'TB' | 'LR';
  /** Horizontal separation between nodes (default: 60) */
  nodeSep?: number;
  /** Vertical separation between ranks (default: 100) */
  rankSep?: number;
  /** Graph margin X (default: 20) */
  marginX?: number;
  /** Graph margin Y (default: 20) */
  marginY?: number;
  /** Node width for layout calculation (default: 200) */
  nodeWidth?: number;
  /** Node height for layout calculation (default: 80) */
  nodeHeight?: number;
}

/**
 * Apply dagre hierarchical layout to ReactFlow nodes and edges
 * Returns new nodes array with calculated positions
 *
 * @param nodes - ReactFlow nodes to layout
 * @param edges - ReactFlow edges defining relationships
 * @param options - Layout configuration options
 * @returns Nodes with updated position properties
 */
export function applyDagreLayout<T = unknown>(
  nodes: Node<T>[],
  edges: Edge[],
  options: DagreLayoutOptions = {}
): Node<T>[] {
  const {
    direction = 'TB',
    nodeSep = 60,
    rankSep = 100,
    marginX = 20,
    marginY = 20,
    nodeWidth = DEFAULT_NODE_WIDTH,
    nodeHeight = DEFAULT_NODE_HEIGHT,
  } = options;

  // Create dagre graph
  const g = new dagre.graphlib.Graph();
  g.setGraph({
    rankdir: direction,
    nodesep: nodeSep,
    ranksep: rankSep,
    marginx: marginX,
    marginy: marginY,
  });
  g.setDefaultEdgeLabel(() => ({}));

  // Add nodes with dimensions
  nodes.forEach(node => {
    // Allow individual nodes to specify their dimensions
    const width = (node.data as { width?: number })?.width ?? nodeWidth;
    const height = (node.data as { height?: number })?.height ?? nodeHeight;
    g.setNode(node.id, { width, height });
  });

  // Add edges
  edges.forEach(edge => {
    g.setEdge(edge.source, edge.target);
  });

  // Run dagre layout algorithm
  dagre.layout(g);

  // Apply calculated positions to nodes, centering based on dimensions
  return nodes.map(node => {
    const nodeWithPosition = g.node(node.id);
    const width = (node.data as { width?: number })?.width ?? nodeWidth;
    const height = (node.data as { height?: number })?.height ?? nodeHeight;

    return {
      ...node,
      position: {
        x: nodeWithPosition.x - width / 2,
        y: nodeWithPosition.y - height / 2,
      },
    };
  });
}

/**
 * Calculate graph bounds from laid out nodes
 * Useful for fitting view to content
 */
export function calculateGraphBounds<T = unknown>(
  nodes: Node<T>[],
  nodeWidth = DEFAULT_NODE_WIDTH,
  nodeHeight = DEFAULT_NODE_HEIGHT
): { minX: number; minY: number; maxX: number; maxY: number; width: number; height: number } {
  if (nodes.length === 0) {
    return { minX: 0, minY: 0, maxX: 0, maxY: 0, width: 0, height: 0 };
  }

  let minX = Infinity;
  let minY = Infinity;
  let maxX = -Infinity;
  let maxY = -Infinity;

  nodes.forEach(node => {
    const width = (node.data as { width?: number })?.width ?? nodeWidth;
    const height = (node.data as { height?: number })?.height ?? nodeHeight;
    const x = node.position?.x ?? 0;
    const y = node.position?.y ?? 0;

    minX = Math.min(minX, x);
    minY = Math.min(minY, y);
    maxX = Math.max(maxX, x + width);
    maxY = Math.max(maxY, y + height);
  });

  return {
    minX,
    minY,
    maxX,
    maxY,
    width: maxX - minX,
    height: maxY - minY,
  };
}
