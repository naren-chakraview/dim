import { useMemo } from 'react';
import { RouteConfig, Route, DAGNode, DAGEdge, DAGLayout, STEP_ICONS, ADAPTER_ICONS } from '../types/route';

/**
 * Computes DAG layout for a route's step pipeline.
 * Uses simple hierarchical layout:
 * - Level 0: Sources
 * - Level 1-N: Steps (one level per step in sequence)
 * - Level N+1: Sinks + Error sinks
 */
export function useDagLayout(route?: RouteConfig): DAGLayout {
  return useMemo(() => {
    if (!route) {
      return { nodes: [], edges: [] };
    }

    const nodes: DAGNode[] = [];
    const edges: DAGEdge[] = [];
    const levelMap = new Map<string, number>(); // Track which level each node is on
    const nodeMap = new Map<string, DAGNode>(); // Track nodes by ID for edge creation

    const NODE_WIDTH = 140;
    const NODE_HEIGHT = 60;
    const LEVEL_WIDTH = 200;
    const LEVEL_HEIGHT = 120;

    let currentLevel = 0;

    // Add source nodes
    Object.entries(route.sources || {}).forEach(([name, config]) => {
      const nodeId = `source-${name}`;
      const icon = ADAPTER_ICONS[config.type] || '📥';
      const node: DAGNode = {
        id: nodeId,
        label: `${icon} ${name}`,
        type: 'source',
        position: { x: currentLevel * LEVEL_WIDTH, y: 0 },
        data: {
          label: name,
          config,
          icon,
        },
      };
      nodes.push(node);
      nodeMap.set(nodeId, node);
      levelMap.set(nodeId, currentLevel);
    });

    currentLevel += 1;

    // Add step nodes for each route
    Object.entries(route.routes || {}).forEach(([routeName, routeConfig]: [string, Route]) => {
      let prevNodeId = Array.from(nodeMap.values())
        .filter(n => n.type === 'source')
        .map(n => n.id)[0];

      (routeConfig.steps || []).forEach((step, stepIndex) => {
        const stepName = Object.keys(step)[0];
        const stepConfig = step[stepName];
        const icon = STEP_ICONS[stepName] || '⚙️';
        const nodeId = `step-${routeName}-${stepIndex}`;

        const x = currentLevel * LEVEL_WIDTH + (stepIndex * 20);
        const y = 100 + (stepIndex * 30);

        const node: DAGNode = {
          id: nodeId,
          label: `${icon} ${stepName}`,
          type: 'step',
          position: { x, y },
          data: {
            label: `${stepName} (step ${stepIndex})`,
            config: stepConfig,
            icon,
          },
        };

        nodes.push(node);
        nodeMap.set(nodeId, node);
        levelMap.set(nodeId, currentLevel + stepIndex);

        // Add edge from previous node
        if (prevNodeId) {
          edges.push({
            id: `edge-${prevNodeId}-${nodeId}`,
            source: prevNodeId,
            target: nodeId,
          });
        }

        prevNodeId = nodeId;
      });

      // Add edge from last step to sink
      if (prevNodeId && routeConfig.to) {
        edges.push({
          id: `edge-${prevNodeId}-sink-${routeConfig.to}`,
          source: prevNodeId,
          target: `sink-${routeConfig.to}`,
        });
      }
    });

    currentLevel += 3; // Space for steps

    // Add sink nodes
    Object.entries(route.sinks || {}).forEach(([name, config]) => {
      const nodeId = `sink-${name}`;
      const icon = ADAPTER_ICONS[config.type] || '📤';
      const node: DAGNode = {
        id: nodeId,
        label: `${icon} ${name}`,
        type: 'sink',
        position: { x: currentLevel * LEVEL_WIDTH, y: 0 },
        data: {
          label: name,
          config,
          icon,
        },
      };
      nodes.push(node);
      nodeMap.set(nodeId, node);
      levelMap.set(nodeId, currentLevel);
    });

    // Add error sink nodes if error path exists
    if (route.error_path?.sinks) {
      Object.entries(route.error_path.sinks).forEach(([name, config]) => {
        const nodeId = `error-sink-${name}`;
        const icon = '⚠️';
        const node: DAGNode = {
          id: nodeId,
          label: `${icon} ${name}`,
          type: 'error_sink',
          position: { x: currentLevel * LEVEL_WIDTH, y: 200 },
          data: {
            label: `${name} (error)`,
            config,
            icon,
          },
        };
        nodes.push(node);
        nodeMap.set(nodeId, node);
        levelMap.set(nodeId, currentLevel);
      });
    }

    return { nodes, edges };
  }, [route]);
}

/**
 * Extract step name and type from a route's step pipeline.
 * Returns array of { name, type, config }
 */
export function getRouteSteps(route?: Route): Array<{ name: string; type: string; config: any }> {
  if (!route?.steps) return [];

  return route.steps.map(step => {
    const name = Object.keys(step)[0];
    return {
      name,
      type: name,
      config: step[name],
    };
  });
}
