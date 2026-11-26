<script setup lang="ts">
import type { ServerMode } from '~/types/items';

definePageMeta({ layout: 'blank' });

const { t } = useI18n();
const api = useApi();
const route = useRoute();
const { data: server } = await useAsyncData('add-user', () => api<ServerMode>('/api/add-user'));
const multi = computed(() => server.value?.mode === 'multi');
const name = ref('');
const password = ref('');
const loading = ref(false);

async function submit() {
  loading.value = true;
  try {
    const body = multi.value ? { name: name.value, password: password.value } : { password: password.value };
    await api('/api/login', { method: 'POST', body });
    clearNuxtData('me');
    const redirect = route.query.redirect;
    await navigateTo(typeof redirect === 'string' && redirect.startsWith('/') ? redirect : '/');
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
      <h1 class="text-lg font-semibold">{{ t('login.title') }}</h1>
    </template>
    <form class="space-y-4" @submit.prevent="submit">
      <UFormField v-if="multi" :label="t('login.name')">
        <UInput v-model="name" autocomplete="username" autofocus class="w-full" required />
      </UFormField>
      <UFormField :label="t('login.password')">
        <UInput
          v-model="password"
          type="password"
          autocomplete="current-password"
          :autofocus="!multi"
          class="w-full"
          required
        />
      </UFormField>
      <UButton type="submit" block :loading="loading" :label="t('login.submit')" />
    </form>
    <template v-if="multi && server?.add_user_mode === 'public'" #footer>
      <p class="text-center text-sm text-muted">
        {{ t('login.noAccount') }}
        <ULink to="/signup" class="text-primary">{{ t('signup.title') }}</ULink>
      </p>
    </template>
  </UCard>
</template>
