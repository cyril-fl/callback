<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui';

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
const { t } = useI18n();

const trigger = computed(
  () =>
    ({
      label: props.label,
      color: props.active ? 'primary' : 'neutral',
      variant: props.active ? 'soft' : 'ghost',
      trailingIcon: 'i-lucide-chevron-down',
    }) as const,
);
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
    <USelectMenu
      v-else-if="options"
      :model-value="model ?? 'all'"
      :items="items"
      value-key="value"
      :color="trigger.color"
      :variant="trigger.variant"
      :trailing-icon="trigger.trailingIcon"
      :search-input="{ placeholder: t('table.search') }"
      :content="{ align: 'start' }"
      :ui="{ content: 'min-w-48', trailingIcon: 'text-current' }"
      class="w-auto font-medium"
      @update:model-value="(v: string) => (model = v === 'all' ? undefined : v)"
    >
      {{ label }}
    </USelectMenu>
    <UPopover v-else-if="search" :content="{ align: 'start' }">
      <UButton v-bind="trigger" />
      <template #content>
        <UInput
          v-model="model"
          icon="i-lucide-search"
          :placeholder="t('table.search')"
          :aria-label="t('table.search')"
          autofocus
          class="m-2 w-56"
        />
      </template>
    </UPopover>
    <span v-else class="px-2.5 font-medium">{{ label }}</span>
    <UButton
      :color="sorted ? 'primary' : 'neutral'"
      variant="ghost"
      size="xs"
      :class="!sorted && 'text-dimmed'"
      :icon="
        sorted === 'asc'
          ? 'i-lucide-arrow-up-narrow-wide'
          : sorted === 'desc'
            ? 'i-lucide-arrow-down-wide-narrow'
            : 'i-lucide-arrow-up-down'
      "
      :aria-label="`${t('table.sortBy')} ${label}`"
      @click="emit('sort')"
    />
  </div>
</template>
