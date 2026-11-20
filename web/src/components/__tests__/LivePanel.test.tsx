import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { LivePanel } from '../LivePanel';

afterEach(() => {
  vi.unstubAllGlobals();
});

function stubLive(body: unknown) {
  vi.stubGlobal('fetch', vi.fn(async () => ({ json: async () => body })));
}

describe('LivePanel', () => {
  it('renders nothing without a route', () => {
    const { container } = render(<LivePanel routeName={null} />);
    expect(container.innerHTML).toBe('');
  });

  it('shows the viewer counts for the route', async () => {
    stubLive({
      available: true,
      state: { name: 'order-payment', totalProcessed: 7, totalErrors: 1, p50LatencyMs: 2.5, p99LatencyMs: 9 },
    });
    render(<LivePanel routeName="order-payment" />);
    expect(await screen.findByText('Live: order-payment')).toBeTruthy();
    expect(screen.getByText('7')).toBeTruthy();
    expect(screen.getByText('1')).toBeTruthy();
  });

  it('says when live state is unavailable, instead of showing zeros', async () => {
    stubLive({ available: false, reason: 'viewer unreachable at http://localhost:8081' });
    render(<LivePanel routeName="order-payment" />);
    expect(await screen.findByText('Live state unavailable')).toBeTruthy();
    expect(screen.getByText(/viewer unreachable/)).toBeTruthy();
  });
});
