import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import { expect, test } from 'vitest';
import { Gate } from './Gate';

const renderGated = (feature: 'servers' | 'trace') =>
  render(
    <QueryClientProvider client={new QueryClient()}>
      <Gate feature={feature}>
        <button type="button">Open</button>
      </Gate>
    </QueryClientProvider>
  );

test('unavailable feature is dimmed, unusable and says why', async () => {
  renderGated('servers');
  expect(
    await screen.findByText('Requires a system account user')
  ).toBeTruthy();
  expect(screen.getByText('Open').closest('[inert]')).toBeTruthy();
});

test('available feature renders as is', async () => {
  renderGated('trace');
  await new Promise((r) => setTimeout(r, 50)); // let capabilities load
  expect(screen.queryByText(/Requires/)).toBeNull();
  expect(screen.getByText('Open').closest('[inert]')).toBeNull();
});
