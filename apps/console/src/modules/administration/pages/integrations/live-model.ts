import { Refusals } from "../../../../service/refusal";
import type { RowDiagnostic } from "../../../../service/command-diagnostics";
import { forgetLost, lostDiagnostics } from "./live-rights";
import type { Delivery, IntegrationsPage, PluginInstall, PluginProposal, PluginVersion, RightId, Subscription } from "../../../../service/integrations";

export const views = ["clients", "modules", "channels", "events"];

export function viewOf(v: string): string {
  return views.includes(v) ? v : "clients";
}

export const shownLimit = 50;
export const maxReason = 256;
export const maxTypes = 8;
export const webhooksTarget = "webhooks";

export type CommandKey =
  | "integrations.plugin_approve"
  | "integrations.plugin_withdraw"
  | "integrations.plugin_propose"
  | "integrations.plugin_stop"
  | "integrations.plugin_disable"
  | "integrations.subscription_propose"
  | "integrations.subscription_approve"
  | "integrations.subscription_pause"
  | "integrations.subscription_request_resume"
  | "integrations.subscription_approve_resume"
  | "integrations.subscription_revoke"
  | "integrations.letter_request_redeliver"
  | "integrations.letter_approve_redeliver"
  | "integrations.letter_request_dismiss"
  | "integrations.letter_approve_dismiss"
  | "integrations.subscription_withdraw"
  | "integrations.subscription_withdraw_resume"
  | "integrations.letter_withdraw_redeliver"
  | "integrations.letter_withdraw_dismiss"
  | "integrations.redelivery_reapprove";

export const commandKeys: CommandKey[] = [
  "integrations.plugin_approve",
  "integrations.plugin_withdraw",
  "integrations.plugin_propose",
  "integrations.plugin_stop",
  "integrations.plugin_disable",
  "integrations.subscription_propose",
  "integrations.subscription_approve",
  "integrations.subscription_pause",
  "integrations.subscription_request_resume",
  "integrations.subscription_approve_resume",
  "integrations.subscription_revoke",
  "integrations.letter_request_redeliver",
  "integrations.letter_approve_redeliver",
  "integrations.letter_request_dismiss",
  "integrations.letter_approve_dismiss",
  "integrations.subscription_withdraw",
  "integrations.subscription_withdraw_resume",
  "integrations.letter_withdraw_redeliver",
  "integrations.letter_withdraw_dismiss",
  "integrations.redelivery_reapprove",
];

export const commandRight: Record<CommandKey, RightId> = {
  "integrations.plugin_approve": "approve_modules",
  "integrations.plugin_withdraw": "administer_modules",
  "integrations.plugin_propose": "administer_modules",
  "integrations.plugin_stop": "stop_modules",
  "integrations.plugin_disable": "stop_modules",
  "integrations.subscription_propose": "administer_channels",
  "integrations.subscription_approve": "approve_channels",
  "integrations.subscription_pause": "stop_channels",
  "integrations.subscription_request_resume": "administer_channels",
  "integrations.subscription_approve_resume": "approve_channels",
  "integrations.subscription_revoke": "stop_channels",
  "integrations.letter_request_redeliver": "reprocess_events",
  "integrations.letter_approve_redeliver": "reprocess_events",
  "integrations.letter_request_dismiss": "reprocess_events",
  "integrations.letter_approve_dismiss": "reprocess_events",
  "integrations.subscription_withdraw": "administer_channels",
  "integrations.subscription_withdraw_resume": "administer_channels",
  "integrations.letter_withdraw_redeliver": "reprocess_events",
  "integrations.letter_withdraw_dismiss": "reprocess_events",
  "integrations.redelivery_reapprove": "reprocess_events",
};

const authorCommands: CommandKey[] = [
  "integrations.plugin_withdraw",
  "integrations.subscription_withdraw",
  "integrations.subscription_withdraw_resume",
  "integrations.letter_withdraw_redeliver",
  "integrations.letter_withdraw_dismiss",
];

export function authorOnly(command: CommandKey): boolean {
  return authorCommands.includes(command);
}

export type Group = "proposal" | "plugin" | "propose_plugin" | "propose_subscription" | "subscription" | "letter" | "resume_request" | "letter_request" | "redelivery";

export function groupOf(command: CommandKey): Group {
  const kind = command.slice("integrations.".length);
  if (kind === "plugin_approve" || kind === "plugin_withdraw") return "proposal";
  if (kind === "plugin_propose") return "propose_plugin";
  if (kind === "plugin_stop" || kind === "plugin_disable") return "plugin";
  if (kind === "subscription_propose") return "propose_subscription";
  if (kind === "subscription_withdraw_resume") return "resume_request";
  if (kind === "letter_withdraw_redeliver" || kind === "letter_withdraw_dismiss") return "letter_request";
  if (kind === "redelivery_reapprove") return "redelivery";
  if (kind.startsWith("subscription_")) return "subscription";
  return "letter";
}

export function viewOfCommand(command: CommandKey): string {
  const g = groupOf(command);
  return g === "proposal" || g === "plugin" || g === "propose_plugin" ? "modules" : g === "letter" || g === "letter_request" || g === "redelivery" ? "events" : "channels";
}

const idChars = /^[A-Za-z0-9._:-]{1,128}$/;

export function validTarget(t: string): boolean {
  return idChars.test(t);
}

export function proposalTarget(p: PluginProposal): string {
  if (p.number === null || !/^plugin-proposal-[1-9][0-9]*$/.test(p.target)) return "";
  return p.target === "plugin-proposal-" + p.number && validTarget(p.target) ? p.target : "";
}

export function subscriptionTarget(s: Subscription): string {
  return s.id && s.target === "webhook-subscription:" + s.id && validTarget(s.target) ? s.target : "";
}

export function letterTarget(d: Delivery): string {
  return d.id && d.target === "webhook-letter:" + d.id && validTarget(d.target) ? d.target : "";
}

export function redeliveryTarget(d: Delivery): string {
  return d.id && d.hold && d.target === "webhook-redelivery:" + d.id && validTarget(d.target) ? d.target : "";
}

export function resumeRequestTarget(r: { id: string; target: string }): string {
  return r.id && r.target === "webhook-resume-request:" + r.id && validTarget(r.target) ? r.target : "";
}

export function letterRequestTarget(r: { id: string; target: string }): string {
  return r.id && r.target === "webhook-letter-request:" + r.id && validTarget(r.target) ? r.target : "";
}

export function pluginTarget(plugin: string): string {
  const t = "plugin:" + plugin;
  return plugin && validTarget(t) ? t : "";
}

export function commandsAllowed(page: IntegrationsPage): boolean {
  return page.revision !== "";
}

export function timeKnown(iso: string): boolean {
  if (!iso) return false;
  const d = new Date(iso);
  return !Number.isNaN(d.getTime()) && d.getUTCFullYear() > 1;
}

export function actor(login: string, id: string): string {
  return login || id;
}

export function ordered(list: PluginProposal[]): { pending: PluginProposal[]; closed: PluginProposal[] } {
  const sorted = [...list].sort((a, b) => (b.number ?? -1) - (a.number ?? -1));
  return { pending: sorted.filter((p) => p.state === "pending"), closed: sorted.filter((p) => p.state !== "pending") };
}

export const proposalKinds = ["publisher", "key", "version", "workspace", "runtime", "install", "upgrade", "rollback", "enable", "remove", "endpoint"];

export type SummaryPart = { text: string } | { id: string } | { name: string } | { date: string };
export type Summary = { kind: string; known: boolean; parts: Record<string, SummaryPart> };

export function summaryOf(p: PluginProposal): Summary {
  const s = p.summary;
  const known = proposalKinds.includes(p.kind);
  const id = (v: string): SummaryPart => ({ id: v });
  const parts: Record<string, SummaryPart> = {
    name: { name: s.publisher_name },
    id: id(s.publisher),
    key: id(s.key_version),
    publisher: id(s.publisher),
    from: { date: s.key_not_before },
    to: { date: s.key_not_after },
    version: id(s.version),
    plugin: id(s.plugin),
    workspace: id(s.workspace),
    runtime: id(s.runtime),
    endpoint: id(s.endpoint),
    fingerprint: id(s.fingerprint),
  };
  return { kind: p.kind, known, parts };
}

export type Explain = "hidden" | "explain";

const hiddenCodes = ["not_active", "not_dead", "not_proposer", "not_enabled", "not_allowed_state"];

export function blockerPolicy(code: string, pendingProposal = false): Explain {
  if (code === "not_pending") return pendingProposal ? "explain" : "hidden";
  return hiddenCodes.includes(code) ? "hidden" : "explain";
}

export const dryRunBlocked = ["signature_invalid", "vulnerable", "revoked", "key_inactive", "publisher_inactive", "incompatible", "license_not_permitted"];

export const semver = /^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$/;

export function compareVersions(a: string, b: string): number | null {
  const x = semver.exec(a);
  const y = semver.exec(b);
  if (!x || !y) return null;
  for (let i = 1; i <= 3; i++) {
    const d = Number(x[i]) - Number(y[i]);
    if (d !== 0) return d > 0 ? 1 : -1;
  }
  return 0;
}

export const installStates = ["installed", "enabled", "stopped", "removed"];
export const changeKinds = ["install", "upgrade", "rollback", "enable", "remove"] as const;
export type ChangeKind = (typeof changeKinds)[number];
export type Change = { kind: ChangeKind; version: string };

export function changeKey(c: Change): string {
  return c.kind + ":" + c.version;
}

export function wellFormedVersion(v: PluginVersion): boolean {
  return !!v.plugin && semver.test(v.version);
}

export function pluginIds(page: IntegrationsPage): string[] {
  const out: string[] = [];
  for (const i of page.modules.installations ?? []) if (i.plugin && !out.includes(i.plugin)) out.push(i.plugin);
  for (const v of page.modules.plugins ?? []) if (v.plugin && !out.includes(v.plugin)) out.push(v.plugin);
  return out;
}

export function installOf(page: IntegrationsPage, plugin: string): PluginInstall | undefined {
  return (page.modules.installations ?? []).find((i) => i.plugin === plugin);
}

export function trusted(page: IntegrationsPage, v: PluginVersion): boolean {
  const publisher = (page.modules.publishers ?? []).find((p) => p.id === v.publisher);
  if (!publisher || publisher.revocation !== null) return false;
  const key = (publisher.keys ?? []).find((k) => k.version === v.key_version);
  return !!key && key.compromise === null;
}

export function permitted(v: PluginVersion): boolean {
  return v.license_state === "permitted";
}

export function refusedByList(v: PluginVersion | undefined): boolean {
  return v?.license_state === "not_permitted";
}

export function runnable(page: IntegrationsPage, v: PluginVersion): boolean {
  if (!trusted(page, v) || !permitted(v)) return false;
  return v.vulnerable === null || page.modules.workspace?.verification === true;
}

export function versionOf(page: IntegrationsPage, plugin: string, version: string): PluginVersion | undefined {
  return (page.modules.plugins ?? []).find((v) => v.plugin === plugin && v.version === version);
}

export function licenseRefused(page: IntegrationsPage, plugin: string, version: string): boolean {
  if (!plugin || !version) return false;
  const v = versionOf(page, plugin, version);
  return !v || !permitted(v);
}

export type ProposeBlocker = { code: "no_command" | "no_workspace" | "no_version" | "unknown_state" | "unknown_list" | "untrusted" | "license" | "license_unknown" } | { code: "duplicate"; number: string; own: boolean };

export type ProposeOffer = { options: Change[]; blocker: ProposeBlocker | null; withheld: PluginVersion[] };

export function pendingChange(page: IntegrationsPage, plugin: string, c: Change): PluginProposal | undefined {
  return (page.modules.proposals ?? []).find((p) => p.state === "pending" && p.kind === c.kind && p.summary.plugin === plugin && p.summary.version === c.version);
}

export function proposeOffer(page: IntegrationsPage, me: string, plugin: string): ProposeOffer {
  let withheld: PluginVersion[] = [];
  const none = (blocker: ProposeBlocker): ProposeOffer => ({ options: [], blocker, withheld });
  if (!pluginTarget(plugin)) return none({ code: "no_command" });
  if (!page.modules.workspace) return none({ code: "no_workspace" });
  if (page.modules.installations === null) return none({ code: "unknown_state" });
  if (page.modules.proposals === null) return none({ code: "unknown_list" });
  const versions = (page.modules.plugins ?? []).filter((v) => v.plugin === plugin && wellFormedVersion(v));
  if (versions.length === 0) return none({ code: "no_version" });
  const install = installOf(page, plugin);
  if (install && !installStates.includes(install.state)) return none({ code: "unknown_state" });
  const candidates: Change[] = [];
  const ok = versions.filter((v) => runnable(page, v));
  const licenseOnly = versions.filter((v) => !permitted(v) && runnable(page, { ...v, license_state: "permitted" }));
  const settled = install && install.state !== "removed" && install.state !== "installed" && install.state !== "stopped";
  withheld = licenseOnly.filter((v) => !settled || v.version !== install.version);
  if (!install || install.state === "removed") {
    for (const v of ok) candidates.push({ kind: "install", version: v.version });
  } else {
    for (const v of ok) {
      const d = compareVersions(v.version, install.version);
      if (d === 1) candidates.push({ kind: "upgrade", version: v.version });
      if (d === -1) candidates.push({ kind: "rollback", version: v.version });
    }
    const current = versions.find((v) => v.version === install.version);
    if ((install.state === "installed" || install.state === "stopped") && current && runnable(page, current)) candidates.push({ kind: "enable", version: "" });
    candidates.push({ kind: "remove", version: "" });
  }
  const options: Change[] = [];
  let dup: PluginProposal | undefined;
  for (const c of candidates) {
    const same = pendingChange(page, plugin, c);
    if (same) dup = dup ?? same;
    else options.push(c);
  }
  if (options.length > 0) return { options, blocker: null, withheld };
  if (dup) return none({ code: "duplicate", number: dup.number === null ? "" : String(dup.number), own: !!me && dup.proposer === me });
  if (ok.length < versions.length && ok.length + licenseOnly.length === versions.length) return none({ code: licenseOnly.every(refusedByList) ? "license" : "license_unknown" });
  if (ok.length < versions.length) return none({ code: "untrusted" });
  return none({ code: "no_version" });
}

export function proposeValues(c: Change): Record<string, string> {
  return c.kind === "install" || c.kind === "upgrade" || c.kind === "rollback" ? { kind: c.kind, version: c.version } : { kind: c.kind };
}

export type SubscribeBlocker = "no_types" | "no_destination" | "no_subscriber" | "unknown_list";

export function subscribeBlocker(page: IntegrationsPage): SubscribeBlocker | null {
  if ((page.events.types ?? []).filter((t) => validTarget(t.name)).length === 0) return "no_types";
  if (!(page.channels.destinations ?? []).some((d) => d.state === "active" && validTarget(d.id))) return "no_destination";
  const subs = page.channels.subscribers ?? [];
  if (subs.length !== 1 || !validTarget(subs[0])) return "no_subscriber";
  if (page.channels.subscriptions === null) return "unknown_list";
  return null;
}

export function normalTypes(types: string[]): string[] {
  return [...new Set(types)].sort((a, b) => (a < b ? -1 : a > b ? 1 : 0));
}

export type Covered = { id: string; own: boolean } | null;

export function coveredBy(page: IntegrationsPage, me: string, destination: string, types: string[]): Covered {
  const want = normalTypes(types).join(",");
  const subscriber = (page.channels.subscribers ?? [])[0] ?? "";
  if (page.channels.subscriptions === null) return { id: "", own: false };
  for (const s of page.channels.subscriptions) {
    if (s.state !== "proposed" && s.state !== "active") continue;
    if (s.destination !== destination || s.subscriber !== subscriber || s.types === null) continue;
    if (normalTypes(s.types).join(",") !== want) continue;
    return { id: s.id, own: s.state === "proposed" && !!me && s.proposer === me };
  }
  return null;
}

export type ChoiceProblem = "destination" | "types_none" | "types_many";

export function subscribeProblems(page: IntegrationsPage, destination: string, types: string[]): ChoiceProblem[] {
  const out: ChoiceProblem[] = [];
  const active = (page.channels.destinations ?? []).some((d) => d.id === destination && d.state === "active");
  if (!active) out.push("destination");
  const n = normalTypes(types).length;
  if (n === 0) out.push("types_none");
  if (n > maxTypes) out.push("types_many");
  return out;
}

export function listed(page: IntegrationsPage, command: CommandKey, target: string): boolean {
  switch (groupOf(command)) {
    case "proposal":
      return (page.modules.proposals ?? []).some((p) => p.target === target);
    case "plugin":
    case "propose_plugin":
      return pluginIds(page).some((p) => "plugin:" + p === target);
    case "propose_subscription":
      return target === webhooksTarget;
    case "subscription":
      return (page.channels.subscriptions ?? []).some((s) => s.target === target);
    case "resume_request":
    case "letter_request":
      return (page.own_pending ?? []).some((p) => p.target === target);
    default:
      return (page.events.deliveries ?? []).some((d) => d.target === target);
  }
}

const subAction = (command: CommandKey) => command.slice("integrations.subscription_".length) as keyof Subscription["actions"];
const letterAction = (command: CommandKey) => command.slice("integrations.letter_".length) as keyof Delivery["actions"];

export function stillAllowed(page: IntegrationsPage, me: string, command: CommandKey, target: string): boolean {
  switch (groupOf(command)) {
    case "proposal": {
      const p = (page.modules.proposals ?? []).find((x) => x.target === target);
      if (!p || p.state !== "pending") return false;
      return command === "integrations.plugin_approve" ? p.can_approve : p.can_withdraw;
    }
    case "plugin": {
      const i = (page.modules.installations ?? []).find((x) => "plugin:" + x.plugin === target);
      if (!i) return false;
      return command === "integrations.plugin_stop" ? i.can_stop : i.can_disable;
    }
    case "propose_plugin":
      return proposeOffer(page, me, target.slice("plugin:".length)).options.length > 0;
    case "propose_subscription":
      return subscribeBlocker(page) === null;
    case "subscription": {
      const s = (page.channels.subscriptions ?? []).find((x) => x.target === target);
      return !!s && s.actions[subAction(command)].allowed;
    }
    case "resume_request":
    case "letter_request":
      return (page.own_pending ?? []).some((p) => p.target === target);
    case "redelivery": {
      const d = (page.events.deliveries ?? []).find((x) => x.target === target);
      return !!d && d.actions.reapprove.allowed;
    }
    default: {
      const d = (page.events.deliveries ?? []).find((x) => x.target === target);
      return !!d && d.actions[letterAction(command)].allowed;
    }
  }
}

export type Selection = { proposal: string; subscription: string; delivery: string };

export function selectionOf(params: URLSearchParams, page: IntegrationsPage | null, view: string): Selection {
  const out: Selection = { proposal: "", subscription: "", delivery: "" };
  if (!page) return out;
  const proposal = params.get("proposal") ?? "";
  if (view === "modules" && /^[1-9][0-9]*$/.test(proposal) && (page.modules.proposals ?? []).some((p) => p.number !== null && String(p.number) === proposal)) out.proposal = proposal;
  const subscription = params.get("subscription") ?? "";
  if ((view === "channels" || view === "events") && subscription && (page.channels.subscriptions ?? []).some((s) => s.id === subscription)) out.subscription = subscription;
  const delivery = params.get("delivery") ?? "";
  if (view === "events" && delivery && (page.events.deliveries ?? []).some((d) => d.id === delivery)) out.delivery = delivery;
  return out;
}

export function deliveriesCut(page: IntegrationsPage, subscription: string): "cut" | "complete" | "unknown" {
  const more = page.events.deliveries_more;
  if (more === null) return "unknown";
  return (subscription ? more.includes(subscription) : more.length > 0) ? "cut" : "complete";
}

export function deliveriesShown(page: IntegrationsPage, subscription: string): Delivery[] | null {
  const all = page.events.deliveries;
  if (all === null) return null;
  return subscription ? all.filter((d) => d.subscription === subscription) : all;
}

export function waitsForResume(page: IntegrationsPage, d: Delivery): boolean {
  if (d.state !== "waiting") return false;
  const s = (page.channels.subscriptions ?? []).find((x) => x.id === d.subscription);
  return !!s && (s.state === "paused" || s.state === "suspended");
}

export function subscriptionRevoked(page: IntegrationsPage, d: Delivery): boolean {
  const s = (page.channels.subscriptions ?? []).find((x) => x.id === d.subscription);
  return !!s && s.state === "revoked";
}

export type Capability = { kind: string; target: string } | { raw: string };

export const capabilityKinds = ["read_events", "call_command", "serve_screen"];

export function capabilityOf(c: string): Capability {
  const at = c.indexOf(":");
  if (at <= 0) return { raw: c };
  const kind = c.slice(0, at);
  return capabilityKinds.includes(kind) ? { kind, target: c.slice(at + 1) } : { raw: c };
}

export type Reaction = "field" | "reason" | "hint" | "conflict" | "denied" | "signin" | "wait" | "retry" | "renew" | "partial" | "reauth" | "failed";

export function react(failure: string, fields: string[]): Reaction {
  switch (failure) {
    case "invalid":
      return fields.length > 0 ? "field" : "reason";
    case "refused":
    case "stage":
      return "hint";
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
  return r === "retry" || r === "reauth";
}

export type FieldReaction = "built" | "kind" | "changed" | "choice" | "reason";

export function fieldReaction(command: CommandKey, fields: string[]): FieldReaction {
  const own: string[] =
    command === "integrations.plugin_propose" ? ["kind", "version"] : command === "integrations.subscription_propose" ? ["destination", "types"] : [];
  if (fields.some((f) => f !== "rationale" && !own.includes(f))) return "built";
  if (fields.includes("kind")) return "kind";
  if (fields.includes("version")) return "changed";
  if (fields.includes("destination") || fields.includes("types")) return "choice";
  return "reason";
}

export function utf8Bytes(v: string): number {
  return new TextEncoder().encode(v).length;
}

export function reasonProblem(v: string): "" | "missing" | "long" {
  const t = v.trim();
  if (!t) return "missing";
  return utf8Bytes(t) > maxReason ? "long" : "";
}

export type Pending = { key: string; request: string; expected: string; values: string; snapshot: string[]; context: Record<string, string> };
const pending = new Map<string, Pending>();
const drafts = new Map<string, Record<string, string>>();
const refusals = new Refusals<string>();

let scope = "";
let scopeGeneration = 0;

export function currentScope(): number { return scopeGeneration; }

export function setScope(workspace: string, user: string, session: string): void {
  const next = workspace + "\u0000" + user + "\u0000" + session;
  if (next !== scope) {
    scopeGeneration++;
    forgetLost();
    pending.clear();
    drafts.clear();
    refusals.clear();
    scope = next;
  }
}

export function forget(): void {
  scopeGeneration++;
  pending.clear();
  drafts.clear();
  refusals.clear();
  scope = "";
}

function slot(command: string, target: string): string {
  return scope + "\u0001" + command + "\u0000" + target;
}

function unslot(id: string): { command: CommandKey; target: string } | null {
  if (!id.startsWith(scope + "\u0001")) return null;
  const rest = id.slice(scope.length + 1);
  const at = rest.indexOf("\u0000");
  return { command: rest.slice(0, at) as CommandKey, target: rest.slice(at + 1) };
}

export function pendingNow(command: string, target: string): Pending | undefined {
  return pending.get(slot(command, target));
}

export function pendingFor(
  command: string,
  target: string,
  values: Record<string, string>,
  expected: string,
  fresh: () => { request: string; key: string },
  snapshot: string[] = [],
  context: Record<string, string> = {},
): Pending {
  const id = slot(command, target);
  const frozen = JSON.stringify(values);
  const kept = pending.get(id);
  if (kept && kept.values === frozen) return kept;
  refusals.delete(id);
  const tried = fresh();
  const next = { key: tried.key, request: tried.request, expected, values: frozen, snapshot: [...snapshot], context: { ...context } };
  pending.set(id, next);
  return next;
}

export function frozenValues(p: Pending | undefined): Record<string, string> | null {
  if (!p) return null;
  try {
    const v = JSON.parse(p.values) as unknown;
    return v && typeof v === "object" ? (v as Record<string, string>) : null;
  } catch {
    return null;
  }
}

export function discard(command: string, target: string, values: Record<string, string>): void {
  const kept = pending.get(slot(command, target));
  if (kept && kept.values !== JSON.stringify(values)) pending.delete(slot(command, target));
}

export function drop(command: string, target: string): void {
  pending.delete(slot(command, target));
}

export function settle(command: string, target: string, reaction: Reaction | "ok", diagnostic?: RowDiagnostic): void {
  if (reaction !== "ok" && keepsAttempt(reaction)) return;
  const values = pending.get(slot(command, target))?.values ?? "";
  pending.delete(slot(command, target));
  if (reaction === "hint") refusals.set(slot(command, target), values, diagnostic);
  else refusals.delete(slot(command, target));
}

export function refusedNow(command: string, target: string): boolean {
  return refusals.has(slot(command, target));
}

export function refusedValues(command: string, target: string): Record<string, string> {
  try {
    const v = JSON.parse(refusals.get(slot(command, target)) ?? "") as unknown;
    return v && typeof v === "object" && !Array.isArray(v) ? (v as Record<string, string>) : {};
  } catch {
    return {};
  }
}

export function refusedLicense(page: IntegrationsPage, plugin: string, kind: string, version: string): string {
  const target = kind === "enable" ? installOf(page, plugin)?.version ?? "" : kind === "install" || kind === "upgrade" || kind === "rollback" ? version : "";
  return target && licenseRefused(page, plugin, target) ? target : "";
}

export function pruneRefusals(page: IntegrationsPage, me: string): void {
  for (const id of [...refusals.keys()]) {
    const at = unslot(id);
    if (!at) {
      refusals.delete(id);
      continue;
    }
    if (!listed(page, at.command, at.target) || !stillAllowed(page, me, at.command, at.target)) refusals.delete(id);
  }
}

export type Vanished = { command: CommandKey; target: string; context: Record<string, string> };

export function vanished(page: IntegrationsPage, view: string): Vanished[] {
  const out: Vanished[] = [];
  for (const [id, p] of pending) {
    const at = unslot(id);
    if (!at || viewOfCommand(at.command) !== view || listed(page, at.command, at.target)) continue;
    out.push({ command: at.command, target: at.target, context: p.context });
  }
  return out;
}

export function draftOf(command: string, target: string): Record<string, string> {
  return drafts.get(slot(command, target)) ?? {};
}

export function keepDraft(command: string, target: string, values: Record<string, string>): void {
  drafts.set(slot(command, target), { ...draftOf(command, target), ...values });
}

export function clearDraft(command: string, target: string): void {
  drafts.delete(slot(command, target));
}

export function snapshotOf(page: IntegrationsPage, command: CommandKey): string[] {
  if (command === "integrations.plugin_propose") return (page.modules.proposals ?? []).flatMap((p) => (p.number === null ? [] : [String(p.number)]));
  if (command === "integrations.subscription_propose") return (page.channels.subscriptions ?? []).map((s) => s.id);
  return [];
}

export function duplicateIn(snapshot: string[], id: string): boolean {
  return !!id && snapshot.includes(id);
}

export type Denied = "applied_letter" | "applied_proposal" | "gone" | "no_right";

export function deniedOutcome(command: CommandKey, target: string, hadPending: boolean, page: IntegrationsPage): Denied {
  if (page.rights !== null && !page.rights[commandRight[command]] && !authorOnly(command)) return "no_right";
  if (!hadPending || listed(page, command, target)) return "gone";
  if (command === "integrations.letter_approve_redeliver" || command === "integrations.letter_approve_dismiss") return "applied_letter";
  if (groupOf(command) === "proposal") return "applied_proposal";
  return "gone";
}

export function refusalDiagnostics(target: string): RowDiagnostic[] {
  return commandKeys.flatMap((command) => {
    const entry = refusals.detail(slot(command, target));
    return entry && entry.command === command && entry.detail.target === target ? [entry] : [];
  });
}

export function feedbackDiagnostics(choices: Array<[CommandKey, string]>): RowDiagnostic[] {
  return choices.flatMap(([command, target]) => {
    const denied = lostDiagnostics(target).find((entry) => entry.command === command);
    const entry = denied ?? refusals.detail(slot(command, target));
    return entry && entry.command === command && entry.detail.target === target ? [entry] : [];
  });
}
