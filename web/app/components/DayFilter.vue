<script setup lang="ts">
import type { CalendarDate, DateValue } from '@internationalized/date';

// A date column's filter: all days, from a day, until a day, on a day, or between two, picked on a calendar.
const mode = defineModel<DayMode>('mode', { required: true });
const day = defineModel<CalendarDate>('day');
const range = defineModel<{ start: DateValue | undefined; end: DateValue | undefined }>('range');
const { t } = useI18n();
const modes = DAY_MODES.map((value) => ({ value, label: t(`dayFilter.${value}`) }));
</script>

<template>
  <div class="space-y-2 p-2">
    <UTabs v-model="mode" :items="modes" :content="false" size="xs" />
    <UCalendar v-if="mode === 'between'" v-model="range" range />
    <UCalendar v-else-if="mode !== 'all'" v-model="day" />
  </div>
</template>
