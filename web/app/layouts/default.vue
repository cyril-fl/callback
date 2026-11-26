<script setup lang="ts">
const { t } = useI18n();
const { icons } = useIcons();
const { pro } = usePreferences();
// One word each; the pro mode's dashboard has a page of its own.
const links = computed(() => [
  ...(pro.value ? [{ to: '/dashboard', icon: icons.dashboard, label: t('nav.dashboard') }] : []),
  { to: '/', icon: icons.applications, label: t('nav.applications') },
  { to: '/settings', icon: icons.settings, label: t('nav.settings') },
]);
</script>

<template>
  <div class="min-h-dvh">
    <!-- Three icons fit any width: no burger menu. -->
    <UHeader :toggle="false" :ui="{ left: 'shrink-0', right: 'min-w-0' }">
      <template #left>
        <NuxtLink to="/" class="mr-3 font-semibold">{{ t('app.name') }}</NuxtLink>
        <UTooltip v-for="link in links" :key="link.to" :text="link.label">
          <UButton
            :to="link.to"
            :icon="link.icon"
            :aria-label="link.label"
            variant="ghost"
            color="neutral"
            active-variant="soft"
          />
        </UTooltip>
      </template>
      <template #right>
        <!-- A page's own actions, teleported here (the list's view, export and add). -->
        <!-- Too wide for a phone: the actions scroll sideways, not the page. -->
        <div id="header-actions" class="flex min-w-0 items-center gap-2 overflow-x-auto *:shrink-0" />
      </template>
    </UHeader>
    <UContainer class="py-6">
      <slot />
    </UContainer>
  </div>
</template>
