import { fireEvent, render, screen } from '@testing-library/react';
import { expect, test, vi } from 'vitest';
import type { MetricsWindow } from '@/lib/api';
import { formatRate } from '@/lib/format';
import fixture from '@/mocks/fixtures/metrics.json';
import { TimeSeriesChart } from './TimeSeriesChart';

// jsdom has no canvas; the plot itself is uPlot's job.
vi.mock('uplot', () => ({
  default: class {
    setData() {}
    setSize() {}
    destroy() {}
  },
}));
vi.mock('uplot/dist/uPlot.min.css', () => ({}));

// History started 10 minutes ago, inside the 1h range, whenever the test runs.
const data = {
  ...(fixture as MetricsWindow),
  history_since: new Date(Date.now() - 10 * 60_000).toISOString(),
};
const chart = (d?: MetricsWindow) =>
  render(
    <TimeSeriesChart
      title="Messages in"
      data={d}
      series="msgs_in"
      format={(v) => formatRate(v)}
      rangeSeconds={3600}
    />
  );

test('legend names every server and history start is stated', () => {
  chart(data);
  for (const server of ['n1', 'n2', 'n3']) {
    expect(screen.getByText(server)).toBeTruthy();
  }
  expect(screen.getByText(/^History since \d\d:\d\d$/)).toBeTruthy();
});

test('table view lists every point', () => {
  chart(data);
  fireEvent.click(screen.getByRole('button', { name: 'Show table' }));
  expect(screen.getAllByRole('row')).toHaveLength(data.t.length + 1);
});

test('empty and loading states', () => {
  const { unmount } = chart({ ...data, t: [], series: { msgs_in: {} } });
  expect(screen.getByText(/Collecting data/)).toBeTruthy();
  unmount();
  chart(undefined);
  expect(screen.getByText('Loading…')).toBeTruthy();
});
