<script setup lang="ts">
import type { Application, Stats } from '~/types/items';

// The pro mode's page: in base the stats stay above the list.
definePageMeta({ middleware: 'pro' });

const { t } = useI18n();
const api = useApi();
const sources = useSources();

const { data: stats } = await useAsyncData('stats', () => api<Stats>('/api/stats'));
// shortcut: counts in the browser from every application's few fields, an API route once there are thousands.
const { data: applications } = await useAsyncData(
  'dashboard',
  () => api<Application[]>('/api/applications', { query: { fields: 'applied_at,location,country,source' } }),
  { default: () => [] },
);

const count = (values: string[], top = 8) => {
  const counts = new Map<string, number>();
  for (const v of values) counts.set(v, (counts.get(v) ?? 0) + 1);
  return [...counts]
    .map(([label, count]) => ({ label, count }))
    .sort((a, b) => b.count - a.count)
    .slice(0, top);
};
const days = computed(() => applications.value.flatMap((a) => (a.applied_at ? [a.applied_at] : [])));
const cities = computed(() =>
  count(applications.value.flatMap((a) => (a.location ? [[a.location, a.country].filter(Boolean).join(', ')] : []))),
);
const bySource = computed(() => count(applications.value.map((a) => sources.label(a.source))));
</script>

<template>
  <div class="space-y-8">
    <div class="flex items-baseline justify-between gap-4">
      <h1 class="text-xl font-semibold">{{ t('nav.dashboard') }}</h1>
      <p class="text-sm text-muted">{{ t('stats.total') }} : {{ stats?.total ?? 0 }}</p>
    </div>

    <StatsCards :stats="stats" />

    <section>
      <h2 class="mb-3 text-sm font-semibold">{{ t('stats.activity') }}</h2>
      <ActivityHeatmap :days="days" />
    </section>

    <div class="grid gap-8 md:grid-cols-2">
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
  </div>
</template>
