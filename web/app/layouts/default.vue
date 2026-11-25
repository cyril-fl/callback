<script setup lang="ts">
const { t } = useI18n();
const logout = useLogout();
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
    <header class="border-b border-default">
      <UContainer class="flex h-14 items-center justify-between">
        <nav class="flex items-center gap-1">
          <NuxtLink to="/" class="mr-3 font-semibold">{{ t('app.name') }}</NuxtLink>
          <UButton
            v-for="link in links"
            :key="link.to"
            :to="link.to"
            :icon="link.icon"
            :label="link.label"
            :ui="{ label: 'max-sm:sr-only' }"
            variant="ghost"
            color="neutral"
            active-variant="soft"
          />
          <USeparator orientation="vertical" class="mx-1 h-5" />
          <UTooltip :text="t('nav.logout')">
            <UButton
              variant="ghost"
              color="neutral"
              icon="i-lucide-log-out"
              :aria-label="t('nav.logout')"
              @click="logout"
            />
          </UTooltip>
        </nav>
        <!-- A page's own actions, teleported here (the list's view, export and add). -->
        <div id="header-actions" class="flex items-center gap-2" />
      </UContainer>
    </header>
    <UContainer class="py-6">
      <slot />
    </UContainer>
  </div>
</template>
