import type { ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { actor, generationNumber, lostOf, timeKnown, type Basis } from "./live-model";
import { k } from "./live-copy";

const tags: Record<Locale, string> = { en: "en-US", ko: "ko-KR", ar: "ar" };

export function stamp(l: Locale, iso: string): string {
  if (!timeKnown(iso)) return k.unknown(l);
  return new Intl.DateTimeFormat(tags[l], { dateStyle: "medium", timeStyle: "short", timeZone: "UTC" }).format(new Date(iso)) + " UTC";
}

export const fsi = (s: string) => "⁨" + s + "⁩";

export function stampParts(l: Locale, at: string): { date: string; time: string } | null {
  if (!timeKnown(at)) return null;
  const parts = new Intl.DateTimeFormat(tags[l], { dateStyle: "medium", timeStyle: "short", timeZone: "UTC" }).formatToParts(new Date(at));
  const first = parts.findIndex((x) => x.type === "hour" || x.type === "dayPeriod");
  if (first <= 0) return { date: parts.map((x) => x.value).join(""), time: "UTC" };
  const sep = parts[first - 1].type === "literal" ? first - 1 : first;
  const date = parts.slice(0, sep).map((x) => x.value).join("") + (sep < first ? parts[sep].value.trimEnd() : "");
  return { date, time: parts.slice(first).map((x) => x.value).join("") + " UTC" };
}

export function Stamp({ l, at }: { l: Locale; at: string }) {
  const s = stampParts(l, at);
  if (!s) return <>{k.unknown(l)}</>;
  return <span className="p03-stamp"><span className="p03-nobr">{s.date}</span> <span className="p03-nobr">{s.time}</span></span>;
}

export function Rich({ text }: { text: string }) {
  const pieces = text.split(/(\uE000[^\uE001]*\uE001)/);
  return (
    <>
      {pieces.map((piece, n) =>
        piece.startsWith("\uE000") ? <bdi key={n} dir="ltr" className="live-mono p03-nobr">{piece.slice(1, -1)}</bdi> : <span key={n}>{piece}</span>,
      )}
    </>
  );
}

export const ltr = (s: string, wrap = false) => <bdi dir="ltr" className={wrap ? "live-mono p03-wrap" : "live-mono"}>{s}</bdi>;

export function Known({ l, value, wrap = false }: { l: Locale; value: string; wrap?: boolean }) {
  if (!value) return <>{k.unknown(l)}</>;
  return ltr(value, wrap);
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
  return who ? <bdi dir="ltr" className="p03-login">{who}</bdi> : <>{k.unknown(l)}</>;
}

export function Prose({ l, value, empty }: { l: Locale; value: string; empty?: string }) {
  return value ? <bdi>{value}</bdi> : <>{empty ?? k.unknown(l)}</>;
}

export function ListOr<T>({ l, items, empty, children }: { l: Locale; items: T[] | null; empty?: string; children: (items: T[]) => ReactNode }) {
  if (items === null) return <>{k.unknown(l)}</>;
  if (items.length === 0) return <>{empty ?? k.none(l)}</>;
  return <>{children(items)}</>;
}

export function Template({ text, slots }: { text: string; slots: Record<string, ReactNode> }) {
  const pieces = text.split(/(\{[a-z_]+\})/);
  return (
    <>
      {pieces.map((piece, n) => {
        const m = /^\{([a-z_]+)\}$/.exec(piece);
        if (m && m[1] in slots) return <span key={n}>{slots[m[1]]}</span>;
        return <span key={n}>{piece}</span>;
      })}
    </>
  );
}

export function href(l: Locale, view: string, extra: Record<string, string> = {}): string {
  const q = new URLSearchParams({ lang: l, view, ...extra });
  return "#/recovery?" + q.toString();
}

export function LostText({ l, list }: { l: Locale; list: number[] | null }) {
  const x = lostOf(list);
  if (x.kind === "unknown") return <>{k.unknown(l)}</>;
  if (x.kind === "none") return <>{k.none(l)}</>;
  return <>{ltr(x.shown.join(", "), true)}{x.rest > 0 ? k.andMore(l, x.rest) : null}</>;
}

export function ModeImpact({ l, b, lead }: { l: Locale; b: Basis; lead: boolean }) {
  const body =
    b.mode === "lock_release" ? k.releaseImpact(l, b.installation, generationNumber(b))
      : b.mode === "recovery_resume" ? k.resumeImpact(l, b.installation, b.recovery || k.unknown(l))
        : "";
  return <p>{lead ? k.proposeLead(l) + (body ? " " : "") : ""}<Rich text={body} /></p>;
}

export function InstallNow({ l, b }: { l: Locale; b: Basis }) {
  const g = generationNumber(b);
  return (
    <>
      <Code l={l} value={b.mode} say={k.modeWord} /> · <Code l={l} value={b.state} say={k.installState} /> · {k.colGeneration(l)} {g === null ? k.unknown(l) : <bdi dir="ltr">{g}</bdi>}
    </>
  );
}
