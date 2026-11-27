import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { expect, test } from 'vitest';
import { TimeRangePicker, useTimeRange } from './TimeRangePicker';

function Page() {
  const [range, setRange] = useTimeRange();
  return (
    <>
      <TimeRangePicker value={range} onChange={setRange} />
      <output>{useLocation().search}</output>
    </>
  );
}

const renderAt = (url: string) =>
  render(
    <MemoryRouter initialEntries={[url]}>
      <Page />
    </MemoryRouter>
  );

test('range is read from the URL and written back', () => {
  renderAt('/overview?range=6h&q=x');
  expect(
    screen.getByRole('button', { name: '6h' }).getAttribute('aria-pressed')
  ).toBe('true');

  fireEvent.click(screen.getByRole('button', { name: '15m' }));
  expect(screen.getByRole('status').textContent).toBe('?range=15m&q=x');
});

test('unknown range falls back to 1h', () => {
  renderAt('/overview?range=7d');
  expect(
    screen.getByRole('button', { name: '1h' }).getAttribute('aria-pressed')
  ).toBe('true');
});
