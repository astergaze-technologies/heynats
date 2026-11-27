import type { DiagnosticsReport } from '@/lib/api';

/** Findings as Markdown, for pasting into an incident channel or ticket. */
export function toMarkdown(r: DiagnosticsReport): string {
  const lines = [
    `# NATS health: ${r.verdict}`,
    '',
    `Checked ${new Date(r.checked_at).toISOString()} (${r.source} account data).`,
    '',
  ];
  if (r.findings.length === 0) lines.push('No open findings.');
  for (const f of r.findings) {
    lines.push(
      `## [${f.severity}] ${f.title}`,
      '',
      `\`${f.resource}\` · ${f.rule_id} · since ${new Date(f.first_seen).toISOString()}`,
      '',
      f.detail,
      ''
    );
    for (const e of f.evidence)
      lines.push(`- ${e.label}: ${e.value} (${e.source})`);
    for (const x of f.fixes ?? [])
      if (x.cli) lines.push(`- Fix: ${x.label}: \`${x.cli}\``);
    lines.push('');
  }
  return lines.join('\n');
}
