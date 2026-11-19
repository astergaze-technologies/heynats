import { isLosslessNumber, parse, stringify } from 'lossless-json';

export { isLosslessNumber };

export function parseJSON(data: string): unknown {
  return parse(data);
}

export function formatJSON(data: string): string {
  try {
    return stringify(parse(data), null, 2) ?? data;
  } catch {
    return data;
  }
}

export function isValidJSON(data: string): boolean {
  try {
    parse(data);
    return true;
  } catch {
    return false;
  }
}
