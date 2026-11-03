export const asOf = "2026-11-03T12:00:00Z";
export const cyclePeriods = ["2026-09", "2026-10"] as const;
export const states = [
  "incomplete",
  "blocked",
  "held",
  "exempt",
  "complete",
] as const;
export const stages = [
  "collection",
  "allocation",
  "calculation",
  "approval",
  "issuance",
  "delivery",
  "accounting",
] as const;
export type CloseState = (typeof states)[number];
export type Stage = (typeof stages)[number];
export type Target = {
  id: string;
  organization: string;
  contract: string;
  entity: string;
  owner: string;
  period: string;
  stage: Stage;
  state: CloseState;
  internal: boolean;
  original: string;
  current: string;
  schedule: string;
  revision: string;
  calendar: string;
  erpRequired: boolean;
  erp: "confirmed" | "unknown" | "na";
};
export type Filter = {
  query: string;
  owner: string;
  stage: string;
  state: string;
  path: string;
  overdue: boolean;
};
export const emptyFilter: Filter = {
  query: "",
  owner: "",
  stage: "",
  state: "",
  path: "",
  overdue: false,
};
export function deadlineMonth(period: string) {
  return period === "2026-09" ? "2026-10" : "2026-11";
}
export function fixtureTargets(scope: string, period: string): Target[] {
  if (!cyclePeriods.includes(period as (typeof cyclePeriods)[number]))
    return [];
  return Array.from({ length: 1200 }, (_, i) => {
    const day = (i % 26) + 1,
      month = deadlineMonth(period),
      state = states[i % 5],
      erpRequired = i % 3 === 0 || state === "blocked";
    return {
      id: `${scope}:CYCLE:${period}:${String(i + 1).padStart(6, "0")}`,
      organization: `ORG-${String((i % 240) + 1).padStart(4, "0")}`,
      contract: `CONTRACT-${String(i + 1).padStart(6, "0")}`,
      entity: `ENTITY-${(i % 12) + 1}`,
      owner: `OWNER-${(i % 8) + 1}`,
      period,
      stage:
        i % 2 === 0
          ? (["collection", "calculation", "approval", "accounting"] as const)[
              Math.floor(i / 2) % 4
            ]
          : stages[i % stages.length],
      state,
      internal: i % 2 === 0,
      original: `${month}-${String(day).padStart(2, "0")}T09:00:00Z`,
      current: `${month}-${String(day + (i % 7 === 0 ? 2 : 0)).padStart(2, "0")}T09:00:00Z`,
      schedule: `SCHEDULE-${String(i + 1).padStart(6, "0")}`,
      revision: i % 7 === 0 ? "r2" : "r1",
      calendar: "DEMO-CALENDAR-UNCONFIGURED",
      erpRequired,
      erp: state === "blocked" ? "unknown" : erpRequired ? "confirmed" : "na",
    };
  });
}
export function isOverdue(row: Target) {
  return (
    row.state !== "complete" && row.state !== "exempt" && row.current < asOf
  );
}
export function filterTargets(rows: Target[], filter: Filter) {
  const q = filter.query.trim().toLowerCase();
  return rows
    .filter(
      (r) =>
        (!q ||
          `${r.organization} ${r.contract} ${r.entity}`
            .toLowerCase()
            .includes(q)) &&
        (!filter.owner || r.owner === filter.owner) &&
        (!filter.stage || r.stage === filter.stage) &&
        (!filter.state || r.state === filter.state) &&
        (!filter.path ||
          (filter.path === "internal" && r.internal) ||
          (filter.path === "external" && !r.internal)) &&
        (!filter.overdue || isOverdue(r)),
    )
    .sort(
      (a, b) => a.current.localeCompare(b.current) || a.id.localeCompare(b.id),
    );
}
export function dateCounts(rows: Target[]) {
  const counts = new Map<string, number>();
  for (const row of rows) {
    const date = row.current.slice(0, 10);
    counts.set(date, (counts.get(date) ?? 0) + 1);
  }
  return counts;
}
