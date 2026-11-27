import { render, screen } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { beforeEach, expect, test, vi } from 'vitest';
import type { DiagnosticsReport } from '@/lib/api';
import degraded from '@/mocks/fixtures/diagnostics-degraded.json';
import { server } from '@/mocks/setup';
import { toMarkdown } from './markdown';

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

test('healthy cluster', async () => {
  await open('/dashboard/health');
  expect(await screen.findByText('Healthy')).toBeTruthy();
  expect(screen.getByText(/All checks passing/)).toBeTruthy();
  expect(screen.getByText('n3 · ok')).toBeTruthy();
});

test('stopped node: degraded verdict, finding with evidence and a fix', async () => {
  server.use(
    http.get('/api/nats/diagnostics', () => HttpResponse.json(degraded))
  );
  await open('/dashboard/health');
  expect(await screen.findByText('Degraded')).toBeTruthy();
  expect(
    screen.getByRole('heading', { name: 'n3 is not responding' })
  ).toBeTruthy();
  expect(screen.getByText('n3 · not responding')).toBeTruthy();
  expect(screen.getByText('Missed checks')).toBeTruthy();
  expect(
    screen.getByRole('button', { name: 'Copy nats server list' })
  ).toBeTruthy();
});

test('overview shows open findings', async () => {
  server.use(
    http.get('/api/nats/diagnostics', () => HttpResponse.json(degraded))
  );
  await open('/dashboard');
  expect(await screen.findByText('n3 is not responding')).toBeTruthy();
  expect(screen.getByRole('link', { name: 'Open health' })).toBeTruthy();
});

test('markdown export lists findings with their fix', () => {
  const md = toMarkdown(degraded as DiagnosticsReport);
  expect(md).toMatch(/^# NATS health: degraded/);
  expect(md).toContain('## [warning] n3 is not responding');
  expect(md).toContain('`server:n3`');
  expect(md).toContain('`nats server list`');
});
