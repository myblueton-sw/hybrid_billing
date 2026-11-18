import type { Locale } from "../../../../shared/types";
import { AdminError, type EpochRow, type PolicyPage, type TrustChange, type TrustItem } from "../../../../service/admin";
import { w } from "../users/live-copy";
import { policyFailure } from "./live-model";
import { ty } from "./trust-copy";

export const openStates = ["activation_unknown", "independently_approved", "validated", "draft"];
export const closedStates = ["activated", "rejected", "cancelled", "superseded"];

export function versionOf(e: { generation: number; number: number }): string {
  return `${Math.trunc(e.generation)}.${Math.trunc(e.number)}`;
}

export function currentVersion(page: PolicyPage): string {
  return versionOf({ generation: page.trust.epoch_generation, number: page.trust.epoch_number });
}

export function expectedNext(page: PolicyPage): { generation: string; number: string; text: string } {
  const e0 = (BigInt(Math.trunc(page.trust.epoch_generation)) << 32n) + BigInt(Math.trunc(page.trust.epoch_number));
  const floor = (BigInt(Math.trunc(page.ledger.generation)) << 32n) + 1n;
  const next = e0 + 1n > floor ? e0 + 1n : floor;
  const generation = (next >> 32n).toString();
  const number = (next & 0xffffffffn).toString();
  return { generation, number, text: `${generation}.${number}` };
}

export type Candidate = { methods: string[]; mfa: string; maxAgeSeconds: number; stepUp: { action: string; maxAgeSeconds: number; mfa: boolean }[] };

export function candidateOf(maxAgeMinutes: number, stepMinutes: number): Candidate {
  return {
    methods: ["password"], mfa: "none", maxAgeSeconds: maxAgeMinutes * 60,
    stepUp: stepMinutes > 0 ? [{ action: "policy.change", maxAgeSeconds: stepMinutes * 60, mfa: false }] : [],
  };
}

function u64(n: bigint): Uint8Array {
  const out = new Uint8Array(8);
  new DataView(out.buffer).setBigUint64(0, n, false);
  return out;
}

export function configEncoding(c: Candidate): Uint8Array {
  const parts: Uint8Array[] = [];
  const enc = new TextEncoder();
  const text = (s: string) => { const b = enc.encode(s); parts.push(u64(BigInt(b.length)), b); };
  const number = (n: bigint) => parts.push(u64(n));
  const nanos = (seconds: number) => BigInt(Math.trunc(seconds)) * 1000000000n;
  text("trust_config_v1");
  number(BigInt(c.methods.length));
  for (const m of c.methods) text(m);
  text(c.mfa);
  number(nanos(c.maxAgeSeconds));
  number(BigInt(c.stepUp.length));
  for (const r of c.stepUp) {
    text(r.action);
    number(nanos(r.maxAgeSeconds));
    number(r.mfa ? 1n : 0n);
  }
  const out = new Uint8Array(parts.reduce((n, p) => n + p.length, 0));
  let at = 0;
  for (const p of parts) { out.set(p, at); at += p.length; }
  return out;
}

export async function configDigest(c: Candidate): Promise<string> {
  const sum = await globalThis.crypto.subtle.digest("SHA-256", configEncoding(c) as BufferSource);
  return Array.from(new Uint8Array(sum), (b) => b.toString(16).padStart(2, "0")).join("");
}

export function unavailable(page: PolicyPage): boolean {
  return page.trust.hold !== null || !["ok", "recovered", "full"].includes(page.ledger.state);
}

export function canPropose(page: PolicyPage): boolean {
  return page.rights.propose && !unavailable(page);
}

export type Condition = { kind: "hold" | "isolated" | "durable" | "stopped" | "reset" | "carried"; line: number; older?: boolean };

export function conditions(page: PolicyPage): Condition[] {
  const out: Condition[] = [];
  if (page.trust.hold) out.push({ kind: "hold", line: 0 });
  if (page.ledger.state === "isolated") {
    const lined = page.ledger.isolation_reason === "integrity_failed" || page.ledger.isolation_reason === "epoch_order";
    out.push({ kind: "isolated", line: lined ? page.ledger.problem_line : 0, older: page.trust.older_record_ignored });
  }
  if (page.trust.durable_unconfirmed || page.ledger.not_durable) out.push({ kind: "durable", line: 0 });
  else if (page.ledger.state === "stopped" && !page.trust.hold) out.push({ kind: "stopped", line: 0 });
  if (page.trust.reset_at) out.push({ kind: "reset", line: 0 });
  if (page.trust.origin === "carried" && page.trust.carried_line > 0) out.push({ kind: "carried", line: page.trust.carried_line });
  return out;
}

export function firstSentence(text: string): [string, string] {
  const m = /^(.+?[.。])(\s+|$)(.*)$/su.exec(text);
  return m ? [m[1], m[3]] : [text, ""];
}

export function conditionText(l: Locale, c: Condition, at: string): string[] {
  switch (c.kind) {
    case "hold": return [ty.holdLong(l)];
    case "isolated": return [ty.isolatedLead(l), ty.isolatedRest(l) + (c.line ? " " + ty.problemLine(l, c.line) : ""), ...(c.older ? [ty.olderIgnored(l)] : [])];
    case "durable": return [ty.durableLead(l), ty.durableRest(l)];
    case "stopped": return [ty.stoppedLead(l), ty.durableRest(l)];
    case "reset": return [ty.resetLine(l, at)];
    default: return [ty.carriedLine(l, c.line)];
  }
}

export function grouped(page: PolicyPage): { open: TrustChange[]; closed: TrustChange[] } {
  const byState = (states: string[]) =>
    page.changes.filter((c) => states.includes(c.state)).sort((a, b) => states.indexOf(a.state) - states.indexOf(b.state) || b.number - a.number || (a.id < b.id ? 1 : -1));
  return { open: byState(openStates), closed: byState(closedStates) };
}

export function blocked(c: TrustChange): boolean {
  return c.validation !== null && c.validation.result === "blocked";
}

export function stale(c: TrustChange): boolean {
  return c.validation !== null && !c.validation.evidence_current && (c.state === "validated" || c.state === "independently_approved");
}

export function stateSentence(l: Locale, c: TrustChange): string {
  if (stale(c)) return ty.sentence(l, c.state === "validated" ? "stale_validated" : "stale_approved");
  if (c.state === "draft" && blocked(c)) return ty.sentence(l, "blocked");
  return ty.sentence(l, c.state);
}

export function nextStep(l: Locale, c: TrustChange): string {
  if (stale(c)) return ty.next.stale(l);
  switch (c.state) {
    case "draft": return blocked(c) ? ty.next.blocked(l) : ty.next.draft(l);
    case "validated": return ty.next.validated(l);
    case "independently_approved": return ty.next.approved(l);
    case "activation_unknown": return ty.next.unknown(l);
    case "activated": return c.invalidation && c.invalidation.state !== "complete" ? ty.next.progress(l) : ty.next.activated(l);
    default: return ty.next.closed(l);
  }
}

const whyKeys = ["self", "approver", "no_right", "not_validated", "needs_approval", "evidence_stale", "trust_unavailable"] as const;

export function whyNot(l: Locale, c: TrustChange): string[] {
  if (!openStates.includes(c.state) || c.state === "activation_unknown") return [];
  if (stale(c)) return [];
  const out: string[] = [];
  const reasons = c.state === "independently_approved" ? [c.why_not.activate] : [c.why_not.approve, c.why_not.activate];
  for (const reason of reasons) {
    const key = whyKeys.find((k) => k === reason);
    const text = key ? ty.why[key](l) : "";
    if (text && !out.includes(text)) out.push(text);
  }
  return out;
}

export type ActionKey = "activate" | "approve" | "validate" | "revise" | "reject" | "cancel";

export function actions(c: TrustChange): { primary: ActionKey | null; secondary: ActionKey[] } {
  const order: ActionKey[] = ["activate", "approve", "validate"];
  const primary = order.find((k) => c.can[k]) ?? null;
  const secondary: ActionKey[] = [];
  if (c.can.revise) secondary.push("revise");
  if (c.can.validate && primary !== "validate") secondary.push("validate");
  if (c.can.reject) secondary.push("reject");
  if (c.can.cancel) secondary.push("cancel");
  return { primary, secondary };
}

export function rightsList(l: Locale, page: PolicyPage): string {
  const held = (["propose", "validate", "approve", "activate"] as const).filter((k) => page.rights[k]).map((k) => ty.role(l, "trust_" + k));
  return held.length ? ty.rightsLine(l, held.join(l === "ar" ? "، " : ", ")) : ty.noRights(l);
}

export function enrolled(page: PolicyPage): boolean {
  return page.holders.ready;
}

export function findingReason(l: Locale, item: TrustItem): string {
  if (item.item === "recovery_path" && item.result === "blocked") {
    if (item.detail.startsWith("missing:")) return ty.missingRoles(l, item.detail.slice(8).split(",").map((r) => ty.role(l, r)).join(l === "ar" ? "، " : ", "));
    return ty.reason(l, "recovery_unverified");
  }
  if (item.item === "recovery_path" && item.result === "verified") return ty.reason(l, "recovery_verified");
  if (item.item === "test_login" && item.result === "blocked") return ty.reason(l, "test_login_failed");
  return item.reason ? ty.reason(l, item.reason) : "";
}

export const checkItems = ["test_login", "recovery_path", "issuer_subject", "provider_diagnostics", "external_status", "record_size"];

export function epochText(e: EpochRow | null): string {
  return e ? versionOf(e) : "";
}

export const fatalFailures = ["self", "epoch", "approval_stale", "evidence_stale", "needs_approval", "trust_unavailable", "stage", "denied", "stale", "conflict"];

export function asTrustFailure(e: unknown, page: PolicyPage | null): unknown {
  if (e instanceof AdminError && e.failure === "busy" && page && unavailable(page)) return new AdminError("trust_unavailable", e.fields, 0, e.operation, e.code);
  return e;
}

export function trustFailure(l: Locale, e: unknown, kind = ""): { title: string; body: string } {
  if (!(e instanceof AdminError)) return policyFailure(l, e);
  if (e.failure === "evidence_stale" && kind === "approve") return { title: ty.fail.title(l), body: ty.sentence(l, "stale_validated") };
  switch (e.failure) {
    case "self":
    case "epoch":
    case "approval_stale":
    case "evidence_stale":
    case "needs_approval":
    case "trust_unavailable":
      return { title: ty.fail.title(l), body: ty.fail[e.failure](l) };
    case "stage":
      return { title: ty.fail.title(l), body: ty.fail.stage(l) };
    case "denied":
      return { title: ty.fail.title(l), body: ty.fail.denied(l) };
    case "unauthenticated":
      return { title: ty.fail.title(l), body: w.ended(l) };
    default:
      return policyFailure(l, e);
  }
}

export function fatal(e: unknown): boolean {
  return e instanceof AdminError && fatalFailures.includes(e.failure);
}

export function revoked(e: unknown): boolean {
  return e instanceof AdminError && (e.failure === "denied" || e.failure === "unauthenticated");
}

export type Proposed = { maxAge: number; step: number };

export type Confirming = { kind: Exclude<ActionKey, "revise"> | "propose"; change: TrustChange | null; proposed: Proposed | null };

export function valuesFor(page: PolicyPage, k: Confirming): Record<string, string> {
  const ch = k.change;
  const epoch = page.trust.epoch;
  switch (k.kind) {
    case "propose": {
      const v: Record<string, string> = { max_age_minutes: String(k.proposed?.maxAge ?? 0), step_up_minutes: String(k.proposed?.step ?? 0), expected_epoch: epoch,
        expected_policy: String(page.trust.policy_number) };
      if (ch) v.expected_change_revision = String(ch.change_revision);
      return v;
    }
    case "validate":
      return { expected_change_revision: String(ch?.change_revision ?? 0), change_digest: ch?.change_digest ?? "", expected_epoch: epoch };
    case "approve":
      return { change_digest: ch?.change_digest ?? "", evidence_digest: ch?.validation?.evidence_digest ?? "", expected_epoch: epoch };
    case "activate":
      return { change_digest: ch?.change_digest ?? "", evidence_digest: ch?.validation?.evidence_digest ?? "", expected_epoch: epoch, expected_policy: String(page.trust.policy_number) };
    default:
      return { expected_change_revision: String(ch?.change_revision ?? 0) };
  }
}

export function rawEpochText(raw: string): string {
  if (!/^(0|[1-9][0-9]*)$/.test(raw)) return raw;
  const v = BigInt(raw);
  return `${(v >> 32n).toString()}.${(v & 0xffffffffn).toString()}`;
}
