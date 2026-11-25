<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui';
import type { Token, TokenInput, TokenResult } from '~/types/items';

const { t } = useI18n();
const api = useApi();
const toast = useToast();
const colorMode = useColorMode();
const mode = computed({
  get: () => colorMode.preference as 'light' | 'dark' | 'system',
  set: (v: 'light' | 'dark' | 'system') => (colorMode.preference = v),
});

const modes = ['light', 'dark', 'system'] as const;
const modeItems = modes.map((value) => ({ value, label: t(`settings.${value}`) }));

const { data: tokens, refresh } = await useAsyncData('tokens', () => api<Token[]>('/api/tokens'));

const when = (v?: string) => (v ? new Date(v).toLocaleDateString('fr-FR') : t('settings.never'));
const columns = computed<TableColumn<Token>[]>(() => [
  { accessorKey: 'name', header: t('settings.name') },
  { accessorKey: 'kind', header: t('settings.kind') },
  { accessorKey: 'read_only', header: t('settings.readOnly') },
  { accessorKey: 'created_at', header: t('settings.createdAt') },
  { accessorKey: 'last_used_at', header: t('settings.lastUsedAt') },
  { accessorKey: 'expires_at', header: t('settings.expiresAt') },
  { id: 'actions', header: () => h('span', { class: 'sr-only' }, t('table.actions')) },
]);

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
async function copyToken() {
  if (!created.value) return;
  await copy(created.value);
  toast.add({ title: t('settings.copied'), color: 'success' });
}

const confirmOpen = ref(false);
const revoking = ref<Token>();
function askRevoke(token: Token) {
  revoking.value = token;
  confirmOpen.value = true;
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
  <div class="space-y-8">
    <h1 class="text-xl font-semibold">{{ t('settings.title') }}</h1>

    <section class="space-y-3">
      <h2 class="font-medium">{{ t('settings.colorMode') }}</h2>
      <USelect v-model="mode" :items="modeItems" :aria-label="t('settings.colorMode')" class="w-48" />
    </section>

    <section class="space-y-4">
      <div>
        <h2 class="font-medium">{{ t('settings.tokens') }}</h2>
        <p class="text-sm text-muted">{{ t('settings.tokensHelp') }}</p>
      </div>

      <UForm :state="form" class="flex flex-wrap items-end gap-3" @submit="create">
        <UFormField :label="t('settings.name')" name="name" required>
          <UInput v-model="form.name" required />
        </UFormField>
        <UFormField :label="t('settings.expiresIn')" name="expires_in">
          <USelect v-model="form.expires_in" :items="expiryItems" class="w-40" />
        </UFormField>
        <UCheckbox v-model="form.read_only" :label="t('settings.readOnly')" class="pb-2" />
        <UButton type="submit" icon="i-lucide-plus" :loading="creating" :label="t('settings.create')" />
      </UForm>

      <UAlert
        v-if="created"
        color="warning"
        variant="subtle"
        icon="i-lucide-key-round"
        :title="t('settings.newToken')"
        :description="t('settings.tokenOnce')"
      >
        <template #actions>
          <code class="break-all rounded bg-elevated px-2 py-1 text-sm">{{ created }}</code>
          <UButton icon="i-lucide-copy" :label="t('settings.copy')" color="neutral" @click="copyToken" />
        </template>
      </UAlert>

      <UTable :data="tokens ?? []" :columns="columns">
        <template #empty>{{ t('settings.noTokens') }}</template>
        <template #kind-cell="{ row }">{{ t(`settings.kinds.${row.original.kind}`) }}</template>
        <template #read_only-cell="{ row }">
          <UIcon v-if="row.original.read_only" name="i-lucide-check" :aria-label="t('settings.readOnly')" />
        </template>
        <template #created_at-cell="{ row }">{{ when(row.original.created_at) }}</template>
        <template #last_used_at-cell="{ row }">{{ when(row.original.last_used_at) }}</template>
        <template #expires_at-cell="{ row }">{{ when(row.original.expires_at) }}</template>
        <template #actions-cell="{ row }">
          <div class="flex justify-end">
            <UButton
              icon="i-lucide-trash-2"
              color="error"
              variant="ghost"
              :aria-label="t('settings.revoke')"
              @click="askRevoke(row.original)"
            />
          </div>
        </template>
      </UTable>
    </section>

    <ConfirmModal
      v-model:open="confirmOpen"
      :title="t('settings.revokeTitle')"
      :message="t('settings.revokeMessage', { name: revoking?.name ?? '' })"
      :confirm-label="t('settings.revoke')"
      @confirm="revoke"
    />
  </div>
</template>
