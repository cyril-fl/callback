<script setup lang="ts">
import type { SourceCount, Stats } from '~/types/items';

// The pro mode's page: in base the stats stay above the list.
definePageMeta({ middleware: () => (usePlan().value === 'pro' ? undefined : navigateTo('/', { replace: true })) });

const { t } = useI18n();
const api = useApi();
const sources = useSources();

const { data: stats } = await useAsyncData('stats', () => api<Stats>('/api/stats'));
const { data: counts } = await useAsyncData('source-counts', () => api<SourceCount[]>('/api/sources'), {
  default: () => [],
});
const most = computed(() => Math.max(1, ...counts.value.map((s) => s.count)));
</script>

<template>
  <div class="space-y-8">
    <div class="flex items-baseline justify-between gap-4">
      <h1 class="text-xl font-semibold">{{ t('nav.dashboard') }}</h1>
      <p class="text-sm text-muted">{{ t('stats.total') }} : {{ stats?.total ?? 0 }}</p>
    </div>

    <StatsCards :stats="stats" />

    <section>
      <h2 class="mb-3 text-sm font-semibold">{{ t('stats.bySource') }}</h2>
      <ul v-if="counts.length" class="divide-y divide-default rounded-lg border border-default">
        <li v-for="s in counts" :key="s.source" class="flex items-center gap-4 px-4 py-3">
          <p class="w-40 shrink-0 truncate text-sm">{{ sources.label(s.source) }}</p>
          <div class="h-2 flex-1 overflow-hidden rounded-full bg-elevated">
            <div class="h-full rounded-full bg-primary" :style="{ width: `${(s.count / most) * 100}%` }" />
          </div>
          <span class="w-8 shrink-0 text-right text-sm text-muted">{{ s.count }}</span>
        </li>
      </ul>
      <p v-else class="text-xs text-muted">{{ t('common.empty') }}</p>
    </section>
  </div>
</template>
