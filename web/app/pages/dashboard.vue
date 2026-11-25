<script setup lang="ts">
import { getLocalTimeZone, today as todayDate } from '@internationalized/date';
import type { Application, StatusChange } from '~/types/items';

// The pro mode's page: in base the stats stay above the list.
definePageMeta({ middleware: 'pro' });

const { t } = useI18n();
const api = useApi();
const sources = useSources();
const { preferences } = usePreferences();

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
const filtered = computed(() => inPeriod(applications.value, since.value, city.value));
const stats = computed(() => statsOf(filtered.value));
const days = computed(() => filtered.value.flatMap((a) => (a.applied_at ? [a.applied_at] : [])));
const cities = computed(() =>
  countBy(filtered.value.flatMap((a) => (a.location ? [[a.location, a.country].filter(Boolean).join(', ')] : []))),
);
const bySource = computed(() => countBy(filtered.value.map((a) => sources.label(a.source))));
const weeks = computed(() =>
  weeksOf(days.value, since.value, today).map((w) => ({
    label: w.start.toDate(getLocalTimeZone()).toLocaleDateString('fr-FR', { day: 'numeric', month: 'short' }),
    count: w.count,
  })),
);
const rateBySource = computed(() => responseRates(filtered.value, sources.label));
const history = computed(() => historyOf(changes.value, filtered.value));
const funnel = computed(() => funnelOf(history.value).map((s) => ({ label: t(`status.${s.status}`), count: s.count })));
const daysToResponse = computed(() => averageDaysToResponse(filtered.value, history.value));
const followUps = computed(() =>
  toFollowUp(
    filtered.value,
    today.subtract({ days: preferences.value?.follow_up_days ?? 7 }).toDate(getLocalTimeZone()),
  ),
);
const upcoming = computed(() => upcomingInterviews(filtered.value, new Date()));

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
      <ActivityHeatmap :days="days" :since="since" />
    </section>

    <section>
      <h2 class="mb-3 text-sm font-semibold">{{ t('stats.perWeek') }}</h2>
      <WeekBars v-if="weeks.length" :weeks="weeks" />
      <p v-else class="text-xs text-muted">{{ t('common.empty') }}</p>
    </section>

    <div class="grid gap-8 md:grid-cols-2">
      <section>
        <h2 class="mb-3 text-sm font-semibold">{{ t('stats.toFollowUp') }}</h2>
        <ul v-if="followUps.length" class="divide-y divide-default rounded-lg border border-default">
          <li v-for="a in followUps.slice(0, 8)" :key="a.id">
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
