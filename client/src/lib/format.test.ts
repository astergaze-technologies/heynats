import { expect, test } from 'vitest';
import {
  formatBytes,
  formatClock,
  formatCount,
  formatRate,
  formatUptime,
} from './format';

test('formatCount', () => {
  expect(formatCount(0)).toBe('0');
  expect(formatCount(999)).toBe('999');
  expect(formatCount(1000)).toBe('1K');
  expect(formatCount(1234)).toBe('1.2K');
  expect(formatCount(1234, { fullBelow: 10_000 })).toBe('1,234');
  expect(formatCount(-1500)).toBe('-1.5K');
  expect(formatCount(Number.NaN)).toBe('—');
});

test('formatRate', () => {
  expect(formatRate(0)).toBe('0 msg/s');
  expect(formatRate(0.42)).toBe('0.4 msg/s');
  expect(formatRate(999)).toBe('999 msg/s');
  expect(formatRate(1250, 'B')).toBe('1.3K B/s');
  expect(formatRate(Number.POSITIVE_INFINITY)).toBe('—');
});

test('formatBytes', () => {
  expect(formatBytes(0)).toBe('0 B');
  expect(formatBytes(0.5)).toBe('0.5 B');
  expect(formatBytes(999)).toBe('999 B');
  expect(formatBytes(1024)).toBe('1 KB');
  expect(formatBytes(12_923)).toBe('12.62 KB');
  expect(formatBytes(-2048)).toBe('-2 KB');
  expect(formatBytes(Number.NaN)).toBe('—');
  expect(formatBytes(1024 ** 7)).toBe('1048576 PB');
});

test('formatClock', () => {
  expect(formatClock(0, true)).toBe('00:00 UTC');
  expect(formatClock(3600 * 10 + 42 * 60, true)).toBe('10:42 UTC');
});

test('formatUptime', () => {
  const now = Date.parse('2026-01-03T05:30:00Z');
  expect(formatUptime('2026-01-03T05:29:15Z', now)).toBe('45s');
  expect(formatUptime('2026-01-03T05:18:00Z', now)).toBe('12m');
  expect(formatUptime('2026-01-03T02:18:00Z', now)).toBe('3h 12m');
  expect(formatUptime('2026-01-01T02:00:00Z', now)).toBe('2d 3h');
  expect(formatUptime('2026-01-04T00:00:00Z', now)).toBe('0s');
  expect(formatUptime('not a date', now)).toBe('—');
});
