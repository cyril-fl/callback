<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui';
import type {
  BatchResult,
  Backup,
  DeletedApplication,
  ImportResult,
  Invite,
  InviteResult,
  Me,
  PasswordReset,
  PurgeResult,
  Token,
  TokenInput,
  TokenResult,
  User,
} from '~/types/items';

const { t } = useI18n();
const { icons } = useIcons();
const api = useApi();
const logout = useLogout();
const toast = useToast();
const theme = useTheme();
const { preferences, pro, save } = usePreferences();
const places = usePlaces();
const countries = useCountries();
const me = useNuxtData<Me>('me').data;
const planItems = (['base', 'pro'] as const).map((value) => ({ value, label: t(`settings.plans.${value}`) }));
// /settings opens the first section; an unknown one goes back to it.
const route = useRoute();
const admin = computed(() => me.value?.mode === 'multi' && me.value.admin);
const sections = computed(() => settingsSections.filter(({ id }) => id !== 'accounts' || admin.value));
const section = computed({
  get: () => sections.value.find(({ id }) => id === route.params.section)?.id ?? 'general',
  set: (id: SettingsSection) => navigateTo(`/settings/${id}`, { replace: true }),
});
const sectionItems = computed(() =>
  sections.value.map(({ id, icon }) => ({
    value: id,
    label: t(`settings.sections.${id}`),
    icon: icons[icon],
  })),
);

const colorMode = useColorMode();
const mode = computed({
  get: () => colorMode.preference as 'light' | 'dark' | 'system',
  set: (v: 'light' | 'dark' | 'system') => save({ color_mode: v }),
});

const modes = ['light', 'dark', 'system'] as const;
const modeIcons = { light: icons.lightMode, dark: icons.darkMode, system: icons.systemMode };
const modeItems = modes.map((value) => ({
  value,
  label: t(`settings.${value}`),
  icon: modeIcons[value],
  ui: { label: 'max-sm:sr-only' },
}));

const { data: tokens, refresh } = await useAsyncData('tokens', () => api<Token[]>('/api/tokens'));
const { data: trash, refresh: refreshTrash } = await useAsyncData('trash', () =>
  api<DeletedApplication[]>('/api/applications/deleted'),
);
// One application from its row, or the whole trash.
const forgetting = ref<{ body: { ids: string[] } | { all: true }; message: string }>();
const forgetOpen = ref(false);
function askForget(deleted?: DeletedApplication) {
  forgetting.value = deleted
    ? { body: { ids: [deleted.id] }, message: t('settings.forgetMessage', { company: deleted.application.company }) }
    : { body: { all: true }, message: t('settings.emptyTrashMessage', trash.value?.length ?? 0) };
  forgetOpen.value = true;
}
async function forget() {
  if (!forgetting.value) return;
  try {
    await api('/api/applications/deleted', { method: 'DELETE', body: forgetting.value.body });
  } catch {
    // useApi already toasted.
  } finally {
    await refreshTrash();
  }
}
async function restore(deleted: DeletedApplication) {
  try {
    await api('/api/applications/restore', { method: 'POST', body: { ids: [deleted.id] } });
    toast.add({ title: t('settings.restored', { company: deleted.application.company }), color: 'success' });
  } catch {
    // useApi already toasted.
  } finally {
    await refreshTrash();
  }
}

// The values the applications name, a field at a time: one fixed here is renamed in each of its applications.
// Countries are codes of a fixed list: no typo to fix.
const valueKind = ref<'company' | 'location'>('company');
const valueKindItems = (['company', 'location'] as const).map((value) => ({
  value,
  label: t(`settings.valueKinds.${value}`),
}));
const valueSearch = ref('');
const values = computed(() =>
  groupValues(places.applications.value, valueKind.value).map(({ value, ids }) => ({ value, ids, count: ids.length })),
);
type ValueRow = (typeof values.value)[number];
// A page at a time: a long search names hundreds of companies. The filter searches them all.
const valuesPerPage = 20;
const valuePage = ref(1);
watch([valueKind, valueSearch], () => (valuePage.value = 1));
const foundValues = computed(() => {
  const q = valueSearch.value.trim().toLocaleLowerCase();
  return q ? values.value.filter(({ value }) => value.toLocaleLowerCase().includes(q)) : values.value;
});
const pagedValues = computed(() =>
  foundValues.value.slice((valuePage.value - 1) * valuesPerPage, valuePage.value * valuesPerPage),
);
const valueColumns = computed<TableColumn<ValueRow>[]>(() => [
  { accessorKey: 'value', header: t(`settings.valueKinds.${valueKind.value}`) },
  {
    accessorKey: 'count',
    header: t('settings.valueCount'),
    meta: { class: { th: 'w-32 text-right', td: 'text-right' } },
  },
]);
async function rename(row: ValueRow, event: Event) {
  const input = event.target as HTMLInputElement;
  const to = input.value.trim();
  if (!to || to === row.value) {
    input.value = row.value;
    return;
  }
  try {
    const result = await api<BatchResult>('/api/applications', {
      method: 'PATCH',
      body: { ops: [{ ids: row.ids, set: { [valueKind.value]: to } }] },
    });
    // A batch's failures come back in the answer: useApi only tells a refused request.
    const failure = result.failed?.[0];
    if (failure) toast.add({ title: failure.error, color: 'error' });
    else
      toast.add({
        title: t('settings.renamed', { from: row.value, to, count: result.count }, result.count),
        color: 'success',
      });
  } catch {
    // useApi already toasted.
  } finally {
    await places.refresh();
  }
}

const when = (v?: string) => (v ? new Date(v).toLocaleDateString('fr-FR') : t('settings.never'));
const expiryItems = (['7d', '30d', '90d', '1y', 'never'] as const).map((value) => ({
  value,
  label: t(`settings.expiry.${value}`),
}));
const form = reactive<Required<TokenInput>>({ name: '', expires_in: '90d', read_only: false });
const created = ref<string>();
const creating = ref(false);

async function create() {
  creating.value = true;
  try {
    const result = await api<TokenResult>('/api/tokens', { method: 'POST', body: { ...form } });
    created.value = result.token;
    form.name = '';
    await refresh();
  } catch {
    // useApi already toasted.
  } finally {
    creating.value = false;
  }
}

const { copy } = useClipboard();
async function copyText(text?: string) {
  if (!text) return;
  await copy(text);
  toast.add({ title: t('settings.copied'), color: 'success' });
}

// Accounts: an admin's, in the multi mode only.
const { data: users, refresh: refreshUsers } = await useAsyncData('users', () =>
  admin.value ? api<User[]>('/api/users') : Promise.resolve(null),
);
const { data: invites, refresh: refreshInvites } = await useAsyncData('invites', () =>
  admin.value ? api<Invite[]>('/api/invites') : Promise.resolve(null),
);
// A generated password, or an invitation link when the server keeps none (no CALLBACK_SECRET): shown once.
const shown = ref<{ label: string; value: string }>();
const inviteLink = (token: string) => `${location.origin}/signup?invite=${token}`;
const inviting = ref(false);
async function createInvite() {
  inviting.value = true;
  try {
    const { token } = await api<InviteResult>('/api/invites', { method: 'POST' });
    await refreshInvites();
    if (invites.value?.some((invite) => invite.token)) await copyText(inviteLink(token));
    else shown.value = { label: t('settings.inviteLink'), value: inviteLink(token) };
  } catch {
    // useApi already toasted.
  } finally {
    inviting.value = false;
  }
}
// The account to delete or whose password to reset, or the invitation to revoke, behind a confirmation.
const accountAction = ref<
  { kind: 'delete'; user: User } | { kind: 'reset'; user: User } | { kind: 'revokeInvite'; invite: Invite }
>();
const accountOpen = ref(false);
function askAccount(action: NonNullable<typeof accountAction.value>) {
  accountAction.value = action;
  accountOpen.value = true;
}
const accountConfirm = computed(() => {
  const action = accountAction.value;
  if (action?.kind === 'delete')
    return {
      title: t('settings.deleteUser'),
      message: t('settings.deleteUserMessage', { name: action.user.name }),
      label: t('common.delete'),
    };
  if (action?.kind === 'reset')
    return {
      title: t('settings.resetPassword'),
      message: t('settings.resetPasswordMessage', { name: action.user.name }),
      label: t('settings.resetPassword'),
    };
  return { title: t('settings.revokeInvite'), message: t('settings.revokeInviteMessage'), label: t('settings.revoke') };
});
async function confirmAccount() {
  const action = accountAction.value;
  if (!action) return;
  try {
    if (action.kind === 'delete') {
      await api(`/api/users/${action.user.id}`, { method: 'DELETE' });
    } else if (action.kind === 'reset') {
      const { password } = await api<PasswordReset>(`/api/users/${action.user.id}/password`, { method: 'POST' });
      shown.value = { label: t('settings.newPassword', { name: action.user.name }), value: password };
    } else {
      await api(`/api/invites/${action.invite.id}`, { method: 'DELETE' });
    }
  } catch {
    // useApi already toasted.
  } finally {
    await Promise.all([refreshUsers(), refreshInvites()]);
  }
}

// The account's own name, the one it logs in with: saved when the field is left.
const nameDraft = ref(me.value?.user ?? '');
watch(
  () => me.value?.user,
  (name) => (nameDraft.value = name ?? ''),
);
async function renameAccount() {
  const name = nameDraft.value.trim();
  if (!name || name === me.value?.user) return void (nameDraft.value = me.value?.user ?? '');
  try {
    await api('/api/me/name', { method: 'PATCH', body: { name } });
    await refreshNuxtData('me');
    toast.add({ title: t('settings.renamed'), color: 'success' });
  } catch {
    nameDraft.value = me.value?.user ?? ''; // useApi already toasted.
  }
}

// The account's own password: the current one proves it is the owner, the other devices log out.
const passwordOpen = ref(false);
const passwordForm = reactive({ current: '', new: '' });
const changingPassword = ref(false);
async function changePassword() {
  changingPassword.value = true;
  try {
    await api('/api/me/password', { method: 'PATCH', body: passwordForm });
    passwordOpen.value = false;
    Object.assign(passwordForm, { current: '', new: '' });
    toast.add({ title: t('settings.passwordChanged'), color: 'success' });
  } catch {
    // useApi already toasted.
  } finally {
    changingPassword.value = false;
  }
}

const confirmOpen = ref(false);
const revoking = ref<Token>();
function askRevoke(token: Token) {
  revoking.value = token;
  confirmOpen.value = true;
}
async function exportJson() {
  try {
    const backup = await api<Backup>('/api/export');
    const url = URL.createObjectURL(new Blob([JSON.stringify(backup, null, 2)], { type: 'application/json' }));
    Object.assign(document.createElement('a'), {
      href: url,
      download: `callback-${backup.exported_at.slice(0, 10)}.json`,
    }).click();
    URL.revokeObjectURL(url);
  } catch {
    // useApi already toasted.
  }
}
const importInput = ref<HTMLInputElement>();
async function importJson(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = ''; // The same file can be picked again.
  if (!file) return;
  try {
    const result = await api<ImportResult>('/api/import', { method: 'POST', body: JSON.parse(await file.text()) });
    toast.add({
      title: t('settings.imported', {
        added: result.added + result.deleted,
        skipped: result.skipped,
        failed: result.failed?.length ?? 0,
      }),
      color: result.failed?.length ? 'warning' : 'success',
    });
    clearNuxtData(); // Every page read applications that are not all there.
    await refreshTrash();
  } catch (error) {
    if (error instanceof SyntaxError) toast.add({ title: t('settings.importInvalid'), color: 'error' });
    // Otherwise useApi already toasted.
  }
}

const purgeOpen = ref(false);
async function purge() {
  try {
    const { count } = await api<PurgeResult>('/api/data', { method: 'DELETE' });
    toast.add({ title: t('settings.purged', { count }), color: 'success' });
    clearNuxtData(); // Every page read applications that are gone.
    await navigateTo('/');
  } catch {
    // useApi already toasted.
  }
}

async function revoke() {
  if (!revoking.value) return;
  try {
    await api(`/api/tokens/${revoking.value.id}`, { method: 'DELETE' });
    await refresh();
  } catch {
    // useApi already toasted.
  }
}
</script>

<template>
  <div class="mx-auto max-w-3xl space-y-6">
    <h1 class="text-xl font-semibold">{{ t('settings.title') }}</h1>

    <div>
      <UTabs
        v-model="section"
        :items="sectionItems"
        :content="false"
        variant="link"
        :ui="{ label: 'max-sm:sr-only' }"
        :aria-label="t('settings.title')"
      />
      <p class="mt-3 text-xs text-muted">{{ t(`settings.sections.${section}Description`) }}</p>
    </div>

    <div v-if="section === 'general'" class="space-y-10">
      <section>
        <ul class="divide-y divide-default rounded-lg border border-default">
          <li class="flex flex-wrap items-center gap-4 px-4 py-3">
            <div class="flex-1 basis-48">
              <p class="text-sm">{{ t('settings.plan') }}</p>
              <p class="text-xs text-muted">{{ t(`settings.planHelp.${preferences?.plan ?? 'base'}`) }}</p>
            </div>
            <!-- Clicking the pressed option would empty the group: it is ignored. -->
            <UToggleGroup
              :model-value="preferences?.plan"
              :items="planItems"
              color="neutral"
              variant="outline"
              size="sm"
              attached
              :aria-label="t('settings.plan')"
              @update:model-value="(v) => v && save({ plan: v as 'base' | 'pro' })"
            />
          </li>
          <li class="flex flex-wrap items-center gap-4 px-4 py-3">
            <div class="flex-1 basis-48">
              <p class="text-sm">{{ t('settings.defaultPlace') }}</p>
              <p class="text-xs text-muted">{{ t('settings.defaultPlaceHelp') }}</p>
            </div>
            <div class="flex gap-2">
              <SuggestInput
                :model-value="preferences?.default_location ?? ''"
                :suggestions="places.cities.value"
                :placeholder="t('form.city')"
                :aria-label="t('form.city')"
                clear
                class="w-40"
                @update:model-value="(v: string) => save({ default_location: v })"
              />
              <USelectMenu
                :model-value="preferences?.default_country ?? ''"
                :items="countries.items.value"
                value-key="value"
                :placeholder="t('form.country')"
                :aria-label="t('form.country')"
                clear
                class="w-40"
                @update:model-value="(v?: string | null) => save({ default_country: v ?? '' })"
              />
            </div>
          </li>
          <li v-if="pro" class="flex flex-wrap items-center gap-4 px-4 py-3">
            <div class="flex-1 basis-48">
              <p class="text-sm">{{ t('settings.followUp') }}</p>
              <p class="text-xs text-muted">{{ t('settings.followUpHelp') }}</p>
            </div>
            <UInputNumber
              :model-value="preferences?.follow_up_days"
              :min="1"
              :max="90"
              :format-options="{ style: 'unit', unit: 'day', unitDisplay: 'long' }"
              :aria-label="t('settings.followUp')"
              class="w-40"
              @update:model-value="(v) => v && v !== preferences?.follow_up_days && save({ follow_up_days: v })"
            />
          </li>
          <li class="flex flex-wrap items-center gap-4 px-4 py-3">
            <div class="flex-1 basis-48">
              <p class="text-sm">{{ t('settings.noReply') }}</p>
              <p class="text-xs text-muted">{{ t('settings.noReplyHelp') }}</p>
            </div>
            <UInputNumber
              :model-value="preferences?.no_reply_days"
              :min="0"
              :max="365"
              :format-options="{ style: 'unit', unit: 'day', unitDisplay: 'long' }"
              :aria-label="t('settings.noReply')"
              class="w-40"
              @update:model-value="(v) => v != null && v !== preferences?.no_reply_days && save({ no_reply_days: v })"
            />
          </li>
        </ul>
      </section>

      <section>
        <h2 class="mb-3 text-sm font-semibold">{{ t('settings.account') }}</h2>
        <ul class="divide-y divide-default rounded-lg border border-default">
          <li v-if="me?.mode === 'multi'" class="flex flex-wrap items-center gap-4 px-4 py-3">
            <p class="min-w-0 flex-1 text-sm text-muted">{{ t('settings.accountName') }}</p>
            <UInput
              v-model="nameDraft"
              size="sm"
              maxlength="64"
              autocomplete="username"
              :aria-label="t('settings.accountName')"
              @change="renameAccount"
            />
          </li>
          <li v-if="me?.version" class="flex flex-wrap items-center gap-4 px-4 py-3">
            <p class="min-w-0 flex-1 text-sm text-muted">{{ t('settings.version') }}</p>
            <span class="font-mono text-xs text-muted">{{ me.version }}</span>
          </li>
          <li class="flex flex-wrap items-center gap-4 px-4 py-3">
            <p class="min-w-0 flex-1 text-sm">{{ t('settings.password') }}</p>
            <UModal v-model:open="passwordOpen" :title="t('settings.changePassword')">
              <UButton
                :icon="icons.resetPassword"
                :label="t('settings.changePassword')"
                :ui="{ label: 'max-sm:sr-only' }"
                color="neutral"
                variant="ghost"
              />
              <template #body>
                <form id="password-form" class="space-y-4" @submit.prevent="changePassword">
                  <UFormField :label="t('settings.currentPassword')">
                    <UInput
                      v-model="passwordForm.current"
                      type="password"
                      autocomplete="current-password"
                      required
                      class="w-full"
                    />
                  </UFormField>
                  <UFormField :label="t('settings.newOwnPassword')" :hint="t('settings.passwordHint')">
                    <UInput
                      v-model="passwordForm.new"
                      type="password"
                      autocomplete="new-password"
                      required
                      minlength="8"
                      class="w-full"
                    />
                  </UFormField>
                </form>
              </template>
              <template #footer>
                <UButton
                  type="submit"
                  form="password-form"
                  :label="t('settings.changePassword')"
                  :loading="changingPassword"
                />
              </template>
            </UModal>
          </li>
          <li class="flex flex-wrap items-center gap-4 px-4 py-3">
            <p class="min-w-0 flex-1 text-sm">{{ t('nav.logout') }}</p>
            <UButton
              :icon="icons.logout"
              :label="t('nav.logout')"
              :ui="{ label: 'max-sm:sr-only' }"
              color="neutral"
              variant="ghost"
              @click="logout"
            />
          </li>
        </ul>
      </section>
    </div>
    <div v-else-if="section === 'appearance'">
      <section>
        <ul class="divide-y divide-default rounded-lg border border-default">
          <li class="px-4 py-3">
            <p class="mb-3 text-sm">{{ t('settings.theme') }}</p>
            <div class="grid grid-cols-[repeat(auto-fill,minmax(10rem,1fr))] gap-3">
              <ThemePreview
                v-for="item in themes"
                :key="item.id"
                :theme="item"
                :active="theme === item.id"
                @click="theme = item.id"
              />
            </div>
          </li>
          <li class="flex flex-wrap items-center gap-4 px-4 py-3">
            <p class="flex-1 basis-48 text-sm">{{ t('settings.colorMode') }}</p>
            <UToggleGroup
              :model-value="mode"
              :items="modeItems"
              color="neutral"
              variant="outline"
              size="sm"
              attached
              :aria-label="t('settings.colorMode')"
              @update:model-value="(v) => v && (mode = v as 'light' | 'dark' | 'system')"
            />
          </li>
        </ul>
      </section>
    </div>
    <div v-else-if="section === 'data'" class="space-y-10">
      <section>
        <h2 class="mb-1 text-sm font-semibold">{{ t('settings.values') }}</h2>
        <p class="mb-3 text-xs text-muted">{{ t('settings.valuesHelp') }}</p>
        <div class="space-y-3">
          <div class="flex flex-wrap items-center gap-3">
            <UTabs v-model="valueKind" :items="valueKindItems" :content="false" size="sm" />
            <UInput
              v-model="valueSearch"
              :icon="icons.search"
              :placeholder="t('settings.valueSearch')"
              :aria-label="t('settings.valueSearch')"
              size="sm"
              class="ms-auto w-48"
            />
          </div>
          <UTable
            :data="pagedValues"
            :columns="valueColumns"
            :get-row-id="(row: ValueRow) => row.value"
            class="rounded-lg border border-default"
          >
            <template #empty>{{ t('settings.valuesEmpty') }}</template>
            <template #value-cell="{ row }">
              <UInput
                :key="row.original.value"
                :default-value="row.original.value"
                :aria-label="t(`settings.valueKinds.${valueKind}`)"
                variant="ghost"
                class="w-full"
                @change="rename(row.original, $event)"
                @keydown.enter="($event.target as HTMLInputElement).blur()"
              />
            </template>
          </UTable>
          <UPagination
            v-if="foundValues.length > valuesPerPage"
            v-model:page="valuePage"
            :items-per-page="valuesPerPage"
            :total="foundValues.length"
            size="sm"
            class="flex justify-center"
          />
        </div>
      </section>

      <section>
        <h2 class="mb-3 text-sm font-semibold">{{ t('settings.backup') }}</h2>
        <div class="flex flex-wrap items-center gap-4 rounded-lg border border-default px-4 py-3">
          <p class="flex-1 basis-48 text-xs text-muted">{{ t('settings.backupHelp') }}</p>
          <UButton
            :icon="icons.export"
            :label="t('settings.export')"
            color="neutral"
            variant="outline"
            @click="exportJson"
          />
          <UButton
            :icon="icons.import"
            :label="t('settings.import')"
            color="neutral"
            variant="outline"
            @click="importInput?.click()"
          />
          <input ref="importInput" type="file" accept="application/json,.json" class="hidden" @change="importJson" />
        </div>
      </section>

      <section>
        <div class="mb-3 flex items-center justify-between gap-4">
          <h2 class="text-sm font-semibold">{{ t('settings.trash') }}</h2>
          <UButton
            v-if="trash?.length"
            :label="t('settings.emptyTrash')"
            color="error"
            variant="ghost"
            size="sm"
            @click="askForget()"
          />
        </div>
        <ul class="divide-y divide-default rounded-lg border border-default">
          <li class="flex flex-wrap items-center gap-4 px-4 py-3">
            <div class="flex-1 basis-48">
              <p class="text-sm">{{ t('settings.trashDays') }}</p>
              <p class="text-xs text-muted">{{ t('settings.trashDaysHelp') }}</p>
            </div>
            <UInputNumber
              :model-value="preferences?.trash_days"
              :min="1"
              :max="365"
              :format-options="{ style: 'unit', unit: 'day', unitDisplay: 'long' }"
              :aria-label="t('settings.trashDays')"
              class="w-40"
              @update:model-value="(v) => v && v !== preferences?.trash_days && save({ trash_days: v })"
            />
          </li>
          <li v-for="deleted in trash" :key="deleted.id" class="flex flex-wrap items-center gap-3 px-4 py-2">
            <UIcon :name="icons.trash" class="size-4 shrink-0 text-muted" />
            <div class="min-w-0 flex-1 basis-48">
              <p class="truncate text-sm">
                {{ [deleted.application.company, deleted.application.location].filter(Boolean).join(' · ') }}
              </p>
              <p class="text-xs text-muted">{{ t('settings.deletedAt') }} {{ when(deleted.deleted_at) }}</p>
            </div>
            <UButton
              :icon="icons.restore"
              :label="t('settings.restore')"
              :ui="{ label: 'max-sm:sr-only' }"
              color="neutral"
              variant="ghost"
              @click="restore(deleted)"
            />
            <UButton
              :icon="icons.delete"
              :aria-label="t('settings.forget')"
              color="error"
              variant="ghost"
              @click="askForget(deleted)"
            />
          </li>
          <li v-if="trash && !trash.length" class="px-4 py-3 text-xs text-muted">{{ t('settings.trashEmpty') }}</li>
        </ul>
      </section>

      <section>
        <h2 class="mb-3 text-sm font-semibold text-error">{{ t('settings.danger') }}</h2>
        <div class="flex flex-wrap items-center gap-4 rounded-lg border border-error/25 bg-error/5 px-4 py-3">
          <div class="flex-1 basis-48">
            <p class="text-sm">{{ t('settings.purge') }}</p>
            <p class="text-xs text-muted">{{ t('settings.purgeHelp') }}</p>
          </div>
          <UButton :label="t('settings.purgeAction')" color="error" variant="ghost" @click="purgeOpen = true" />
        </div>
      </section>
    </div>
    <div v-else-if="section === 'tokens'">
      <section>
        <form class="flex flex-wrap items-end gap-4" @submit.prevent="create">
          <UFormField :label="t('settings.name')">
            <UInput v-model="form.name" class="w-56" />
          </UFormField>
          <UFormField :label="t('settings.expiresIn')">
            <USelect v-model="form.expires_in" :items="expiryItems" class="w-40" />
          </UFormField>
          <UCheckbox v-model="form.read_only" :label="t('settings.readOnly')" class="mb-2" />
          <UButton
            type="submit"
            :icon="icons.apiToken"
            color="neutral"
            variant="outline"
            :loading="creating"
            :disabled="!form.name.trim()"
            :label="t('settings.create')"
          />
        </form>

        <div v-if="created" class="mt-3 flex flex-wrap items-center gap-3 rounded-lg border border-default px-4 py-2">
          <span class="text-xs text-muted">{{ t('settings.tokenOnce') }}</span>
          <UInput
            :model-value="created"
            readonly
            size="sm"
            class="min-w-0 flex-1 basis-64 font-mono"
            :aria-label="t('settings.newToken')"
          >
            <template #trailing>
              <UButton
                :icon="icons.copy"
                :aria-label="t('settings.copy')"
                color="neutral"
                variant="link"
                size="sm"
                @click="copyText(created)"
              />
            </template>
          </UInput>
        </div>

        <ul v-if="tokens?.length" class="mt-3 divide-y divide-default rounded-lg border border-default">
          <li v-for="token in tokens" :key="token.id" class="flex flex-wrap items-center gap-3 px-4 py-2">
            <UIcon :name="icons.apiToken" class="size-4 shrink-0 text-muted" />
            <div class="min-w-0 flex-1 basis-48">
              <p class="flex min-w-0 items-center gap-2 text-sm">
                <span class="truncate">{{ token.name }}</span>
                <UBadge :label="t(`settings.kinds.${token.kind}`)" size="sm" color="neutral" variant="soft" />
                <UBadge
                  v-if="token.read_only"
                  :label="t('settings.readOnly')"
                  size="sm"
                  color="neutral"
                  variant="soft"
                />
              </p>
              <p class="text-xs text-muted">
                {{ t('settings.createdAt') }} {{ when(token.created_at) }} · {{ t('settings.lastUsedAt') }}
                {{ when(token.last_used_at) }} · {{ t('settings.expiresAt') }} {{ when(token.expires_at) }}
              </p>
            </div>
            <UButton
              :icon="icons.delete"
              :label="t('settings.revoke')"
              :ui="{ label: 'max-sm:sr-only' }"
              color="neutral"
              variant="ghost"
              @click="askRevoke(token)"
            />
          </li>
        </ul>
        <p v-else-if="tokens" class="mt-3 text-xs text-muted">{{ t('settings.noTokens') }}</p>
      </section>
    </div>
    <div v-else class="space-y-10">
      <div v-if="shown" class="flex flex-wrap items-center gap-3 rounded-lg border border-default px-4 py-2">
        <span class="text-xs text-muted">{{ shown.label }} · {{ t('settings.tokenOnce') }}</span>
        <UInput
          :model-value="shown.value"
          readonly
          size="sm"
          class="min-w-0 flex-1 basis-64 font-mono"
          :aria-label="shown.label"
        >
          <template #trailing>
            <UButton
              :icon="icons.copy"
              :aria-label="t('settings.copy')"
              color="neutral"
              variant="link"
              size="sm"
              @click="copyText(shown.value)"
            />
          </template>
        </UInput>
      </div>

      <section>
        <h2 class="mb-3 text-sm font-semibold">{{ t('settings.users') }}</h2>
        <ul class="divide-y divide-default rounded-lg border border-default">
          <li v-for="user in users" :key="user.id" class="flex flex-wrap items-center gap-3 px-4 py-2">
            <UIcon :name="icons.recruiter" class="size-4 shrink-0 text-muted" />
            <div class="min-w-0 flex-1 basis-48">
              <p class="flex min-w-0 items-center gap-2 text-sm">
                <span class="truncate">{{ user.name }}</span>
                <UBadge v-if="user.admin" :label="t('settings.admin')" size="sm" color="neutral" variant="soft" />
                <UBadge
                  v-if="user.name === me?.user"
                  :label="t('settings.you')"
                  size="sm"
                  color="primary"
                  variant="soft"
                />
              </p>
              <p class="text-xs text-muted">{{ t('settings.createdAt') }} {{ when(user.created_at) }}</p>
            </div>
            <template v-if="user.name !== me?.user">
              <UButton
                :icon="icons.resetPassword"
                :label="t('settings.resetPassword')"
                :ui="{ label: 'max-sm:sr-only' }"
                color="neutral"
                variant="ghost"
                @click="askAccount({ kind: 'reset', user })"
              />
              <UButton
                :icon="icons.delete"
                :aria-label="t('settings.deleteUser')"
                color="error"
                variant="ghost"
                @click="askAccount({ kind: 'delete', user })"
              />
            </template>
          </li>
        </ul>
      </section>

      <section>
        <div class="mb-3 flex items-center justify-between gap-4">
          <h2 class="text-sm font-semibold">{{ t('settings.invites') }}</h2>
          <UButton
            :icon="icons.invite"
            :label="t('settings.createInvite')"
            color="neutral"
            variant="outline"
            size="sm"
            :loading="inviting"
            @click="createInvite"
          />
        </div>
        <ul v-if="invites?.length" class="divide-y divide-default rounded-lg border border-default">
          <li v-for="invite in invites" :key="invite.id" class="flex flex-wrap items-center gap-3 px-4 py-2">
            <UIcon :name="icons.invite" class="size-4 shrink-0 text-muted" />
            <p class="min-w-0 flex-1 basis-48 text-xs text-muted">
              {{ t('settings.createdAt') }} {{ when(invite.created_at) }} · {{ t('settings.expiresAt') }}
              {{ when(invite.expires_at) }}
            </p>
            <UInput
              v-if="invite.token"
              :model-value="inviteLink(invite.token)"
              readonly
              size="sm"
              class="min-w-0 flex-1 basis-64 font-mono"
              :aria-label="t('settings.inviteLink')"
            >
              <template #trailing>
                <UButton
                  :icon="icons.copy"
                  :aria-label="t('settings.copy')"
                  color="neutral"
                  variant="link"
                  size="sm"
                  @click="copyText(inviteLink(invite.token))"
                />
              </template>
            </UInput>
            <UButton
              :icon="icons.delete"
              :label="t('settings.revoke')"
              :ui="{ label: 'max-sm:sr-only' }"
              color="neutral"
              variant="ghost"
              @click="askAccount({ kind: 'revokeInvite', invite })"
            />
          </li>
        </ul>
        <p v-else-if="invites" class="text-xs text-muted">{{ t('settings.noInvites') }}</p>
      </section>
    </div>

    <ConfirmModal
      v-model:open="forgetOpen"
      :title="forgetting && 'all' in forgetting.body ? t('settings.emptyTrash') : t('settings.forget')"
      :message="forgetting?.message ?? ''"
      :confirm-label="t('common.delete')"
      @confirm="forget"
    />
    <ConfirmModal
      v-model:open="purgeOpen"
      :title="t('settings.purge')"
      :message="t('settings.purgeMessage')"
      :confirm-label="t('settings.purgeAction')"
      :expected="t('settings.purgeWord')"
      @confirm="purge"
    />
    <ConfirmModal
      v-model:open="confirmOpen"
      :title="t('settings.revokeTitle')"
      :message="t('settings.revokeMessage', { name: revoking?.name ?? '' })"
      :confirm-label="t('settings.revoke')"
      @confirm="revoke"
    />
    <ConfirmModal
      v-model:open="accountOpen"
      :title="accountConfirm.title"
      :message="accountConfirm.message"
      :confirm-label="accountConfirm.label"
      @confirm="confirmAccount"
    />
  </div>
</template>
