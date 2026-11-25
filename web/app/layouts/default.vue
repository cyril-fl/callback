<script setup lang="ts">
const { t } = useI18n();
const { pro } = usePreferences();
// One word each; the pro mode's dashboard has a page of its own.
const links = computed(() => [
  ...(pro.value ? [{ to: '/dashboard', icon: 'i-lucide-chart-column', label: t('nav.dashboard') }] : []),
  { to: '/', icon: 'i-lucide-list', label: t('nav.applications') },
  { to: '/settings', icon: 'i-lucide-settings', label: t('nav.settings') },
]);
</script>

<template>
  <div class="min-h-dvh">
    <UHeader :title="t('app.name')" to="/">
      <UNavigationMenu :items="links" variant="link" />
      <template #right>
        <!-- A page's own actions, teleported here (the list's view, export and add). -->
        <div id="header-actions" class="flex items-center gap-2" />
      </template>
      <template #body>
        <UNavigationMenu :items="links" orientation="vertical" class="-mx-2.5" />
      </template>
    </UHeader>
    <UContainer class="py-6">
      <slot />
    </UContainer>
  </div>
</template>
