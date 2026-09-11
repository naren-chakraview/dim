import React, { useCallback, useState } from 'react';
import ReactFlow, {
  Node,
  Edge,
  Controls,
  Background,
  useNodesState,
  useEdgesState,
} from 'reactflow';
import 'reactflow/dist/style.css';
import { RouteConfig, DAGNode as DAGNodeType } from '../types/route';
import { useDagLayout } from '../hooks/useDagLayout';
import '../styles/Canvas.css';

interface CanvasProps {
  route?: RouteConfig;
  onStepSelect?: (stepName: string) => void;
}

/**
 * Canvas component renders route's step DAG as interactive node-graph.
 * Features:
 * - Click step to select for editing
 * - Drag to pan, scroll to zoom
 * - Highlight selected step
 * - Show step type icon
 */
export function Canvas({ route, onStepSelect }: CanvasProps) {
  const { nodes: layoutNodes, edges: layoutEdges } = useDagLayout(route);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);

  // Convert layout nodes to react-flow nodes with click handlers
  const nodes = layoutNodes.map((node: DAGNodeType): Node => ({
    id: node.id,
    data: {
      label: node.label,
      ...node.data,
    },
    position: node.position,
    style: {
      background: getNodeColor(node.type, selectedNodeId === node.id),
      border: selectedNodeId === node.id ? '3px solid #0066cc' : '2px solid #ccc',
      borderRadius: '8px',
      padding: '10px',
      fontSize: '12px',
      fontWeight: selectedNodeId === node.id ? 'bold' : 'normal',
      cursor: 'pointer',
      transition: 'all 0.2s ease',
      minWidth: '120px',
      textAlign: 'center',
    },
    className: `node node-${node.type}`,
  }));

  // Convert layout edges to react-flow edges
  const edges = layoutEdges.map((edge): Edge => ({
    id: edge.id,
    source: edge.source,
    target: edge.target,
    animated: true,
    style: {
      stroke: '#999',
      strokeWidth: 2,
    },
  }));

  const [nodesState, setNodes, onNodesChange] = useNodesState(nodes);
  const [edgesState, setEdges, onEdgesChange] = useEdgesState(edges);

  // Handle node click for selection
  const handleNodeClick = useCallback(
    (event: React.MouseEvent, node: Node) => {
      const stepName = node.data.label.split(' ').pop();
      if (stepName) {
        setSelectedNodeId(node.id);
        onStepSelect?.(stepName);
      }
    },
    [onStepSelect]
  );

  // Update nodes when layout changes
  React.useEffect(() => {
    setNodes(nodes);
  }, [layoutNodes, selectedNodeId, setNodes]);

  React.useEffect(() => {
    setEdges(edges);
  }, [layoutEdges, setEdges]);

  return (
    <div className="canvas-container">
      {!route ? (
        <div className="canvas-empty">
          <p>Open a route file to view the step DAG</p>
        </div>
      ) : (
        <ReactFlow
          nodes={nodesState}
          edges={edgesState}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onNodeClick={handleNodeClick}
          fitView
        >
          <Background color="#aaa" gap={16} />
          <Controls />
        </ReactFlow>
      )}
    </div>
  );
}

function getNodeColor(type: string, isSelected: boolean): string {
  if (isSelected) {
    return '#e6f2ff';
  }

  switch (type) {
    case 'source':
      return '#e6f9e6'; // Light green
    case 'step':
      return '#fff9e6'; // Light yellow
    case 'sink':
      return '#e6f0ff'; // Light blue
    case 'error_sink':
      return '#ffe6e6'; // Light red
    default:
      return '#f5f5f5';
  }
}
