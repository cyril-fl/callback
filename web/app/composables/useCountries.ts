import type { Country } from '~/types/items';

// The countries of the server's countries.yaml: an application stores the code, the app shows the translated name.
export function useCountries() {
  const { t } = useI18n();
  const api = useApi();
  const { data } = useAsyncData('countries', () => api<Country[]>('/api/countries'), { default: () => [] });

  const keys = computed(() => new Map(data.value.map((c) => [c.code, c.key])));
  // A code it does not know (the list not loaded yet) shows as is.
  const label = (code?: string) => {
    const key = code && keys.value.get(code);
    return key ? t(key) : (code ?? '');
  };
  const items = computed(() =>
    data.value.map((c) => ({ value: c.code, label: t(c.key) })).sort((a, b) => a.label.localeCompare(b.label, 'fr')),
  );
  return { items, label };
}
