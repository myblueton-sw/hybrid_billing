import type { OrgHistory, OrgMember, OrgParty, OrgTarget, RelationKind } from "../../../../service/admin";

export function label(name: string, id: string): string {
  return name || id;
}

export type Side =
  | { state: "target"; target: OrgTarget }
  | { state: "conflict"; targets: OrgTarget[] }
  | { state: "unattributed" }
  | { state: "none" };
export type AccountRow = { id: string; name: string; unit: Side; payer: Side; other: Side | null; entryFrom: string };

export function sideOf(m: OrgMember | undefined): Side {
  if (!m || m.history.length === 0) return { state: "none" };
  if (m.targets.length > 1) return { state: "conflict", targets: m.targets };
  return m.targets[0] ? { state: "target", target: m.targets[0] } : { state: "unattributed" };
}

export function applies(h: OrgHistory, on: string): boolean {
  return h.from <= on && (!h.until || on < h.until);
}

export function covering(history: OrgHistory[], on: string): OrgHistory | undefined {
  return history.find((h) => applies(h, on));
}

export const relationKinds: RelationKind[] = ["hierarchy", "payer", "contract_pool", "allocation_pool", "view_group"];

export function joinKinds(kind: RelationKind): RelationKind[] {
  return (["hierarchy", "payer"] as RelationKind[]).filter((k) => k !== kind);
}

export function accountRows(primary: OrgMember[], kind: RelationKind, joined: Partial<Record<RelationKind, OrgMember[]>>, on: string): AccountRow[] {
  const of = (k: RelationKind, id: string, m: OrgMember) => (k === kind ? m : joined[k]?.find((x) => x.member === id));
  return primary.map((m) => {
    const hierarchy = of("hierarchy", m.member, m);
    const unit = sideOf(hierarchy);
    return {
      id: m.member, name: label(m.name, m.member), unit, payer: sideOf(of("payer", m.member, m)), other: joinKinds(kind).length === 2 ? sideOf(m) : null,
      entryFrom: hierarchy && unit.state === "target" ? covering(hierarchy.history, on)?.from ?? "" : "",
    };
  });
}

export type Filters = { entity: string; rk: RelationKind; ua: boolean; q: string };
export type FilterKey = keyof Filters;
export const noFilters: Filters = { entity: "", rk: "hierarchy", ua: false, q: "" };
export const searchBytes = 64;

const idPattern = /^[A-Za-z0-9._:-]{1,128}$/;

export function validId(id: string): boolean {
  return idPattern.test(id);
}

export function linkedId(p: URLSearchParams, key: string): { id: string; invalid: boolean } {
  const raw = p.get(key);
  if (raw === null || raw === "") return { id: "", invalid: false };
  return validId(raw) ? { id: raw, invalid: false } : { id: "", invalid: true };
}

export function legacyKind(p: URLSearchParams): RelationKind | null {
  const kind = p.get("kind");
  return p.get("rk") === null && kind !== null && (relationKinds as string[]).includes(kind) ? (kind as RelationKind) : null;
}

export class FilterRefused extends Error {}

export function refusal(err: unknown, linked: string): "store" | "drop" | "error" {
  if (!(err instanceof FilterRefused)) return "store";
  return linked ? "drop" : "error";
}

export async function pinnedRead<T>(read: (pin: number | undefined) => Promise<T>, pin: number | null, refused: (err: unknown) => boolean): Promise<T> {
  try {
    return await read(pin ?? undefined);
  } catch (err) {
    if (!refused(err)) throw err;
    if (pin === null) throw new FilterRefused();
    try {
      await read(undefined);
    } catch (again) {
      if (refused(again)) throw new FilterRefused();
      throw again;
    }
    throw err;
  }
}

export function validSearch(q: string): boolean {
  return new TextEncoder().encode(q).length <= searchBytes;
}

export function readFilters(p: URLSearchParams): { filters: Filters; invalid: FilterKey[] } {
  const filters = { ...noFilters };
  const invalid: FilterKey[] = [];
  const entity = p.get("entity");
  if (entity !== null) {
    if (idPattern.test(entity)) filters.entity = entity;
    else invalid.push("entity");
  }
  const rk = p.get("rk");
  if (rk !== null) {
    if ((relationKinds as string[]).includes(rk)) filters.rk = rk as RelationKind;
    else invalid.push("rk");
  }
  const ua = p.get("ua");
  if (ua !== null) {
    if (ua === "1") filters.ua = true;
    else invalid.push("ua");
  }
  const q = p.get("q");
  if (q !== null) {
    if (q !== "" && validSearch(q)) filters.q = q;
    else invalid.push("q");
  }
  return { filters, invalid };
}

export function filterParams(f: Filters): Record<string, string | null> {
  return { entity: f.entity || null, rk: f.rk === "hierarchy" ? null : f.rk, ua: f.ua ? "1" : null, q: f.q || null };
}

export function appliedKeys(view: string, f: Filters): FilterKey[] {
  const on: Record<FilterKey, boolean> = { entity: f.entity !== "", rk: f.rk !== "hierarchy", ua: f.ua, q: f.q !== "" };
  const shown: Record<string, FilterKey[]> = { accounts: ["q", "entity", "rk", "ua"], groups: ["q", "entity"], history: ["rk"] };
  return (shown[view] ?? []).filter((k) => on[k]);
}

export function narrowing(view: string, f: Filters): FilterKey[] {
  return appliedKeys(view, f).filter((k) => k !== "rk" || (view === "accounts" && f.ua));
}

export function cleared(f: Filters, key: FilterKey): Filters {
  return { ...f, [key]: noFilters[key] };
}

export function clearedAll(view: string, f: Filters): Filters {
  return appliedKeys(view, f).reduce(cleared, f);
}

export function nextFocus(keys: FilterKey[], removed: FilterKey): FilterKey | "" {
  const at = keys.indexOf(removed);
  return keys[at + 1] ?? keys[at - 1] ?? "";
}

export type Paged<T> = { rows: T[]; next: string; stopped: boolean; status: "loading" | "ready" | "error" | "more" | "moreFailed"; error: unknown };

export function firstPage<T>(): Paged<T> {
  return { rows: [], next: "", stopped: false, status: "loading", error: null };
}

export function appended<T>(prev: Paged<T>, rows: T[], next: string, stopped: boolean): Paged<T> {
  return { rows: [...prev.rows, ...rows], next, stopped, status: "ready", error: null };
}

export type Count = { n: number; more: boolean };

export function countOf(p: Paged<unknown>): Count {
  return { n: p.rows.length, more: p.next !== "" };
}

export function newer(active: number, pinnedAt: number): boolean {
  return active > pinnedAt;
}

export function validDay(raw: string): boolean {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(raw)) return false;
  const d = new Date(raw + "T00:00:00Z");
  return !Number.isNaN(d.getTime()) && d.toISOString().slice(0, 10) === raw;
}

export function todayUTC(now: Date): string {
  return now.toISOString().slice(0, 10);
}
