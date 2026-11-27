import { render, screen, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { beforeEach, expect, test, vi } from 'vitest';
import degraded from '@/mocks/fixtures/jetstream-health-degraded.json';
import system from '@/mocks/fixtures/jetstream-health-system.json';
import { server } from '@/mocks/setup';

vi.mock('uplot', () => ({
  default: class {
    setData() {}
    setSize() {}
    destroy() {}
  },
}));
vi.mock('uplot/dist/uPlot.min.css', () => ({}));

beforeEach(() => vi.resetModules());

async function open(url: string) {
  window.history.pushState({}, '', url);
  const { default: App } = await import('@/App');
  render(<App />);
}
const health = (body: unknown) =>
  server.use(
    http.get('/api/nats/jetstream/health', () => HttpResponse.json(body))
  );
const rowOf = async (name: string) =>
  (await screen.findByText(name, { selector: 'span.font-medium' })).closest(
    'tr'
  ) as HTMLElement;

test('meta group, replicated stream and consumers', async () => {
  health(system);
  await open('/dashboard/health?tab=jetstream');
  expect(await screen.findByText('n2 · leader')).toBeTruthy();
  const orders = within(await rowOf('ORDERS'));
  expect(orders.getByText('Healthy')).toBeTruthy();
  expect(orders.getByText('n1 · current')).toBeTruthy();
  expect(within(await rowOf('billing')).getByText('800')).toBeTruthy();
});

test('an offline replica shows on the stream', async () => {
  health(degraded);
  await open('/dashboard/health?tab=jetstream');
  const orders = within(await rowOf('ORDERS'));
  expect(orders.getByText('1 replica offline')).toBeTruthy();
  expect(orders.getByText('n3 · offline')).toBeTruthy();
});

test('account scope explains the missing meta group', async () => {
  await open('/dashboard/health?tab=jetstream');
  expect(
    await screen.findByText(/visible to system account users only/)
  ).toBeTruthy();
});

test('overview lists the most lagging consumers', async () => {
  await open('/dashboard');
  const card = within(
    (await screen.findByText('Top lagging consumers')).closest(
      'section'
    ) as HTMLElement
  );
  expect(card.getAllByText('800 pending')).toHaveLength(2);
});

test('streams list gets a health column', async () => {
  server.use(
    http.get('/api/nats/streams', () =>
      HttpResponse.json({
        streams: [
          {
            config: { name: 'ORDERS', subjects: ['orders.>'] },
            state: { messages: 800, bytes: 1, consumer_count: 2 },
          },
        ],
      })
    )
  );
  await open('/dashboard/streams');
  const row = (await screen.findByRole('link', { name: 'ORDERS' })).closest(
    'tr'
  ) as HTMLElement;
  expect(await within(row).findByText('Healthy')).toBeTruthy();
});
