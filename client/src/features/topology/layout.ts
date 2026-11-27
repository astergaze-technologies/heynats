import type { TopoEdge, TopoNode } from '@/lib/api';

export const WIDTH = 1000;
export const HEIGHT = 560;

export type Point = { x: number; y: number };

/**
 * Deterministic layout for a few dozen nodes: each cluster's servers on a
 * circle, clusters side by side, remote gateways in a column on the right,
 * leaf nodes in a row under the server they attach to.
 */
export function layout(
  nodes: TopoNode[],
  edges: TopoEdge[]
): Map<string, Point> {
  const pos = new Map<string, Point>();
  const servers = nodes.filter((n) => n.kind === 'server');
  const clusters = [...new Set(servers.map((n) => n.cluster ?? ''))].sort();
  const gateways = nodes.filter((n) => n.kind === 'gateway');
  const usable = gateways.length ? WIDTH * 0.8 : WIDTH;

  clusters.forEach((cluster, ci) => {
    const members = servers.filter((n) => (n.cluster ?? '') === cluster);
    const cx = (usable / clusters.length) * (ci + 0.5);
    const cy = HEIGHT * 0.38;
    const r =
      members.length === 1 ? 0 : Math.min(150, usable / clusters.length / 2.6);
    members.forEach((n, i) => {
      const a = -Math.PI / 2 + (2 * Math.PI * i) / members.length;
      pos.set(n.id, { x: cx + r * Math.cos(a), y: cy + r * Math.sin(a) });
    });
  });

  gateways.forEach((n, i) => {
    pos.set(n.id, {
      x: WIDTH * 0.92,
      y: (HEIGHT / (gateways.length + 1)) * (i + 1),
    });
  });

  const leafsUnder = new Map<string, number>();
  for (const e of edges) {
    if (e.kind !== 'leaf') continue;
    const at = pos.get(e.from);
    if (!at || pos.has(e.to)) continue;
    const k = leafsUnder.get(e.from) ?? 0;
    leafsUnder.set(e.from, k + 1);
    pos.set(e.to, {
      x: at.x + ((k % 3) - 1) * 90,
      y: HEIGHT * 0.82 + Math.floor(k / 3) * 50,
    });
  }
  return pos;
}
