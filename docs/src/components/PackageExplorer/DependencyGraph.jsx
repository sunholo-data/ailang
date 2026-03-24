import React, { useState, useEffect, useRef, useCallback } from 'react';
import { usePackageIndex } from '@site/src/hooks/useRegistryData';
import styles from './styles.module.css';

const EFFECT_COLORS = {
  pure: '#38a169',    // green
  io: '#2b6cb0',      // blue
  net: '#dd6b20',     // orange
  mixed: '#e73c17',   // red
};

function getNodeColor(effects) {
  if (!effects || effects.length === 0) return EFFECT_COLORS.pure;
  if (effects.length === 1 && effects[0] === 'IO') return EFFECT_COLORS.io;
  if (effects.some(e => e === 'Net' || e === 'FS')) return EFFECT_COLORS.net;
  return EFFECT_COLORS.mixed;
}

function getEffectLabel(effects) {
  if (!effects || effects.length === 0) return 'Pure';
  return effects.join(', ');
}

/**
 * DependencyGraph — interactive SVG-based package dependency visualization.
 * Uses a simple force-directed layout computed in JS (no d3-force dependency).
 */
export default function DependencyGraph() {
  const { data: indexData, loading, error, stale } = usePackageIndex();
  const svgRef = useRef(null);
  const [tooltip, setTooltip] = useState(null);
  const [nodes, setNodes] = useState([]);
  const [edges, setEdges] = useState([]);

  const packages = indexData?.packages || [];

  // Compute layout
  useEffect(() => {
    if (packages.length === 0) return;

    // Build dependency graph
    const depCounts = {};
    packages.forEach((p) => {
      (p.dependencies || []).forEach((dep) => {
        depCounts[dep] = (depCounts[dep] || 0) + 1;
      });
    });

    // Simple layered layout: packages with no deps at top, most deps at bottom
    const depDepth = {};
    function getDepth(name, visited = new Set()) {
      if (depDepth[name] !== undefined) return depDepth[name];
      if (visited.has(name)) return 0;
      visited.add(name);
      const pkg = packages.find((p) => p.name === name);
      if (!pkg || !pkg.dependencies || pkg.dependencies.length === 0) {
        depDepth[name] = 0;
        return 0;
      }
      const maxChildDepth = Math.max(...pkg.dependencies.map((d) => getDepth(d, visited)));
      depDepth[name] = maxChildDepth + 1;
      return depDepth[name];
    }
    packages.forEach((p) => getDepth(p.name));

    // Group by depth level
    const levels = {};
    packages.forEach((p) => {
      const depth = depDepth[p.name] || 0;
      if (!levels[depth]) levels[depth] = [];
      levels[depth].push(p);
    });

    const svgWidth = 800;
    const svgHeight = 500;
    const levelCount = Object.keys(levels).length;
    const ySpacing = svgHeight / (levelCount + 1);

    const computedNodes = [];
    const nodePositions = {};

    Object.entries(levels)
      .sort(([a], [b]) => Number(a) - Number(b))
      .forEach(([depth, pkgs], levelIndex) => {
        const xSpacing = svgWidth / (pkgs.length + 1);
        pkgs.forEach((p, i) => {
          const x = xSpacing * (i + 1);
          const y = ySpacing * (levelIndex + 1);
          const dependentCount = depCounts[p.name] || 0;
          const radius = Math.max(18, Math.min(35, 18 + dependentCount * 5));

          const node = {
            id: p.name,
            shortName: p.name.split('/')[1],
            x, y, radius,
            color: getNodeColor(p.effects),
            effects: p.effects,
            aiSummary: p.ai_summary,
            latest: p.latest,
            stability: p.stability,
            dependentCount,
          };
          computedNodes.push(node);
          nodePositions[p.name] = { x, y };
        });
      });

    // Build edges
    const computedEdges = [];
    packages.forEach((p) => {
      (p.dependencies || []).forEach((dep) => {
        if (nodePositions[p.name] && nodePositions[dep]) {
          computedEdges.push({
            from: p.name,
            to: dep,
            x1: nodePositions[p.name].x,
            y1: nodePositions[p.name].y,
            x2: nodePositions[dep].x,
            y2: nodePositions[dep].y,
          });
        }
      });
    });

    setNodes(computedNodes);
    setEdges(computedEdges);
  }, [packages]);

  const handleNodeHover = useCallback((node, event) => {
    const rect = svgRef.current.getBoundingClientRect();
    setTooltip({
      x: event.clientX - rect.left + 15,
      y: event.clientY - rect.top - 10,
      node,
    });
  }, []);

  if (loading) {
    return (
      <div className={styles.loading}>
        <div className={styles.spinner} />
        <p>Loading dependency graph...</p>
      </div>
    );
  }

  if (packages.length === 0) {
    return (
      <div className={styles.emptyState}>
        <p>No packages in registry.</p>
      </div>
    );
  }

  return (
    <div>
      {stale && (
        <div className={styles.staleBanner}>
          Showing cached data — graph may not reflect latest publishes.
        </div>
      )}

      <div className={styles.graphContainer} ref={svgRef}>
        <svg className={styles.graphSvg} viewBox="0 0 800 500">
          <defs>
            <marker
              id="arrowhead"
              viewBox="0 0 10 7"
              refX="10"
              refY="3.5"
              markerWidth="8"
              markerHeight="6"
              orient="auto-start-reverse"
            >
              <polygon
                points="0 0, 10 3.5, 0 7"
                fill="var(--ifm-color-emphasis-400)"
              />
            </marker>
          </defs>

          {/* Edges */}
          {edges.map((edge, i) => {
            const dx = edge.x2 - edge.x1;
            const dy = edge.y2 - edge.y1;
            const len = Math.sqrt(dx * dx + dy * dy);
            const targetNode = nodes.find((n) => n.id === edge.to);
            const targetRadius = targetNode ? targetNode.radius : 18;
            // Shorten line to stop at node edge
            const ratio = (len - targetRadius - 4) / len;

            return (
              <line
                key={i}
                className={styles.graphEdge}
                x1={edge.x1}
                y1={edge.y1}
                x2={edge.x1 + dx * ratio}
                y2={edge.y1 + dy * ratio}
              />
            );
          })}

          {/* Nodes */}
          {nodes.map((node) => (
            <g
              key={node.id}
              className={styles.graphNode}
              onMouseEnter={(e) => handleNodeHover(node, e)}
              onMouseLeave={() => setTooltip(null)}
              onClick={() => {
                window.location.href = `/docs/packages/${node.id}`;
              }}
            >
              <circle
                cx={node.x}
                cy={node.y}
                r={node.radius}
                fill={node.color}
                opacity={0.85}
                stroke={node.color}
                strokeWidth={2}
                strokeOpacity={0.3}
              />
              <text
                className={styles.graphNodeLabel}
                x={node.x}
                y={node.y}
              >
                {node.shortName}
              </text>
            </g>
          ))}
        </svg>

        {/* Tooltip */}
        {tooltip && (
          <div
            className={styles.graphTooltip}
            style={{ left: tooltip.x, top: tooltip.y }}
          >
            <h4>{tooltip.node.id}</h4>
            <p>{tooltip.node.aiSummary}</p>
            <p>
              v{tooltip.node.latest} · {getEffectLabel(tooltip.node.effects)} · {tooltip.node.stability || 'experimental'}
              {tooltip.node.dependentCount > 0 && ` · ${tooltip.node.dependentCount} dependents`}
            </p>
          </div>
        )}
      </div>

      {/* Legend */}
      <div className={styles.graphLegend}>
        <div className={styles.legendItem}>
          <span className={styles.legendDot} style={{ background: EFFECT_COLORS.pure }} />
          Pure
        </div>
        <div className={styles.legendItem}>
          <span className={styles.legendDot} style={{ background: EFFECT_COLORS.io }} />
          IO only
        </div>
        <div className={styles.legendItem}>
          <span className={styles.legendDot} style={{ background: EFFECT_COLORS.net }} />
          Net / FS
        </div>
        <div className={styles.legendItem}>
          <span className={styles.legendDot} style={{ background: EFFECT_COLORS.mixed }} />
          Mixed
        </div>
        <span style={{ marginLeft: '1rem', fontStyle: 'italic' }}>
          Node size = dependent count · Click to navigate
        </span>
      </div>
    </div>
  );
}
