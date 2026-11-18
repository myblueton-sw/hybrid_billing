import type { Locale } from "../../../../shared/types";
import type { Membership, UserRow } from "../../../../service/admin";
import { tags } from "./live-model";

export function offsetLabel(minutes: number): string {
  const sign = minutes > 0 ? "-" : "+";
  const abs = Math.abs(minutes);
  return `UTC${sign}${String(Math.floor(abs / 60)).padStart(2, "0")}:${String(abs % 60).padStart(2, "0")}`;
}

export function zoneOf(d: Date): string {
  return offsetLabel(d.getTimezoneOffset());
}

const pad = (n: number) => String(n).padStart(2, "0");

export function localInput(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export function utcOf(local: string): string {
  const m = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$/.exec(local);
  if (!m) return "";
  const d = new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]), Number(m[4]), Number(m[5]), 0, 0);
  if (Number.isNaN(d.getTime()) || d.getFullYear() !== Number(m[1]) || d.getMonth() !== Number(m[2]) - 1 || d.getDate() !== Number(m[3])) return "";
  return d.toISOString().replace(/:00\.000Z$/, ":00Z");
}

export function dateParts(l: Locale, iso: string): { date: string; time: string; zone: string } {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return { date: "", time: "", zone: "" };
  return {
    date: new Intl.DateTimeFormat(tags[l], { dateStyle: "medium" }).format(d),
    time: new Intl.DateTimeFormat(tags[l], { timeStyle: "short" }).format(d),
    zone: zoneOf(d),
  };
}

export type Remaining = { days: number; today: boolean };

export function remaining(until: string, serverTime: string): Remaining {
  const ms = new Date(until).getTime() - new Date(serverTime).getTime();
  if (!Number.isFinite(ms)) return { days: 0, today: false };
  const day = 24 * 3600 * 1000;
  if (ms < day) return { days: 0, today: true };
  return { days: Math.floor(ms / day), today: false };
}

export type PeriodView = { kind: "open" | "bounded" | "ends_soon" | "ended" | "none"; until: string };

export function periodOf(u: UserRow): PeriodView {
  const m = u.membership;
  if (!m) return { kind: "none", until: "" };
  return { kind: m.role_period, until: m.role_until };
}

export type Direction = "shorter" | "longer" | "same";

export function direction(m: Membership, next: string): Direction {
  const restoring = m.suspended_for === "role_period_ended" || m.role_period === "ended";
  if (restoring) return "longer";
  if (m.role_until === next) return "same";
  if (!next) return "longer";
  if (!m.role_until) return "shorter";
  return new Date(next).getTime() > new Date(m.role_until).getTime() ? "longer" : "shorter";
}

export type Bounds = { min: string; max: string; own: string };

export function bounds(serverTime: string, maxDays: number, own: string): Bounds {
  const now = new Date(serverTime).getTime();
  if (!Number.isFinite(now)) return { min: "", max: "", own };
  const min = new Date(Math.ceil((now + 60000) / 60000) * 60000).toISOString();
  let max = new Date(Math.floor((now + maxDays * 24 * 3600 * 1000) / 60000) * 60000).toISOString();
  if (own && new Date(own).getTime() < new Date(max).getTime()) max = own;
  return { min, max, own };
}

export type EndProblem = "" | "missing" | "early" | "late" | "own";

export function endProblem(utc: string, b: Bounds): EndProblem {
  if (!utc) return "missing";
  const t = new Date(utc).getTime();
  if (b.min && t < new Date(b.min).getTime()) return "early";
  if (b.own && t > new Date(b.own).getTime()) return "own";
  if (b.max && t > new Date(b.max).getTime()) return "late";
  return "";
}

export function setting(rows: { name: string; value: number }[], name: string, fallback: number): number {
  return rows.find((r) => r.name === name)?.value ?? fallback;
}

export type EndValue = { kind: "at" | "none"; local: string };

export function initialEnd(stored: string, own: string): EndValue {
  if (stored) return { kind: "at", local: localInput(stored) };
  if (own) return { kind: "at", local: localInput(own) };
  return { kind: "none", local: "" };
}

export function endFields(v: EndValue): { values: Record<string, string>; utc: string } {
  if (v.kind === "none") return { values: {}, utc: "" };
  const utc = utcOf(v.local);
  return { values: utc ? { role_until: utc } : {}, utc };
}
