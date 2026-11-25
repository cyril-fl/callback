<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui';
import type { Application, BatchResult, Stats, Status } from '~/types/items';
import { SOURCES, STATUSES } from '~/types/items';

const { t } = useI18n();
const api = useApi();

const q = ref('');
const qDebounced = refDebounced(q, 300);
const statusFilter = ref<Status[]>([]);
const sourceFilter = ref<string>('all');
const sorting = ref<{ id: string; desc: boolean }[]>([{ id: 'applied_at', desc: true }]);

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
const refresh = () => Promise.all([refreshList(), refreshStats()]);

const statusItems = STATUSES.map((value) => ({ value, label: t(`status.${value}`) }));
const sourceItems = [
  { value: 'all', label: t('common.all') },
  ...SOURCES.map((value) => ({ value, label: t(`source.${value}`) })),
];
const statusColor: Record<Status, 'neutral' | 'info' | 'primary' | 'success' | 'error'> = {
  applied: 'neutral',
  responded: 'info',
  interview: 'primary',
  offer: 'success',
  rejected: 'error',
};

const UButton = resolveComponent('UButton');
const sortable =
  (id: string, label: string): TableColumn<Application>['header'] =>
  ({ column }) => {
    const dir = column.getIsSorted();
    return h(UButton, {
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
      'aria-label': label,
      id,
    });
  };

const columns = computed<TableColumn<Application>[]>(() => [
  { accessorKey: 'company', header: sortable('company', t('table.company')) },
  { accessorKey: 'location', header: t('table.location') },
  { id: 'source', header: t('table.source') },
  { accessorKey: 'status', header: sortable('status', t('table.status')) },
  { accessorKey: 'rating', header: sortable('rating', t('table.rating')) },
  { accessorKey: 'applied_at', header: sortable('applied_at', t('table.appliedAt')) },
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
  <div class="space-y-6">
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
      <USelectMenu
        v-model="statusFilter"
        multiple
        value-key="value"
        :items="statusItems"
        :placeholder="t('table.statusFilter')"
        :aria-label="t('table.statusFilter')"
        class="w-48"
      />
      <USelect v-model="sourceFilter" :items="sourceItems" :aria-label="t('table.sourceFilter')" class="w-40" />
      <UButton class="ml-auto" icon="i-lucide-plus" :label="t('form.add')" @click="openForm()" />
    </div>

    <UTable
      v-model:sorting="sorting"
      :data="applications ?? []"
      :columns="columns"
      :loading="loadStatus === 'pending'"
      :sorting-options="{ manualSorting: true }"
    >
      <template #empty>{{ t('common.empty') }}</template>
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
        <span v-else>{{ t(`source.${row.original.source ?? 'spontaneous'}`) }}</span>
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
        <span role="img" class="flex" :aria-label="t('table.stars', { n: row.original.rating ?? 0 })">
          <UIcon
            v-for="n in 5"
            :key="n"
            name="i-lucide-star"
            :class="n <= (row.original.rating ?? 0) ? 'text-warning' : 'text-muted/40'"
          />
        </span>
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
