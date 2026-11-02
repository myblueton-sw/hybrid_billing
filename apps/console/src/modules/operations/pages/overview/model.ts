export const months = [
  "2026-05",
  "2026-06",
  "2026-07",
  "2026-08",
  "2026-09",
  "2026-10",
];
export const sources = ["AWS", "Azure", "GCP", "On-prem"] as const;
export const organizations = ["A", "B", "C"] as const;
export const stages = [
  "collection",
  "reconciliation",
  "allocation",
  "calculation",
  "approval",
  "issuance",
  "accounting",
  "delivery",
  "receipt",
] as const;
export type Currency = "USD" | "KRW";
export type Stage = (typeof stages)[number];
export type Status = "complete" | "pending" | "blocked" | "unknown" | "na";
export type Cost = {
  id: string;
  workspace: string;
  month: string;
  source: string;
  organization: string;
  currency: Currency;
  minor: bigint | null;
};
export type Closing = {
  id: string;
  workspace: string;
  month: string;
  organization: string;
  sources: string[];
  currency: Currency;
  internal: boolean;
  status: Record<Stage, Status>;
};
export const snapshot = "DEMO-2026-11-02";
export function fixtureCosts(workspace: string): Cost[] {
  return months.flatMap((month, m) =>
    organizations.flatMap((organization, o) =>
      sources.map((source, s) => ({
        id: `${workspace}:${month}:${organization}:${source}`,
        workspace,
        month,
        organization,
        source,
        currency: s === 3 ? ("KRW" as const) : ("USD" as const),
        minor:
          m === 5 && o === 1 && s === 2
            ? null
            : o === 2 && s === 3
              ? 0n
              : BigInt(
                  (s === 3 ? 2800000 : 120000 + s * 40000) +
                    m * (s === 3 ? 80000 : 9000) +
                    o * (s === 3 ? 500000 : 23000),
                ) * (workspace === "sample-b" ? 2n : 1n),
      })),
    ),
  );
}
export function selectCosts(
  costs: Cost[],
  selectedSources: string[],
  selectedOrganizations: string[],
  month?: string,
) {
  return costs.filter(
    (r) =>
      selectedSources.includes(r.source) &&
      selectedOrganizations.includes(r.organization) &&
      (!month || r.month === month),
  );
}
export function total(costs: Cost[], currency: Currency) {
  const rows = costs.filter((r) => r.currency === currency);
  const known = rows.filter((r) => r.minor !== null);
  return {
    value: known.length ? known.reduce((sum, r) => sum + r.minor!, 0n) : null,
    known: known.length,
    expected: rows.length,
    missing: rows.length - known.length,
  };
}
export function formatMoney(
  value: bigint | null,
  currency: Currency,
  locale: string,
) {
  if (value === null) return "—";
  const scale = currency === "USD" ? 100n : 1n;
  const magnitude = value < 0n ? -value : value;
  const whole = new Intl.NumberFormat(locale, {
    maximumFractionDigits: 0,
  }).format(magnitude / scale);
  return `${currency} ${value < 0n ? "-" : ""}${whole}${scale === 100n ? "." + (magnitude % scale).toString().padStart(2, "0") : ""}`;
}
export function fixtureClosings(workspace: string, month: string): Closing[] {
  if (!months.includes(month)) return [];
  return organizations.flatMap((organization, o) =>
    [false, true].map((internal) => {
      const status = Object.fromEntries(
        stages.map((s) => [s, "complete"]),
      ) as Record<Stage, Status>;
      if (internal) {
        status.issuance = "na";
        status.delivery = "na";
        status.receipt = "na";
      }
      if (month === months.at(-1)) {
        if (!internal && o === 1) {
          status.collection = "blocked";
          status.reconciliation = "blocked";
          for (const stage of stages.slice(2)) status[stage] = "pending";
        } else if (internal) {
          status.approval = "pending";
          status.accounting = "pending";
        } else {
          status.accounting = "unknown";
          status.receipt = "pending";
        }
      }
      return {
        id: `${workspace}:${month}:${organization}:${internal ? "internal" : "external"}`,
        workspace,
        month,
        organization,
        sources: internal ? ["On-prem"] : ["AWS", "Azure", "GCP"],
        currency: internal ? ("KRW" as const) : ("USD" as const),
        internal,
        status,
      };
    }),
  );
}
export function selectClosings(
  rows: Closing[],
  selectedSources: string[],
  selectedOrganizations: string[],
) {
  return rows.filter(
    (r) =>
      selectedOrganizations.includes(r.organization) &&
      r.sources.some((s) => selectedSources.includes(s)),
  );
}
