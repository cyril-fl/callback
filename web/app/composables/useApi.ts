import type { FetchError, FetchOptions } from 'ofetch';

type ApiOptions = FetchOptions<'json'> & {
  /** The page shows the error itself: always, or for these statuses. */
  quiet?: boolean | number[];
};

/** Calls the Go API and tells the user when it fails; without a session, opens the login page. */
export function useApi() {
  const toast = useToast();
  const { t } = useI18n();
  const route = useRoute();
  const baseURL = useRuntimeConfig().public.apiBase;

  return async function api<T>(path: string, { quiet, ...options }: ApiOptions = {}): Promise<T> {
    try {
      // include: in development the API is on another port, and its session cookie must go along.
      return await $fetch<T>(path, { baseURL, credentials: 'include', ...options } as object);
    } catch (error) {
      const status = (error as FetchError).status;
      if (status === 401 && route.path !== '/login') {
        await navigateTo({ path: '/login', query: { redirect: route.fullPath } });
      } else if (quiet !== true && !(Array.isArray(quiet) && quiet.includes(status ?? 0))) {
        const message = status ? ((error as FetchError).data?.error ?? t('api.failed')) : t('api.down');
        toast.add({ title: message, color: 'error' });
      }
      throw error;
    }
  };
}
