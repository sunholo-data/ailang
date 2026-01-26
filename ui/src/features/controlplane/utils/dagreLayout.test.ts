import { describe, it, expect } from 'vitest';
import {
  applyDagreLayout,
  calculateGraphBounds,
  DEFAULT_NODE_WIDTH,
  DEFAULT_NODE_HEIGHT,
} from './dagreLayout';
import type { Node, Edge } from '@xyflow/react';

describe('applyDagreLayout', () => {
  it('returns empty array for empty input', () => {
    const result = applyDagreLayout([], []);
    expect(result).toEqual([]);
  });

  it('positions a single node', () => {
    const nodes: Node[] = [
      { id: '1', position: { x: 0, y: 0 }, data: {} },
    ];
    const result = applyDagreLayout(nodes, []);

    expect(result).toHaveLength(1);
    expect(result[0].position).toBeDefined();
    expect(typeof result[0].position.x).toBe('number');
    expect(typeof result[0].position.y).toBe('number');
  });

  it('positions connected nodes hierarchically', () => {
    const nodes: Node[] = [
      { id: 'parent', position: { x: 0, y: 0 }, data: {} },
      { id: 'child', position: { x: 0, y: 0 }, data: {} },
    ];
    const edges: Edge[] = [
      { id: 'e1', source: 'parent', target: 'child' },
    ];

    const result = applyDagreLayout(nodes, edges, { direction: 'TB' });

    expect(result).toHaveLength(2);

    const parent = result.find(n => n.id === 'parent')!;
    const child = result.find(n => n.id === 'child')!;

    // In TB layout, parent should be above child (smaller y)
    expect(parent.position.y).toBeLessThan(child.position.y);
  });

  it('respects direction option', () => {
    const nodes: Node[] = [
      { id: 'a', position: { x: 0, y: 0 }, data: {} },
      { id: 'b', position: { x: 0, y: 0 }, data: {} },
    ];
    const edges: Edge[] = [
      { id: 'e1', source: 'a', target: 'b' },
    ];

    const tbResult = applyDagreLayout(nodes, edges, { direction: 'TB' });
    const lrResult = applyDagreLayout(nodes, edges, { direction: 'LR' });

    const tbA = tbResult.find(n => n.id === 'a')!;
    const tbB = tbResult.find(n => n.id === 'b')!;
    const lrA = lrResult.find(n => n.id === 'a')!;
    const lrB = lrResult.find(n => n.id === 'b')!;

    // TB: a is above b (smaller y)
    expect(tbA.position.y).toBeLessThan(tbB.position.y);

    // LR: a is left of b (smaller x)
    expect(lrA.position.x).toBeLessThan(lrB.position.x);
  });

  it('respects custom node dimensions in bounds calculation', () => {
    const nodes: Node[] = [
      { id: '1', position: { x: 0, y: 0 }, data: {} },
    ];

    // Layout with small dimensions
    const smallResult = applyDagreLayout(nodes, [], { nodeWidth: 50, nodeHeight: 25 });
    const smallBounds = calculateGraphBounds(smallResult, 50, 25);

    // Layout with large dimensions
    const largeResult = applyDagreLayout(nodes, [], { nodeWidth: 400, nodeHeight: 200 });
    const largeBounds = calculateGraphBounds(largeResult, 400, 200);

    // Bounds should reflect different node dimensions
    expect(smallBounds.width).toBeLessThan(largeBounds.width);
    expect(smallBounds.height).toBeLessThan(largeBounds.height);
  });

  it('handles complex graph with multiple edges', () => {
    const nodes: Node[] = [
      { id: 'root', position: { x: 0, y: 0 }, data: {} },
      { id: 'left', position: { x: 0, y: 0 }, data: {} },
      { id: 'right', position: { x: 0, y: 0 }, data: {} },
      { id: 'leaf', position: { x: 0, y: 0 }, data: {} },
    ];
    const edges: Edge[] = [
      { id: 'e1', source: 'root', target: 'left' },
      { id: 'e2', source: 'root', target: 'right' },
      { id: 'e3', source: 'left', target: 'leaf' },
    ];

    const result = applyDagreLayout(nodes, edges);

    expect(result).toHaveLength(4);

    // All nodes should have valid positions
    result.forEach(node => {
      expect(node.position).toBeDefined();
      expect(Number.isFinite(node.position.x)).toBe(true);
      expect(Number.isFinite(node.position.y)).toBe(true);
    });
  });
});

describe('calculateGraphBounds', () => {
  it('returns zeros for empty array', () => {
    const bounds = calculateGraphBounds([]);
    expect(bounds).toEqual({
      minX: 0,
      minY: 0,
      maxX: 0,
      maxY: 0,
      width: 0,
      height: 0,
    });
  });

  it('calculates bounds for single node', () => {
    const nodes: Node[] = [
      { id: '1', position: { x: 100, y: 50 }, data: {} },
    ];

    const bounds = calculateGraphBounds(nodes);

    expect(bounds.minX).toBe(100);
    expect(bounds.minY).toBe(50);
    expect(bounds.maxX).toBe(100 + DEFAULT_NODE_WIDTH);
    expect(bounds.maxY).toBe(50 + DEFAULT_NODE_HEIGHT);
    expect(bounds.width).toBe(DEFAULT_NODE_WIDTH);
    expect(bounds.height).toBe(DEFAULT_NODE_HEIGHT);
  });

  it('calculates bounds for multiple nodes', () => {
    const nodes: Node[] = [
      { id: '1', position: { x: 0, y: 0 }, data: {} },
      { id: '2', position: { x: 300, y: 200 }, data: {} },
    ];

    const bounds = calculateGraphBounds(nodes);

    expect(bounds.minX).toBe(0);
    expect(bounds.minY).toBe(0);
    expect(bounds.maxX).toBe(300 + DEFAULT_NODE_WIDTH);
    expect(bounds.maxY).toBe(200 + DEFAULT_NODE_HEIGHT);
    expect(bounds.width).toBe(300 + DEFAULT_NODE_WIDTH);
    expect(bounds.height).toBe(200 + DEFAULT_NODE_HEIGHT);
  });

  it('respects custom node dimensions', () => {
    const nodes: Node[] = [
      { id: '1', position: { x: 100, y: 100 }, data: {} },
    ];

    const bounds = calculateGraphBounds(nodes, 100, 50);

    expect(bounds.width).toBe(100);
    expect(bounds.height).toBe(50);
  });
});

describe('DEFAULT constants', () => {
  it('exports default dimensions', () => {
    expect(DEFAULT_NODE_WIDTH).toBe(200);
    expect(DEFAULT_NODE_HEIGHT).toBe(80);
  });
});
