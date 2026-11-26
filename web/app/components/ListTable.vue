<script setup lang="ts" generic="Row extends Record<string, unknown>">
import type { Component } from 'vue';
import type { TableColumn } from '@nuxt/ui';
import { DateTimeInput, UInput } from '#components';
import type { SortingState } from '@tanstack/vue-table';

// An editable list of an application (interviews, contacts): each cell is an input, the headers sort, a row is added
// at the bottom and deleted by its button. name is the form's field: errors show under their cell.
const props = defineProps<{
  name: string;
  title: string;
  hint?: string;
  columns: readonly {
    key: keyof Row & string;
    label: string;
    type?: 'text' | 'email' | 'datetime-local';
    items?: readonly { value: string; label: string }[];
  }[];
  blank: () => Row;
  sort?: SortingState;
}>();
const rows = defineModel<Row[]>({ required: true });
const { t } = useI18n();
const { icons } = useIcons();
const ColumnHeader = resolveComponent('ColumnHeader');
// The input of each column type: a datetime-local column gets the Nuxt UI date picker, not the browser's.
type ColumnType = NonNullable<(typeof props.columns)[number]['type']>;
const inputs: Record<ColumnType, { is: Component; type?: string }> = {
  text: { is: UInput, type: 'text' },
  email: { is: UInput, type: 'email' },
  'datetime-local': { is: DateTimeInput },
};
const sorting = ref<SortingState>(props.sort ?? []);

const tableColumns = computed<TableColumn<Row>[]>(() => [
  ...props.columns.map((c): TableColumn<Row> => ({
    accessorKey: c.key,
    header: ({ column }) =>
      h(ColumnHeader, {
        label: c.label,
        sorted: column.getIsSorted(),
        onSort: () => column.toggleSorting(column.getIsSorted() === 'asc'),
      }),
  })),
  { id: 'actions', meta: { class: { th: 'w-10', td: 'w-10' } } },
]);
</script>

<template>
  <div class="space-y-1 rounded-md bg-elevated p-2">
    <p class="px-1 text-sm font-medium">{{ title }}</p>
    <p v-if="hint" class="px-1 text-xs text-muted">{{ hint }}</p>
    <UTable
      v-model:sorting="sorting"
      :data="rows"
      :columns="tableColumns"
      :ui="{ base: 'table-fixed', th: 'py-1 px-1', td: 'p-1 align-top', empty: 'py-3' }"
    >
      <template #empty>
        <span class="text-sm text-muted">{{ t('form.noRow') }}</span>
      </template>
      <template v-for="c in columns" :key="c.key" #[`${c.key}-cell`]="{ row }">
        <UFormField :name="`${name}.${row.index}.${c.key}`">
          <USelect
            v-if="c.items"
            v-model="(row.original as Record<string, unknown>)[c.key] as string"
            :items="[...c.items]"
            size="sm"
            class="w-full"
          />
          <component
            :is="inputs[c.type ?? 'text'].is"
            v-else
            v-model="(row.original as Record<string, unknown>)[c.key] as string"
            :type="inputs[c.type ?? 'text'].type"
            :aria-label="c.label"
            size="sm"
            :ui="{ base: 'min-w-0' }"
            class="w-full"
          />
        </UFormField>
      </template>
      <template #actions-cell="{ row }">
        <UButton
          :icon="icons.delete"
          color="neutral"
          variant="ghost"
          size="sm"
          :aria-label="t('common.delete')"
          @click="rows = rows.filter((_, i) => i !== row.index)"
        />
      </template>
    </UTable>
    <UFormField :name="name" />
    <UButton
      :icon="icons.add"
      variant="ghost"
      color="neutral"
      size="sm"
      :label="t('common.add')"
      @click="rows = [...rows, blank()]"
    />
  </div>
</template>
