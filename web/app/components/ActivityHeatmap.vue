<script setup lang="ts">
// The days applied on over the last year, a column a week from Monday, as GitHub's contributions.
const props = defineProps<{ days: string[] }>();
const { t } = useI18n();

const iso = (d: Date) =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
const counts = computed(() => {
  const m = new Map<string, number>();
  for (const d of props.days) m.set(d.slice(0, 10), (m.get(d.slice(0, 10)) ?? 0) + 1);
  return m;
});

const weeks = computed(() => {
  const today = new Date();
  today.setHours(12, 0, 0, 0);
  const start = new Date(today);
  start.setDate(start.getDate() - 52 * 7 - ((today.getDay() + 6) % 7));
  const out: { key: string; date: Date; count: number; future: boolean }[][] = [];
  for (const d = new Date(start); d <= today || out.at(-1)!.length < 7; d.setDate(d.getDate() + 1)) {
    if (!out.length || out.at(-1)!.length === 7) out.push([]);
    const key = iso(d);
    out.at(-1)!.push({ key, date: new Date(d), count: counts.value.get(key) ?? 0, future: d > today });
  }
  return out;
});
const most = computed(() => Math.max(1, ...weeks.value.flat().map((d) => d.count)));
const total = computed(() => weeks.value.flat().reduce((n, d) => n + d.count, 0));

const levels = ['bg-elevated', 'bg-primary/25', 'bg-primary/50', 'bg-primary/75', 'bg-primary'];
const level = (count: number) => levels[count && Math.ceil((count / most.value) * 4)];
// A month's name above the week it starts in.
const month = (week: { date: Date }[], i: number) =>
  i === 0 || week[0]!.date.getMonth() !== weeks.value[i - 1]![0]!.date.getMonth()
    ? week[0]!.date.toLocaleDateString('fr-FR', { month: 'short' })
    : '';

// On a phone the recent weeks show first.
const scroller = ref<HTMLElement>();
onMounted(() => scroller.value && (scroller.value.scrollLeft = scroller.value.scrollWidth));
</script>

<template>
  <div class="rounded-lg border border-default px-4 py-3">
    <div ref="scroller" class="overflow-x-auto pb-1">
      <div class="flex w-max gap-1" role="img" :aria-label="t('stats.activityTotal', total)">
        <div v-for="(week, i) in weeks" :key="week[0]!.key" class="flex flex-col gap-1">
          <span class="h-4 w-3 overflow-visible text-[10px] whitespace-nowrap text-muted">{{ month(week, i) }}</span>
          <div
            v-for="day in week"
            :key="day.key"
            class="size-3 rounded-sm"
            :class="day.future ? 'invisible' : level(day.count)"
            :title="t('stats.dayCount', { date: day.date.toLocaleDateString('fr-FR') }, day.count)"
          />
        </div>
      </div>
    </div>
    <div class="mt-2 flex flex-wrap items-center justify-between gap-2 text-xs text-muted">
      <span>{{ t('stats.activityTotal', total) }}</span>
      <span class="flex items-center gap-1">
        {{ t('stats.less') }}
        <span v-for="c in levels" :key="c" class="size-3 rounded-sm" :class="c" />
        {{ t('stats.more') }}
      </span>
    </div>
  </div>
</template>
