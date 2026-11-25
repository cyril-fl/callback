import type { Me } from '~/types/items';

/**
 * The session before a page renders, kept as `me`; none without a session.
 * Not useApi: its useI18n needs a component, which a route middleware has not.
 */
export async function loadMe() {
  const cached = useNuxtData<Me>('me').data;
  cached.value ??= await $fetch<Me>('/api/me', {
    baseURL: useRuntimeConfig().public.apiBase,
    credentials: 'include',
  }).catch(() => undefined);
  return cached.value;
}
