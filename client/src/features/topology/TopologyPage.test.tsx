import { fireEvent, render, screen, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { beforeEach, expect, test, vi } from 'vitest';
import type { TopoEdge, Topology, TopoNode } from '@/lib/api';
import capabilitiesSystem from '@/mocks/fixtures/capabilities-system.json';
import topology from '@/mocks/fixtures/topology.json';
import { server } from '@/mocks/setup';
import { HEIGHT, layout, WIDTH } from './layout';

beforeEach(() => vi.resetModules());

async function open(url: string) {
  window.history.pushState({}, '', url);
  const { default: App } = await import('@/App');
  render(<App />);
}
const asSystem = (...extra: Parameters<typeof server.use>) =>
  server.use(
    http.get('/api/nats/capabilities', () =>
      HttpResponse.json(capabilitiesSystem)
    ),
    ...extra
  );

test('layout puts every node inside the canvas, apart, leafs under their server', () => {
  const nodes: TopoNode[] = [
    ...['n1', 'n2', 'n3'].map((id) => ({
      id,
      kind: 'server' as const,
      label: id,
      cluster: 'west',
      responding: true,
    })),
    { id: 'gateway:east', kind: 'gateway', label: 'east', responding: true },
    { id: 'leaf:edge-1', kind: 'leaf', label: 'edge-1', responding: true },
  ];
  const edges: TopoEdge[] = [
    {
      from: 'n2',
      to: 'leaf:edge-1',
      kind: 'leaf',
      links: 1,
      in_msgs: 0,
      out_msgs: 0,
    },
  ];
  const pos = layout(nodes, edges);
  const points = nodes.map((n) => pos.get(n.id));
  for (const p of points) {
    expect(p && p.x > 0 && p.x < WIDTH && p.y > 0 && p.y < HEIGHT).toBe(true);
  }
  expect(new Set(points.map((p) => `${p?.x},${p?.y}`)).size).toBe(nodes.length);
  expect(pos.get('leaf:edge-1')?.y).toBeGreaterThan(pos.get('n2')?.y ?? 0);
  expect(pos.get('gateway:east')?.x).toBeGreaterThan(pos.get('n3')?.x ?? 0);
});

test('every server and link is shown; a node opens its details', async () => {
  asSystem();
  await open('/dashboard/topology');
  for (const name of ['n1', 'n2', 'n3']) {
    expect(
      await screen.findByRole('button', { name: `Server ${name}` })
    ).toBeTruthy();
  }
  expect(screen.getByText(/3 of 3 servers responding · 3 links/)).toBeTruthy();

  fireEvent.click(screen.getByRole('button', { name: 'Server n2' }));
  const panel = within(
    await screen.findByRole('complementary', { name: 'Node details' })
  );
  expect(
    panel.getByRole('link', { name: 'Open server' }).getAttribute('href')
  ).toBe('/dashboard/servers/n2');
  expect(window.location.search).toBe('?node=n2');

  fireEvent.click(screen.getByRole('button', { name: 'Show as list' }));
  expect(screen.getAllByRole('row')).toHaveLength(4);
});

test('a stopped node stays on the graph, greyed', async () => {
  const t = topology as Topology;
  asSystem(
    http.get('/api/nats/topology', () =>
      HttpResponse.json({
        ...t,
        partial: true,
        nodes: t.nodes.map((n) =>
          n.id === 'n3' ? { ...n, responding: false } : n
        ),
        edges: t.edges.filter((e) => e.from !== 'n3' && e.to !== 'n3'),
      })
    )
  );
  await open('/dashboard/topology');
  const n3 = await screen.findByRole('button', {
    name: 'Server n3, not responding',
  });
  expect(within(n3).getByText('not responding')).toBeTruthy();
  expect(screen.getByText(/2 of 3 servers responding · 1 links/)).toBeTruthy();
});

test('account users are told why', async () => {
  await open('/dashboard/topology');
  expect(
    await screen.findByText('Topology needs the system account')
  ).toBeTruthy();
});
