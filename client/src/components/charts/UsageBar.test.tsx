import { render, screen } from '@testing-library/react';
import { expect, test } from 'vitest';
import { UsageBar } from './UsageBar';

test.each([
  [50, '50%', false],
  [85, '85% · high', false],
  [97, '97% · critical', false],
  [150, '100% · critical', false],
])('%i of 100 reads %s', (value, text) => {
  render(<UsageBar label="Storage" value={value} max={100} />);
  expect(screen.getByText(`${value} of 100 · ${text}`)).toBeTruthy();
  const meter = screen.getByRole('meter', { name: 'Storage' });
  expect((meter as HTMLMeterElement).value).toBe(Math.min(value, 100));
});
