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
const { icons } = useIcons();

// The ends nobody works on share one column, "closed": a drop there is a rejection, no_reply is set by the server.
const CLOSED: Status[] = ['rejected', 'no_reply'];
const columns = computed(() =>
  [...STATUSES.filter((s) => !CLOSED.includes(s)), 'closed' as const].map((key) => {
    const statuses = key === 'closed' ? CLOSED : [key];
    return {
      key,
      status: statuses[0]!,
      statuses,
      items: props.applications.filter((a) => statuses.includes(a.status ?? 'applied')),
    };
  }),
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
  ({ spontaneous: icons.spontaneous, job_post: icons.jobPost, recruiter: icons.recruiter })[source ?? 'spontaneous'] ??
  icons.otherSource;
const sources = useSources();

// A drag moves a card to another column; a click opens it. The form changes the status from the keyboard.
const dragged = ref<Application>();
const over = ref<string>();
function drop(column: { status: Status; statuses: Status[] }) {
  if (dragged.value && !column.statuses.includes(dragged.value.status ?? 'applied'))
    emit('move', dragged.value, column.status);
  dragged.value = undefined;
  over.value = undefined;
}

const label = (key: string) => (key === 'closed' ? t('kanban.closed') : t(`status.${key}`));

// "2026-01-31" is a day, not an instant: no timezone shift.
const day = (v?: string) =>
  v ? new Date(`${v.slice(0, 10)}T00:00:00`).toLocaleDateString('fr-FR', { day: 'numeric', month: 'short' }) : '';
</script>

<template>
  <div class="flex min-h-80 flex-1 gap-3 overflow-x-auto pb-1">
    <section
      v-for="column in columns"
      :key="column.key"
      class="flex min-w-52 flex-1 shrink-0 flex-col rounded-xl bg-elevated/60 ring-1 transition-colors ring-inset"
      :class="over === column.key ? 'bg-primary/5 ring-primary' : 'ring-default'"
      :aria-label="label(column.key)"
      @dragover.prevent="over = column.key"
      @dragleave.self="over = undefined"
      @drop.prevent="drop(column)"
    >
      <header class="flex items-center gap-2 px-3 pt-3 pb-2">
        <span class="size-2.5 rounded-full" :class="dot[statusColor[column.status]]" aria-hidden="true" />
        <h3 class="text-sm font-medium">{{ label(column.key) }}</h3>
        <UBadge
          :label="column.items.length"
          color="neutral"
          variant="soft"
          size="sm"
          class="rounded-full text-xs text-muted tabular-nums"
        />
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
          <div class="flex items-center gap-2">
            <p class="truncate text-sm font-medium">{{ app.company }}</p>
            <UTooltip v-if="column.key === 'closed'" :text="t(`status.${app.status}`)">
              <UIcon
                :name="app.status === 'no_reply' ? icons.noReply : icons.rejected"
                class="ml-auto size-3.5 shrink-0 text-muted"
                :aria-label="t(`status.${app.status}`)"
              />
            </UTooltip>
          </div>
          <p class="mt-0.5 flex items-center gap-1 truncate text-xs text-muted">
            <UIcon :name="icons.location" class="size-3 shrink-0" />{{
              [app.location, app.country].filter(Boolean).join(', ')
            }}
          </p>
          <div class="mt-3 flex items-center gap-2 text-xs text-muted">
            <UTooltip :text="sources.label(app.source)">
              <UIcon :name="sourceIcon(app.source)" class="size-3.5" :aria-label="sources.label(app.source)" />
            </UTooltip>
            <UInputRating
              v-if="app.rating"
              :icon="icons.rating"
              :empty-icon="icons.ratingEmpty"
              :model-value="app.rating"
              readonly
              size="xs"
            />
            <UTooltip v-if="canFollowUp(app, followUpSince)" :text="t('table.markFollowedUp')">
              <UButton
                :icon="icons.followUp"
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
