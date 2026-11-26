import type { Locale } from "../../../../shared/types";
import { AdminError, type Attempt, type CredentialRow } from "../../../../service/admin";
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

export function wholeMinutes(raw: string, min: number, max: number): number | null {
  if (!/^[1-9][0-9]{0,5}$/.test(raw)) return null;
  const n = Number(raw);
  return n >= min && n <= max ? n : null;
}

export function rotationEnd(now: string, notAfter: string, minutes: number): { at: string; sooner: boolean } | null {
  const from = Date.parse(now);
  const present = Date.parse(notAfter);
  if (Number.isNaN(from) || Number.isNaN(present)) return null;
  const wanted = from + minutes * 60000;
  return present < wanted ? { at: new Date(present).toISOString(), sooner: true } : { at: new Date(wanted).toISOString(), sooner: false };
}

export function utcTime(l: Locale, iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return new Intl.DateTimeFormat({ en: "en-US", ko: "ko-KR", ar: "ar" }[l], { year: "numeric", month: l === "en" ? "short" : "numeric", day: "numeric", hour: "numeric", minute: "2-digit", timeZone: "UTC", timeZoneName: "short" }).format(d);
}

export function canRotate(service: "active" | "suspended" | "retired", c: { state: string; rotation?: "replacing" | "ending"; not_before: string }, now: string): boolean {
  return service === "active" && c.state === "active" && c.rotation !== "ending" && !(Date.parse(c.not_before) > Date.parse(now));
}

export function serviceState(l: Locale, state: "active" | "suspended" | "retired"): string {
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

export function keepsHold(failure: string, holding: boolean): boolean {
  if (failure === "busy" || failure === "unavailable") return true;
  if (failure === "partial") return !holding;
  return holding && (failure === "limited" || failure === "unauthenticated");
}

export function panelShown(wide: boolean, form: boolean, open: boolean): boolean {
  return wide || !form || open;
}

export function pinIssue(service: string): { chosen: string; open: boolean } {
  return { chosen: service, open: true };
}

export function attemptState(fresh: () => Attempt) {
  let tried = fresh();
  let held = "";
  let sending = false;
  let spent = false;
  return {
    held: () => held,
    request: () => tried.request,
    revised() {
      if (held || sending) return;
      tried = fresh();
      spent = false;
    },
    send(revision: string): { attempt: Attempt; expected: string } {
      if (!held && spent) { tried = fresh(); spent = false; }
      sending = true;
      return { attempt: tried, expected: held || revision };
    },
    succeeded() { held = ""; spent = true; },
    keep(revision: string) { held = held || revision; },
    renew(): boolean {
      if (sending) return false;
      held = "";
      tried = fresh();
      spent = false;
      return true;
    },
    sending: () => sending,
    idle() { sending = false; },
  };
}

export type Track = ReturnType<typeof attemptState>;
export type NextPhase = "unknown" | "lapsed" | "remove";

export function nextPhase(retry: boolean, err: unknown, outside: boolean): NextPhase {
  if (!(err instanceof AdminError)) return "unknown";
  switch (err.failure) {
    case "busy":
    case "unavailable":
    case "limited":
    case "unauthenticated":
      return "unknown";
    case "partial":
      return retry ? "lapsed" : "unknown";
    case "reauth":
    case "reauth_cancelled":
      return retry ? "unknown" : "remove";
    case "conflict":
    case "refused":
    case "stage":
    case "invalid":
      return retry && outside ? "lapsed" : "remove";
    case "denied":
      return retry ? "lapsed" : "remove";
  }
  return "lapsed";
}

export function outsideWindow(elapsedMs: number, windowSeconds: number, unknownIsOutside = false): boolean {
  if (windowSeconds <= 0) return unknownIsOutside;
  const margin = Math.min(60, windowSeconds / 10);
  return elapsedMs >= (windowSeconds - margin) * 1000;
}

export function settleFailure(track: Track, err: unknown, holdUnknown: boolean, revision: string, outside = false): NextPhase | "kept" {
  const retry = track.held() !== "";
  track.idle();
  if (!holdUnknown) {
    if (err instanceof AdminError && (err.failure === "reauth" || err.failure === "reauth_cancelled")) return "kept";
    track.renew();
    return "remove";
  }
  const next = nextPhase(retry, err, outside);
  if (next === "unknown") track.keep(revision);
  if (next === "remove") track.renew();
  return next;
}

export function returnFocus(from: string, credentialsView: boolean): string[] {
  return [from, credentialsView ? "#svc-cred-title" : "#svc-list-title"].filter(Boolean);
}

export function pickupLabel(l: Locale, c: CredentialRow): { text: string; time: string; never: boolean } | null {
  if (c.state !== "active") return null;
  if (c.secret_pickup === "never_collected") return { text: v.pickupNever(l), time: "", never: true };
  if (c.secret_pickup === "open" && c.secret_pickup_until) return { text: v.pickupOpen(l), time: when(l, c.secret_pickup_until), never: false };
  return null;
}

export function formVisible(o: { view: string; mode: string; current: string; service: string; wide: boolean; open: boolean; visibility: string }): boolean {
  return o.view === "credentials" && o.mode === "form" && o.current === o.service && panelShown(o.wide, true, o.open) && o.visibility === "visible";
}
