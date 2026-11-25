import type { PreferenceChanges, Preferences } from '~/types/items';

/**
 * The preferences before a page renders, kept as `preferences`: the auth middleware loads them with the session.
 * Not useApi, as loadMe: a route middleware has no component for its useI18n.
 */
export async function loadPreferences() {
  const cached = useNuxtData<Preferences>('preferences').data;
  cached.value ??= await $fetch<Preferences>('/api/preferences', {
    baseURL: useRuntimeConfig().public.apiBase,
    credentials: 'include',
  }).catch(() => undefined);
  return cached.value;
}

/** The preferences of the account, pro for the pro plan, and save to change some. */
export function usePreferences() {
  const api = useApi();
  const preferences = useNuxtData<Preferences>('preferences').data;
  const pro = computed(() => preferences.value?.plan === 'pro');
  async function save(changes: PreferenceChanges) {
    try {
      preferences.value = await api<Preferences>('/api/preferences', { method: 'PATCH', body: changes });
    } catch {
      // useApi already toasted.
    }
  }
  return { preferences, pro, save };
}
