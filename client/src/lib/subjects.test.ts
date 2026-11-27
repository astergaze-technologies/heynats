import { expect, test } from 'vitest';
import cases from '../../../testdata/subject-match.json';
import { subjectMatches, subjectsOverlap } from './subjects';

// The same table runs against the Go matcher.
test.each(cases)(
  '$pattern ~ $subject → $match',
  ({ pattern, subject, match }) => {
    expect(subjectMatches(pattern, subject)).toBe(match);
    if (match) expect(subjectsOverlap(pattern, subject)).toBe(true);
  }
);
