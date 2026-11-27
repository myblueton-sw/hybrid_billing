export const sizeChoices = [10, 25, 50, 100] as const;

export function pageSizeOptions(limit: number | null, current?: number): number[] {
  if (limit === null) return [];
  const out = new Set<number>(sizeChoices.filter((n) => n <= limit));
  out.add(limit);
  if (current !== undefined && current >= 1 && current <= limit) out.add(current);
  return [...out].sort((a, b) => a - b);
}

export function sizeSelectShown(limit: number | null, current?: number): boolean {
  return pageSizeOptions(limit, current).length > 1;
}

export function lowered(requested: number, limit: number | null): boolean {
  return limit !== null && requested > limit;
}

export function nextFocusAfterRemoval(keys: string[], removed: string): string | null {
  const at = keys.indexOf(removed);
  if (at < 0) return null;
  return keys[at + 1] ?? keys[at - 1] ?? null;
}
