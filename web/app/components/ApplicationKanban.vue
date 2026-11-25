<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui';
import type { Application, Status } from '~/types/items';
import { STATUSES } from '~/types/items';

const props = defineProps<{
  applications: Application[];
  statusColor: Record<Status, 'neutral' | 'info' | 'primary' | 'success' | 'error'>;
}>();
const emit = defineEmits<{ move: [app: Application, status: Status]; edit: [app: Application] }>();

const { t } = useI18n();

const columns = computed(() =>
  STATUSES.map((status) => ({ status, items: props.applications.filter((a) => a.status === status) })),
);

// Drag and drop moves a card; the menu of each card does the same from the keyboard.
const dragged = ref<Application>();
const over = ref<Status>();
function drop(status: Status) {
  if (dragged.value && dragged.value.status !== status) emit('move', dragged.value, status);
  dragged.value = undefined;
  over.value = undefined;
}

const moveMenu = (app: Application): DropdownMenuItem[] =>
  STATUSES.filter((s) => s !== app.status).map((s) => ({
    label: t(`status.${s}`),
    onSelect: () => emit('move', app, s),
  }));
</script>

<template>
  <div class="grid gap-3 overflow-x-auto md:grid-cols-5">
    <section
      v-for="column in columns"
      :key="column.status"
      class="flex min-h-48 flex-col gap-2 rounded-lg border border-default bg-elevated/50 p-2 transition-colors"
      :class="over === column.status ? 'border-primary bg-primary/5' : ''"
      :aria-label="t(`status.${column.status}`)"
      @dragover.prevent="over = column.status"
      @dragleave="over = over === column.status ? undefined : over"
      @drop.prevent="drop(column.status)"
    >
      <header class="flex items-center justify-between px-1">
        <UBadge :color="statusColor[column.status]" variant="subtle" :label="t(`status.${column.status}`)" />
        <span class="text-sm text-muted">{{ column.items.length }}</span>
      </header>
      <UCard
        v-for="app in column.items"
        :key="app.id"
        draggable="true"
        class="cursor-grab active:cursor-grabbing"
        :class="dragged?.id === app.id ? 'opacity-50' : ''"
        :ui="{ body: 'p-3 sm:p-3' }"
        @dragstart="dragged = app"
        @dragend="
          dragged = undefined;
          over = undefined;
        "
      >
        <div class="flex items-start justify-between gap-1">
          <div class="min-w-0">
            <p class="truncate font-medium">{{ app.company }}</p>
            <p class="truncate text-sm text-muted">{{ app.location }}</p>
          </div>
          <UDropdownMenu :items="moveMenu(app)">
            <UButton
              icon="i-lucide-ellipsis-vertical"
              color="neutral"
              variant="ghost"
              size="xs"
              :aria-label="t('table.changeStatus')"
            />
          </UDropdownMenu>
        </div>
        <div class="mt-2 flex items-center justify-between">
          <span role="img" class="flex" :aria-label="t('table.stars', { n: app.rating ?? 0 })">
            <UIcon
              v-for="n in 5"
              :key="n"
              name="i-lucide-star"
              class="size-3"
              :class="n <= (app.rating ?? 0) ? 'text-warning' : 'text-muted/40'"
            />
          </span>
          <UButton
            icon="i-lucide-pencil"
            color="neutral"
            variant="ghost"
            size="xs"
            :aria-label="t('common.edit')"
            @click="emit('edit', app)"
          />
        </div>
      </UCard>
    </section>
  </div>
</template>
