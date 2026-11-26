<script setup lang="ts">
import type { ServerMode } from '~/types/items';

definePageMeta({ layout: 'blank' });

const { t } = useI18n();
const api = useApi();
const route = useRoute();
const { data: server } = await useAsyncData('add-user', () => api<ServerMode>('/api/add-user'));
const invite = typeof route.query.invite === 'string' ? route.query.invite : '';
// The API refuses a sign-up in the single mode, and without an invitation in the invitation mode.
const closed = computed(() => {
  if (!server.value) return undefined;
  if (server.value.mode === 'single') return t('signup.single');
  if (server.value.add_user_mode === 'invitation' && !invite) return t('signup.inviteNeeded');
  return undefined;
});
const name = ref('');
const password = ref('');
const loading = ref(false);

async function submit() {
  loading.value = true;
  try {
    await api('/api/signup', {
      method: 'POST',
      body: { name: name.value, password: password.value, ...(invite && { invite }) },
    });
    clearNuxtData('me');
    await navigateTo('/');
  } catch {
    // useApi already toasted.
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <UCard class="w-full max-w-sm">
    <template #header>
      <h1 class="text-lg font-semibold">{{ t('signup.title') }}</h1>
    </template>
    <p v-if="closed" class="text-sm text-muted">{{ closed }}</p>
    <form v-else class="space-y-4" @submit.prevent="submit">
      <UFormField :label="t('login.name')">
        <UInput v-model="name" autocomplete="username" autofocus class="w-full" required />
      </UFormField>
      <UFormField :label="t('login.password')" :help="t('signup.passwordHelp')">
        <UInput v-model="password" type="password" autocomplete="new-password" minlength="8" class="w-full" required />
      </UFormField>
      <UButton type="submit" block :loading="loading" :label="t('signup.submit')" />
    </form>
    <template #footer>
      <p class="text-center text-sm text-muted">
        {{ t('signup.haveAccount') }}
        <ULink to="/login" class="text-primary">{{ t('login.submit') }}</ULink>
      </p>
    </template>
  </UCard>
</template>
