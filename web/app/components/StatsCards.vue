<script setup lang="ts">
import type { Stats } from '~/types/items';
import { STATUSES } from '~/types/items';

defineProps<{ stats?: Stats | null }>();
const { t } = useI18n();
</script>

<template>
  <div class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
    <!-- no_reply is only a step of the dashboard's funnel: a card of it would be gloomy. -->
    <UCard v-for="s in STATUSES.filter((s) => s !== 'no_reply')" :key="s">
      <p class="text-sm text-muted">{{ t(`status.${s}`) }}</p>
      <p class="text-2xl font-semibold">{{ stats?.by_status[s] ?? 0 }}</p>
    </UCard>
    <UCard>
      <p class="text-sm text-muted">{{ t('stats.responseRate') }}</p>
      <p class="text-2xl font-semibold">
        {{ stats?.response_rate === undefined ? '—' : `${Math.round(stats.response_rate * 100)} %` }}
      </p>
    </UCard>
  </div>
</template>
