import { expect, test } from 'vitest';
import type { SubjectMap } from '@/lib/api';
import { buildTree, find, sortedChildren, totals } from './tree';

const map: SubjectMap = {
  source: 'account',
  partial: false,
  hidden_inboxes: 0,
  listening: {
    'orders.eu.created': [{ server: 'n2', cid: 7, name: 'eu', msgs: 0 }],
    'orders.*.created': [{ server: 'n1', cid: 9, queue: 'workers', msgs: 0 }],
  },
  streams: [
    {
      name: 'ORDERS',
      subjects: ['orders.>'],
      counts: { 'orders.eu.created': 3, 'orders.us.created': 1 },
    },
  ],
};

test('tree groups by token, keeps wildcards as their own nodes', () => {
  const root = buildTree(map);
  const orders = find(root, 'orders');
  expect(orders && sortedChildren(orders).map((c) => c.token)).toEqual([
    'eu',
    'us',
    '*',
    '>',
  ]);
  expect(find(root, 'orders.eu.created')?.listeners[0].name).toBe('eu');
  expect(find(root, 'orders.*.created')?.listeners[0].queue).toBe('workers');
  expect(find(root, 'orders.>')?.patternOf[0].name).toBe('ORDERS');
  expect(orders && totals(orders)).toEqual({
    listeners: 2,
    stored: 4,
    streams: 1,
  });
});
