import type { Locale } from "../../../../shared/types";
import { decimal } from "../../../sources/collected-focus/values";
import type { Charge } from "./contract";
export function money(value: string, currency: string, locale: Locale) {
  const n = BigInt(value);
  return `${n < 0n ? "−" : ""}${currency} ${decimal((n < 0n ? -n : n).toString(), locale)}`;
}
export function rowStatus(row: Charge): "used" | "unused" | "unknown" {
  if (row.version !== "1.0" || row.fields.status.state !== "value")
    return "unknown";
  return row.fields.status.value === "Used"
    ? "used"
    : row.fields.status.value === "Unused"
      ? "unused"
      : "unknown";
}
export const range = (after: number, length: number, total: number) =>
  `${length ? after + 1 : 0}–${after + length} / ${total}`;
