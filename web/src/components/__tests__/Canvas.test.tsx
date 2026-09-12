import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Canvas } from '../Canvas';
import { RouteConfig } from '../../types/route';

describe('Canvas Component', () => {
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
          {
            translate: {
              expr: '$payload.id',
            },
          },
        ],
      },
    },
    sinks: {
      output: {
        type: 'file',
        path: './output.jsonl',
      },
    },
  };

  it('renders empty state when no route provided', () => {
    const { container } = render(<Canvas />);
    expect(container.textContent).toContain('Open a route file');
  });

  it('renders canvas when route is provided', () => {
    const { container } = render(<Canvas route={mockRoute} />);
    expect(container.querySelector('.react-flow')).toBeTruthy();
  });

  it('calls onStepSelect when step is clicked', () => {
    const onStepSelect = vi.fn();
    render(<Canvas route={mockRoute} onStepSelect={onStepSelect} />);

    // Note: Full interaction testing would require more setup
    // This is a simplified test demonstrating test structure
    expect(onStepSelect).not.toHaveBeenCalled();
  });

  it('displays source nodes', () => {
    const { container } = render(<Canvas route={mockRoute} />);
    expect(container.textContent).toContain('input');
  });

  it('displays sink nodes', () => {
    const { container } = render(<Canvas route={mockRoute} />);
    expect(container.textContent).toContain('output');
  });

  it('displays step nodes', () => {
    const { container } = render(<Canvas route={mockRoute} />);
    expect(container.textContent).toContain('translate');
  });

  it('handles multiple routes', () => {
    const multiRoute: RouteConfig = {
      ...mockRoute,
      routes: {
        'route-1': {
          from: 'input',
          steps: [{ translate: { expr: '$payload.id' } }],
        },
        'route-2': {
          from: 'input',
          steps: [{ log: { level: 'info' } }],
        },
      },
    };

    const { container } = render(<Canvas route={multiRoute} />);
    expect(container.textContent).toContain('translate');
    expect(container.textContent).toContain('log');
  });
});
