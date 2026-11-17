import type { Locale } from "../../../../shared/types";
import type { OrgParty } from "../../../../service/admin";
import { validDay } from "./live-model";

const tags: Record<Locale, string> = { en: "en-US", ko: "ko-KR", ar: "ar" };

export function day(l: Locale, iso: string): string {
  if (!validDay(iso)) return iso;
  return new Intl.DateTimeFormat(tags[l], { dateStyle: "medium", timeZone: "UTC" }).format(new Date(iso + "T00:00:00Z"));
}

export function stamp(l: Locale, iso: string): string {
  const d = new Date(iso);
  if (!iso || Number.isNaN(d.getTime())) return iso;
  return new Intl.DateTimeFormat(tags[l], { dateStyle: "medium", timeStyle: "short", timeZone: "UTC" }).format(d) + " UTC";
}

export const ltr = (s: string) => <bdi dir="ltr" className="live-mono">{s}</bdi>;

export function nameOf(id: string, parties: OrgParty[]): string {
  if (!id) return "";
  return parties.find((p) => p.id === id)?.name || id;
}
