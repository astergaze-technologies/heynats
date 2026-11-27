import { render, screen, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { beforeEach, expect, test, vi } from 'vitest';
import capabilities from '@/mocks/fixtures/capabilities.json';
import capabilitiesSystem from '@/mocks/fixtures/capabilities-system.json';
import jetstreamSystem from '@/mocks/fixtures/jetstream-system.json';
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

async function openOverview() {
  window.history.pushState({}, '', '/dashboard');
  const { default: App } = await import('@/App');
  render(<App />);
  await screen.findByText('n1');
}

const tile = (title: string) =>
  within(
    screen
      .getByText(title, { selector: 'h3' })
      .closest('div.rounded-lg') as HTMLElement
  );

test('account scope: own account totals, no limit bar', async () => {
  await openOverview();
  expect(await tile('Streams · consumers').findByText('2 · 2')).toBeTruthy();
  expect(tile('JetStream storage').getByText('369.14 KB')).toBeTruthy();
  expect(tile('JetStream storage').getByText('no limit set')).toBeTruthy();
  expect(tile('Servers').getByText('account scope')).toBeTruthy();
  expect(screen.getByText('Messages in per server')).toBeTruthy();
  expect(screen.queryByRole('status')).toBeNull();
});

test('system account: every server, storage against the cluster limit', async () => {
  server.use(
    http.get('/api/nats/capabilities', () =>
      HttpResponse.json(capabilitiesSystem)
    ),
    http.get('/api/nats/jetstream', () => HttpResponse.json(jetstreamSystem))
  );
  await openOverview();
  expect(await tile('Servers').findByText('via system account')).toBeTruthy();
  expect(await tile('JetStream storage').findByRole('meter')).toBeTruthy();
});

test('partial answers show a banner and "2 of 3"', async () => {
  server.use(
    http.get('/api/nats/capabilities', () =>
      HttpResponse.json({ ...capabilities, servers: 2, partial: true })
    )
  );
  await openOverview();
  expect((await screen.findByRole('status')).textContent).toMatch(
    /Only 2 of 3 servers answered/
  );
  expect(tile('Servers').getByText('2 of 3')).toBeTruthy();
});
