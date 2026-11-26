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
  <div class="flex h-dvh flex-col">
    <!-- Three icons fit any width: no burger menu. -->
    <UHeader :toggle="false" class="shrink-0">
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
        <div id="header-actions" class="flex items-center gap-2" />
      </template>
    </UHeader>
    <!-- Only the content scrolls: the header and its toolbar stay put. -->
    <main class="min-h-0 flex-1 overflow-y-auto">
      <UContainer class="py-6">
        <slot />
      </UContainer>
    </main>
  </div>
</template>
