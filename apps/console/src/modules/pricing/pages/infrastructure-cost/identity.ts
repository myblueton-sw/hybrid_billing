import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import type { Basis, FocusRow } from "../../../sources/collected-focus/api";
import { valueState } from "../../../sources/collected-focus/fields";
import {
  missingLabel,
  zeroLabel,
} from "../../../sources/collected-focus/labels";
import { money } from "../../../sources/collected-focus/values";
import { basisLabel } from "./copy";
export type IdentityRow = Pick<
  FocusRow,
  "id" | "billing_account" | "sub_account" | "ordinal"
>;
const ltr = (value: string) => `\u2066${value}\u2069`;
export function shortAccount(value: string) {
  const characters = [...value];
  return characters.length > 28
    ? `${characters.slice(0, 12).join("")}…${characters.slice(-12).join("")}`
    : value;
}
export function accountToken(row: IdentityRow) {
  return row.sub_account === null
    ? { kind: "billing_account" as const, value: row.billing_account }
    : { kind: "sub_account" as const, value: row.sub_account };
}
export function identityText(l: Locale, row: IdentityRow) {
  const account = accountToken(row);
  return [
    `${t(l, "Ordinal", "순번", "الترتيب")} ${ltr(row.ordinal)}`,
    `${
      account.kind === "sub_account"
        ? t(l, "Subaccount", "하위 계정", "الحساب الفرعي")
        : t(l, "Billing account", "결제 계정", "حساب الفوترة")
    } ${ltr(shortAccount(account.value))}`,
    `${t(l, "Row ID", "행 ID", "معرّف الصف")} ${ltr(row.id.slice(0, 8))}`,
  ].join(" · ");
}
export function amountText(
  l: Locale,
  basis: Basis,
  row: Pick<FocusRow, Basis | "currency">,
) {
  const coefficient = row[basis],
    state = valueState(coefficient),
    label = basisLabel(l, basis);
  if (state === "missing") return `${label}: ${missingLabel(l)}`;
  const amount = ltr(money(coefficient, row.currency, l));
  return state === "zero"
    ? `${label}: ${amount} (${zeroLabel(l)})`
    : `${label}: ${amount}`;
}
export const optionText = (
  l: Locale,
  basis: Basis,
  row: IdentityRow & Pick<FocusRow, Basis | "currency">,
) => `${identityText(l, row)} · ${amountText(l, basis, row)}`;
export function deliveryCount(rows: Pick<FocusRow, "batch">[]) {
  return new Set(rows.map((row) => row.batch)).size;
}
