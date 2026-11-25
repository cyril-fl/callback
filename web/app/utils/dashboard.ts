// The pro dashboard's numbers, computed in the browser from the applications and their status history.
// shortcut: every application and status change is read, API routes once there are thousands.
import { parseDate, startOfWeek, type CalendarDate } from '@internationalized/date';
import type { Application, Stats, Status, StatusChange } from '../types/items';
import { STATUSES } from '../types/items';

/** The applications applied on since (a day, '' for all) in city ('all' for every one). */
export function inPeriod(applications: Application[], since: string, city: string) {
  return applications.filter((a) => (a.applied_at ?? '') >= since && (city === 'all' || a.location === city));
}

/** The counts of /api/stats, on these applications only. */
export function statsOf(applications: Application[]): Stats {
  const by_status = Object.fromEntries(STATUSES.map((s) => [s, 0])) as Record<Status, number>;
  const by_source: Record<string, number> = {};
  const responded: Record<string, number> = {};
  for (const a of applications) {
    by_status[a.status ?? 'applied']++;
    const source = a.source ?? 'spontaneous';
    by_source[source] = (by_source[source] ?? 0) + 1;
    if ((a.status ?? 'applied') !== 'applied') responded[source] = (responded[source] ?? 0) + 1;
  }
  const response_rate_by_source = Object.fromEntries(Object.entries(by_source).map(([s, n]) => [s, (responded[s] ?? 0) / n]));
  const total = applications.length;
  return { total, by_status, by_source, response_rate: total ? (total - by_status.applied) / total : undefined, response_rate_by_source };
}

/** The values and how many times each comes, the most first, top of them. */
export function countBy(values: string[], top = 8) {
  const counts = new Map<string, number>();
  for (const v of values) counts.set(v, (counts.get(v) ?? 0) + 1);
  return [...counts]
    .map(([label, count]) => ({ label, count }))
    .sort((a, b) => b.count - a.count)
    .slice(0, top);
}

/** A count a week (from Monday) from since, or the first day, to today's week, the empty weeks included. */
export function weeksOf(days: string[], since: string, today: CalendarDate) {
  const counts = new Map<string, number>();
  for (const d of days) {
    const week = startOfWeek(parseDate(d), 'fr-FR').toString();
    counts.set(week, (counts.get(week) ?? 0) + 1);
  }
  const first = since || [...counts.keys()].sort()[0];
  if (!first) return [];
  const weeks: { start: CalendarDate; count: number }[] = [];
  for (let w = startOfWeek(parseDate(first), 'fr-FR'); w.compare(today) <= 0; w = w.add({ weeks: 1 }))
    weeks.push({ start: w, count: counts.get(w.toString()) ?? 0 });
  return weeks;
}

/** The share in percent of each source's applications that got past applied, the best first. */
export function responseRates(applications: Application[], label: (source?: string) => string, top = 8) {
  const totals = new Map<string, { all: number; answered: number }>();
  for (const a of applications) {
    const total = totals.get(label(a.source)) ?? { all: 0, answered: 0 };
    total.all++;
    if (a.status !== 'applied') total.answered++;
    totals.set(label(a.source), total);
  }
  return [...totals]
    .map(([label, { all, answered }]) => ({ label, count: Math.round((answered / all) * 100) }))
    .sort((a, b) => b.count - a.count)
    .slice(0, top);
}

/** The status changes of these applications by application, in the API's order (oldest first). */
export function historyOf(changes: StatusChange[], applications: Application[]) {
  const ids = new Set(applications.map((a) => a.id));
  const byApplication = new Map<string, StatusChange[]>();
  for (const c of changes) {
    if (ids.has(c.application_id))
      byApplication.set(c.application_id, [...(byApplication.get(c.application_id) ?? []), c]);
  }
  return byApplication;
}

/** How many applications ever reached each status. */
export function funnelOf(history: Map<string, StatusChange[]>) {
  return STATUSES.map((status) => ({
    status,
    count: [...history.values()].filter((list) => list.some((c) => c.status === status)).length,
  }));
}

/** The days from applying to the first status past applied, on average over the applications that got one. */
export function averageDaysToResponse(applications: Application[], history: Map<string, StatusChange[]>) {
  const delays = applications.flatMap((a) => {
    const first = history.get(a.id)?.find((c) => c.status !== 'applied');
    if (!first || !a.applied_at) return [];
    return [Math.max(0, (new Date(first.at).getTime() - new Date(`${a.applied_at}T00:00:00`).getTime()) / 86_400_000)];
  });
  return delays.length ? Math.round((delays.reduce((n, d) => n + d, 0) / delays.length) * 10) / 10 : undefined;
}

/** The last news of an application: its last status change or follow-up, as the server's untouched_since. */
export function lastNews(a: Application, history: Map<string, StatusChange[]>) {
  const dates = [...(history.get(a.id) ?? []).map((c) => c.at), a.followed_up_at ?? a.applied_at ?? a.created_at ?? ''];
  return dates.reduce((last, d) => (new Date(d) > new Date(last) ? d : last));
}

/** Applied or responded and without news since limit, as the list's « À relancer »: the oldest news first. */
export function toFollowUp(applications: Application[], history: Map<string, StatusChange[]>, limit: Date) {
  return applications
    .filter((a) => a.status === 'applied' || a.status === 'responded')
    .map((a) => ({ ...a, news: lastNews(a, history) }))
    .filter((a) => new Date(a.news) < limit)
    .sort((a, b) => new Date(a.news).getTime() - new Date(b.news).getTime());
}

/** The interviews from now on, the nearest first; interview_at is a local "2026-01-31T14:30". */
export function upcomingInterviews(applications: Application[], now: Date) {
  return applications
    .filter((a) => a.interview_at && new Date(a.interview_at) >= now)
    .sort((a, b) => a.interview_at!.localeCompare(b.interview_at!));
}

/** Applied or responded and neither followed up nor applied since the follow-up delay started (a YYYY-MM-DD day): the
 * follow-up button shows. Never followed up, the delay runs from the day it applied. */
export const canFollowUp = (a: Application, since: string) =>
  (a.status === 'applied' || a.status === 'responded') &&
  (a.followed_up_at ?? a.applied_at ?? a.created_at ?? '').slice(0, 10) < since;
