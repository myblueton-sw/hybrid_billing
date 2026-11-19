import type { Delivery, IntegrationsPage, OwnPending, PendingKind, PluginInstall, PluginProposal, RightId, Subscription } from "../../../../service/integrations";
import { rightIds } from "../../../../service/integrations";
import type { CommandKey } from "./live-model";

export const actionRight: Record<string, RightId> = {
  plugin_approve: "approve_modules",
  propose_plugin: "administer_modules",
  stop: "stop_modules",
  disable: "stop_modules",
  approve: "approve_channels",
  approve_resume: "approve_channels",
  pause: "stop_channels",
  revoke: "stop_channels",
  request_resume: "administer_channels",
  propose_subscription: "administer_channels",
  request_redeliver: "reprocess_events",
  approve_redeliver: "reprocess_events",
  request_dismiss: "reprocess_events",
  approve_dismiss: "reprocess_events",
  reapprove: "reprocess_events",
};

export function heldRights(page: IntegrationsPage): RightId[] {
  if (page.rights === null) return [];
  return rightIds.filter((id) => page.rights?.[id] === true);
}

export const areaOf: Record<string, "clients" | "modules" | "channels" | "events"> = { clients: "clients", modules: "modules", channels: "channels", events: "events" };

export function withheld(page: IntegrationsPage, view: string): boolean {
  const area = areaOf[view];
  return !!area && page[area].state === "not_permitted";
}

const pendingView: Record<PendingKind, string> = {
  plugin_proposal: "modules",
  subscription_proposal: "channels",
  resume_request: "channels",
  redeliver_request: "events",
  dismiss_request: "events",
};

export const pendingCommand: Record<PendingKind, CommandKey> = {
  plugin_proposal: "integrations.plugin_withdraw",
  subscription_proposal: "integrations.subscription_withdraw",
  resume_request: "integrations.subscription_withdraw_resume",
  redeliver_request: "integrations.letter_withdraw_redeliver",
  dismiss_request: "integrations.letter_withdraw_dismiss",
};

export function ownPendingFor(page: IntegrationsPage, view: string): OwnPending[] {
  return (page.own_pending ?? []).filter((p) => pendingView[p.kind] === view && p.state === "pending" && /^[A-Za-z0-9._:-]{1,128}$/.test(p.target));
}

export function installApplies(i: PluginInstall, action: "stop" | "disable"): boolean {
  return action === "stop" ? i.state === "enabled" : i.state === "enabled" || i.state === "stopped";
}

export function proposalApplies(p: PluginProposal): boolean {
  return p.state === "pending";
}

export function subscriptionApplies(s: Subscription, action: string): boolean {
  const paused = s.state === "paused" || s.state === "suspended";
  switch (action) {
    case "approve":
      return s.state === "proposed";
    case "approve_resume":
      return paused && s.resumer !== "";
    case "pause":
      return s.state === "active";
    case "request_resume":
      return paused && s.resumer === "";
    case "revoke":
      return s.state !== "revoked";
  }
  return false;
}

export function letterApplies(d: Delivery, action: string): boolean {
  if (action === "reapprove") return d.hold !== "";
  if (d.kind !== "dead") return false;
  const request = action.endsWith("redeliver") ? d.redeliver_request : d.dismiss_request;
  return action.startsWith("request_") ? request === null : request !== null && request !== "unknown";
}

const lost = new Map<string, RightId>();

function slot(command: string, target: string): string {
  return command + "\u0000" + target;
}

export function markLost(command: CommandKey, target: string, right: RightId): void {
  lost.set(slot(command, target), right);
}

export function lostRight(command: CommandKey, target: string): RightId | "" {
  return lost.get(slot(command, target)) ?? "";
}

export function lostAny(commands: CommandKey[], target: string): RightId | "" {
  for (const c of commands) {
    const r = lostRight(c, target);
    if (r) return r;
  }
  return "";
}

export function pruneLost(page: IntegrationsPage): void {
  for (const [id, right] of [...lost]) if (page.rights !== null && page.rights[right]) lost.delete(id);
}

export function forgetLost(): void {
  lost.clear();
}
