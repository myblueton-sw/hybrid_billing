import type { Locale } from "../../../../shared/types";
import { money as sharedMoney } from "../../../sources/collected-focus/values";
import type { Charge } from "./contract";
export const money = (value: string, currency: string, locale: Locale) =>
  sharedMoney(value, currency, locale);
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
