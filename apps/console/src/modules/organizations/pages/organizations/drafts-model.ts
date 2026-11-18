import type { DraftMember, DraftState, GroupDraft } from "../../../../service/admin";
import { validId } from "./live-model";

export const perCall = 31;
export const draftsPage = 10;
export const shownStep = 50;

export type Allowed = { edit: boolean; plan: boolean; discard: boolean; openPlan: boolean };
export function allowed(d: Pick<GroupDraft, "state" | "plan">): Allowed {
  const open = d.state === "open";
  return { edit: open, plan: open, discard: open, openPlan: (d.state === "proposed" || d.state === "activated") && !!d.plan };
}

export type Block = "rejected" | "planDiscarded" | "held" | "openPlan" | "activated" | "discarded" | null;
export function block(d: Pick<GroupDraft, "state" | "plan">): Block {
  if (d.state === "discarded") return "discarded";
  if (d.state === "activated") return "activated";
  if (d.state === "proposed") return d.plan?.blocked === "impact_not_computed" || d.plan?.blocked === "impact_unavailable" ? "held" : "openPlan";
  if (d.state === "open" && d.plan?.state === "rejected") return "rejected";
  if (d.state === "open" && d.plan?.state === "discarded") return "planDiscarded";
  return null;
}

export function lastPlanNote(d: Pick<GroupDraft, "state" | "plan">): "rejected" | "discarded" | "" {
  const b = block(d);
  return b === "rejected" ? "rejected" : b === "planDiscarded" ? "discarded" : "";
}

export function chunks<T>(items: T[], size = perCall): T[][] {
  const out: T[][] = [];
  for (let i = 0; i < items.length; i += size) out.push(items.slice(i, i + size));
  return out;
}

export type EditCall = { add: string[]; remove: string[]; name: boolean };
export function editCalls(size: number, adds: string[], removes: string[], max: number, renamed: boolean): EditCall[] | null {
  const final = size + adds.length - removes.length;
  if (final < 1 || final > max) return null;
  const a = [...adds];
  const r = [...removes];
  const out: EditCall[] = [];
  let n = size;
  while (a.length > 0 || r.length > 0 || (renamed && out.length === 0)) {
    let take = Math.min(perCall, a.length);
    let drop = Math.min(perCall, r.length);
    drop = Math.min(drop, n + take - 1);
    take = Math.min(take, max - n + drop);
    if (take <= 0 && drop <= 0 && !(renamed && out.length === 0)) return null;
    out.push({ add: a.splice(0, Math.max(take, 0)), remove: r.splice(0, Math.max(drop, 0)), name: renamed && out.length === 0 });
    n += Math.max(take, 0) - Math.max(drop, 0);
  }
  return out;
}

export function addPending(list: DraftMember[], rows: DraftMember[], existing: Set<string>, cap: number): { list: DraftMember[]; added: number; already: number; over: number } {
  const have = new Set([...existing, ...list.map((m) => m.id)]);
  const out = [...list];
  let added = 0;
  let already = 0;
  let over = 0;
  for (const m of rows) {
    if (have.has(m.id)) { already++; continue; }
    if (existing.size + out.length >= cap) { over++; continue; }
    have.add(m.id);
    out.push(m);
    added++;
  }
  return { list: out, added, already, over };
}

export function parseIds(text: string): { ids: string[]; bad: string[] } {
  const ids: string[] = [];
  const bad: string[] = [];
  for (const raw of text.split(/\s+/)) {
    const id = raw.trim();
    if (!id || ids.includes(id) || bad.includes(id)) continue;
    (validId(id) ? ids : bad).push(id);
  }
  return { ids, bad };
}

export function validName(raw: string): boolean {
  const name = raw.trim();
  const n = [...name].length;
  return n >= 1 && n <= 80 && !/[\u0000-\u001f\u007f-\u009f]/.test(name);
}

export type Snapshot = { name: string; mode: "new" | "existing"; group: string; adds: string[]; removes: string[] };
export function dirty(start: Snapshot, now: Snapshot): boolean {
  return start.name.trim() !== now.name.trim() || start.mode !== now.mode || start.group !== now.group || now.adds.length > 0 || now.removes.length > 0;
}

export function toggled(set: Set<string>, id: string, cap: number): Set<string> {
  const next = new Set(set);
  if (next.has(id)) next.delete(id);
  else if (next.size < cap) next.add(id);
  return next;
}

export function allShown(set: Set<string>, ids: string[], cap: number): Set<string> {
  const next = new Set(set);
  if (ids.every((id) => next.has(id))) {
    for (const id of ids) next.delete(id);
    return next;
  }
  for (const id of ids) if (next.size < cap) next.add(id);
  return next;
}

export type HeaderState = "none" | "some" | "all";
export function header(set: Set<string>, ids: string[]): HeaderState {
  const n = ids.filter((id) => set.has(id)).length;
  return n === 0 ? "none" : n === ids.length ? "all" : "some";
}

export function capLeft(max: number, inDraft: number): number {
  return Math.max(0, max - inDraft);
}

export function settingValue(settings: { name: string; value: number }[], name: string, fallback: number): number {
  const found = settings.find((s) => s.name === name);
  return found && Number.isFinite(found.value) && found.value > 0 ? found.value : fallback;
}

export function draftHref(id: string, on: string, today: string, base: number, lang: string): string {
  const q = new URLSearchParams({ view: "targets", draft: id });
  if (on >= today) q.set("from", on);
  q.set("base", String(base));
  if (lang) q.set("lang", lang);
  return "#/organization-changes?" + q.toString();
}

export function planHref(plan: string, lang: string): string {
  const q = new URLSearchParams({ view: "impact", plan });
  if (lang) q.set("lang", lang);
  return "#/organization-changes?" + q.toString();
}

export type Refusal = "field" | "limit" | "stage" | "conflict" | "missing" | "dependency" | "denied" | "signin" | "failed";
export function refusalOf(failure: string, code: string): Refusal {
  if (code === "unsupported_command") return "missing";
  if (failure === "conflict") return "conflict";
  if (failure === "invalid") return "field";
  if (failure === "refused") return "limit";
  if (failure === "stage") return "stage";
  if (failure === "denied" && code === "permission_denied") return "denied";
  if (failure === "unauthenticated") return "signin";
  if (failure === "busy") return "dependency";
  return "failed";
}

export function stateKey(s: DraftState): "stOpen" | "stProposed" | "stActivated" | "stDiscarded" | "stUnknown" {
  return s === "open" ? "stOpen" : s === "proposed" ? "stProposed" : s === "activated" ? "stActivated" : s === "discarded" ? "stDiscarded" : "stUnknown";
}

export function partitionPrecheck(pending: DraftMember[], saved: Set<string>, known: Set<string>): { keep: DraftMember[]; already: number; strangers: string[] } {
  const left = pending.filter((m) => !saved.has(m.id));
  return { keep: left.filter((m) => known.has(m.id)), already: pending.length - left.length, strangers: left.filter((m) => !known.has(m.id)).map((m) => m.id) };
}

export type PendingPartial = { kind: "new" | "edit"; draftId: string; done: number };
export type SavePlan = { mode: "continue"; draftId: string; done: number } | { mode: "new" } | { mode: "edit"; draftId: string } | { mode: "none" };
export function savePlan(partial: PendingPartial | null, target: "new" | "draft" | "gone" | null, draftId: string): SavePlan {
  if (partial?.kind === "new" && partial.draftId) return { mode: "continue", draftId: partial.draftId, done: partial.done };
  if (target === "new") return { mode: "new" };
  if (target === "draft" && draftId) return { mode: "edit", draftId };
  return { mode: "none" };
}

export function isLatest(mine: number, current: number): boolean {
  return mine === current;
}

export function sameView(started: number, now: number): boolean {
  return started === now;
}

export function callRevision(index: number, continuing: boolean, loaded: string): string | null {
  return index === 0 && !continuing && loaded ? loaded : null;
}

export type ReadOutcome = "found" | "missing" | "denied" | "signin" | "failed";
export function readOutcome(failure: string): ReadOutcome {
  return failure === "denied" ? "denied" : failure === "unauthenticated" ? "signin" : "failed";
}

export type AfterRefusal = { view: "gone" | "denied" | "signin" | "failed" } | { view: "reloaded"; notice: "dataChanged" | "draftChanged" | "draftChangedClosed" };
export function afterRefusal(read: ReadOutcome, cause: "conflict" | "stage" | "denied", open: boolean): AfterRefusal {
  if (read === "missing") return { view: "gone" };
  if (read !== "found") return { view: read };
  if (!open) return { view: "reloaded", notice: "draftChangedClosed" };
  return { view: "reloaded", notice: cause === "conflict" ? "dataChanged" : "draftChanged" };
}

export function conflictCopy(draftExists: boolean): "dataChanged" | "dataChangedNew" {
  return draftExists ? "dataChanged" : "dataChangedNew";
}
