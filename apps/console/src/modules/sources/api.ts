export type EvidenceRow = {
  id: string;
  scope: string;
  provider: string;
  state: "ready" | "missing";
  period: string;
};
export type Query = {
  scope: string;
  filter: string;
  search: string;
  page: number;
  scenario: string;
};
export async function readSyntheticEvidence(
  q: Query,
  signal: AbortSignal,
): Promise<{ rows: EvidenceRow[]; total: number }> {
  await new Promise<void>((resolve, reject) => {
    const finish = () => {
      signal.removeEventListener("abort", abort);
      resolve();
    };
    const timer = setTimeout(finish, q.scope === "sample-a" ? 450 : 60);
    const abort = () => {
      clearTimeout(timer);
      reject(new DOMException("Aborted", "AbortError"));
    };
    if (signal.aborted) abort();
    else signal.addEventListener("abort", abort, { once: true });
  });
  if (q.scenario === "error") throw Error("synthetic-unavailable");
  const all =
    q.scenario === "empty"
      ? []
      : Array.from(
          { length: 57 },
          (_, i): EvidenceRow => ({
            id: `${q.scope}-REC-${String(i + 1).padStart(4, "0")}`,
            scope: q.scope,
            provider: ["AWS", "Azure", "GCP"][i % 3],
            state: i % 4 === 0 ? "missing" : "ready",
            period: "2026-10",
          }),
        );
  const found = all.filter(
    (r) =>
      (q.filter === "all" || r.state === q.filter) && r.id.includes(q.search),
  );
  return {
    rows: found.slice((q.page - 1) * 10, q.page * 10),
    total: found.length,
  };
}
export function latestRequest() {
  let epoch = 0;
  return {
    next: () => ++epoch,
    isCurrent: (value: number) => value === epoch,
    invalidate: () => {
      epoch++;
    },
  };
}
