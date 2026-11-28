import type { Locale } from "../../../../shared/types";
import { AdminError, type PolicyPage, type RevisionRow } from "../../../../service/admin";
import { c, retryPhrase } from "../../../../service/copy";
import { w } from "../users/live-copy";
import { p } from "./live-copy";

const tags: Record<Locale, string> = { en: "en-US", ko: "ko-KR", ar: "ar" };

export function duration(l: Locale, seconds: number): string {
  const minutes = Math.round(seconds / 60);
  const unit = (n: number, kind: "minute" | "hour") => new Intl.NumberFormat(tags[l], { style: "unit", unit: kind, unitDisplay: "long" }).format(n);
  return minutes >= 60 && minutes % 60 === 0 ? `${unit(minutes, "minute")} (${unit(minutes / 60, "hour")})` : unit(minutes, "minute");
}

export function when(l: Locale, iso: string): string {
  const d = new Date(iso);
  if (!iso || Number.isNaN(d.getTime())) return "";
  return new Intl.DateTimeFormat(tags[l], { dateStyle: "medium", timeStyle: "short" }).format(d);
}

export function policyStep(r: RevisionRow): number {
  return r.step_up.find((s) => s.action === "policy.change" && !s.mfa)?.max_age_seconds ?? 0;
}

export const defaultStepUp = 900;

export function stepText(l: Locale, r: RevisionRow): string {
  const step = policyStep(r);
  return step ? p.within(l, duration(l, step)) : p.stepDefault(l, duration(l, defaultStepUp));
}

export function effectiveStep(page: PolicyPage): number {
  return page.step_up_seconds > 0 ? page.step_up_seconds : defaultStepUp;
}

export function stepMinutes(raw: string, maxAge: number | null): number | null {
  const trimmed = raw.trim();
  if (trimmed === "" || trimmed === "0") return 0;
  return wholeMinutes(trimmed, 5, maxAge ?? 720);
}

export function methodsText(l: Locale, r: RevisionRow): string {
  return r.methods.length === 1 && r.methods[0] === "password" ? p.passwordOnly(l) : r.methods.join(", ");
}

export function stale(page: PolicyPage): boolean {
  return page.act_until !== "" && page.now > page.act_until;
}

export function wholeMinutes(raw: string, low: number, high: number): number | null {
  if (!/^[1-9][0-9]{0,3}$/.test(raw)) return null;
  const n = Number(raw);
  return n >= low && n <= high ? n : null;
}

export function policyFailure(l: Locale, e: unknown): { title: string; body: string } {
  if (!(e instanceof AdminError)) return { title: w.unknownTitle(l), body: w.unknownBody(l) };
  switch (e.failure) {
    case "unavailable":
      return { title: w.unknownTitle(l), body: w.unknownBody(l) };
    case "refused":
      return { title: p.refusedTitle(l), body: p.refusedBody(l) };
    case "stage":
      return { title: p.saveFailed(l), body: p.stageBody(l) };
    case "conflict":
    case "stale":
      return { title: p.saveFailed(l), body: p.conflictBody(l) };
    case "busy":
      return { title: p.saveFailed(l), body: p.busyBody(l) };
    case "denied":
      return { title: p.deniedTitle(l), body: p.denied(l) };
    case "unauthenticated":
      return { title: p.saveFailed(l), body: w.ended(l) };
    case "limited":
      return { title: p.saveFailed(l), body: c.limited(l, retryPhrase(l, e.retryAfter || 60)) };
    case "partial":
      return { title: w.unknownTitle(l), body: w.partial(l, e.operation) };
    case "invalid":
      return e.code === "invalid_request" ? { title: c.check(l), body: w.invalidFallback(l) } : { title: w.unknownTitle(l), body: w.unknownBody(l) };
    case "reauth":
    case "reauth_cancelled":
      return { title: p.saveFailed(l), body: c.reauthText(l, e.failure) };
    default:
      return { title: p.saveFailed(l), body: w.unavailable(l) };
  }
}
