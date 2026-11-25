import assert from 'node:assert/strict';
import { test } from 'node:test';
import { parseDate } from '@internationalized/date';
import type { Application, StatusChange } from '../app/types/items.ts';
import {
  averageDaysToResponse,
  countBy,
  funnelOf,
  historyOf,
  inPeriod,
  responseRates,
  statsOf,
  toFollowUp,
  upcomingInterviews,
  weeksOf,
} from '../app/utils/dashboard.ts';

const apps: Application[] = [
  {
    id: 'a',
    company: 'Acme',
    location: 'Paris',
    source: 'job_post',
    status: 'applied',
    applied_at: '2026-01-05',
    updated_at: '2026-01-05T10:00:00Z',
  },
  {
    id: 'b',
    company: 'Globex',
    location: 'Lyon',
    source: 'job_post',
    status: 'interview',
    applied_at: '2026-01-07',
    updated_at: '2026-01-20T10:00:00Z',
    interview_at: '2026-02-02T14:30',
  },
  {
    id: 'c',
    company: 'Initech',
    location: 'Paris',
    source: 'spontaneous',
    status: 'responded',
    applied_at: '2026-01-20',
    updated_at: '2026-01-25T10:00:00Z',
    interview_at: '2026-01-10T09:00',
  },
];
const changes: StatusChange[] = [
  { application_id: 'a', status: 'applied', at: '2026-01-05T10:00:00Z' },
  { application_id: 'b', status: 'applied', at: '2026-01-07T10:00:00Z' },
  { application_id: 'c', status: 'applied', at: '2026-01-20T10:00:00Z' },
  { application_id: 'b', status: 'responded', at: '2026-01-10T00:00:00' },
  { application_id: 'c', status: 'responded', at: '2026-01-21T00:00:00' },
  { application_id: 'b', status: 'interview', at: '2026-01-20T10:00:00Z' },
  { application_id: 'gone', status: 'applied', at: '2026-01-01T10:00:00Z' },
];

test('the period and the city filter the applications', () => {
  assert.deepEqual(
    inPeriod(apps, '', 'all').map((a) => a.id),
    ['a', 'b', 'c'],
  );
  assert.deepEqual(
    inPeriod(apps, '2026-01-07', 'all').map((a) => a.id),
    ['b', 'c'],
  );
  assert.deepEqual(
    inPeriod(apps, '', 'Paris').map((a) => a.id),
    ['a', 'c'],
  );
});

test('the stats count every status and the share past applied', () => {
  assert.deepEqual(statsOf(apps), {
    total: 3,
    by_status: { applied: 1, responded: 1, interview: 1, offer: 0, rejected: 0 },
    by_source: { job_post: 2, spontaneous: 1 },
    response_rate: 2 / 3,
  });
  assert.equal(statsOf([]).response_rate, 0);
});

test('countBy sorts the most first and keeps the top', () => {
  assert.deepEqual(countBy(['x', 'y', 'y', 'z', 'y', 'x'], 2), [
    { label: 'y', count: 3 },
    { label: 'x', count: 2 },
  ]);
});

test('a bar a week from Monday, the empty weeks included', () => {
  const weeks = weeksOf(['2026-01-05', '2026-01-07', '2026-01-20'], '', parseDate('2026-01-21'));
  assert.deepEqual(
    weeks.map((w) => [w.start.toString(), w.count]),
    [
      ['2026-01-05', 2],
      ['2026-01-12', 0],
      ['2026-01-19', 1],
    ],
  );
  assert.equal(weeksOf([], '2026-01-01', parseDate('2026-01-21'))[0]!.start.toString(), '2025-12-29');
  assert.deepEqual(weeksOf([], '', parseDate('2026-01-21')), []);
});

test('the response rate is per source, in percent', () => {
  assert.deepEqual(
    responseRates(apps, (s) => s ?? ''),
    [
      { label: 'spontaneous', count: 100 },
      { label: 'job_post', count: 50 },
    ],
  );
});

test('the funnel counts the applications that ever reached a status, in the filters only', () => {
  const history = historyOf(changes, apps);
  assert.equal(history.has('gone'), false);
  assert.deepEqual(
    funnelOf(history).map((s) => s.count),
    [3, 2, 1, 0, 0],
  );
});

test('the days to a first response average the applications that got one', () => {
  // b: 3 days (applied the 7th, responded the 10th), c: 1 day; a never answered.
  assert.equal(averageDaysToResponse(apps, historyOf(changes, apps)), 2);
  assert.equal(averageDaysToResponse([apps[0]!], historyOf(changes, apps)), undefined);
});

test('to follow up: applied or responded, no status change nor follow-up since the limit, the oldest news first', () => {
  const history = historyOf(changes, apps);
  const ids = (list: Application[], limit: string) => toFollowUp(list, history, new Date(limit)).map((a) => a.id);
  assert.deepEqual(ids(apps, '2026-02-01T00:00:00Z'), ['a', 'c']);
  assert.deepEqual(ids(apps, '2026-01-10T00:00:00Z'), ['a']);
  // A follow-up is news.
  const followed = apps.map((a) => (a.id === 'a' ? { ...a, followed_up_at: '2026-01-25T10:00:00Z' } : a));
  assert.deepEqual(ids(followed, '2026-02-01T00:00:00Z'), ['c', 'a']);
  assert.deepEqual(ids(followed, '2026-01-22T00:00:00Z'), ['c']);
});

test('the upcoming interviews are the ones from now on, the nearest first', () => {
  assert.deepEqual(
    upcomingInterviews(apps, new Date('2026-01-15T00:00:00')).map((a) => a.id),
    ['b'],
  );
  assert.deepEqual(
    upcomingInterviews(apps, new Date('2026-01-01T00:00:00')).map((a) => a.id),
    ['c', 'b'],
  );
});
