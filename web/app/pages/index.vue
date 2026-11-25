<script setup lang="ts">
import type { DropdownMenuItem, TableColumn } from '@nuxt/ui';
import type { Application, BatchResult, Stats, Status } from '~/types/items';
import { STATUSES } from '~/types/items';

const { t } = useI18n();
const api = useApi();
const sources = useSources();

const q = ref('');
const qDebounced = refDebounced(q, 300);
const statusFilter = ref<Status[]>([]);
const sourceFilter = ref<string>('all');
const defaultSorting = [{ id: 'applied_at', desc: true }];
const sorting = ref<{ id: string; desc: boolean }[]>([...defaultSorting]);
// On the kanban the page is the window's height (less the layout's header, h-14 and its border, and its padding,
// py-6): the board takes what is left and its columns scroll inside.
const view = useLocalStorage<'table' | 'kanban'>('callback-view', 'table');

const filtered = computed(
  () =>
    q.value !== '' ||
    statusFilter.value.length > 0 ||
    sourceFilter.value !== 'all' ||
    JSON.stringify(sorting.value) !== JSON.stringify(defaultSorting),
);
function reset() {
  q.value = '';
  statusFilter.value = [];
  sourceFilter.value = 'all';
  sorting.value = [...defaultSorting];
}

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
        status: statusFilter.value.length ? statusFilter.value.join(',') : undefined,
        source: sourceFilter.value === 'all' ? undefined : sourceFilter.value,
        sort: sort.value,
      },
    }),
  { watch: [qDebounced, statusFilter, sourceFilter, sort] },
);
const refresh = () => Promise.all([refreshList(), refreshStats(), sources.refresh()]);

// The statuses never take primary: it marks the actions, and the blue theme's info is close to it.
const statusColor: Record<Status, 'neutral' | 'info' | 'warning' | 'success' | 'error'> = {
  applied: 'neutral',
  responded: 'info',
  interview: 'warning',
  offer: 'success',
  rejected: 'error',
};

const UButton = resolveComponent('UButton');
const UDropdownMenu = resolveComponent('UDropdownMenu');

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

// A column header: its label sorts asc then desc, the funnel beside it filters.
const header =
  (label: string, filter?: { menu: DropdownMenuItem[][]; active: boolean }): TableColumn<Application>['header'] =>
  ({ column }) => {
    const dir = column.getIsSorted();
    const sort = h(UButton, {
      color: 'neutral',
      variant: 'ghost',
      label,
      class: '-mx-2.5',
      icon: dir
        ? dir === 'asc'
          ? 'i-lucide-arrow-up-narrow-wide'
          : 'i-lucide-arrow-down-wide-narrow'
        : 'i-lucide-arrow-up-down',
      onClick: () => column.toggleSorting(dir === 'asc'),
      'aria-label': `${t('table.sortBy')} ${label}`,
    });
    if (!filter) return sort;
    return h('div', { class: 'flex items-center gap-1' }, [
      sort,
      h(
        UDropdownMenu,
        { items: filter.menu, content: { align: 'start' } },
        {
          default: () =>
            h(UButton, {
              icon: 'i-lucide-list-filter',
              color: filter.active ? 'primary' : 'neutral',
              variant: filter.active ? 'soft' : 'ghost',
              size: 'xs',
              'aria-label': `${t('table.filterBy')} ${label}`,
            }),
        },
      ),
    ]);
  };

const columns = computed<TableColumn<Application>[]>(() => [
  { accessorKey: 'company', header: header(t('table.company')) },
  { accessorKey: 'location', header: t('table.location') },
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
  { id: 'actions', header: () => h('span', { class: 'sr-only' }, t('table.actions')) },
]);

// "2026-01-31" is a day, not an instant: no timezone shift.
const day = (v?: string) => (v ? new Date(`${v.slice(0, 10)}T00:00:00`).toLocaleDateString('fr-FR') : '');
const stamp = (v?: string) => (v ? new Date(v).toLocaleString('fr-FR') : '');

async function setStatus(app: Application, status: Status) {
  if (app.status === status) return;
  try {
    await api<BatchResult>('/api/applications', {
      method: 'PATCH',
      body: { ops: [{ ids: [app.id], set: { status } }] },
    });
    await refresh();
  } catch {
    // useApi already toasted.
  }
}

const statusMenu = (app: Application) =>
  STATUSES.map((s) => ({
    label: t(`status.${s}`),
    type: 'checkbox' as const,
    checked: app.status === s,
    onSelect: () => setStatus(app, s),
  }));

const slideover = ref(false);
const editing = ref<Application>();
function openForm(app?: Application) {
  editing.value = app;
  slideover.value = true;
}

const confirmOpen = ref(false);
const deleting = ref<Application>();
function askDelete(app: Application) {
  deleting.value = app;
  confirmOpen.value = true;
}
async function doDelete() {
  if (!deleting.value) return;
  try {
    await api<BatchResult>('/api/applications', { method: 'DELETE', body: { ids: [deleting.value.id] } });
    useToast().add({ title: t('delete.done'), color: 'success' });
    await refresh();
  } catch {
    // useApi already toasted.
  }
}
</script>

<template>
  <div class="flex flex-col gap-6" :class="view === 'kanban' ? 'h-[calc(100dvh-3.5rem-1px-3rem)]' : ''">
    <div class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
      <UCard v-for="s in STATUSES" :key="s">
        <p class="text-sm text-muted">{{ t(`status.${s}`) }}</p>
        <p class="text-2xl font-semibold">{{ stats?.by_status[s] ?? 0 }}</p>
      </UCard>
      <UCard>
        <p class="text-sm text-muted">{{ t('stats.responseRate') }}</p>
        <p class="text-2xl font-semibold">{{ Math.round((stats?.response_rate ?? 0) * 100) }} %</p>
      </UCard>
    </div>

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
        <UButton icon="i-lucide-plus" :label="t('form.add')" @click="openForm()" />
      </div>
    </div>

    <ApplicationKanban
      v-if="view === 'kanban'"
      :applications="applications ?? []"
      :status-color="statusColor"
      @move="setStatus"
      @edit="openForm"
    />
    <UTable
      v-else
      v-model:sorting="sorting"
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
        <UInputRating :model-value="row.original.rating ?? 0" readonly size="xs" />
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

    <ApplicationSlideover v-model:open="slideover" :application="editing" @saved="refresh" />
    <ConfirmModal
      v-model:open="confirmOpen"
      :title="t('delete.title')"
      :message="t('delete.message', { company: deleting?.company ?? '' })"
      :confirm-label="t('common.delete')"
      @confirm="doDelete"
    />
  </div>
</template>
