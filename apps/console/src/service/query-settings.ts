import { AdminError, type AccessSetting } from "./admin";

export const pageLimitName = "service_query_page_size";

export function pageLimitOf(rows: AccessSetting[]): number | null {
  const row = rows.find((x) => x.name === pageLimitName);
  return row && Number.isSafeInteger(row.value) && row.value >= 1 && row.value <= 100 ? row.value : null;
}

export function sizeFor(wanted: number, limit: number | null): number | undefined {
  return limit === null ? undefined : Math.min(wanted, limit);
}

export function sized(wanted: number, limit: number | null): { limit: number } | Record<string, never> {
  const size = sizeFor(wanted, limit);
  return size === undefined ? {} : { limit: size };
}

export function sizeRefused(error: unknown): boolean {
  return error instanceof AdminError && error.code === "invalid_request" && error.fields.includes("limit");
}
