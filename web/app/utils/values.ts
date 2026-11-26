/** The values a field takes across items, each with the ids that carry it, most used first, then by name. */
export function groupValues<T extends { id: string }>(items: T[], key: keyof T) {
  const groups = new Map<string, string[]>();
  for (const item of items) {
    const value = item[key];
    if (typeof value === 'string' && value) groups.set(value, [...(groups.get(value) ?? []), item.id]);
  }
  return [...groups]
    .map(([value, ids]) => ({ value, ids }))
    .sort((a, b) => b.ids.length - a.ids.length || a.value.localeCompare(b.value));
}
