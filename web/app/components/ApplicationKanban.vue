<script setup lang="ts">
import type { Application, Status } from '~/types/items';
import { STATUSES } from '~/types/items';

const props = defineProps<{
  applications: Application[];
  statusColor: Record<Status, 'neutral' | 'info' | 'warning' | 'success' | 'error'>;
  followUpSince: string;
}>();
const emit = defineEmits<{
  move: [app: Application, status: Status];
  edit: [app: Application];
  followUp: [app: Application];
}>();

const { t } = useI18n();

const columns = computed(() =>
  STATUSES.map((status) => ({ status, items: props.applications.filter((a) => a.status === status) })),
);

const dot: Record<string, string> = {
  neutral: 'bg-inverted/40',
  info: 'bg-info',
  warning: 'bg-warning',
  success: 'bg-success',
  error: 'bg-error',
};
// A source the user named gets a tag.
const sourceIcon = (source?: string) =>
  ({ spontaneous: 'i-lucide-send', job_post: 'i-lucide-briefcase', recruiter: 'i-lucide-user-round' })[
    source ?? 'spontaneous'
  ] ?? 'i-lucide-tag';
const sources = useSources();

// A drag moves a card to another column; a click opens it. The form changes the status from the keyboard.
const dragged = ref<Application>();
const over = ref<Status>();
function drop(status: Status) {
  if (dragged.value && dragged.value.status !== status) emit('move', dragged.value, status);
  dragged.value = undefined;
  over.value = undefined;
}

// "2026-01-31" is a day, not an instant: no timezone shift.
const day = (v?: string) =>
  v ? new Date(`${v.slice(0, 10)}T00:00:00`).toLocaleDateString('fr-FR', { day: 'numeric', month: 'short' }) : '';
</script>

<template>
  <div class="flex min-h-80 flex-1 gap-3 overflow-x-auto pb-1">
    <section
      v-for="column in columns"
      :key="column.status"
      class="flex min-w-52 flex-1 shrink-0 flex-col rounded-xl bg-elevated/60 ring-1 transition-colors ring-inset"
      :class="over === column.status ? 'bg-primary/5 ring-primary' : 'ring-default'"
      :aria-label="t(`status.${column.status}`)"
      @dragover.prevent="over = column.status"
      @dragleave.self="over = undefined"
      @drop.prevent="drop(column.status)"
    >
      <header class="flex items-center gap-2 px-3 pt-3 pb-2">
        <span class="size-2.5 rounded-full" :class="dot[statusColor[column.status]]" aria-hidden="true" />
        <h3 class="text-sm font-medium">{{ t(`status.${column.status}`) }}</h3>
        <span class="rounded-full bg-accented/60 px-1.5 text-xs text-muted tabular-nums">{{
          column.items.length
        }}</span>
      </header>

      <div class="flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto px-2 pb-2">
        <div
          v-for="app in column.items"
          :key="app.id"
          role="button"
          tabindex="0"
          draggable="true"
          class="group cursor-pointer rounded-lg border border-default bg-default p-3 shadow-xs transition hover:border-accented hover:shadow-sm focus-visible:outline-2 focus-visible:outline-primary active:cursor-grabbing"
          :class="dragged?.id === app.id ? 'opacity-40' : ''"
          :aria-label="`${app.company}, ${app.location}`"
          @click="emit('edit', app)"
          @keydown.enter.prevent="emit('edit', app)"
          @dragstart="dragged = app"
          @dragend="
            dragged = undefined;
            over = undefined;
          "
        >
          <p class="truncate text-sm font-medium">{{ app.company }}</p>
          <p class="mt-0.5 flex items-center gap-1 truncate text-xs text-muted">
            <UIcon name="i-lucide-map-pin" class="size-3 shrink-0" />{{
              [app.location, app.country].filter(Boolean).join(', ')
            }}
          </p>
          <div class="mt-3 flex items-center gap-2 text-xs text-muted">
            <UTooltip :text="sources.label(app.source)">
              <UIcon :name="sourceIcon(app.source)" class="size-3.5" :aria-label="sources.label(app.source)" />
            </UTooltip>
            <UInputRating
              v-if="app.rating"
              icon="i-callback-star-filled"
              empty-icon="i-lucide-star"
              :model-value="app.rating"
              readonly
              size="xs"
            />
            <UTooltip v-if="canFollowUp(app, followUpSince)" :text="t('table.markFollowedUp')">
              <UButton
                icon="i-lucide-bell-ring"
                color="neutral"
                variant="ghost"
                size="xs"
                class="-my-1"
                :aria-label="t('table.markFollowedUp')"
                @click.stop="emit('followUp', app)"
                @keydown.enter.stop
              />
            </UTooltip>
            <span class="ml-auto tabular-nums">{{ day(app.applied_at) }}</span>
          </div>
        </div>
        <p
          v-if="column.items.length === 0"
          class="rounded-lg border border-dashed border-default p-4 text-center text-xs text-muted"
        >
          {{ t('kanban.dropHere') }}
        </p>
      </div>
    </section>
  </div>
</template>
