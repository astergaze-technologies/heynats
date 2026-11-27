import { fireEvent, render, screen, within } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { beforeEach, expect, test, vi } from 'vitest';
import accounts from '@/mocks/fixtures/accounts.json';
import { server } from '@/mocks/setup';

beforeEach(() => vi.resetModules());

async function open(url: string) {
  window.history.pushState({}, '', url);
  const { default: App } = await import('@/App');
  render(<App />);
}

test('every account with usage, system account marked', async () => {
  await open('/dashboard/accounts');
  for (const name of ['APP', 'USER', 'SYS']) {
    expect(await screen.findByRole('link', { name })).toBeTruthy();
  }
  const sys = screen
    .getByRole('link', { name: 'SYS' })
    .closest('tr') as HTMLElement;
  expect(within(sys).getByText('system')).toBeTruthy();
  const app = screen
    .getByRole('link', { name: 'APP' })
    .closest('tr') as HTMLElement;
  expect(within(app).getByText(/no limit set/)).toBeTruthy();
});

test("USER's import links back to APP's export", async () => {
  await open('/dashboard/accounts/USER?tab=imports');
  // The subject and its local name are both events.>; take the first cell.
  const [cell] = await screen.findAllByText('events.>', { selector: 'td' });
  const row = cell.closest('tr') as HTMLElement;
  expect(within(row).getByText('stream')).toBeTruthy();
  const builtIns = screen.getAllByText('built-in');
  expect(builtIns.length).toBeGreaterThan(0);

  fireEvent.click(within(row).getByRole('link', { name: 'APP' }));
  expect(await screen.findByText('Exports (2)')).toBeTruthy();
  expect(screen.getAllByText('Any account')).toHaveLength(2);
});

test('mappings and JWT tabs', async () => {
  await open('/dashboard/accounts/APP?tab=mappings');
  expect(await screen.findByText('orders.legacy.>')).toBeTruthy();
  expect(screen.getByText('100%')).toBeTruthy();
  fireEvent.click(screen.getByRole('tab', { name: 'JWT' }));
  expect(await screen.findByText('No JWT for this account')).toBeTruthy();
});

test('account scope: own account only, details explain why not', async () => {
  const own = accounts.accounts.filter((a) => a.name === 'APP');
  server.use(
    http.get('/api/nats/accounts', () =>
      HttpResponse.json({ ...accounts, source: 'account', accounts: own })
    )
  );
  await open('/dashboard/accounts');
  expect(await screen.findByText(/yours only/)).toBeTruthy();
  expect(screen.queryByRole('link', { name: 'USER' })).toBeNull();

  fireEvent.click(screen.getByRole('link', { name: 'APP' }));
  fireEvent.click(
    await screen.findByRole('tab', { name: 'Imports / Exports' })
  );
  expect(await screen.findByText('Needs the system account')).toBeTruthy();
});
