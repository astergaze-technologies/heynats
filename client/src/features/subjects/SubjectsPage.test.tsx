import { fireEvent, render, screen, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { beforeEach, expect, test, vi } from 'vitest';
import health from '@/mocks/fixtures/jetstream-health-subjects.json';
import { server } from '@/mocks/setup';

beforeEach(() => vi.resetModules());

async function open(url: string) {
  window.history.pushState({}, '', url);
  const { default: App } = await import('@/App');
  render(<App />);
}

test('top level shows who listens and what is stored below', async () => {
  await open('/dashboard/subjects');
  const tree = within(await screen.findByRole('list', { name: 'Subjects' }));
  const orders = tree.getByRole('button', { name: 'orders' })
    .parentElement as HTMLElement;
  expect(within(orders).getByText('2 listening')).toBeTruthy();
  expect(within(orders).getByText('5 stored')).toBeTruthy();
  fireEvent.click(tree.getByRole('button', { name: 'Expand orders' }));
  expect(tree.getByRole('button', { name: 'eu' })).toBeTruthy();
  expect(tree.getByRole('button', { name: '*' })).toBeTruthy();
  expect(screen.getByText(/3 reply-inbox subscriptions hidden/)).toBeTruthy();
});

test('searching by prefix opens the way to matches', async () => {
  await open('/dashboard/subjects?q=orders.eu');
  const tree = within(await screen.findByRole('list', { name: 'Subjects' }));
  expect(tree.getByRole('button', { name: 'created' })).toBeTruthy();
  expect(tree.getByRole('button', { name: 'shipped' })).toBeTruthy();
  expect(tree.queryByRole('button', { name: 'us' })).toBeNull();
  expect(tree.queryByRole('button', { name: 'billing' })).toBeNull();
});

test('inspector: subscribers, wildcard listeners, streams and consumers', async () => {
  server.use(
    http.get('/api/nats/jetstream/health', () => HttpResponse.json(health))
  );
  await open('/dashboard/subjects?subject=orders.eu.created');
  const panel = within(
    await screen.findByRole('complementary', { name: 'Subject details' })
  );
  expect(panel.getByText(/eu-listener/)).toBeTruthy();
  expect(panel.getByText(/orders\.\*\.created \(1\)/)).toBeTruthy();
  expect(panel.getByText(/3 stored here/)).toBeTruthy();
  expect(await panel.findByText('eu-only')).toBeTruthy();
});

test('wildcard tester', async () => {
  await open('/dashboard/subjects');
  expect(await screen.findByText('yes, it matches')).toBeTruthy();
  fireEvent.change(screen.getByLabelText('Subject'), {
    target: { value: 'orders.eu.west.created' },
  });
  expect(screen.getByText("no, it doesn't match")).toBeTruthy();
});
