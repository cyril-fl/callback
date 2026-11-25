<script setup lang="ts">
import type { Token, TokenInput, TokenResult } from '~/types/items';

const { t } = useI18n();
const api = useApi();
const toast = useToast();
const theme = useTheme();
const colorMode = useColorMode();
const mode = computed({
  get: () => colorMode.preference as 'light' | 'dark' | 'system',
  set: (v: 'light' | 'dark' | 'system') => (colorMode.preference = v),
});

const modes = ['light', 'dark', 'system'] as const;
const modeIcons = { light: 'i-lucide-sun', dark: 'i-lucide-moon', system: 'i-lucide-monitor' };
const modeItems = modes.map((value) => ({ value, label: t(`settings.${value}`), icon: modeIcons[value] }));

const { data: tokens, refresh } = await useAsyncData('tokens', () => api<Token[]>('/api/tokens'));

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
  <div class="mx-auto max-w-3xl space-y-10">
    <h1 class="text-xl font-semibold">{{ t('settings.title') }}</h1>

    <section>
      <h2 class="mb-3 text-sm font-semibold">{{ t('settings.appearance') }}</h2>
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
          <UTabs
            v-model="mode"
            :items="modeItems"
            :content="false"
            size="sm"
            :ui="{ label: 'max-sm:sr-only' }"
            :aria-label="t('settings.colorMode')"
          />
        </li>
      </ul>
    </section>

    <section>
      <h2 class="mb-1 text-sm font-semibold">{{ t('settings.tokens') }}</h2>
      <p class="mb-3 text-xs text-muted">{{ t('settings.tokensHelp') }}</p>

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
          icon="i-lucide-key-round"
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
              icon="i-lucide-copy"
              :aria-label="t('settings.copy')"
              color="neutral"
              variant="link"
              size="sm"
              @click="copyToken"
            />
          </template>
        </UInput>
      </div>

      <ul v-if="tokens?.length" class="mt-3 divide-y divide-default rounded-lg border border-default">
        <li v-for="token in tokens" :key="token.id" class="flex flex-wrap items-center gap-3 px-4 py-2">
          <UIcon name="i-lucide-key-round" class="size-4 shrink-0 text-muted" />
          <div class="min-w-0 flex-1 basis-48">
            <p class="flex min-w-0 items-center gap-2 text-sm">
              <span class="truncate">{{ token.name }}</span>
              <UBadge :label="t(`settings.kinds.${token.kind}`)" size="sm" color="neutral" variant="subtle" />
              <UBadge
                v-if="token.read_only"
                :label="t('settings.readOnly')"
                size="sm"
                color="neutral"
                variant="subtle"
              />
            </p>
            <p class="text-xs text-muted">
              {{ t('settings.createdAt') }} {{ when(token.created_at) }} · {{ t('settings.lastUsedAt') }}
              {{ when(token.last_used_at) }} · {{ t('settings.expiresAt') }} {{ when(token.expires_at) }}
            </p>
          </div>
          <UButton
            icon="i-lucide-trash-2"
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

    <ConfirmModal
      v-model:open="confirmOpen"
      :title="t('settings.revokeTitle')"
      :message="t('settings.revokeMessage', { name: revoking?.name ?? '' })"
      :confirm-label="t('settings.revoke')"
      @confirm="revoke"
    />
  </div>
</template>
