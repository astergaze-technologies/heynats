import { render, screen } from '@testing-library/react';
import { expect, test } from 'vitest';
import { SourceBadge } from './SourceBadge';

test('names the source', () => {
  render(<SourceBadge source="system" />);
  expect(screen.getByText('via system account')).toBeTruthy();
});

test('says how many servers answered when partial', () => {
  render(<SourceBadge source="account" partial servers={2} expected={3} />);
  expect(
    screen.getByText('account scope · partial: 2 of 3 servers')
  ).toBeTruthy();
});
