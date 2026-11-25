<script setup lang="ts">
const { t } = useI18n();
const logout = useLogout();
</script>

<template>
  <div class="min-h-dvh">
    <header class="border-b border-default">
      <UContainer class="flex h-14 items-center justify-between">
        <NuxtLink to="/" class="font-semibold">{{ t('app.name') }}</NuxtLink>
        <nav class="flex items-center gap-1">
          <UButton
            to="/"
            variant="ghost"
            color="neutral"
            icon="i-lucide-layout-dashboard"
            :label="t('nav.dashboard')"
          />
          <UButton
            to="/settings"
            variant="ghost"
            color="neutral"
            icon="i-lucide-settings"
            :aria-label="t('nav.settings')"
          />
          <UButton
            variant="ghost"
            color="neutral"
            icon="i-lucide-log-out"
            :aria-label="t('nav.logout')"
            @click="logout"
          />
        </nav>
      </UContainer>
    </header>
    <UContainer class="py-6">
      <slot />
    </UContainer>
  </div>
</template>
