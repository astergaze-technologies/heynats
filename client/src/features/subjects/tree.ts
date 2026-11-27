import type { StoredStream, SubjectMap, Subscriber } from '@/lib/api';

export interface SubjectNode {
  token: string;
  path: string;
  children: Map<string, SubjectNode>;
  /** Subscriptions on exactly this subject (a wildcard subscription has its own node). */
  listeners: Subscriber[];
  /** Messages stored under exactly this subject, per stream. */
  stored: { stream: StoredStream; count: number }[];
  /** Streams whose subject pattern is exactly this node. */
  patternOf: StoredStream[];
}

const node = (token: string, path: string): SubjectNode => ({
  token,
  path,
  children: new Map(),
  listeners: [],
  stored: [],
  patternOf: [],
});

/** One node per token, so `orders.eu.created` sits under `orders` → `orders.eu`. */
export function buildTree(map: SubjectMap): SubjectNode {
  const root = node('', '');
  const at = (subject: string) => {
    let n = root;
    for (const token of subject.split('.')) {
      const path = n.path ? `${n.path}.${token}` : token;
      let next = n.children.get(token);
      if (!next) {
        next = node(token, path);
        n.children.set(token, next);
      }
      n = next;
    }
    return n;
  };
  for (const [subject, subs] of Object.entries(map.listening))
    at(subject).listeners.push(...subs);
  for (const s of map.streams) {
    for (const pattern of s.subjects ?? []) at(pattern).patternOf.push(s);
    for (const [subject, count] of Object.entries(s.counts ?? {}))
      at(subject).stored.push({ stream: s, count });
  }
  return root;
}

/** Totals for a node and everything under it, for the badges. */
export function totals(n: SubjectNode): {
  listeners: number;
  stored: number;
  streams: number;
} {
  let listeners = n.listeners.length;
  let stored = n.stored.reduce((a, s) => a + s.count, 0);
  let streams = n.patternOf.length;
  for (const c of n.children.values()) {
    const t = totals(c);
    listeners += t.listeners;
    stored += t.stored;
    streams += t.streams;
  }
  return { listeners, stored, streams };
}

export function find(root: SubjectNode, path: string): SubjectNode | undefined {
  let n: SubjectNode | undefined = root;
  for (const token of path.split('.')) n = n?.children.get(token);
  return n;
}

/** Children sorted with literals first, then "*", then ">". */
export function sortedChildren(n: SubjectNode): SubjectNode[] {
  const rank = (t: string) => (t === '>' ? 2 : t === '*' ? 1 : 0);
  return [...n.children.values()].sort(
    (a, b) => rank(a.token) - rank(b.token) || a.token.localeCompare(b.token)
  );
}
