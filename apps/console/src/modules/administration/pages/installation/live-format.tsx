import type { Locale } from "../../../../shared/types";
import { timeKnown } from "./live-model";
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
