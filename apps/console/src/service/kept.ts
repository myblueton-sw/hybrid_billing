export type KeptFacts = { total: number; next: string; kept: number; dropped: number };
export type KeptPage<T> = { rows: T[]; next: string; matching: number; history: KeptFacts };

const whole = (x: unknown): number => (typeof x === "number" && Number.isInteger(x) && x >= 0 ? x : 0);

export function keptFactsOf(v: unknown): KeptFacts {
  const c = (v && typeof v === "object" && !Array.isArray(v) ? v : {}) as Record<string, unknown>;
  return { total: whole(c.total), next: typeof c.next === "string" ? c.next : "", kept: whole(c.kept), dropped: whole(c.dropped) };
}

export function keptPageOf<T>(body: Record<string, unknown>, row: (r: Record<string, unknown>) => T): KeptPage<T> | null {
  if (!Array.isArray(body.proposals) || typeof body.next !== "string") return null;
  const rows = body.proposals.filter((r): r is Record<string, unknown> => !!r && typeof r === "object" && !Array.isArray(r)).map(row);
  return { rows, next: body.next, matching: whole(body.matching), history: keptFactsOf(body.proposal_history) };
}
