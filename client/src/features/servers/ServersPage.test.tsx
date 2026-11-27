import { render, screen } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { beforeEach, expect, test, vi } from 'vitest';
import capabilitiesSystem from '@/mocks/fixtures/capabilities-system.json';
import servers from '@/mocks/fixtures/servers.json';
import { server } from '@/mocks/setup';

vi.mock('uplot', () => ({
  default: class {
    setData() {}
    setSize() {}
    destroy() {}
  },
}));
vi.mock('uplot/dist/uPlot.min.css', () => ({}));

// Fresh router and query cache per test: both are created on import.
beforeEach(() => vi.resetModules());

const asSystem = (...extra: Parameters<typeof server.use>) =>
  server.use(
    http.get('/api/nats/capabilities', () =>
      HttpResponse.json(capabilitiesSystem)
    ),
    ...extra
  );

async function open(url: string) {
  window.history.pushState({}, '', url);
  const { default: App } = await import('@/App');
  render(<App />);
}

test('lists every server with health', async () => {
  asSystem();
  await open('/dashboard/servers');
  for (const name of ['n1', 'n2', 'n3']) {
    expect(await screen.findByRole('link', { name })).toBeTruthy();
  }
  expect(screen.getAllByText('Healthy')).toHaveLength(3);
  expect(screen.getByText('3 of 3 servers responding')).toBeTruthy();
});

test('a stopped node stays listed as not responding; odd versions are flagged', async () => {
  const [n1, n2, n3] = servers.servers;
  asSystem(
    http.get('/api/nats/servers', () =>
      HttpResponse.json({
        partial: true,
        expected: 3,
        servers: [
          n1,
          { ...n2, version: '2.11.4' },
          { ...n3, responding: false, health: undefined },
        ],
      })
    )
  );
  await open('/dashboard/servers');
  expect(await screen.findByText('Not responding')).toBeTruthy();
  expect(screen.getByRole('link', { name: 'n3' })).toBeTruthy();
  expect(screen.getByText('differs from 2.12.0')).toBeTruthy();
  expect(screen.getByText('2 of 3 servers responding')).toBeTruthy();
});

test('account users are told why', async () => {
  await open('/dashboard/servers');
  expect(
    await screen.findByText('Servers need the system account')
  ).toBeTruthy();
});

test('detail tabs read the server itself', async () => {
  asSystem();
  await open('/dashboard/servers/n1?tab=config');
  expect(await screen.findByText('Max payload')).toBeTruthy();
  expect(screen.getByText('1 MB')).toBeTruthy();

  screen.getByRole('tab', { name: 'Routes / Gateways / Leafs' }).click();
  expect(await screen.findByText(/^\(8\)$/)).toBeTruthy();
  expect(screen.getByText('No gateways configured.')).toBeTruthy();
});

test('unknown server name', async () => {
  asSystem();
  await open('/dashboard/servers/zz');
  expect(await screen.findByText('No server named zz')).toBeTruthy();
});
