// NATS subject matching ("*" one token, ">" one or more trailing tokens).
// Kept in step with internal/subject: both test against testdata/subject-match.json.

/** Does a literal subject match the pattern? */
export function subjectMatches(pattern: string, subject: string): boolean {
  const p = pattern.split('.');
  const s = subject.split('.');
  for (let i = 0; i < p.length; i++) {
    if (p[i] === '>') return s.length > i;
    if (i >= s.length) return false;
    if (p[i] !== '*' && p[i] !== s[i]) return false;
  }
  return p.length === s.length;
}

/** Can some subject match both patterns? */
export function subjectsOverlap(a: string, b: string): boolean {
  const at = a.split('.');
  const bt = b.split('.');
  for (let i = 0; ; i++) {
    if (i === at.length && i === bt.length) return true;
    if (i === at.length || i === bt.length) return false;
    if (at[i] === '>' || bt[i] === '>') return true;
    if (at[i] !== '*' && bt[i] !== '*' && at[i] !== bt[i]) return false;
  }
}

export const isWildcard = (s: string) =>
  s.split('.').some((t) => t === '*' || t === '>');
