<script setup lang="ts">
import { getLocalTimeZone, parseDate, startOfWeek, today as todayDate } from '@internationalized/date';
import type { Application, Stats, Status, StatusChange } from '~/types/items';
import { STATUSES } from '~/types/items';

// The pro mode's page: in base the stats stay above the list.
definePageMeta({ middleware: 'pro' });

const { t } = useI18n();
const api = useApi();
const sources = useSources();
const { preferences } = usePreferences();

// shortcut: counts in the browser from every application and every status change, API routes once there are thousands.
const { data: applications, refresh: refreshApplications } = await useAsyncData(
  'dashboard',
  () => api<Application[]>('/api/applications'),
  { default: () => [] },
);
const { data: changes, refresh: refreshChanges } = await useAsyncData(
  'status-changes',
  () => api<StatusChange[]>('/api/status-changes'),
  { default: () => [] },
);
const refresh = () => Promise.all([refreshApplications(), refreshChanges()]);

// The period and the city filter every block.
const periods = { '3m': { months: 3 }, '6m': { months: 6 }, '1y': { years: 1 }, all: undefined } as const;
type Period = keyof typeof periods;
const period = ref<Period>('6m');
const periodItems = (Object.keys(periods) as Period[]).map((value) => ({ value, label: t(`stats.periods.${value}`) }));
const city = ref('all');
const cityItems = computed(() => [
  { value: 'all', label: t('stats.allCities') },
  ...[...new Set(applications.value.flatMap((a) => (a.location ? [a.location] : [])))]
    .sort((a, b) => a.localeCompare(b))
    .map((value) => ({ value, label: value })),
]);

const today = todayDate(getLocalTimeZone());
const since = computed(() => {
  const duration = periods[period.value];
  return duration ? today.subtract(duration).toString() : '';
});
const filtered = computed(() =>
  applications.value.filter(
    (a) => (a.applied_at ?? '') >= since.value && (city.value === 'all' || a.location === city.value),
  ),
);

const stats = computed<Stats>(() => {
  const by_status = Object.fromEntries(STATUSES.map((s) => [s, 0])) as Record<Status, number>;
  for (const a of filtered.value) by_status[a.status ?? 'applied']++;
  const total = filtered.value.length;
  return { total, by_status, response_rate: total ? (total - by_status.applied) / total : 0 };
});

const count = (values: string[], top = 8) => {
  const counts = new Map<string, number>();
  for (const v of values) counts.set(v, (counts.get(v) ?? 0) + 1);
  return [...counts]
    .map(([label, count]) => ({ label, count }))
    .sort((a, b) => b.count - a.count)
    .slice(0, top);
};
const days = computed(() => filtered.value.flatMap((a) => (a.applied_at ? [a.applied_at] : [])));
const cities = computed(() =>
  count(filtered.value.flatMap((a) => (a.location ? [[a.location, a.country].filter(Boolean).join(', ')] : []))),
);
const bySource = computed(() => count(filtered.value.map((a) => sources.label(a.source))));

// A bar a week from the period's start (or the first application) to this week, empty weeks included.
const weeks = computed(() => {
  const counts = new Map<string, number>();
  for (const d of days.value) {
    const week = startOfWeek(parseDate(d), 'fr-FR').toString();
    counts.set(week, (counts.get(week) ?? 0) + 1);
  }
  const first = since.value || [...counts.keys()].sort()[0];
  if (!first) return [];
  const out = [];
  for (let w = startOfWeek(parseDate(first), 'fr-FR'); w.compare(today) <= 0; w = w.add({ weeks: 1 })) {
    const date = w.toDate(getLocalTimeZone()).toLocaleDateString('fr-FR', { day: 'numeric', month: 'short' });
    out.push({ label: date, count: counts.get(w.toString()) ?? 0 });
  }
  return out;
});

// The share of each source's applications that got past applied, in percent.
const rateBySource = computed(() => {
  const totals = new Map<string, { all: number; answered: number }>();
  for (const a of filtered.value) {
    const label = sources.label(a.source);
    const total = totals.get(label) ?? { all: 0, answered: 0 };
    total.all++;
    if (a.status !== 'applied') total.answered++;
    totals.set(label, total);
  }
  return [...totals]
    .map(([label, { all, answered }]) => ({ label, count: Math.round((answered / all) * 100) }))
    .sort((a, b) => b.count - a.count)
    .slice(0, 8);
});

// The history of each application of the filters, oldest first as the API gives it.
const history = computed(() => {
  const ids = new Set(filtered.value.map((a) => a.id));
  const byApplication = new Map<string, StatusChange[]>();
  for (const c of changes.value) {
    if (!ids.has(c.application_id)) continue;
    byApplication.set(c.application_id, [...(byApplication.get(c.application_id) ?? []), c]);
  }
  return byApplication;
});

// How many applications ever reached each status.
const funnel = computed(() =>
  STATUSES.map((s) => ({
    label: t(`status.${s}`),
    count: [...history.value.values()].filter((list) => list.some((c) => c.status === s)).length,
  })),
);

// The days from applying to the first status past applied, on average over the applications that got one.
const daysToResponse = computed(() => {
  const delays = filtered.value.flatMap((a) => {
    const first = history.value.get(a.id)?.find((c) => c.status !== 'applied');
    if (!first || !a.applied_at) return [];
    const applied = new Date(`${a.applied_at}T00:00:00`).getTime();
    return [Math.max(0, (new Date(first.at).getTime() - applied) / 86_400_000)];
  });
  return delays.length ? Math.round((delays.reduce((n, d) => n + d, 0) / delays.length) * 10) / 10 : undefined;
});

// Applied or responded, untouched for the follow-up delay: the list's « À relancer ».
const toFollowUp = computed(() => {
  const limit = today.subtract({ days: preferences.value?.follow_up_days ?? 7 }).toDate(getLocalTimeZone());
  return filtered.value
    .filter(
      (a) => (a.status === 'applied' || a.status === 'responded') && a.updated_at && new Date(a.updated_at) < limit,
    )
    .sort((a, b) => (a.updated_at ?? '').localeCompare(b.updated_at ?? ''));
});

// The interviews from now on, the nearest first; interview_at is a local "2026-01-31T14:30".
const upcoming = computed(() => {
  const now = new Date();
  return filtered.value
    .filter((a) => a.interview_at && new Date(a.interview_at) >= now)
    .sort((a, b) => a.interview_at!.localeCompare(b.interview_at!));
});

const day = (v?: string) => (v ? new Date(v.length === 10 ? `${v}T00:00:00` : v).toLocaleDateString('fr-FR') : '');
const stamp = (v?: string) =>
  v ? new Date(v).toLocaleString('fr-FR', { dateStyle: 'medium', timeStyle: 'short' }) : '';

const slideover = ref(false);
const editing = ref<Application>();
function openForm(app: Application) {
  editing.value = app;
  slideover.value = true;
}
</script>

<template>
  <div class="space-y-8">
    <div class="flex flex-wrap items-center justify-between gap-4">
      <h1 class="text-xl font-semibold">{{ t('nav.dashboard') }}</h1>
      <div class="flex flex-wrap items-center gap-2">
        <UTabs v-model="period" :items="periodItems" :content="false" size="sm" :aria-label="t('stats.period')" />
        <USelect v-model="city" :items="cityItems" :aria-label="t('form.city')" class="w-44" />
      </div>
    </div>

    <StatsCards :stats="stats" />

    <section>
      <h2 class="mb-3 text-sm font-semibold">{{ t('stats.activity') }}</h2>
      <ActivityHeatmap :days="days" />
    </section>

    <section>
      <h2 class="mb-3 text-sm font-semibold">{{ t('stats.perWeek') }}</h2>
      <WeekBars v-if="weeks.length" :weeks="weeks" />
      <p v-else class="text-xs text-muted">{{ t('common.empty') }}</p>
    </section>

    <div class="grid gap-8 md:grid-cols-2">
      <section>
        <h2 class="mb-3 text-sm font-semibold">{{ t('stats.toFollowUp') }}</h2>
        <ul v-if="toFollowUp.length" class="divide-y divide-default rounded-lg border border-default">
          <li v-for="a in toFollowUp.slice(0, 8)" :key="a.id">
            <button
              type="button"
              class="flex w-full items-center gap-4 px-4 py-2.5 text-left hover:bg-elevated"
              @click="openForm(a)"
            >
              <span class="flex-1 truncate text-sm">{{ a.company }}</span>
              <span class="text-xs text-muted">{{ t('stats.untouchedSince', { date: day(a.updated_at) }) }}</span>
            </button>
          </li>
        </ul>
        <p v-else class="text-xs text-muted">{{ t('common.empty') }}</p>
      </section>
      <section>
        <h2 class="mb-3 text-sm font-semibold">{{ t('stats.upcoming') }}</h2>
        <ul v-if="upcoming.length" class="divide-y divide-default rounded-lg border border-default">
          <li v-for="a in upcoming.slice(0, 8)" :key="a.id">
            <button
              type="button"
              class="flex w-full items-center gap-4 px-4 py-2.5 text-left hover:bg-elevated"
              @click="openForm(a)"
            >
              <span class="flex-1 truncate text-sm">{{ a.company }}</span>
              <span class="text-xs text-muted">{{ stamp(a.interview_at) }}</span>
            </button>
          </li>
        </ul>
        <p v-else class="text-xs text-muted">{{ t('common.empty') }}</p>
      </section>

      <section>
        <h2 class="mb-3 text-sm font-semibold">{{ t('stats.funnel') }}</h2>
        <BarList :items="funnel" />
        <p class="mt-2 text-xs text-muted">
          {{
            daysToResponse === undefined
              ? t('stats.noResponseYet')
              : t('stats.daysToResponse', { days: daysToResponse.toLocaleString('fr-FR') })
          }}
        </p>
      </section>
      <section>
        <h2 class="mb-3 text-sm font-semibold">{{ t('stats.rateBySource') }}</h2>
        <BarList v-if="rateBySource.length" :items="rateBySource" :max="100" unit=" %" />
        <p v-else class="text-xs text-muted">{{ t('common.empty') }}</p>
      </section>

      <section>
        <h2 class="mb-3 text-sm font-semibold">{{ t('stats.byCity') }}</h2>
        <BarList v-if="cities.length" :items="cities" />
        <p v-else class="text-xs text-muted">{{ t('common.empty') }}</p>
      </section>
      <section>
        <h2 class="mb-3 text-sm font-semibold">{{ t('stats.bySource') }}</h2>
        <BarList v-if="bySource.length" :items="bySource" />
        <p v-else class="text-xs text-muted">{{ t('common.empty') }}</p>
      </section>
    </div>

    <ApplicationSlideover v-model:open="slideover" :application="editing" @saved="refresh" />
  </div>
</template>
