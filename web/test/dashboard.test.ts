import assert from 'node:assert/strict';
import { test } from 'node:test';
import { parseDate } from '@internationalized/date';
import type { Application, StatusChange } from '../app/types/items.ts';
import {
  addressOf,
  appliedBounds,
  averageDaysToResponse,
  canFollowUp,
  countBy,
  dayBounds,
  funnelOf,
  historyOf,
  inPeriod,
  interviewsOf,
  responseRates,
  statsOf,
  toFollowUp,
  websiteHref,
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
    by_status: { applied: 1, responded: 1, interview: 1, offer: 0, rejected: 0, no_reply: 0 },
    by_source: { job_post: 2, spontaneous: 1 },
    response_rate: 2 / 3,
    response_rate_by_source: { job_post: 0.5, spontaneous: 1 },
  });
  assert.equal(statsOf([]).response_rate, undefined);
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
    [3, 2, 1, 0, 0, 0],
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
  // An interview is news, and an interview to come keeps it off the list.
  const interviewed = apps.map((a) =>
    a.id === 'a' ? { ...a, status: 'interview' as const, interview_at: '2026-01-28T10:00' } : a,
  );
  assert.deepEqual(ids(interviewed, '2026-02-01T00:00:00Z'), ['c', 'a']);
  assert.deepEqual(ids(interviewed, '2026-01-25T00:00:00Z'), ['c']);
});

test('the follow-up button: waiting on the employer, and no follow-up nor interview, else no application, since the delay started', () => {
  // Applied on 2026-01-05, entered later: the delay runs from the day it applied.
  const app: Application = { ...apps[0]!, created_at: '2026-01-20T10:00:00Z' };
  assert.equal(canFollowUp(app, '2026-01-01'), false);
  assert.equal(canFollowUp(app, '2026-01-06'), true);
  // Followed up since the delay started: no button, created long ago or not.
  assert.equal(canFollowUp({ ...app, followed_up_at: '2026-01-20T10:00:00Z' }, '2026-01-15'), false);
  assert.equal(canFollowUp({ ...app, followed_up_at: '2026-01-10T10:00:00Z' }, '2026-01-15'), true);
  // An interview waits on the employer too, once it is past the start of the delay.
  assert.equal(canFollowUp({ ...app, status: 'interview', interview_at: '2026-01-03T10:00' }, '2026-01-06'), true);
  assert.equal(canFollowUp({ ...app, status: 'interview', interview_at: '2026-01-08T10:00' }, '2026-01-06'), false);
  // Past the follow-up statuses: never.
  assert.equal(canFollowUp({ ...app, status: 'offer' }, '2026-01-06'), false);
  assert.equal(canFollowUp({ ...app, status: 'no_reply' }, '2026-01-06'), false);
});

test('interviewsOf lists every interview between two days, the earliest first', () => {
  const list: Application[] = [
    { id: 'a', interviews: [{ at: '2026-02-02T14:30' }, { at: '2026-01-10T09:00' }] },
    { id: 'b', interviews: [{ at: '2026-01-20T10:00' }] },
    { id: 'c' },
  ];
  const at = (from?: string, to?: string) => interviewsOf(list, from, to).map((r) => `${r.application.id} ${r.at}`);
  assert.deepEqual(at(), ['a 2026-01-10T09:00', 'b 2026-01-20T10:00', 'a 2026-02-02T14:30']);
  assert.deepEqual(at('2026-01-20'), ['b 2026-01-20T10:00', 'a 2026-02-02T14:30']);
  assert.deepEqual(at(undefined, '2026-01-20'), ['a 2026-01-10T09:00', 'b 2026-01-20T10:00']);
  assert.deepEqual(at('2026-01-20', '2026-01-20'), ['b 2026-01-20T10:00']);
});

test('dayBounds turns a day filter into from and to', () => {
  assert.deepEqual(dayBounds('all', '2026-01-20'), [undefined, undefined]);
  assert.deepEqual(dayBounds('after', '2026-01-20'), ['2026-01-20', undefined]);
  assert.deepEqual(dayBounds('before', '2026-01-20'), [undefined, '2026-01-20']);
  assert.deepEqual(dayBounds('on', '2026-01-20'), ['2026-01-20', '2026-01-20']);
  assert.deepEqual(dayBounds('between', '2026-01-20', { start: '2026-01-01', end: '2026-01-31' }), [
    '2026-01-01',
    '2026-01-31',
  ]);
});

test('appliedBounds moves the strict bounds out by a day', () => {
  assert.deepEqual(appliedBounds(['2026-03-01', '2026-02-28']), {
    applied_after: '2026-02-28',
    applied_before: '2026-03-01',
  });
  assert.deepEqual(appliedBounds([undefined, undefined]), { applied_after: undefined, applied_before: undefined });
});

test('addressOf joins the parts it has', () => {
  assert.equal(
    addressOf({ id: 'a', street: '1 rue X', postal_code: '75001', location: 'Paris' }, 'France'),
    '1 rue X, 75001 Paris, France',
  );
  assert.equal(addressOf({ id: 'a', location: 'Paris' }, ''), 'Paris');
});

test('websiteHref mails an address and links a site', () => {
  assert.equal(websiteHref('jobs@acme.fr'), 'mailto:jobs@acme.fr');
  assert.equal(websiteHref('https://acme.fr/@team'), 'https://acme.fr/@team');
});
