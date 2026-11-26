<script setup lang="ts">
import { getLocalTimeZone, today as todayDate, type DateValue } from '@internationalized/date';
import type { TableColumn, TableRow } from '@nuxt/ui';
import type { VNode } from 'vue';
import type { Application, Contact, StatusChange } from '~/types/items';

// The pro mode's page: in base the stats stay above the list.
definePageMeta({ middleware: 'pro' });

const { t } = useI18n();
const api = useApi();
const sources = useSources();
const countries = useCountries();
const { preferences } = usePreferences();
const { icons } = useIcons();
const ColumnHeader = resolveComponent('ColumnHeader');
const DayFilter = resolveComponent('DayFilter');

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
const periods = { '1w': { weeks: 1 }, '1m': { months: 1 }, '3m': { months: 3 }, all: undefined } as const;
type Period = keyof typeof periods;
const period = ref<Period>('1m');
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
  // The day after: a week is the last 7 days, today included.
  return duration ? today.subtract(duration).add({ days: 1 }).toString() : '';
});
const filtered = computed(() => inPeriod(applications.value, since.value, city.value));
const stats = computed(() => statsOf(filtered.value));
const days = computed(() => filtered.value.flatMap((a) => (a.applied_at ? [a.applied_at] : [])));
const cities = computed(() =>
  countBy(
    filtered.value.flatMap((a) =>
      a.location ? [[a.location, countries.label(a.country)].filter(Boolean).join(', ')] : [],
    ),
  ),
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
    history.value,
    today.subtract({ days: preferences.value?.follow_up_days ?? 7 }).toDate(getLocalTimeZone()),
  ),
);
// The interviews follow the city but not the period (it counts applications); their headers filter them, the day
// from today on by default.
const dayMode = ref<DayMode>('after');
const interviewDay = shallowRef(today);
const interviewRange = shallowRef<{ start: DateValue | undefined; end: DateValue | undefined }>({
  start: today,
  end: today.add({ weeks: 1 }),
});
const companyQuery = ref('');
const addressQuery = ref('');
const contactFilter = ref<string>();
const contactName = (c: Contact) => c.name || c.email || '';
const has = (v: string, q: string) => v.toLowerCase().includes(q.trim().toLowerCase());
const cityInterviews = computed(() =>
  interviewsOf(inPeriod(applications.value, '', city.value)).map((r) => ({
    ...r,
    company: r.application.company ?? '',
    address: addressOf(r.application, countries.label(r.application.country)),
    contacts: r.application.contacts ?? [],
  })),
);
const contactNames = computed(() =>
  [...new Set(cityInterviews.value.flatMap((r) => r.contacts.map(contactName)))].filter(Boolean).sort(),
);
// The headers' filters but the day's: the calendar marks these days, the day filter then narrows the table.
const matching = computed(() =>
  cityInterviews.value.filter(
    (r) =>
      has(r.company, companyQuery.value) &&
      has(r.address, addressQuery.value) &&
      (!contactFilter.value || r.contacts.some((c) => contactName(c) === contactFilter.value)),
  ),
);
const interviewDays = computed(() => new Set(matching.value.map((r) => r.at.slice(0, 10))));
const interviews = computed(() => {
  const [from, to] = dayBounds(dayMode.value, interviewDay.value?.toString(), {
    start: interviewRange.value?.start?.toString(),
    end: interviewRange.value?.end?.toString(),
  });
  return matching.value.filter((r) => (!from || r.at.slice(0, 10) >= from) && (!to || r.at.slice(0, 10) <= to));
});
// A day picked on the calendar is the day filter's « on »; picked again (the calendar unselects it), the default.
const calendarDay = computed({
  get: () => (dayMode.value === 'on' ? interviewDay.value : undefined),
  set: (v) => {
    dayMode.value = v ? 'on' : 'after';
    interviewDay.value = v ?? today;
  },
});
type InterviewRow = (typeof interviews.value)[number];
const interviewSorting = ref([{ id: 'at', desc: false }]);
// Built once, as in the list: a new header function would close the open filter at every pick.
const header =
  (label: string, filter: () => Record<string, unknown>, slot?: () => VNode): TableColumn<InterviewRow>['header'] =>
  ({ column }) =>
    h(
      ColumnHeader,
      {
        label: t(label),
        sorted: column.getIsSorted(),
        onSort: () => column.toggleSorting(column.getIsSorted() === 'asc'),
        ...filter(),
      },
      slot && { default: slot },
    );
const search = (query: Ref<string>) => () => ({
  search: true,
  modelValue: query.value,
  'onUpdate:modelValue': (v?: string) => (query.value = v ?? ''),
  active: query.value !== '',
});
const interviewColumns: TableColumn<InterviewRow>[] = [
  {
    accessorKey: 'at',
    header: header(
      'form.at',
      () => ({ active: dayMode.value !== 'all' }),
      () =>
        h(DayFilter, {
          mode: dayMode.value,
          'onUpdate:mode': (v: DayMode) => (dayMode.value = v),
          day: interviewDay.value,
          'onUpdate:day': (v: typeof today) => (interviewDay.value = v),
          range: interviewRange.value,
          'onUpdate:range': (v: typeof interviewRange.value) => (interviewRange.value = v),
        }),
    ),
  },
  { accessorKey: 'company', header: header('form.company', search(companyQuery)) },
  { accessorKey: 'address', header: header('form.street', search(addressQuery)) },
  {
    id: 'contacts',
    accessorFn: (r) => r.contacts.map(contactName).join(', '),
    header: header('form.contacts', () => ({
      options: contactNames.value,
      modelValue: contactFilter.value,
      'onUpdate:modelValue': (v?: string) => (contactFilter.value = v),
      active: !!contactFilter.value,
    })),
  },
];

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
    <Teleport defer to="#header-actions">
      <UTabs v-model="period" :items="periodItems" :content="false" size="sm" :aria-label="t('stats.period')" />
      <USelect v-model="city" :items="cityItems" :aria-label="t('form.city')" class="w-44" />
    </Teleport>

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

    <section>
      <h2 class="mb-3 text-sm font-semibold">{{ t('stats.interviews') }}</h2>
      <div class="grid items-start gap-4 xl:grid-cols-[1fr_auto]">
        <UCalendar v-model="calendarDay" class="justify-self-center rounded-lg border border-default p-2 xl:order-last">
          <template #day="{ day: d }">
            <UChip :show="interviewDays.has(d.toString())" size="2xs">{{ d.day }}</UChip>
          </template>
        </UCalendar>
        <UTable
          v-model:sorting="interviewSorting"
          :data="interviews"
          :columns="interviewColumns"
          class="min-w-0 rounded-lg border border-default"
          :ui="{ tr: 'data-[selectable=true]:cursor-pointer' }"
          @select="(_: Event, row: TableRow<InterviewRow>) => openForm(row.original.application)"
        >
          <template #empty><UEmpty :icon="icons.empty" :title="t('common.empty')" variant="naked" /></template>
          <template #at-cell="{ row }">{{ stamp(row.original.at) }}</template>
          <template #contacts-cell="{ row }">
            <UAvatarGroup v-if="row.original.contacts.length" size="xs">
              <UAvatar
                v-for="(c, i) in row.original.contacts"
                :key="i"
                :alt="c.name || c.email"
                :title="[c.name || c.email, c.role].filter(Boolean).join(' · ')"
              />
            </UAvatarGroup>
          </template>
        </UTable>
      </div>
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
              <span class="text-xs text-muted">{{ t('stats.untouchedSince', { date: day(a.news) }) }}</span>
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
