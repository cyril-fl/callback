<script setup lang="ts">
import { fr } from '@nuxt/ui/locale';

const { t } = useI18n();
useHead({ titleTemplate: (title) => (title ? `${t('app.name')} - ${title}` : t('app.name')) });
</script>

<template>
  <UApp :locale="fr" :toaster="{ position: 'bottom-right' }">
    <NuxtLayout>
      <NuxtPage />
    </NuxtLayout>
  </UApp>
</template>
