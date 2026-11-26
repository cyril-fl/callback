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
  remove: [app: Application];
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
const countries = useCountries();

// A drag moves a card to another column; a click opens it. The form changes the status from the keyboard.
const dragged = ref<Application>();
const over = ref<string>();
function drop(column: { status: Status; statuses: Status[] }) {
  if (dragged.value && !column.statuses.includes(dragged.value.status ?? 'applied'))
    emit('move', dragged.value, column.status);
  dragged.value = undefined;
  over.value = undefined;
}

// A right click or the card's "…" button (keyboard, touch): the status and delete, the table's same paths. A click edits.
const menu = (app: Application) => [
  [
    {
      label: t('table.changeStatus'),
      children: STATUSES.map((s) => ({
        label: t(`status.${s}`),
        type: 'checkbox' as const,
        checked: app.status === s,
        onSelect: () => emit('move', app, s),
      })),
    },
  ],
  [{ label: t('common.delete'), icon: icons.delete, color: 'error' as const, onSelect: () => emit('remove', app) }],
];

// The card's facts as icons, one group, the detail in a tooltip.
type Badge = { icon: string; label: string; href?: string };
const badges = (app: Application) =>
  [
    { icon: sourceIcon(app.source), label: sources.label(app.source) },
    CLOSED.includes(app.status ?? 'applied') && {
      icon: app.status === 'no_reply' ? icons.noReply : icons.rejected,
      label: t(`status.${app.status}`),
    },
    (app.contact_name || app.contact_email) && {
      icon: icons.person,
      label: [app.contact_name, app.contact_email].filter(Boolean).join(' · '),
    },
    app.website && { icon: icons.website, label: app.website, href: websiteHref(app.website) },
    app.followed_up_at && {
      icon: icons.followedUp,
      label: t('kanban.followedUpOn', { date: day(app.followed_up_at) }),
    },
    app.comment && { icon: icons.comment, label: app.comment },
  ].filter((b): b is Badge => !!b);

// "2026-01-31T14:30" is a local time: no timezone shift.
const interview = (app: Application) =>
  app.status === 'interview' && app.interview_at
    ? new Date(app.interview_at).toLocaleString('fr-FR', {
        day: 'numeric',
        month: 'short',
        hour: '2-digit',
        minute: '2-digit',
      })
    : '';

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
      class="flex min-w-52 flex-1 shrink-0 flex-col rounded-xl ring-1 ring-default transition-colors ring-inset"
      :class="over === column.key ? 'bg-elevated' : 'bg-elevated/60'"
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

      <div class="flex min-h-0 flex-1 flex-col gap-2.5 overflow-y-auto px-2.5 pb-2.5">
        <UContextMenu v-for="app in column.items" :key="app.id" :items="menu(app)">
          <div
            role="button"
            tabindex="0"
            draggable="true"
            class="group cursor-pointer rounded-lg bg-default p-4 shadow-xs ring-1 ring-default/50 transition hover:shadow-md focus-visible:outline-2 focus-visible:outline-primary active:cursor-grabbing"
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
              <p class="min-w-0 flex-1 truncate text-sm font-medium">{{ app.company }}</p>
              <UInputRating
                :icon="icons.rating"
                :empty-icon="icons.ratingEmpty"
                :model-value="app.rating ?? 0"
                readonly
                size="xs"
                class="shrink-0"
              />
              <UDropdownMenu :items="menu(app)">
                <UButton
                  :icon="icons.more"
                  color="neutral"
                  variant="ghost"
                  size="xs"
                  class="-my-1 -mr-2"
                  :aria-label="t('table.actions')"
                  @click.stop
                  @keydown.enter.stop
                />
              </UDropdownMenu>
            </div>
            <p v-if="app.title" class="truncate text-xs text-muted">{{ app.title }}</p>
            <p class="mt-0.5 flex items-center gap-1 truncate text-xs text-muted">
              <UIcon :name="icons.location" class="size-3 shrink-0" />{{
                [app.location, countries.label(app.country)].filter(Boolean).join(', ')
              }}
            </p>
            <div class="mt-4 flex items-center gap-2 text-xs text-muted">
              <span class="flex items-center gap-1.5">
                <UTooltip
                  v-for="badge in badges(app)"
                  :key="badge.icon"
                  :text="badge.label"
                  :ui="{ content: 'h-auto max-w-72', text: 'line-clamp-6 whitespace-pre-line' }"
                >
                  <a
                    v-if="badge.href"
                    :href="badge.href"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="flex hover:text-highlighted"
                    :aria-label="badge.label"
                    @click.stop
                    @keydown.enter.stop
                  >
                    <UIcon :name="badge.icon" class="size-3.5" />
                  </a>
                  <UIcon v-else :name="badge.icon" class="size-3.5" :aria-label="badge.label" />
                </UTooltip>
              </span>
              <UTooltip v-if="interview(app)" :text="`${t('form.interviewAt')} : ${interview(app)}`">
                <UIcon
                  :name="icons.interview"
                  class="size-3.5 text-primary"
                  :aria-label="`${t('form.interviewAt')} : ${interview(app)}`"
                />
              </UTooltip>
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
              <span class="ml-auto shrink-0 tabular-nums">{{ day(app.applied_at) }}</span>
            </div>
          </div>
        </UContextMenu>
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
