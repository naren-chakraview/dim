import React, { useEffect, useState } from 'react';
import '../styles/LivePanel.css';

/** Live state of one route, as reported by the running viewer (read-only). */
export interface LiveRouteState {
  name: string;
  inFlight?: number;
  totalProcessed?: number;
  totalErrors?: number;
  p50LatencyMs?: number;
  p90LatencyMs?: number;
  p99LatencyMs?: number;
}

export interface LiveResponse {
  available: boolean;
  reason?: string;
  state?: LiveRouteState | null;
}

const REFRESH_MS = 5000;

/**
 * Shows a route's live state beside its definition. It polls /api/live, which reads
 * the running viewer. If the viewer cannot be read, the panel says so rather than
 * showing zero counts.
 */
export function LivePanel({ routeName }: { routeName: string | null }) {
  const [live, setLive] = useState<LiveResponse | null>(null);

  useEffect(() => {
    if (!routeName) {
      setLive(null);
      return;
    }
    let cancelled = false;
    const load = async () => {
      try {
        const response = await fetch(`/api/live?route=${encodeURIComponent(routeName)}`);
        const body: LiveResponse = await response.json();
        if (!cancelled) setLive(body);
      } catch (err) {
        if (!cancelled) {
          setLive({ available: false, reason: `live state request failed: ${String(err)}` });
        }
      }
    };
    load();
    const timer = setInterval(load, REFRESH_MS);
    return () => {
      cancelled = true;
      clearInterval(timer);
    };
  }, [routeName]);

  if (!routeName) return null;

  if (!live) {
    return <div className="live-panel">Loading live state…</div>;
  }

  if (!live.available || !live.state) {
    return (
      <div className="live-panel unavailable">
        <strong>Live state unavailable</strong>
        <p>{live.reason || 'The viewer did not report this route.'}</p>
      </div>
    );
  }

  const s = live.state;
  return (
    <div className="live-panel">
      <strong>Live: {s.name}</strong>
      <dl>
        <dt>Processed</dt>
        <dd>{s.totalProcessed ?? 0}</dd>
        <dt>Errors</dt>
        <dd>{s.totalErrors ?? 0}</dd>
        <dt>In flight</dt>
        <dd>{s.inFlight ?? 0}</dd>
        <dt>p50 / p99 (ms)</dt>
        <dd>
          {(s.p50LatencyMs ?? 0).toFixed(1)} / {(s.p99LatencyMs ?? 0).toFixed(1)}
        </dd>
      </dl>
    </div>
  );
}
