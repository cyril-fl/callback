<script setup lang="ts">
import type { DropdownMenuItem, TableColumn } from '@nuxt/ui';
import type { Application, BatchResult, Stats, Status } from '~/types/items';
import { STATUSES } from '~/types/items';
import { today as todayDate, getLocalTimeZone } from '@internationalized/date';

const { t } = useI18n();
const { icons } = useIcons();
const api = useApi();
const sources = useSources();
const places = usePlaces();
const { pro, preferences } = usePreferences();
const { shortcuts, key } = useShortcuts();
const { record, undo, redo } = useHistory();
const toast = useToast();

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
          ? FOLLOW_UP_STATUSES.join(',')
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
  no_reply: 'neutral',
};

const UButton = resolveComponent('UButton');
const UCheckbox = resolveComponent('UCheckbox');
const ColumnHeader = resolveComponent('ColumnHeader');

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
    ...STATUSES.map((s) => ({
      label: t(`status.${s}`),
      type: 'checkbox' as const,
      checked: statusFilter.value.includes(s),
      onSelect: keepOpen(() => {
        statusFilter.value = statusFilter.value.includes(s)
          ? statusFilter.value.filter((x) => x !== s)
          : [...statusFilter.value, s];
      }),
    })),
  ],
]);

const sourceFilterMenu = computed<DropdownMenuItem[][]>(() => [
  [{ value: 'all', label: t('common.all') }, ...sources.items.value].map(({ value: s, label }) => ({
    label,
    type: 'checkbox' as const,
    checked: sourceFilter.value === s,
    onSelect: () => (sourceFilter.value = s),
  })),
]);

// The follow-up column filters on its button: all rows, or those that show it.
const followUpFilterMenu = computed<DropdownMenuItem[][]>(() => [
  [
    { label: t('common.all'), type: 'checkbox', checked: !followUp.value, onSelect: () => (followUp.value = false) },
    { label: t('table.followUp'), type: 'checkbox', checked: followUp.value, onSelect: () => (followUp.value = true) },
  ],
]);
// The company search, opened by its shortcut: the table's only.
const searchOpen = ref(false);
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

// A header: label and sort from TanStack, the filter's props read here so the header follows them.
// The columns are built once: TanStack renders a header function as a component, a new function is a new component
// and would close the open menu at every pick.
const header =
  (label: string, filter?: () => Record<string, unknown>): TableColumn<Application>['header'] =>
  ({ column }) =>
    h(ColumnHeader, {
      label: t(label),
      sorted: column.getIsSorted(),
      onSort: () => column.toggleSorting(column.getIsSorted() === 'asc'),
      ...filter?.(),
    });
const dataColumns: TableColumn<Application>[] = [
  {
    accessorKey: 'company',
    header: header('table.company', () => ({
      search: true,
      open: searchOpen.value,
      'onUpdate:open': (v: boolean) => (searchOpen.value = v),
      modelValue: q.value,
      'onUpdate:modelValue': (v?: string) => (q.value = v ?? ''),
      active: q.value !== '',
    })),
  },
  {
    accessorKey: 'location',
    header: header('table.location', () => ({
      options: places.cities.value,
      modelValue: locationFilter.value,
      'onUpdate:modelValue': (v?: string) => (locationFilter.value = v),
      active: !!locationFilter.value,
    })),
  },
  {
    accessorKey: 'source',
    header: header('table.source', () => ({ menu: sourceFilterMenu.value, active: sourceFilter.value !== 'all' })),
  },
  {
    accessorKey: 'status',
    header: header('table.status', () => ({ menu: statusFilterMenu.value, active: statusFilter.value.length > 0 })),
  },
  { accessorKey: 'rating', header: header('table.rating') },
  { accessorKey: 'applied_at', header: header('table.appliedAt') },
  {
    accessorKey: 'followed_up_at',
    header: header('table.followUp', () => ({ menu: followUpFilterMenu.value, active: followUp.value })),
    // An empty slot (no bell) falls back to this cell: never the raw timestamp.
    cell: () => '',
  },
  {
    id: 'actions',
    // Always there, hidden when nothing is filtered: the columns do not move when it shows.
    header: () =>
      h('div', { class: 'flex justify-end' }, [
        h('span', { class: 'sr-only' }, t('table.actions')),
        h(UButton, {
          icon: icons.reset,
          color: 'neutral',
          variant: 'ghost',
          class: !filtered.value && 'invisible',
          'aria-label': t('table.reset'),
          title: t('table.reset'),
          onClick: reset,
        }),
      ]),
  },
];
const columns = computed(() => (pro.value ? [selectColumn, ...dataColumns] : dataColumns));

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
const remove = (ids: string[]) => () => api<BatchResult>('/api/applications', { method: 'DELETE', body: { ids } });
const restore = (ids: string[]) => () =>
  api<BatchResult>('/api/applications/restore', { method: 'POST', body: { ids } });

// An undo or a redo, then the list read again. A batch's failures are told here: useApi only tells a refused request.
async function replay(request: () => Promise<BatchResult>) {
  let result: BatchResult | undefined;
  await write([], {}, async () => (result = await request()));
  const failure = result?.failed?.[0];
  if (failure) toast.add({ title: failure.error, color: 'error' });
  return !!result && !failure;
}
// The API takes a status back with its history: the undone step is not counted as reached.
const recordStatus = (ids: string[], status: Status) =>
  record({ undo: () => replay(patch(ids, { undo_status: true })), redo: () => replay(patch(ids, { status })) });

async function setStatus(app: Application, status: Status) {
  if (app.status === status) return;
  const ids = [app.id];
  if (await write(ids, { status }, patch(ids, { status }))) recordStatus(ids, status);
}

async function markFollowedUp(app: Application) {
  const ids = [app.id];
  if (await write(ids, { followed_up_at: new Date().toISOString() }, patch(ids, { followed_up: true }))) {
    record({
      undo: () => replay(patch(ids, { undo_follow_up: true })),
      redo: () => replay(patch(ids, { followed_up: true })),
    });
    toast.add({ title: t('table.followedUp', { company: app.company }), color: 'success' });
  }
}

const statusMenu = (app: Application) =>
  STATUSES.map((s) => ({
    label: t(`status.${s}`),
    type: 'checkbox' as const,
    checked: app.status === s,
    onSelect: () => setStatus(app, s),
  }));

// The form wrote it already: show it now, then refresh. A creation can be undone, not an edit.
function saved(item: Application) {
  const list = applications.value ?? [];
  const created = !list.some((a) => a.id === item.id);
  applications.value = created ? [item, ...list] : list.map((a) => (a.id === item.id ? { ...a, ...item } : a));
  if (created) record({ undo: () => replay(remove([item.id])), redo: () => replay(restore([item.id])) });
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
  if (await write(ids, 'delete', remove(ids))) {
    record({ undo: () => replay(restore(ids)), redo: () => replay(remove(ids)) });
    toast.add({
      title: t('delete.done'),
      color: 'success',
      actions: [{ label: t('common.undo'), color: 'neutral', variant: 'outline', onClick: () => undo() }],
    });
  }
}

// Only the ones whose status changes: an undo takes back their last change, the others have none of this batch.
async function setSelectedStatus(status: Status) {
  const ids = (applications.value ?? [])
    .filter((a) => selected.value.includes(a.id) && a.status !== status)
    .map((a) => a.id);
  if (ids.length && (await write(ids, { status }, patch(ids, { status })))) recordStatus(ids, status);
}
const selectedStatusMenu = computed(() =>
  STATUSES.map((s) => ({ label: t(`status.${s}`), onSelect: () => setSelectedStatus(s) })),
);

// Not in an open menu: its keys are its own.
const outsideMenu = (handler: () => unknown) => () => document.activeElement?.closest('[role="menu"]') || handler();
defineShortcuts({
  [key('search')]: outsideMenu(() => view.value === 'table' && (searchOpen.value = true)),
  [key('add')]: outsideMenu(() => openForm()),
  [key('selectAll')]: outsideMenu(
    () =>
      pro.value &&
      view.value === 'table' &&
      (selection.value = Object.fromEntries((applications.value ?? []).map((a) => [a.id, true]))),
  ),
  // Outside a field, which undoes its own text.
  [key('undo')]: outsideMenu(undo),
  [key('redo')]: outsideMenu(redo),
  [key('redoShift')]: outsideMenu(redo),
});

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

    <!-- The table resets from its header; the kanban has none. -->
    <div v-if="filtered && view === 'kanban'">
      <UButton :icon="icons.reset" color="neutral" variant="ghost" :label="t('table.reset')" @click="reset" />
    </div>
    <Teleport defer to="#header-actions">
      <div class="flex items-center gap-2">
        <!-- Clicking the pressed view would empty the group: it is ignored. -->
        <UToggleGroup
          :model-value="view"
          :items="[
            { value: 'table', icon: icons.table, 'aria-label': t('view.table') },
            { value: 'kanban', icon: icons.kanban, 'aria-label': t('view.kanban') },
          ]"
          color="neutral"
          variant="outline"
          attached
          @update:model-value="(v) => v && (view = v as 'table' | 'kanban')"
        />
        <UButton
          v-if="pro"
          :icon="icons.export"
          color="neutral"
          variant="outline"
          :aria-label="t('table.export')"
          :disabled="!applications?.length"
          @click="exportCsv"
        />
        <UTooltip :text="t('common.add')" :kbds="shortcuts.add">
          <UButton :icon="icons.add" :label="t('common.add')" :ui="{ label: 'max-sm:sr-only' }" @click="openForm()" />
        </UTooltip>
      </div>
    </Teleport>

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
      <template #empty><UEmpty :icon="icons.empty" :title="t('common.empty')" variant="naked" /></template>
      <template #company-cell="{ row }">
        <a
          v-if="row.original.website"
          :href="websiteHref(row.original.website)"
          target="_blank"
          rel="noopener noreferrer"
          class="font-medium text-primary hover:underline"
        >
          {{ row.original.company }}
        </a>
        <span v-else class="font-medium">{{ row.original.company }}</span>
      </template>
      <template #source-cell="{ row }">
        <UBadge
          v-if="row.original.source === 'job_post' && row.original.source_detail"
          as="a"
          :href="row.original.source_detail"
          target="_blank"
          rel="noopener noreferrer"
          :label="t('source.job_post')"
          :trailing-icon="icons.website"
          variant="soft"
          class="hover:underline"
        />
        <UBadge
          v-else-if="row.original.source === 'recruiter' && row.original.source_detail"
          as="a"
          :href="`mailto:${row.original.source_detail}`"
          :label="t('source.recruiter')"
          :trailing-icon="icons.mail"
          variant="soft"
          class="hover:underline"
        />
        <UBadge v-else :label="sources.label(row.original.source)" color="neutral" variant="soft" />
      </template>
      <template #status-cell="{ row }">
        <UDropdownMenu :items="statusMenu(row.original)">
          <UBadge
            as="button"
            :color="statusColor[row.original.status ?? 'applied']"
            variant="soft"
            class="cursor-pointer"
            :aria-label="`${t('table.changeStatus')} : ${t(`status.${row.original.status ?? 'applied'}`)}`"
            :label="t(`status.${row.original.status ?? 'applied'}`)"
            :trailing-icon="icons.chevronDown"
          />
        </UDropdownMenu>
      </template>
      <template #rating-cell="{ row }">
        <UInputRating
          :icon="icons.rating"
          :empty-icon="icons.ratingEmpty"
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
            :icon="icons.followUp"
            color="neutral"
            variant="ghost"
            :aria-label="t('table.markFollowedUp')"
            @click="markFollowedUp(row.original)"
          />
        </UTooltip>
      </template>
      <template #actions-cell="{ row }">
        <div class="flex justify-end gap-1">
          <UButton
            :icon="icons.edit"
            color="neutral"
            variant="ghost"
            :aria-label="t('common.edit')"
            @click="openForm(row.original)"
          />
          <UButton
            :icon="icons.delete"
            color="error"
            variant="ghost"
            :aria-label="t('common.delete')"
            @click="askDelete(row.original)"
          />
        </div>
      </template>
    </UTable>
    <!-- Fixed to the viewport so a short table does not lift it; the page's pb-16 keeps the last rows visible.
         The container gives it the table's width; its buttons are like the toolbar's. -->
    <UContainer v-if="view === 'table' && selected.length" class="fixed inset-x-0 bottom-4 z-10">
      <div class="flex flex-wrap items-center gap-2 rounded-lg border border-default bg-default p-2 shadow-lg">
        <span class="px-2 text-sm font-medium whitespace-nowrap">{{ t('table.selected', selected.length) }}</span>
        <UDropdownMenu :items="selectedStatusMenu" :content="{ side: 'top' }">
          <UButton
            color="neutral"
            variant="outline"
            size="sm"
            :trailing-icon="icons.chevronDown"
            :label="t('table.batchStatus')"
          />
        </UDropdownMenu>
        <div class="ml-auto flex items-center gap-2">
          <UButton color="neutral" variant="outline" size="sm" :label="t('common.cancel')" @click="selection = {}" />
          <UButton
            color="error"
            variant="outline"
            size="sm"
            :icon="icons.delete"
            :label="t('common.delete')"
            @click="askDeleteSelected"
          />
        </div>
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
