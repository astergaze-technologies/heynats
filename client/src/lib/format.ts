// Shared number/time formatting for tiles, tables, chart axes and tooltips.

const compact = new Intl.NumberFormat('en', {
  notation: 'compact',
  maximumFractionDigits: 1,
});

/** 999 -> "999", 1234 -> "1.2K", 12_900_000 -> "12.9M"; whole below 10,000 when asked. */
export function formatCount(n: number, { fullBelow = 1000 } = {}): string {
  if (!Number.isFinite(n)) return '—';
  if (Math.abs(n) < fullBelow) return Math.round(n).toLocaleString('en');
  return compact.format(n);
}

/** Per-second rate: 0.4 -> "0.4 msg/s", 1234 -> "1.2K msg/s". */
export function formatRate(n: number, unit = 'msg'): string {
  if (!Number.isFinite(n)) return '—';
  const value =
    Math.abs(n) > 0 && Math.abs(n) < 10
      ? Number(n.toFixed(1)).toString()
      : formatCount(n);
  return `${value} ${unit}/s`;
}

const UNITS = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];

/** 1024-based, two decimals at most: 12_923 -> "12.62 KB". */
export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes)) return '—';
  if (bytes === 0) return '0 B';
  const sign = bytes < 0 ? '-' : '';
  const abs = Math.abs(bytes);
  const i = Math.min(
    UNITS.length - 1,
    Math.max(0, Math.floor(Math.log(abs) / Math.log(1024)))
  );
  return `${sign}${Number.parseFloat((abs / 1024 ** i).toFixed(2))} ${UNITS[i]}`;
}

/** Unix seconds -> local "10:42", or "10:42 UTC" when utc. */
export function formatClock(unixSeconds: number, utc = false): string {
  const time = new Date(unixSeconds * 1000).toLocaleTimeString('en-GB', {
    hour: '2-digit',
    minute: '2-digit',
    timeZone: utc ? 'UTC' : undefined,
  });
  return utc ? `${time} UTC` : time;
}
