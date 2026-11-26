<script setup lang="ts">
import { fr } from '@nuxt/ui/locale';

const { t } = useI18n();
// The theme picked in the settings sets the colors of Nuxt UI.
const theme = useTheme();
const appConfig = useAppConfig();
watchEffect(() => {
  const { id: _, ...colors } = themes.find((t) => t.id === theme.value) ?? themes[0]!;
  Object.assign(appConfig.ui.colors, colors);
});
// The color mode too is a preference of the account; without a session the device's stays.
const { preferences } = usePreferences();
const colorMode = useColorMode();
watchEffect(() => {
  if (preferences.value?.color_mode) colorMode.preference = preferences.value.color_mode;
});
useHead({ titleTemplate: (title) => (title ? `${t('app.name')} - ${title}` : t('app.name')) });
</script>

<template>
  <UApp :locale="fr" :toaster="{ position: 'bottom-right' }">
    <NuxtLayout>
      <NuxtPage />
    </NuxtLayout>
  </UApp>
</template>
