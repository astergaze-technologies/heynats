import { render, screen } from '@testing-library/react';
import { expect, test } from 'vitest';

test('dashboard renders the connected server from fixtures', async () => {
  window.history.pushState({}, '', '/dashboard');
  // The router is created on import, so import after setting the URL.
  const { default: App } = await import('@/App');
  render(<App />);

  expect(await screen.findByText('n1')).toBeTruthy();
  expect(screen.getByText('nats://localhost:4222')).toBeTruthy();
});
