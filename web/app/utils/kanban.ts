// A card dropped at index of a column (index counted among the column's cards before the drop, the dragged one
// included when it was there): the column's ids in their new order, or undefined when nothing moves.
export function reorder(ids: string[], moved: string, index: number): string[] | undefined {
  const from = ids.indexOf(moved);
  const rest = ids.filter((id) => id !== moved);
  const at = from >= 0 && from < index ? index - 1 : index;
  if (at === from) return undefined;
  return [...rest.slice(0, at), moved, ...rest.slice(at)];
}
