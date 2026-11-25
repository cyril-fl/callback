import type { SourceCount } from '~/types/items';
import { SOURCES } from '~/types/items';

// The sources to offer: the three known ones (translated), then those the applications use, most used first.
export function useSources() {
  const { t } = useI18n();
  const api = useApi();
  const { data, refresh } = useAsyncData('sources', () => api<SourceCount[]>('/api/sources'), { default: () => [] });

  const known = new Set<string>(SOURCES);
  const label = (source?: string) => (!source ? '' : known.has(source) ? t(`source.${source}`) : source);
  const items = computed(() => [
    ...SOURCES.map((value) => ({ value: value as string, label: label(value) })),
    ...data.value.filter((s) => !known.has(s.source)).map((s) => ({ value: s.source, label: s.source })),
  ]);
  return { items, label, refresh, isKnown: (source?: string) => !!source && known.has(source) };
}
