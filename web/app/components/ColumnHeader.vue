<script setup lang="ts">
import type { CommandPaletteItem, DropdownMenuItem } from '@nuxt/ui';

// A column header: the label opens the column's filter (a menu, a list of values with a search, or a text search),
// the arrow after it sorts asc then desc.
const props = defineProps<{
  label: string;
  sorted: false | 'asc' | 'desc';
  active?: boolean;
  menu?: DropdownMenuItem[][];
  // With v-model: a value picked among these, none for all.
  options?: string[];
  // With v-model: free text.
  search?: boolean;
}>();
const emit = defineEmits<{ sort: [] }>();
const model = defineModel<string>();
// The filter's popover, open from outside too (a shortcut).
const open = defineModel<boolean>('open', { default: false });
const { t } = useI18n();
const { icons } = useIcons();

const trigger = computed(
  () =>
    ({
      label: props.label,
      color: props.active ? 'primary' : 'neutral',
      variant: props.active ? 'soft' : 'ghost',
      trailingIcon: icons.chevronDown,
    }) as const,
);
function pick(item?: CommandPaletteItem) {
  if (!item?.value) return;
  model.value = item.value === 'all' ? undefined : item.value;
  open.value = false;
}
const items = computed(() => [
  { label: t('common.all'), value: 'all' },
  ...(props.options ?? []).map((v) => ({ label: v, value: v })),
]);
</script>

<template>
  <!-- -ml-2.5 takes back the button's padding: the label lines up with the cells. -->
  <div class="-ml-2.5 flex items-center gap-0.5">
    <UDropdownMenu v-if="menu" :items="menu" :content="{ align: 'start' }">
      <UButton v-bind="trigger" />
    </UDropdownMenu>
    <!-- A button like the other headers (a USelectMenu trigger is an input: no active color, its own focus). -->
    <UPopover v-else-if="options" v-model:open="open" :content="{ align: 'start' }">
      <UButton v-bind="trigger" />
      <template #content>
        <UCommandPalette
          :model-value="items.find((i) => i.value === (model ?? 'all'))"
          :groups="[{ id: 'options', items }]"
          by="value"
          :placeholder="t('table.search')"
          class="max-h-80 w-56"
          @update:model-value="pick"
        />
      </template>
    </UPopover>
    <UPopover v-else-if="search" v-model:open="open" :content="{ align: 'start' }">
      <UButton v-bind="trigger" />
      <template #content>
        <!-- The input of the UCommandPalette above, so both searches look alike. -->
        <UInput
          v-model="model"
          variant="none"
          :icon="icons.search"
          :placeholder="t('table.search')"
          :aria-label="t('table.search')"
          autofocus
          class="w-56 [&>input]:h-12"
        />
      </template>
    </UPopover>
    <span v-else class="px-2.5 font-medium">{{ label }}</span>
    <UButton
      :color="sorted ? 'primary' : 'neutral'"
      variant="ghost"
      size="xs"
      :class="!sorted && 'text-dimmed'"
      :icon="sorted === 'asc' ? icons.ascending : sorted === 'desc' ? icons.descending : icons.sort"
      :aria-label="`${t('table.sortBy')} ${label}`"
      @click="emit('sort')"
    />
  </div>
</template>
