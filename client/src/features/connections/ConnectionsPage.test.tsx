import { fireEvent, render, screen, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { afterAll, beforeAll, beforeEach, expect, test, vi } from 'vitest';
import type { Conn, ConnPage } from '@/lib/api';
import connections from '@/mocks/fixtures/connections.json';
import { server } from '@/mocks/setup';

// jsdom lays nothing out; give the virtualised list a 600px viewport to fill.
const proto = HTMLElement.prototype;
const original = {
  offsetHeight: Object.getOwnPropertyDescriptor(proto, 'offsetHeight'),
  offsetWidth: Object.getOwnPropertyDescriptor(proto, 'offsetWidth'),
};
beforeAll(() => {
  Object.defineProperty(proto, 'offsetHeight', {
    configurable: true,
    get: () => 600,
  });
  Object.defineProperty(proto, 'offsetWidth', {
    configurable: true,
    get: () => 1200,
  });
});
afterAll(() => {
  for (const [key, desc] of Object.entries(original)) {
    if (desc) Object.defineProperty(proto, key, desc);
  }
});
beforeEach(() => vi.resetModules());

async function open(url: string) {
  window.history.pushState({}, '', url);
  const { default: App } = await import('@/App');
  render(<App />);
}

const table = () => within(screen.getByRole('table', { name: 'Connections' }));

test('open connections from every server, with their server', async () => {
  await open('/dashboard/connections');
  expect(await screen.findByText('billing-api')).toBeTruthy();
  expect(table().getAllByText('orders-worker', { exact: false })).toHaveLength(
    3
  );
  expect(screen.getByText('via system account')).toBeTruthy();
});

test('closed tab shows why each client went away', async () => {
  await open('/dashboard/connections?tab=closed');
  expect(await screen.findByText('short-lived-job')).toBeTruthy();
  expect(table().getAllByText('Client Closed').length).toBeGreaterThan(0);
});

test('filters come from the URL and go to the server', async () => {
  let asked = '';
  server.use(
    http.get('/api/nats/connections', ({ request }) => {
      asked = new URL(request.url).search;
      return HttpResponse.json(connections);
    })
  );
  await open('/dashboard/connections?name=billing&account=APP');
  await screen.findByText('billing-api');
  expect(asked).toContain('name=billing');
  expect(asked).toContain('account=APP');
  expect(
    (screen.getByLabelText('Filter by name') as HTMLInputElement).value
  ).toBe('billing');
});

test('10,000 connections render only the visible rows', async () => {
  const base = connections.connections[0] as Conn;
  const many: ConnPage = {
    ...(connections as ConnPage),
    total: 10_000,
    matched: 10_000,
    connections: Array.from({ length: 10_000 }, (_, i) => ({
      ...base,
      cid: i,
      name: `client-${i}`,
    })),
  };
  server.use(http.get('/api/nats/connections', () => HttpResponse.json(many)));
  await open('/dashboard/connections');
  expect(await screen.findByText('client-0')).toBeTruthy();
  expect(table().getAllByRole('row').length).toBeLessThan(60);
  expect(screen.getByText(/10K\s+of 10K/)).toBeTruthy();
});

test('clicking a row opens its details and subscriptions', async () => {
  await open('/dashboard/connections');
  fireEvent.click(await screen.findByText('billing-api'));
  const panel = within(
    await screen.findByRole('complementary', { name: 'Connection details' })
  );
  expect(await panel.findByText('billing.quote')).toBeTruthy();
  expect(window.location.search).toMatch(/server_detail=n2&cid=56/);
});
