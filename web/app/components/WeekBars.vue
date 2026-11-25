<script setup lang="ts">
// A bar a week, its height the week's count: plain Tailwind, as BarList.
const props = defineProps<{ weeks: { label: string; count: number }[] }>();
const { t } = useI18n();
const most = computed(() => Math.max(1, ...props.weeks.map((w) => w.count)));
</script>

<template>
  <div>
    <ul class="flex h-32 items-end gap-px rounded-lg border border-default px-2 pt-2">
      <li
        v-for="week in weeks"
        :key="week.label"
        class="flex h-full flex-1 items-end"
        :title="t('stats.weekCount', { date: week.label }, week.count)"
      >
        <span class="sr-only">{{ t('stats.weekCount', { date: week.label }, week.count) }}</span>
        <div class="w-full rounded-t-sm bg-primary" :style="{ height: `${(week.count / most) * 100}%` }" />
      </li>
    </ul>
    <div v-if="weeks.length" class="mt-1 flex justify-between text-xs text-muted">
      <span>{{ weeks[0]!.label }}</span>
      <span>{{ weeks.at(-1)!.label }}</span>
    </div>
  </div>
</template>
