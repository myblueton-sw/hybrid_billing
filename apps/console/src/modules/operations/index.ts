export type WorkRecord = {
  id: string;
  scope: string;
  customer: string;
  account: string;
  contract: string;
  provider: string;
  status: "missing" | "ready" | "processing";
  owner: string;
  due: string;
  period: string;
  currency: string;
  source: string;
};
export function workRecords(scope: string): WorkRecord[] {
  return Array.from({ length: 57 }, (_, i) => ({
    id: `${scope}-CASE-${String(i + 1).padStart(4, "0")}`,
    scope,
    customer: `DEMO-CUSTOMER-${String(i + 1).padStart(3, "0")}`,
    account: scope === "sample-b" ? "ACC-000002" : "ACC-000001",
    contract: `DEMO-CONTRACT-${String(i + 1).padStart(3, "0")}`,
    provider: ["AWS", "Azure", "GCP"][i % 3],
    status: i % 4 === 0 ? "missing" : i % 4 === 1 ? "processing" : "ready",
    owner: `DEMO-PERSON-${i % 2 ? "B" : "A"}`,
    due: "2026-11-10",
    period: "2026-10",
    currency: ["USD", "AED", "KRW"][i % 3],
    source: `${scope}-REC-${String(i + 1).padStart(7, "0")}`,
  }));
}
export function findWorkRecord(
  scope: string,
  id: string | null,
  period?: string | null,
) {
  return id
    ? workRecords(scope).find(
        (r) => r.id === id && (!period || r.period === period),
      )
    : undefined;
}
