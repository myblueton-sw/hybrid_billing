import type { FocusRow } from "./api";
export type Origin = "source" | "derived" | "lineage" | "annotation";
export const numericFields = [
  "billed_cost",
  "effective_cost",
  "list_cost",
  "contracted_cost",
  "consumed_quantity",
  "pricing_quantity",
  "list_unit_price",
  "contracted_unit_price",
] as const;
export type NumericField = (typeof numericFields)[number];
export const nullableFields = [
  "sub_account",
  "service",
  "consumed_unit",
  "pricing_unit",
  ...numericFields,
] as const;
export type NullableField = (typeof nullableFields)[number];
export const mappedFields = [
  "provider",
  "service",
  "currency",
  "billing_account",
  "sub_account",
  "consumed_unit",
  "pricing_unit",
  "billing_start",
  "billing_end",
  "charge_start",
  "charge_end",
  ...numericFields,
] as const;
export type MappedField = (typeof mappedFields)[number];
const focusColumns: Record<MappedField, string> = {
  provider: "ProviderName",
  service: "ServiceName",
  currency: "BillingCurrency",
  billing_account: "BillingAccountId",
  sub_account: "SubAccountId",
  consumed_unit: "ConsumedUnit",
  pricing_unit: "PricingUnit",
  billing_start: "BillingPeriodStart",
  billing_end: "BillingPeriodEnd",
  charge_start: "ChargePeriodStart",
  charge_end: "ChargePeriodEnd",
  billed_cost: "BilledCost",
  effective_cost: "EffectiveCost",
  list_cost: "ListCost",
  contracted_cost: "ContractedCost",
  consumed_quantity: "ConsumedQuantity",
  pricing_quantity: "PricingQuantity",
  list_unit_price: "ListUnitPrice",
  contracted_unit_price: "ContractedUnitPrice",
};
export const mappingVersions: Record<string, Record<MappedField, string>> = {
  "1.0": focusColumns,
  "1.2": focusColumns,
  "1.3": { ...focusColumns, provider: "ServiceProviderName" },
};
export const derivedFields = ["charge_month", "id", "account"] as const;
export const lineageFields = ["batch", "ordinal"] as const;
export const annotationFields = ["evidence_class"] as const;
export const annotationColumn = "x_HB_EvidenceClass";
export const provenanceClasses = [
  "official-public",
  "official-public-subset",
  "synthetic-contract-fixture",
  "unclassified",
] as const;
export type ProvenanceClass = (typeof provenanceClasses)[number];
export function provenanceClass(value: string): ProvenanceClass {
  return (provenanceClasses as readonly string[]).includes(value)
    ? (value as ProvenanceClass)
    : "unclassified";
}
export function mappedColumns(field: MappedField, versions: string[]) {
  const mapped = versions
    .filter((version) => version in mappingVersions)
    .map((version) => ({ version, name: mappingVersions[version][field] }));
  return {
    mapped,
    names: [...new Set(mapped.map((entry) => entry.name))],
    unmapped: versions.filter((version) => !(version in mappingVersions)),
  };
}
export type ValueState = "missing" | "zero" | "value";
export function valueState(coefficient: string | null): ValueState {
  if (coefficient === null) return "missing";
  return BigInt(coefficient) === 0n ? "zero" : "value";
}
export function rowVersion(versions: string[]) {
  return versions.length === 1 ? versions[0] : null;
}
export function missingFields(row: FocusRow): NullableField[] {
  return nullableFields.filter((field) => row[field] === null);
}
const utcInstant =
  /^(\d{4}-\d{2}-\d{2})T(\d{2}:\d{2}:\d{2})(?:\.(\d{1,6}))?\+00:00$/;
export function instantParts(instant: string) {
  const match = utcInstant.exec(instant);
  if (!match) return null;
  const fraction = match[3]?.replace(/0+$/, "");
  return {
    date: match[1],
    time: fraction ? `${match[2]}.${fraction}` : match[2],
  };
}
export function utcMonth(instant: string) {
  return instantParts(instant)?.date.slice(0, 7) ?? null;
}
export function chargeMonthBasis(
  row: Pick<FocusRow, "charge_month" | "charge_start">,
  tenants: { timezone: string }[],
) {
  const month = utcMonth(row.charge_start),
    timezones = [...new Set(tenants.map((t) => t.timezone))].sort();
  return {
    utcMonth: month,
    differs: month !== null && month !== row.charge_month,
    timezones,
    rowTimezone: timezones.length === 1 ? timezones[0] : null,
  };
}
export function chargeStartsInBillingPeriod(
  row: Pick<FocusRow, "charge_start" | "billing_start" | "billing_end">,
) {
  const charge = microseconds(row.charge_start),
    start = microseconds(row.billing_start),
    end = microseconds(row.billing_end);
  if (charge === null || start === null || end === null) return null;
  return start <= charge && charge < end;
}
function microseconds(instant: string) {
  const match = utcInstant.exec(instant);
  if (!match) return null;
  const milliseconds = Date.parse(`${match[1]}T${match[2]}Z`);
  if (Number.isNaN(milliseconds)) return null;
  return (
    BigInt(milliseconds) * 1000n +
    BigInt((match[3] ?? "").padEnd(6, "0") || "0")
  );
}
