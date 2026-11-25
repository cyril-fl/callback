<script setup lang="ts">
import type { DropdownMenuItem, TableColumn } from '@nuxt/ui';
import type { Application, BatchResult, Stats, Status } from '~/types/items';
import { STATUSES } from '~/types/items';
import { today as todayDate, getLocalTimeZone } from '@internationalized/date';

const { t } = useI18n();
const api = useApi();
const sources = useSources();
const places = usePlaces();
const { pro, preferences } = usePreferences();

const q = ref('');
const qDebounced = refDebounced(q, 300);
const statusFilter = ref<Status[]>([]);
const sourceFilter = ref<string>('all');
const locationFilter = ref<string>();
// Pro: the applications without news for the follow-up delay, applied or responded.
const followUp = ref(false);
const defaultSorting = [{ id: 'applied_at', desc: true }];
const sorting = ref<{ id: string; desc: boolean }[]>([...defaultSorting]);
// On the kanban the page is the window's height (less the layout's header, h-14 and its border, and its padding,
// py-6): the board takes what is left and its columns scroll inside.
const view = useLocalStorage<'table' | 'kanban'>('callback-view', 'table');

const filtered = computed(
  () =>
    q.value !== '' ||
    followUp.value ||
    statusFilter.value.length > 0 ||
    sourceFilter.value !== 'all' ||
    !!locationFilter.value ||
    JSON.stringify(sorting.value) !== JSON.stringify(defaultSorting),
);
function reset() {
  q.value = '';
  followUp.value = false;
  statusFilter.value = [];
  sourceFilter.value = 'all';
  locationFilter.value = undefined;
  sorting.value = [...defaultSorting];
}

// The day the follow-up delay started: an application without news (status, follow-up) since then is to follow up.
const followUpSince = computed(() => {
  const since = todayDate(getLocalTimeZone()).subtract({ days: preferences.value?.follow_up_days ?? 7 });
  return since.toString();
});

const sort = computed(() => {
  const s = sorting.value[0];
  return s ? (s.desc ? '-' : '') + s.id : undefined;
});

const { data: stats, refresh: refreshStats } = await useAsyncData('stats', () => api<Stats>('/api/stats'));
const {
  data: applications,
  status: loadStatus,
  refresh: refreshList,
} = await useAsyncData(
  'applications',
  () =>
    api<Application[]>('/api/applications', {
      query: {
        q: qDebounced.value || undefined,
        status: followUp.value
          ? 'applied,responded'
          : statusFilter.value.length
            ? statusFilter.value.join(',')
            : undefined,
        untouched_since: followUp.value ? followUpSince.value : undefined,
        source: sourceFilter.value === 'all' ? undefined : sourceFilter.value,
        location: locationFilter.value,
        sort: sort.value,
      },
    }),
  { watch: [qDebounced, followUp, statusFilter, sourceFilter, locationFilter, sort] },
);
const refresh = () => Promise.all([refreshList(), refreshStats(), sources.refresh(), places.refresh()]);

// The statuses never take primary: it marks the actions, and the blue theme's info is close to it.
const statusColor: Record<Status, 'neutral' | 'info' | 'warning' | 'success' | 'error'> = {
  applied: 'neutral',
  responded: 'info',
  interview: 'warning',
  offer: 'success',
  rejected: 'error',
};

const UButton = resolveComponent('UButton');
const UCheckbox = resolveComponent('UCheckbox');
const UDropdownMenu = resolveComponent('UDropdownMenu');
const USelectMenu = resolveComponent('USelectMenu');

// The menu stays open while ticking: preventDefault keeps it from closing on select.
const keepOpen = (fn: () => void) => (e: Event) => {
  e.preventDefault();
  fn();
};

const statusFilterMenu = computed<DropdownMenuItem[][]>(() => [
  [
    {
      label: t('common.all'),
      type: 'checkbox',
      checked: statusFilter.value.length === 0,
      onSelect: keepOpen(() => (statusFilter.value = [])),
    },
  ],
  STATUSES.map((s) => ({
    label: t(`status.${s}`),
    type: 'checkbox' as const,
    checked: statusFilter.value.includes(s),
    onSelect: keepOpen(() => {
      statusFilter.value = statusFilter.value.includes(s)
        ? statusFilter.value.filter((x) => x !== s)
        : [...statusFilter.value, s];
    }),
  })),
]);

const sourceFilterMenu = computed<DropdownMenuItem[][]>(() => [
  [{ value: 'all', label: t('common.all') }, ...sources.items.value].map(({ value: s, label }) => ({
    label,
    type: 'checkbox' as const,
    checked: sourceFilter.value === s,
    onSelect: () => (sourceFilter.value = s),
  })),
]);

// A column header: the arrow beside the label sorts asc then desc; the label opens the filter, a menu for the closed
// values (status, source), a search for the open ones (city).
type Filter = { active: boolean } & (
  { menu: DropdownMenuItem[][] } | { search: { items: string[]; value?: string; set: (v?: string) => void } }
);
const header =
  (label: string, filter?: Filter): TableColumn<Application>['header'] =>
  ({ column }) => {
    const dir = column.getIsSorted();
    const sort = h(UButton, {
      color: dir ? 'primary' : 'neutral',
      variant: 'ghost',
      size: 'xs',
      icon: dir
        ? dir === 'asc'
          ? 'i-lucide-arrow-up-narrow-wide'
          : 'i-lucide-arrow-down-wide-narrow'
        : 'i-lucide-arrow-up-down',
      onClick: () => column.toggleSorting(dir === 'asc'),
      'aria-label': `${t('table.sortBy')} ${label}`,
    });
    const trigger = {
      label,
      color: filter?.active ? 'primary' : 'neutral',
      variant: filter?.active ? 'soft' : 'ghost',
      trailingIcon: 'i-lucide-chevron-down',
    } as const;
    const title = !filter
      ? h('span', { class: 'font-medium' }, label)
      : 'menu' in filter
        ? h(UDropdownMenu, { items: filter.menu, content: { align: 'start' } }, { default: () => h(UButton, trigger) })
        : h(
            USelectMenu,
            {
              ...trigger,
              items: [
                { label: t('common.all'), value: 'all' },
                ...filter.search.items.map((v) => ({ label: v, value: v })),
              ],
              valueKey: 'value',
              modelValue: filter.search.value ?? 'all',
              'onUpdate:modelValue': (v: string) => filter.search.set(v === 'all' ? undefined : v),
              searchInput: { placeholder: t('table.search') },
              content: { align: 'start' },
              class: ['w-auto', filter.active && 'bg-primary/10 text-primary'],
            },
            { default: () => label },
          );
    return h('div', { class: 'flex items-center gap-0.5 -ml-2' }, [sort, title]);
  };

// The follow-up column filters on its button: all rows, or those that show it.
const followUpFilterMenu = computed<DropdownMenuItem[][]>(() => [
  [
    { label: t('common.all'), type: 'checkbox', checked: !followUp.value, onSelect: () => (followUp.value = false) },
    {
      label: t('table.followUp'),
      icon: 'i-lucide-bell-ring',
      type: 'checkbox',
      checked: followUp.value,
      onSelect: () => (followUp.value = true),
    },
  ],
]);
// Pro: rows are picked by id for the batch bar.
const selection = ref<Record<string, boolean>>({});
const selected = computed(() => Object.keys(selection.value).filter((id) => selection.value[id]));
watch(applications, () => (selection.value = {}));
const selectColumn: TableColumn<Application> = {
  id: 'select',
  header: ({ table }) =>
    h(UCheckbox, {
      modelValue: table.getIsSomePageRowsSelected() ? 'indeterminate' : table.getIsAllPageRowsSelected(),
      'onUpdate:modelValue': (v: boolean | 'indeterminate') => table.toggleAllPageRowsSelected(!!v),
      'aria-label': t('table.selectAll'),
    }),
  cell: ({ row }) =>
    h(UCheckbox, {
      modelValue: row.getIsSelected(),
      'onUpdate:modelValue': (v: boolean | 'indeterminate') => row.toggleSelected(!!v),
      'aria-label': t('table.select'),
    }),
};

const columns = computed<TableColumn<Application>[]>(() => [
  ...(pro.value ? [selectColumn] : []),
  { accessorKey: 'company', header: header(t('table.company')) },
  {
    accessorKey: 'location',
    header: header(t('table.location'), {
      search: { items: places.cities.value, value: locationFilter.value, set: (v) => (locationFilter.value = v) },
      active: !!locationFilter.value,
    }),
  },
  {
    accessorKey: 'source',
    header: header(t('table.source'), { menu: sourceFilterMenu.value, active: sourceFilter.value !== 'all' }),
  },
  {
    accessorKey: 'status',
    header: header(t('table.status'), { menu: statusFilterMenu.value, active: statusFilter.value.length > 0 }),
  },
  { accessorKey: 'rating', header: header(t('table.rating')) },
  { accessorKey: 'applied_at', header: header(t('table.appliedAt')) },
  {
    accessorKey: 'followed_up_at',
    header: header(t('table.followUp'), { menu: followUpFilterMenu.value, active: followUp.value }),
  },
  { id: 'actions', header: () => h('span', { class: 'sr-only' }, t('table.actions')) },
]);

// "2026-01-31" is a day, not an instant: no timezone shift.
const day = (v?: string) => (v ? new Date(`${v.slice(0, 10)}T00:00:00`).toLocaleDateString('fr-FR') : '');
const stamp = (v?: string) => (v ? new Date(v).toLocaleString('fr-FR') : '');

// Optimistic: the rows change at once; the refresh then brings the server's truth, or rolls back on a failure.
async function write(ids: string[], local: Partial<Application> | 'delete', request: () => Promise<unknown>) {
  applications.value =
    local === 'delete'
      ? applications.value?.filter((a) => !ids.includes(a.id))
      : applications.value?.map((a) => (ids.includes(a.id) ? { ...a, ...local } : a));
  try {
    await request();
    return true;
  } catch {
    return false; // useApi already toasted.
  } finally {
    await refresh();
  }
}
const patch = (ids: string[], set: Record<string, unknown>) => () =>
  api<BatchResult>('/api/applications', { method: 'PATCH', body: { ops: [{ ids, set }] } });

async function setStatus(app: Application, status: Status) {
  if (app.status !== status) await write([app.id], { status }, patch([app.id], { status }));
}

async function markFollowedUp(app: Application) {
  if (await write([app.id], { followed_up_at: new Date().toISOString() }, patch([app.id], { followed_up: true })))
    useToast().add({ title: t('table.followedUp', { company: app.company }), color: 'success' });
}

const statusMenu = (app: Application) =>
  STATUSES.map((s) => ({
    label: t(`status.${s}`),
    type: 'checkbox' as const,
    checked: app.status === s,
    onSelect: () => setStatus(app, s),
  }));

// The form wrote it already: show it now, then refresh.
function saved(item: Application) {
  const list = applications.value ?? [];
  applications.value = list.some((a) => a.id === item.id)
    ? list.map((a) => (a.id === item.id ? { ...a, ...item } : a))
    : [item, ...list];
  refresh();
}

const slideover = ref(false);
const editing = ref<Application>();
function openForm(app?: Application) {
  editing.value = app;
  slideover.value = true;
}

// One application from its row, or the selected ones from the batch bar.
const confirmOpen = ref(false);
const deleting = ref<{ ids: string[]; message: string }>();
function askDelete(app: Application) {
  deleting.value = { ids: [app.id], message: t('delete.message', { company: app.company ?? '' }) };
  confirmOpen.value = true;
}
function askDeleteSelected() {
  deleting.value = { ids: selected.value, message: t('delete.messageMany', selected.value.length) };
  confirmOpen.value = true;
}
async function doDelete() {
  if (!deleting.value) return;
  const { ids } = deleting.value;
  if (await write(ids, 'delete', () => api<BatchResult>('/api/applications', { method: 'DELETE', body: { ids } })))
    useToast().add({ title: t('delete.done'), color: 'success' });
}

async function setSelectedStatus(status: Status) {
  await write(selected.value, { status }, patch(selected.value, { status }));
}
const selectedStatusMenu = computed(() =>
  STATUSES.map((s) => ({ label: t(`status.${s}`), onSelect: () => setSelectedStatus(s) })),
);

// The list as shown, filters and sort included, to a CSV file.
const csvColumns = [
  'company',
  'location',
  'street',
  'postal_code',
  'country',
  'website',
  'source',
  'source_detail',
  'status',
  'rating',
  'comment',
  'applied_at',
  'interview_at',
  'contact_name',
  'contact_email',
  'created_at',
  'updated_at',
] as const;
function exportCsv() {
  const csv = toCsv(csvColumns, applications.value ?? []);
  const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' }));
  const link = Object.assign(document.createElement('a'), {
    href: url,
    download: `candidatures-${todayDate(getLocalTimeZone())}.csv`,
  });
  link.click();
  URL.revokeObjectURL(url);
}
</script>

<template>
  <div
    class="flex flex-col gap-6"
    :class="view === 'kanban' ? 'h-[calc(100dvh-3.5rem-1px-3rem)]' : selected.length ? 'pb-16' : ''"
  >
    <StatsCards v-if="!pro" :stats="stats" />

    <div class="flex flex-wrap items-center gap-2">
      <UInput v-model="q" icon="i-lucide-search" :placeholder="t('table.search')" :aria-label="t('table.search')" />
      <UButton
        v-if="filtered"
        icon="i-lucide-rotate-ccw"
        color="neutral"
        variant="ghost"
        :label="t('table.reset')"
        @click="reset"
      />
      <div class="ml-auto flex items-center gap-2">
        <UFieldGroup>
          <UButton
            icon="i-lucide-table"
            :color="view === 'table' ? 'primary' : 'neutral'"
            :variant="view === 'table' ? 'soft' : 'outline'"
            :aria-label="t('view.table')"
            :aria-pressed="view === 'table'"
            @click="view = 'table'"
          />
          <UButton
            icon="i-lucide-columns-3"
            :color="view === 'kanban' ? 'primary' : 'neutral'"
            :variant="view === 'kanban' ? 'soft' : 'outline'"
            :aria-label="t('view.kanban')"
            :aria-pressed="view === 'kanban'"
            @click="view = 'kanban'"
          />
        </UFieldGroup>
        <UButton
          v-if="pro"
          icon="i-lucide-download"
          color="neutral"
          variant="outline"
          :aria-label="t('table.export')"
          :disabled="!applications?.length"
          @click="exportCsv"
        />
        <UButton icon="i-lucide-plus" :label="t('common.add')" @click="openForm()" />
      </div>
    </div>

    <ApplicationKanban
      v-if="view === 'kanban'"
      :applications="applications ?? []"
      :status-color="statusColor"
      :follow-up-since="followUpSince"
      @move="setStatus"
      @follow-up="markFollowedUp"
      @edit="openForm"
    />
    <UTable
      v-if="view === 'table'"
      v-model:sorting="sorting"
      v-model:row-selection="selection"
      :get-row-id="(row: Application) => row.id"
      :data="applications ?? []"
      :columns="columns"
      :loading="loadStatus === 'pending'"
      :sorting-options="{ manualSorting: true }"
    >
      <template #empty>{{ t('common.empty') }}</template>
      <template #company-cell="{ row }">
        <a
          v-if="row.original.website"
          :href="
            row.original.website.includes('@') && !row.original.website.includes('://')
              ? `mailto:${row.original.website}`
              : row.original.website
          "
          target="_blank"
          rel="noopener noreferrer"
          class="font-medium text-primary hover:underline"
        >
          {{ row.original.company }}
        </a>
        <span v-else class="font-medium">{{ row.original.company }}</span>
      </template>
      <template #source-cell="{ row }">
        <a
          v-if="row.original.source === 'job_post' && row.original.source_detail"
          :href="row.original.source_detail"
          target="_blank"
          rel="noopener noreferrer"
          class="text-primary hover:underline"
        >
          {{ t('source.job_post') }}
        </a>
        <a
          v-else-if="row.original.source === 'recruiter' && row.original.source_detail"
          :href="`mailto:${row.original.source_detail}`"
          class="text-primary hover:underline"
        >
          {{ t('source.recruiter') }}
        </a>
        <span v-else>{{ sources.label(row.original.source) }}</span>
      </template>
      <template #status-cell="{ row }">
        <UDropdownMenu :items="statusMenu(row.original)">
          <UBadge
            as="button"
            :color="statusColor[row.original.status ?? 'applied']"
            variant="subtle"
            class="cursor-pointer"
            :aria-label="`${t('table.changeStatus')} : ${t(`status.${row.original.status ?? 'applied'}`)}`"
            :label="t(`status.${row.original.status ?? 'applied'}`)"
            trailing-icon="i-lucide-chevron-down"
          />
        </UDropdownMenu>
      </template>
      <template #rating-cell="{ row }">
        <UInputRating
          icon="i-callback-star-filled"
          empty-icon="i-lucide-star"
          :model-value="row.original.rating ?? 0"
          readonly
          size="xs"
        />
      </template>
      <template #location-cell="{ row }">
        {{ [row.original.location, row.original.country].filter(Boolean).join(', ') }}
      </template>
      <template #applied_at-cell="{ row }">
        <UTooltip
          :text="`${t('table.created', { date: stamp(row.original.created_at) })} · ${t('table.updated', { date: stamp(row.original.updated_at) })}`"
        >
          <span>{{ day(row.original.applied_at) }}</span>
        </UTooltip>
      </template>
      <template #followed_up_at-cell="{ row }">
        <UTooltip
          v-if="canFollowUp(row.original, followUpSince)"
          :text="
            row.original.followed_up_at
              ? t('table.followedUpOn', { date: day(row.original.followed_up_at) })
              : t('table.markFollowedUp')
          "
        >
          <UButton
            icon="i-lucide-bell-ring"
            color="neutral"
            variant="ghost"
            :aria-label="t('table.markFollowedUp')"
            @click="markFollowedUp(row.original)"
          />
        </UTooltip>
        <span v-else-if="row.original.followed_up_at">{{ day(row.original.followed_up_at) }}</span>
        <span v-else />
      </template>
      <template #actions-cell="{ row }">
        <div class="flex justify-end gap-1">
          <UButton
            icon="i-lucide-pencil"
            color="neutral"
            variant="ghost"
            :aria-label="t('common.edit')"
            @click="openForm(row.original)"
          />
          <UButton
            icon="i-lucide-trash-2"
            color="error"
            variant="ghost"
            :aria-label="t('common.delete')"
            @click="askDelete(row.original)"
          />
        </div>
      </template>
    </UTable>
    <!-- Fixed to the viewport so a short table does not lift it; the page's pb-16 keeps the last rows visible. -->
    <UContainer v-if="view === 'table' && selected.length" class="fixed inset-x-0 bottom-4 z-10">
      <div class="flex flex-wrap items-center gap-2 rounded-lg border border-default bg-elevated px-3 py-2 shadow-lg">
        <span class="text-sm">{{ t('table.selected', selected.length) }}</span>
        <UDropdownMenu :items="selectedStatusMenu">
          <UButton
            color="neutral"
            variant="outline"
            size="sm"
            trailing-icon="i-lucide-chevron-down"
            :label="t('table.batchStatus')"
          />
        </UDropdownMenu>
        <UButton
          color="error"
          variant="soft"
          size="sm"
          icon="i-lucide-trash-2"
          :label="t('common.delete')"
          @click="askDeleteSelected"
        />
        <UButton
          color="neutral"
          variant="ghost"
          size="sm"
          class="ml-auto"
          :label="t('common.cancel')"
          @click="selection = {}"
        />
      </div>
    </UContainer>

    <ApplicationSlideover v-model:open="slideover" :application="editing" @saved="saved" />
    <ConfirmModal
      v-model:open="confirmOpen"
      :title="t('delete.title')"
      :message="deleting?.message ?? ''"
      :confirm-label="t('common.delete')"
      @confirm="doDelete"
    />
  </div>
</template>
