import type { Application } from '~/types/items';

// The cities and companies the applications name, most used first: the form and the settings suggest them.
// shortcut: reads every application's address, an API route like /api/sources once there are thousands.
export function usePlaces() {
  const api = useApi();
  const { data, refresh } = useAsyncData(
    'places',
    () =>
      api<Application[]>('/api/applications', {
        query: { fields: 'company,location,street,postal_code,country,website,applied_at' },
      }),
    { default: () => [] },
  );
  const byUse = (values: (string | undefined)[]) => {
    const counts = new Map<string, number>();
    for (const v of values) if (v) counts.set(v, (counts.get(v) ?? 0) + 1);
    return [...counts.keys()].sort((a, b) => counts.get(b)! - counts.get(a)!);
  };
  const cities = computed(() => byUse(data.value.map((a) => a.location)));
  const companies = computed(() => byUse(data.value.map((a) => a.company)));
  // The list comes newest first: a city's country is the one it had last.
  const countryOf = (city: string) => data.value.find((a) => a.location === city && a.country)?.country;
  // The newest application to company, but the one excluded (the one being edited).
  const lastTo = (company: string, excluded?: string) =>
    data.value.find((a) => a.company === company && a.id !== excluded);
  return { applications: data, cities, companies, countryOf, lastTo, refresh };
}
