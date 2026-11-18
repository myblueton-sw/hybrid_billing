import { createElement, Fragment, type ReactNode } from "react";
import type { OrgMember, OrgParty, PlanRow, RelationKind } from "../../../../service/admin";
import { applies, label, validDay, validId } from "../organizations/live-model";

export const kinds: RelationKind[] = ["hierarchy", "payer", "contract_pool", "allocation_pool", "view_group"];
export type Action = "set" | "end" | "join" | "leave";
export type Choice = { id: string; name: string };

export function actionsFor(kind: RelationKind): Action[] {
  return kind === "view_group" ? ["join", "leave"] : ["set", "end"];
}

export function needsTarget(action: Action): boolean {
  return action !== "end";
}

function sorted(choices: Map<string, string>): Choice[] {
  return [...choices].map(([id, name]) => ({ id, name: label(name, id) })).sort((a, b) => a.name.localeCompare(b.name) || a.id.localeCompare(b.id));
}

export function memberChoices(parties: OrgParty[], members: OrgMember[]): Choice[] {
  const out = new Map<string, string>();
  for (const p of parties) out.set(p.id, p.name);
  for (const m of members) if (!out.has(m.member)) out.set(m.member, m.name);
  return sorted(out);
}

export function targetChoices(kind: RelationKind, member: string, parties: OrgParty[], members: OrgMember[]): Choice[] {
  const out = new Map<string, string>();
  if (kind === "hierarchy" || kind === "payer") {
    for (const p of parties) if (p.state === "active" && (kind === "hierarchy" || p.kind === "legal_entity")) out.set(p.id, p.name);
  } else {
    for (const m of members) {
      for (const t of m.targets) out.set(t.id, t.name || out.get(t.id) || "");
      for (const h of m.history) if (!out.has(h.target)) out.set(h.target, h.name);
    }
  }
  out.delete(member);
  return sorted(out);
}

export function currentOn(member: string, members: OrgMember[], on: string): Choice[] | null {
  const m = members.find((x) => x.member === member);
  if (!m) return null;
  return m.history.filter((h) => applies(h, on)).map((h) => ({ id: h.target, name: label(h.name, h.target) }));
}

export function isOpen(p: PlanRow): boolean {
  return p.state === "sealed" || p.state === "draft";
}

export function partition(plans: PlanRow[]): { open: PlanRow[]; closed: PlanRow[] } {
  return { open: plans.filter(isOpen), closed: plans.filter((p) => !isOpen(p)) };
}

export function impactDone(p: PlanRow): boolean {
  return p.impact.money === "computed" && p.impact.publication === "computed" && p.impact.erp === "computed";
}

export function impactUnread(p: PlanRow): boolean {
  return p.impact.money === "unknown" || p.impact.publication === "unknown" || p.impact.erp === "unknown";
}

export type Step = "done" | "current" | "held" | "stopped" | "na" | "unknown";
export type Steps = [Step, Step, Step, Step];
export function steps(p: PlanRow): Steps {
  const impact: Step = impactDone(p) ? "done" : "held";
  if (p.state === "draft") return ["current", "na", "na", "na"];
  if (p.state === "rejected") return ["stopped", "na", "na", "na"];
  if (p.state === "activated") return ["done", impact, "done", "done"];
  if (p.state === "discarded") return impact === "done" ? ["done", "done", "stopped", "na"] : ["done", "stopped", "na", "na"];
  if (p.state === "sealed") return impact === "held" ? ["done", "held", "na", "na"] : ["done", "done", canActivate(p) ? "current" : "held", "na"];
  return ["unknown", "unknown", "unknown", "unknown"];
}

export function canActivate(p: PlanRow): boolean {
  return p.state === "sealed" && p.can_activate && impactDone(p);
}

export type Blocked = "own" | "stale" | "passed" | "impact" | "impact_unknown" | "other" | "";
export function blocked(p: PlanRow): Blocked {
  if (p.state !== "sealed" || canActivate(p)) return "";
  if (p.blocked === "own_proposal") return "own";
  if (p.blocked === "stale") return "stale";
  if (p.blocked === "effective_date_passed") return "passed";
  if (p.blocked === "impact_not_computed") return "impact";
  if (p.blocked === "impact_unavailable") return "impact_unknown";
  if (!impactDone(p)) return impactUnread(p) ? "impact_unknown" : "impact";
  return "other";
}

export function held(b: Blocked): boolean {
  return b === "impact" || b === "impact_unknown";
}

export type Refusal = "impact" | "impact_unknown" | "own" | "other";
export function refusal(p: PlanRow | undefined): Refusal {
  const b = p ? blocked(p) : "";
  return b === "impact" || b === "impact_unknown" || b === "own" ? b : "other";
}

export function showOpenCount(openLimit: number): boolean {
  return Number.isFinite(openLimit) && openLimit > 0;
}

export type Closing = "activated" | "rejected" | "discarded" | "discarded_stale" | "discarded_expired" | "other";
export function closing(p: PlanRow): Closing {
  if (p.state === "activated" || p.state === "rejected") return p.state;
  if (p.closed === "discarded" || p.closed === "discarded_stale" || p.closed === "discarded_expired") return p.closed;
  return "other";
}

export const knownRules = ["no_change", "cycle", "depth", "payer_chain", "member_and_target", "plan_limit"];

export function minFrom(today: string, from: string): boolean {
  return from >= today;
}

export function findPlan(plans: PlanRow[], id: string): PlanRow | undefined {
  return id ? plans.find((p) => p.id === id) : undefined;
}

export function allMemberChoices(parties: OrgParty[], byKind: OrgMember[][]): Choice[] {
  return memberChoices(parties, byKind.flat());
}

export function nameOf(id: string, parties: OrgParty[], members: OrgMember[]): string {
  const p = parties.find((x) => x.id === id);
  if (p?.name) return p.name;
  for (const m of members) {
    if (m.member === id && m.name) return m.name;
    for (const h of m.history) if (h.target === id && h.name) return h.name;
  }
  return id;
}

export type Lead = { from: string; target: string };
export function lead(p: PlanRow): Lead {
  return { from: p.change.from, target: p.change.action === "end" || p.change.action === "leave" ? "" : p.change.target };
}

export function membersKept(p: PlanRow): boolean {
  return p.state === "sealed" || p.state === "activated";
}

export type Impact = "computed" | "not_computed" | "unknown";
export function impactOf(v: string): Impact {
  return v === "computed" || v === "not_computed" ? v : "unknown";
}

export const maxReason = 256;

export type Command = "propose" | "activate" | "discard";
export type Reaction =
  | "field"
  | "reason"
  | "limit"
  | "refused"
  | "stage"
  | "conflict"
  | "denied"
  | "signin"
  | "wait"
  | "retry"
  | "renew"
  | "partial"
  | "reauth"
  | "failed";
export function react(cmd: Command, failure: string, fields: string[]): Reaction {
  switch (failure) {
    case "invalid":
      return cmd === "propose" && fields.length > 0 ? "field" : "reason";
    case "refused":
      return cmd === "propose" ? "limit" : "refused";
    case "stage":
      return "stage";
    case "conflict":
      return "conflict";
    case "denied":
      return "denied";
    case "unauthenticated":
      return "signin";
    case "limited":
      return "wait";
    case "busy":
    case "unavailable":
      return "retry";
    case "stale":
      return "renew";
    case "partial":
      return "partial";
    case "reauth":
    case "reauth_cancelled":
      return "reauth";
    default:
      return "failed";
  }
}

export function keepsAttempt(r: Reaction): boolean {
  return r === "retry" || r === "wait" || r === "reauth";
}

export type HandoffLink = { kind: RelationKind; member: string; from: string; base: number; lang: string };
export function handoffHref(h: HandoffLink): string {
  const q = new URLSearchParams({ view: "targets", kind: h.kind, member: h.member });
  if (h.from) q.set("from", h.from);
  q.set("base", String(h.base));
  if (h.lang) q.set("lang", h.lang);
  return "#/organization-changes?" + q.toString();
}

export const handoffKeys = ["kind", "member", "from", "base"] as const;
export type Handoff = { kind: RelationKind | ""; member: string; from: string; passed: string; base: number | null; dropped: boolean };

export function canonicalRevision(raw: string): number | null {
  if (!/^(0|[1-9][0-9]{0,15})$/.test(raw)) return null;
  const n = Number(raw);
  return Number.isSafeInteger(n) ? n : null;
}

export function readHandoff(p: URLSearchParams, today: string): Handoff | null {
  if (!handoffKeys.some((k) => p.has(k))) return null;
  const out: Handoff = { kind: "", member: "", from: "", passed: "", base: null, dropped: false };
  const kind = p.get("kind");
  if (kind !== null) {
    if ((kinds as string[]).includes(kind)) out.kind = kind as RelationKind;
    else out.dropped = true;
  }
  const member = p.get("member");
  if (member !== null) {
    if (validId(member)) out.member = member;
    else out.dropped = true;
  }
  const from = p.get("from");
  if (from !== null) {
    if (!validDay(from)) out.dropped = true;
    else if (minFrom(today, from)) out.from = from;
    else out.passed = from;
  }
  const base = p.get("base");
  if (base !== null) {
    out.base = canonicalRevision(base);
    if (out.base === null) out.dropped = true;
  }
  return out;
}

export function prefilled(h: Handoff): boolean {
  return !!(h.kind || h.member || h.from || h.base !== null);
}

export function baseStale(base: number | null, active: number): boolean {
  return base !== null && base !== active;
}

export type ConflictShown = "base" | "plain";
export function conflictShown(base: number | null, reloaded: number | null): ConflictShown {
  return base !== null && reloaded !== null && base !== reloaded ? "base" : "plain";
}

export function fill(text: string, parts: Record<string, ReactNode>): ReactNode {
  const pieces = text.split(/(\{[a-z]\})/);
  const out: ReactNode[] = [];
  let rest = "";
  const flush = (key: number) => {
    if (rest) out.push(createElement(Fragment, { key: "t" + key }, rest));
    rest = "";
  };
  for (let i = 0; i < pieces.length; i++) {
    const m = /^\{([a-z])\}$/.exec(pieces[i]);
    if (!m || !(m[1] in parts)) {
      rest += pieces[i];
      continue;
    }
    const before = /\S+ ?$/.exec(rest)?.[0] ?? "";
    rest = rest.slice(0, rest.length - before.length);
    flush(i);
    const after = /^\S+/.exec(pieces[i + 1] ?? "")?.[0] ?? "";
    if (after) pieces[i + 1] = pieces[i + 1].slice(after.length);
    out.push(before || after ? createElement("span", { key: i, className: "hb-nowrap", style: { whiteSpace: "nowrap" } }, before, parts[m[1]], after) : createElement(Fragment, { key: i }, parts[m[1]]));
  }
  flush(pieces.length);
  return createElement(Fragment, null, ...out);
}

export type Carried = Handoff & { refused: boolean };

export type FormStep = "edit" | "adopt" | "proposed" | "left";
export type StepEffect = { touched: boolean; clearAttempt: boolean; resetForm: boolean };
export function formStep(step: FormStep, touched: boolean): StepEffect {
  switch (step) {
    case "edit":
      return { touched: true, clearAttempt: true, resetForm: false };
    case "adopt":
      return { touched, clearAttempt: true, resetForm: false };
    case "proposed":
      return { touched: false, clearAttempt: true, resetForm: true };
    default:
      return { touched: false, clearAttempt: false, resetForm: touched };
  }
}

export function adopted(h: Carried, active: number): Carried {
  return { ...h, base: active, refused: false };
}

export function baseBlocks(h: Carried | null, active: number | null): boolean {
  return !!h && active !== null && baseStale(h.base, active);
}

export function baseField(h: Carried | null): Record<string, string> {
  return h && h.base !== null ? { base: String(h.base) } : {};
}

export type MemberMode = "list" | "id" | "keep";
export function memberMode(settled: boolean, listed: boolean, touched: boolean): MemberMode | null {
  if (!settled) return null;
  if (touched) return "keep";
  return listed ? "list" : "id";
}

export function afterConflict(h: Carried | null, cmd: Command, reloaded: number | null): { handoff: Carried | null; shown: ConflictShown } {
  if (cmd !== "propose" || !h || conflictShown(h.base, reloaded) === "plain") return { handoff: h, shown: "plain" };
  return { handoff: { ...h, refused: true }, shown: "base" };
}

export type CurrentView = "loading" | "none_of_kind" | "not_in_listing" | "none" | "targets";
export function currentView(settled: boolean, complete: boolean, got: Choice[] | null): CurrentView {
  if (!settled) return "loading";
  if (got === null) return complete ? "none_of_kind" : "not_in_listing";
  return got.length === 0 ? "none" : "targets";
}
