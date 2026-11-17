import type { ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import type { MarkValue } from "../../../../service/integrations";
import { actor, timeKnown } from "./live-model";
import { k } from "./live-copy";

const tags: Record<Locale, string> = { en: "en-US", ko: "ko-KR", ar: "ar" };

export function stamp(l: Locale, iso: string): string {
  if (!timeKnown(iso)) return k.unknown(l);
  return new Intl.DateTimeFormat(tags[l], { dateStyle: "medium", timeStyle: "short", timeZone: "UTC" }).format(new Date(iso)) + " UTC";
}

export const fsi = (s: string) => "⁨" + s + "⁩";

export const ltr = (s: string) => <bdi dir="ltr" className="live-mono">{s}</bdi>;

export function Known({ l, value, mono = true }: { l: Locale; value: string; mono?: boolean }) {
  if (!value) return <>{k.unknown(l)}</>;
  return mono ? ltr(value) : <bdi dir="ltr">{value}</bdi>;
}

export function Code({ l, value, say }: { l: Locale; value: string; say: (l: Locale, s: string) => string }) {
  const text = say(l, value);
  if (text !== k.unknownCode(l, value)) return <>{text}</>;
  return value ? <>{k.unknown(l)} ({ltr(value)})</> : <>{k.unknown(l)}</>;
}

export function Num({ l, n }: { l: Locale; n: number | null }) {
  return n === null ? <>{k.unknown(l)}</> : <bdi dir="ltr">{n}</bdi>;
}

export function Who({ l, login, id }: { l: Locale; login: string; id: string }) {
  const who = actor(login, id);
  return who ? <bdi dir="ltr" className="p02-login">{who}</bdi> : <>{k.unknown(l)}</>;
}

export function Prose({ l, value, empty }: { l: Locale; value: string; empty?: string }) {
  return value ? <bdi>{value}</bdi> : <>{empty ?? k.unknown(l)}</>;
}

export function MarkText({ l, m, lead }: { l: Locale; m: MarkValue; lead: (l: Locale, who: string) => string }) {
  if (m === null) return <>{k.none(l)}</>;
  if (m === "unknown") return <>{k.unknown(l)}</>;
  const by = actor(m.by_login, m.by) || k.unknown(l);
  const confirmed = actor(m.confirmed_by_login, m.confirmed_by);
  return (
    <>
      {lead(l, by)} <Prose l={l} value={m.reason} />
      {confirmed && <span className="live-sub">{k.confirmedBy(l, confirmed)}</span>}
    </>
  );
}

export function ListOr<T>({ l, items, empty, children }: { l: Locale; items: T[] | null; empty?: string; children: (items: T[]) => ReactNode }) {
  if (items === null) return <>{k.unknown(l)}</>;
  if (items.length === 0) return <>{empty ?? k.none(l)}</>;
  return <>{children(items)}</>;
}

export function href(l: Locale, view: string, extra: Record<string, string> = {}): string {
  const q = new URLSearchParams({ lang: l, view, ...extra });
  return "#/integrations?" + q.toString();
}
