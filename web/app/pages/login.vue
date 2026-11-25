<script setup lang="ts">
definePageMeta({ layout: 'blank' });

const { t } = useI18n();
const api = useApi();
const route = useRoute();
const password = ref('');
const loading = ref(false);

async function submit() {
  loading.value = true;
  try {
    await api('/api/login', { method: 'POST', body: { password: password.value } });
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
      <UFormField :label="t('login.password')">
        <UInput v-model="password" type="password" autocomplete="current-password" autofocus class="w-full" required />
      </UFormField>
      <UButton type="submit" block :loading="loading" :label="t('login.submit')" />
    </form>
  </UCard>
</template>
