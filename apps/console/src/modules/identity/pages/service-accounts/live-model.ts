import type { Locale } from "../../../../shared/types";
import type { CredentialRow, ServiceRow, ServicesPage } from "../../../../service/admin";
import { when } from "../authentication/live-model";
import { v } from "./live-copy";

export const prefix = "svc-";

export function validName(name: string): boolean {
  return /^[a-z0-9]+(-[a-z0-9]+)*$/.test(name) && name.length >= 3 && name.length <= 48;
}

export function wholeDays(raw: string, max: number): number | null {
  if (!/^[1-9][0-9]{0,2}$/.test(raw)) return null;
  const n = Number(raw);
  return n <= max ? n : null;
}

export function split(page: ServicesPage): { current: ServiceRow[]; retired: ServiceRow[] } {
  return { current: page.services.filter((s) => s.state !== "retired"), retired: page.services.filter((s) => s.state === "retired") };
}

export function liveCount(s: ServiceRow): number {
  return s.credentials.filter((c) => c.state === "active").length;
}

export function credentials(s: ServiceRow): { current: CredentialRow[]; ended: CredentialRow[] } {
  return { current: s.credentials.filter((c) => c.state === "active"), ended: s.credentials.filter((c) => c.state !== "active") };
}

export function serviceState(l: Locale, state: ServiceRow["state"]): string {
  return state === "suspended" ? v.stSuspended(l) : state === "retired" ? v.stRetired(l) : v.stActive(l);
}

export function credentialState(l: Locale, c: CredentialRow, retired = false): [string, string] {
  if (c.state === "revoked") return [v.cRevoked(l), v.ended(l, when(l, c.not_after))];
  if (c.state === "expired") return [v.cExpired(l), v.ended(l, when(l, c.not_after))];
  if (retired) return [v.cRevokeRequired(l), v.cRevokeRequiredSub(l)];
  if (!c.usable) return [v.cOnHold(l), v.cOnHoldSub(l)];
  return [v.cActive(l), ""];
}

export type Block = "" | "suspended" | "retired" | "limit";

export function issueBlock(s: ServiceRow, limit: number): Block {
  if (s.state === "retired") return "retired";
  if (s.state === "suspended") return "suspended";
  return liveCount(s) >= limit ? "limit" : "";
}

export function initialService(page: ServicesPage, wanted: string): string {
  if (page.services.some((s) => s.id === wanted)) return wanted;
  return (page.services.find((s) => s.state === "active") ?? page.services.find((s) => s.state !== "retired") ?? page.services[0])?.id ?? "";
}

export function keepsHold(failure: string, holding: boolean): boolean {
  if (failure === "busy" || failure === "unavailable") return true;
  if (failure === "partial") return !holding;
  return holding && (failure === "limited" || failure === "unauthenticated");
}
