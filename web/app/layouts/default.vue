<script setup lang="ts">
const { t } = useI18n();
const logout = useLogout();
const plan = usePlan();
</script>

<template>
  <div class="min-h-dvh">
    <header class="border-b border-default">
      <UContainer class="flex h-14 items-center justify-between">
        <NuxtLink to="/" class="font-semibold">{{ t('app.name') }}</NuxtLink>
        <nav class="flex items-center gap-1">
          <UButton
            :to="plan === 'pro' ? '/dashboard' : '/'"
            variant="ghost"
            color="neutral"
            icon="i-lucide-layout-dashboard"
            :label="t('nav.dashboard')"
            :ui="{ label: 'max-sm:sr-only' }"
          />
          <UButton
            v-if="plan === 'pro'"
            to="/"
            variant="ghost"
            color="neutral"
            icon="i-lucide-list"
            :label="t('nav.applications')"
            :ui="{ label: 'max-sm:sr-only' }"
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
