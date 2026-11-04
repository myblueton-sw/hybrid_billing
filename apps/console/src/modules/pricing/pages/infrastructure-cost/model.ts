import type {
  FocusRow,
  OverviewData,
  Measure,
} from "../../../sources/collected-focus/api";
export const categories = [
  "unclassified",
  "asset",
  "lease",
  "depreciation",
  "license",
  "power",
  "facility",
  "network",
  "other",
] as const;
export type Category = (typeof categories)[number];
export type Entry = {
  row: string;
  category: Category;
  parent: string | null;
  reference: string;
};
export type Draft = { name: string; entries: Entry[] };
export const emptyDraft = (): Draft => ({ name: "", entries: [] });
export function selectedRows(draft: Draft, rows: FocusRow[]) {
  const ids = new Set(draft.entries.map((e) => e.row));
  return rows.filter((r) => ids.has(r.id));
}
export function issues(draft: Draft, data: OverviewData, saving = false) {
  const errors = new Set<string>(),
    ids = new Set(draft.entries.map((e) => e.row)),
    source = new Map(data.rows.map((r) => [r.id, r]));
  if (saving && !draft.name.trim()) errors.add("name_required");
  if (draft.name.length > 120 || draft.entries.length > 25)
    errors.add("draft_bound");
  if (saving && !draft.entries.length) errors.add("source_required");
  if (data.selection.period === "all" && saving) errors.add("month_required");
  if (ids.size !== draft.entries.length) errors.add("duplicate_row");
  const rows = selectedRows(draft, data.rows);
  if (new Set(rows.map((r) => r.currency)).size > 1)
    errors.add("mixed_currency");
  for (const e of draft.entries) {
    const r = source.get(e.row);
    if (!r) errors.add("source_unavailable");
    if (!categories.includes(e.category) || e.reference.length > 300)
      errors.add("entry_invalid");
    if (
      r &&
      data.selection.period !== "all" &&
      r.charge_month !== data.selection.period
    )
      errors.add("month_mismatch");
    if (e.parent) {
      if (!ids.has(e.parent)) errors.add("parent_unavailable");
      if (e.parent === e.row) errors.add("self_inclusion");
      if (!e.reference.trim()) errors.add("inclusion_reference");
      const p = source.get(e.parent);
      if (
        r &&
        p &&
        (r.charge_month !== p.charge_month ||
          r.charge_start !== p.charge_start ||
          r.charge_end !== p.charge_end)
      )
        errors.add("interval_mismatch");
    }
  }
  const parents = new Map(draft.entries.map((e) => [e.row, e.parent]));
  for (const id of ids) {
    const path = new Set<string>();
    let cursor: string | null | undefined = id;
    while (cursor && ids.has(cursor)) {
      if (path.has(cursor)) {
        errors.add("inclusion_cycle");
        break;
      }
      path.add(cursor);
      cursor = parents.get(cursor);
    }
  }
  return [...errors];
}
function sum(
  rows: FocusRow[],
  basis: OverviewData["selection"]["basis"],
): Measure {
  const values = rows
    .map((r) => r[basis])
    .filter((v): v is string => v !== null);
  return {
    coefficient: values.length
      ? values.reduce((n, v) => n + BigInt(v), 0n).toString()
      : null,
    present: values.length,
    missing: rows.length - values.length,
  };
}
export function preview(draft: Draft, data: OverviewData) {
  const rows = selectedRows(draft, data.rows),
    children = new Set(draft.entries.filter((e) => e.parent).map((e) => e.row));
  const errors = issues(draft, data),
    basis = data.selection.basis;
  return {
    currency: errors.includes("mixed_currency")
      ? null
      : (rows[0]?.currency ?? null),
    selected: rows.length,
    gross: errors.includes("mixed_currency") ? null : sum(rows, basis),
    excluded: errors.length
      ? null
      : sum(
          rows.filter((r) => children.has(r.id)),
          basis,
        ),
    review: errors.length
      ? null
      : sum(
          rows.filter((r) => !children.has(r.id)),
          basis,
        ),
    errors,
  };
}
export function removeEntry(draft: Draft, id: string): Draft {
  return {
    ...draft,
    entries: draft.entries
      .filter((e) => e.row !== id)
      .map((e) =>
        e.parent === id ? { ...e, parent: null, reference: "" } : e,
      ),
  };
}
