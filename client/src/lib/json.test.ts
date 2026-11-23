import assert from 'node:assert/strict';
import { test } from 'vitest';
import {
  formatJSON,
  isLosslessNumber,
  isValidJSON,
  parseJSON,
} from './json.ts';

const compact = (s: string) => s.replace(/\n\s*/g, '').replace(/": /g, '":');

test('issue payload keeps int64 snowflake IDs exact', () => {
  const raw =
    '{"taskId":864786760355909,"organizationId":"323425076823199749","assetId":323439282914070533,"mimeType":"text/html","contentId":816983156080709}';
  assert.match(formatJSON(raw), /"assetId": 323439282914070533,/);
  assert.equal(compact(formatJSON(raw)), raw);
});

test('numbers are formatted exactly as stored', () => {
  for (const raw of [
    '{"id":9007199254740993}',
    '{"id":-9223372036854775808}',
    '{"id":18446744073709551615}',
    '{"id":123456789012345678901234567890}',
    '{"x":0.1000000000000000055511151231257827}',
    '{"x":1.0,"y":1e3,"z":2E-7,"w":-0}',
    '[323439282914070533,[{"n":1e400}]]',
    '323439282914070533',
  ]) {
    assert.equal(compact(formatJSON(raw)), raw, raw);
  }
});

test('tree viewer values keep their exact digits', () => {
  const tree = parseJSON(
    '{"a":{"id":323439282914070533},"l":[18446744073709551615]}'
  ) as {
    a: { id: unknown };
    l: unknown[];
  };
  assert.ok(isLosslessNumber(tree.a.id));
  assert.equal(String(tree.a.id), '323439282914070533');
  assert.equal(String(tree.l[0]), '18446744073709551615');
});

test('layout matches JSON.stringify for ordinary JSON', () => {
  for (const v of [
    { a: 1, b: [1, { c: null }], d: { e: 'x,y:{}' } },
    { empty: {}, list: [] },
    [1, 'two'],
    null,
  ]) {
    assert.equal(formatJSON(JSON.stringify(v)), JSON.stringify(v, null, 2));
  }
});

test('randomized: numbers survive formatting', () => {
  let seed = 7;
  const rand = (n: number) => {
    seed = (seed * 1103515245 + 12345) % 2 ** 31;
    return seed % n;
  };
  const nums = [
    '323439282914070533',
    '-0',
    '1.0',
    '1e400',
    '0.30000000000000004',
    '18446744073709551615',
    '-1.50E+10',
  ];
  const gen = (d: number): string => {
    if (d > 3 || rand(3) === 0) return nums[rand(nums.length)];
    const items = Array.from({ length: rand(4) }, () => gen(d + 1));
    return rand(2)
      ? `[${items.join(',')}]`
      : `{${items.map((v, i) => `"k${i}":${v}`).join(',')}}`;
  };
  for (let i = 0; i < 500; i++) {
    const raw = gen(0);
    assert.equal(compact(formatJSON(raw)), raw);
  }
});

test('invalid and duplicate-key JSON fall back to raw', () => {
  assert.equal(formatJSON('hello {not json'), 'hello {not json');
  assert.equal(formatJSON('{"a":1,"a":2}'), '{"a":1,"a":2}');
  assert.equal(isValidJSON('{"a":1,"a":2}'), false);
  assert.equal(isValidJSON('{"id":323439282914070533}'), true);
});
