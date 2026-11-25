import type { Application } from '~/types/items';

// The cities and countries the applications name, most used first: the form and the settings suggest them.
// shortcut: reads every application's address, an API route like /api/sources once there are thousands.
export function usePlaces() {
  const api = useApi();
  const { data, refresh } = useAsyncData(
    'places',
    () => api<Application[]>('/api/applications', { query: { fields: 'location,country' } }),
    { default: () => [] },
  );
  const byUse = (values: (string | undefined)[]) => {
    const counts = new Map<string, number>();
    for (const v of values) if (v) counts.set(v, (counts.get(v) ?? 0) + 1);
    return [...counts.keys()].sort((a, b) => counts.get(b)! - counts.get(a)!);
  };
  const cities = computed(() => byUse(data.value.map((a) => a.location)));
  const countries = computed(() => byUse(data.value.map((a) => a.country)));
  // The list comes newest first: a city's country is the one it had last.
  const countryOf = (city: string) => data.value.find((a) => a.location === city && a.country)?.country;
  return { cities, countries, countryOf, refresh };
}
