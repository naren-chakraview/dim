import { describe, it, expect } from 'vitest';
import { renderHook } from '@testing-library/react';
import { useDagLayout, getRouteSteps } from '../useDagLayout';
import { RouteConfig, Route } from '../../types/route';

describe('useDagLayout Hook', () => {
  const mockRoute: RouteConfig = {
    version: 1,
    domain: 'payments',
    sources: {
      input: {
        type: 'http',
      },
    },
    routes: {
      'order-payment': {
        from: 'input',
        steps: [
          { translate: { expr: '$payload.id' } },
          { log: { level: 'info' } },
        ],
        to: 'output',
      },
    },
    sinks: {
      output: {
        type: 'file',
        path: './output.jsonl',
      },
    },
  };

  it('returns empty layout for undefined route', () => {
    const { result } = renderHook(() => useDagLayout(undefined));
    expect(result.current.nodes).toHaveLength(0);
    expect(result.current.edges).toHaveLength(0);
  });

  it('creates source nodes', () => {
    const { result } = renderHook(() => useDagLayout(mockRoute));
    const sourceNodes = result.current.nodes.filter(n => n.type === 'source');
    expect(sourceNodes).toHaveLength(1);
    expect(sourceNodes[0].label).toContain('input');
  });

  it('creates sink nodes', () => {
    const { result } = renderHook(() => useDagLayout(mockRoute));
    const sinkNodes = result.current.nodes.filter(n => n.type === 'sink');
    expect(sinkNodes).toHaveLength(1);
    expect(sinkNodes[0].label).toContain('output');
  });

  it('creates step nodes for each step in pipeline', () => {
    const { result } = renderHook(() => useDagLayout(mockRoute));
    const stepNodes = result.current.nodes.filter(n => n.type === 'step');
    expect(stepNodes).toHaveLength(2);
  });

  it('creates edges connecting nodes in pipeline', () => {
    const { result } = renderHook(() => useDagLayout(mockRoute));
    expect(result.current.edges.length).toBeGreaterThan(0);

    // Each edge should have source and target
    result.current.edges.forEach(edge => {
      expect(edge.source).toBeTruthy();
      expect(edge.target).toBeTruthy();
    });
  });

  it('positions nodes hierarchically', () => {
    const { result } = renderHook(() => useDagLayout(mockRoute));
    const nodes = result.current.nodes;

    // Source should be on the left
    const sourceNode = nodes.find(n => n.type === 'source');
    expect(sourceNode?.position.x).toBeLessThan(100);

    // Sink should be on the right
    const sinkNode = nodes.find(n => n.type === 'sink');
    expect(sinkNode?.position.x).toBeGreaterThan(sourceNode?.position.x || 0);
  });

  it('handles error path sinks', () => {
    const routeWithError: RouteConfig = {
      ...mockRoute,
      error_path: {
        sinks: {
          dlq: {
            type: 'file',
            path: './dlq.jsonl',
          },
        },
      },
    };

    const { result } = renderHook(() => useDagLayout(routeWithError));
    const errorSinks = result.current.nodes.filter(n => n.type === 'error_sink');
    expect(errorSinks).toHaveLength(1);
  });

  it('assigns proper icons to node types', () => {
    const { result } = renderHook(() => useDagLayout(mockRoute));

    // All nodes should have icons in their labels
    result.current.nodes.forEach(node => {
      expect(node.label).toMatch(/[\p{Emoji}]/u); // Unicode emoji check
    });
  });
});

describe('getRouteSteps', () => {
  const mockRoute: Route = {
    from: 'input',
    steps: [
      { translate: { expr: '$payload.id' } },
      { log: { level: 'info' } },
      { filter: { condition: '$payload.amount > 100' } },
    ],
  };

  it('extracts steps from route', () => {
    const steps = getRouteSteps(mockRoute);
    expect(steps).toHaveLength(3);
  });

  it('includes step name and type', () => {
    const steps = getRouteSteps(mockRoute);
    expect(steps[0].name).toBe('translate');
    expect(steps[0].type).toBe('translate');
    expect(steps[1].name).toBe('log');
    expect(steps[2].name).toBe('filter');
  });

  it('includes step config', () => {
    const steps = getRouteSteps(mockRoute);
    expect(steps[0].config.expr).toBe('$payload.id');
    expect(steps[1].config.level).toBe('info');
    expect(steps[2].config.condition).toBe('$payload.amount > 100');
  });

  it('returns empty array for undefined route', () => {
    expect(getRouteSteps(undefined)).toEqual([]);
  });

  it('returns empty array for route with no steps', () => {
    const route: Route = { from: 'input' };
    expect(getRouteSteps(route)).toEqual([]);
  });
});
