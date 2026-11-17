import type { ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import type { HoldRow, Person } from "../../../../service/retention";
import { actor, textProblem, timeKnown } from "./live-model";
import { k, mono } from "./live-copy";
import { counted, type Plural } from "./live-codes";

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
  return <span className="p04-stamp"><span className="p04-nobr">{s.date}</span> <span className="p04-nobr">{s.time}</span></span>;
}

export function Rich({ text }: { text: string }) {
  const pieces = text.split(/([^]*)/);
  return (
    <>
      {pieces.map((piece, n) =>
        piece.startsWith("") ? <bdi key={n} dir="ltr" className="live-mono p04-nobr">{piece.slice(1, -1)}</bdi> : <span key={n}>{piece}</span>,
      )}
    </>
  );
}

export const ltr = (s: string, wrap = false) => <bdi dir="ltr" className={wrap ? "live-mono p04-wrap" : "live-mono"}>{s}</bdi>;

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

export function Who({ l, p, empty }: { l: Locale; p: Person; empty?: string }) {
  const who = actor(p);
  return who ? <bdi dir="ltr" className="live-mono p04-login">{who}</bdi> : <>{empty ?? k.unknown(l)}</>;
}

export function Free({ l, value, empty }: { l: Locale; value: string; empty?: string }) {
  return value ? <span dir="auto" className="p04-free">{value}</span> : <>{empty ?? k.unknown(l)}</>;
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
  return "#/retention?" + q.toString();
}

export function coversText(l: Locale, h: HoldRow): string {
  const parts: string[] = [];
  if (h.scope.workspace === true) {
    const cls = h.scope.classes;
    const which = cls === null ? k.unknown(l) : cls.length === 0 ? k.allClasses(l) : cls.map((c) => k.className(l, c)).join(", ");
    parts.push(k.thisWorkspace(l) + " · " + which);
  }
  if (h.scope.artifacts === null || h.scope.workspace === null) return parts.length ? parts.join("; ") + "; " + k.unknown(l) : k.unknown(l);
  if (h.scope.artifacts.length > 0) parts.push(h.scope.artifacts.map(mono).join(", "));
  return parts.length ? parts.join("; ") : k.nothingCovered(l);
}

export function holdContext(l: Locale, h: HoldRow | undefined): Record<string, string> {
  if (!h) return { hold_known: "" };
  const c = h.counts;
  const n = (key: string) => (c && c[key] !== null && c[key] !== undefined ? String(c[key]) : "");
  return {
    hold_known: "1",
    hold: h.id,
    hold_owner: actor(h.owner),
    hold_covers: coversText(l, h),
    hold_scope: h.counts_scope === "workspace" && c ? "workspace" : h.counts_scope === "other_scope" ? "other_scope" : "unknown",
    hold_a: n("not_preserved"),
    hold_b: n("outcome_pending"),
    hold_c: n("partially_deleted"),
  };
}

const ctxNum = (v: string | undefined): number | null => (v !== undefined && /^(0|[1-9][0-9]*)$/.test(v) ? Number(v) : null);

export function HoldFacts({ l, ctx, prefix = true }: { l: Locale; ctx: Record<string, string>; prefix?: boolean }) {
  const known = ctx.hold_known === "1";
  const scope = ctx.hold_scope;
  return (
    <>
      {prefix && <div><dt>{k.factHold(l)}</dt><dd><Known l={l} value={ctx.hold ?? ""} /></dd></div>}
      <div><dt>{k.factOwner(l)}</dt><dd>{known && ctx.hold_owner ? <bdi dir="ltr" className="live-mono p04-login">{ctx.hold_owner}</bdi> : k.unknown(l)}</dd></div>
      <div className="p04-span"><dt>{k.covers(l)}</dt><dd>{known ? <Rich text={ctx.hold_covers ?? ""} /> : k.unknown(l)}</dd></div>
      {scope === "other_scope" ? (
        <div className="p04-span"><dt>{k.cannotPreserve(l)}</dt><dd>{k.otherScopeCounts(l)}</dd></div>
      ) : (
        <>
          <div><dt>{k.holdOutcome(l, "not_preserved")}</dt><dd><Num l={l} n={scope === "workspace" ? ctxNum(ctx.hold_a) : null} /></dd></div>
          <div><dt>{k.beingDeleted(l)}</dt><dd><Num l={l} n={scope === "workspace" ? ctxNum(ctx.hold_b) : null} /></dd></div>
          <div><dt>{k.holdOutcome(l, "partially_deleted")}</dt><dd><Num l={l} n={scope === "workspace" ? ctxNum(ctx.hold_c) : null} /></dd></div>
        </>
      )}
    </>
  );
}

export function holdTextError(l: Locale, v: string, release = false): string {
  const p = textProblem(v);
  if (p === "missing") return release ? k.form.releaseReasonMissing(l) : k.form.holdReasonMissing(l);
  if (p === "long") return release ? k.form.releaseReasonLong(l) : k.form.holdReasonLong(l);
  if (p === "control") return k.form.textControl(l);
  return "";
}

export function MoreLine({ l, more, total, forms }: { l: Locale; more: boolean | null; total: number | null; forms: Plural }) {
  if (more === false) return null;
  const text = more === true && total !== null ? counted(l, total, forms) : k.maybeMore(l);
  return <p className="p04-small live-quiet">{text}</p>;
}
