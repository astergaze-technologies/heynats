import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { beforeEach, expect, test, vi } from 'vitest';
import findings from '@/mocks/fixtures/diagnostics-consumers.json';
import health from '@/mocks/fixtures/jetstream-health-consumers.json';
import { server } from '@/mocks/setup';

beforeEach(() => vi.resetModules());

async function open(url: string) {
  window.history.pushState({}, '', url);
  const { default: App } = await import('@/App');
  render(<App />);
}
const withConsumerFindings = (...extra: Parameters<typeof server.use>) =>
  server.use(
    http.get('/api/nats/diagnostics', () => HttpResponse.json(findings)),
    http.get('/api/nats/jetstream/health', () => HttpResponse.json(health)),
    ...extra
  );

test('paused consumer can be resumed from its finding, after confirming', async () => {
  let resumed = '';
  withConsumerFindings(
    http.post(
      '/api/nats/streams/consumers/:stream/:consumer/resume',
      ({ params }) => {
        resumed = `${params.stream}/${params.consumer}`;
        return HttpResponse.json({ resumed: true });
      }
    )
  );
  await open('/dashboard/health');
  const card = (
    await screen.findByRole('heading', {
      name: 'Consumer ORDERS/shipping is paused',
    })
  ).closest('article') as HTMLElement;
  fireEvent.click(within(card).getByRole('button', { name: 'Resume' }));
  const dialog = within(await screen.findByRole('alertdialog'));
  expect(dialog.getByText('Resume shipping?')).toBeTruthy();
  expect(resumed).toBe('');
  fireEvent.click(dialog.getByRole('button', { name: 'Resume' }));
  await waitFor(() => expect(resumed).toBe('ORDERS/shipping'));
});

test('ack limit finding offers the CLI to raise it', async () => {
  withConsumerFindings();
  await open('/dashboard/health');
  const card = (
    await screen.findByRole('heading', {
      name: 'Consumer ORDERS/billing is waiting for acks',
    })
  ).closest('article') as HTMLElement;
  expect(within(card).getByText('5 of max 5')).toBeTruthy();
  expect(
    within(card).getByText('nats consumer edit ORDERS billing --max-pending 10')
  ).toBeTruthy();
});

test('stream page lists its consumers with their findings', async () => {
  withConsumerFindings(
    http.get('/api/nats/streams/:name', () =>
      HttpResponse.json({
        config: { name: 'ORDERS', subjects: ['orders.>'] },
        state: { messages: 50, bytes: 1, consumer_count: 2 },
      })
    )
  );
  await open('/dashboard/streams/ORDERS');
  const section = within(
    (
      await screen.findByRole('heading', { name: 'Consumers', level: 2 })
    ).closest('section') as HTMLElement
  );
  expect(
    await section.findByText('Consumer ORDERS/shipping is paused')
  ).toBeTruthy();
  expect(
    section.getByText('Consumer ORDERS/billing is waiting for acks')
  ).toBeTruthy();
  expect(section.getByText('45 pending')).toBeTruthy();
});
