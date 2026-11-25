import { useLayoutEffect, useState, type ReactNode, type RefObject } from "react";
import type { Locale } from "../../../../shared/types";
import type { Candidate, Person } from "../../../../service/release-evidence";
import { commitPattern, digestPattern, timeKnown, type Absent, type Locator } from "./live-model";
import { k } from "./live-copy";
import { counted, fill, has, type Plural, type Words } from "./live-codes";

const tags: Record<Locale, string> = { en: "en-US", ko: "ko-KR", ar: "ar" };

export function stampParts(l: Locale, at: string | null): { date: string; time: string } | null {
  if (!timeKnown(at)) return null;
  const parts = new Intl.DateTimeFormat(tags[l], { dateStyle: "medium", timeStyle: "short", timeZone: "UTC" }).formatToParts(new Date(at));
  const first = parts.findIndex((x) => x.type === "hour" || x.type === "dayPeriod");
  if (first <= 0) return { date: parts.map((x) => x.value).join(""), time: "UTC" };
  const sep = parts[first - 1].type === "literal" ? first - 1 : first;
  const date = parts.slice(0, sep).map((x) => x.value).join("") + (sep < first ? parts[sep].value.trimEnd() : "");
  return { date, time: parts.slice(first).map((x) => x.value).join("") + " UTC" };
}

export function stamp(l: Locale, at: string | null): string {
  const s = stampParts(l, at);
  return s ? "\u2068" + s.date + " " + s.time + "\u2069" : k.unknown(l);
}

export function Stamp({ l, at }: { l: Locale; at: string | null }) {
  const s = stampParts(l, at);
  if (!s) return <span className="p05-absent">{k.unknown(l)}</span>;
  return <span className="p05-stamp"><span className="p05-nobr">{s.date}</span> <span className="p05-nobr">{s.time}</span></span>;
}

export const ltr = (s: string, wrap = false) => <bdi dir="ltr" className={wrap ? "live-mono p05-wrap" : "live-mono"}>{s}</bdi>;

export function Label({ value }: { value: string }) {
  const at = value.indexOf("/");
  if (at < 0) return <bdi dir="ltr" className="live-mono">{value}</bdi>;
  return <bdi dir="ltr" className="live-mono p05-label"><span className="p05-seg">{value.slice(0, at + 1)}</span><wbr /><span className="p05-id-seg">{value.slice(at + 1)}</span></bdi>;
}

export function Absent({ children }: { children: ReactNode }) {
  return <span className="p05-absent">{children}</span>;
}

export function Gap({ l, why }: { l: Locale; why: Absent }) {
  return <Absent>{why === "not_evaluated" ? k.notEvaluated(l) : k.unknown(l)}</Absent>;
}

export function gapOf(candidate: Candidate): Absent {
  return candidate === null ? "not_evaluated" : "unknown";
}

export function Known({ l, value, wrap = false }: { l: Locale; value: string; wrap?: boolean }) {
  if (!value) return <Absent>{k.unknown(l)}</Absent>;
  return ltr(value, wrap);
}

export function Code({ l, value, table, say }: { l: Locale; value: string; table: Words; say: (l: Locale, s: string) => string }) {
  if (value && has(table, value)) return <>{say(l, value)}</>;
  return value ? <Absent>{k.unknown(l)} (<bdi dir="ltr" className="live-mono">{value}</bdi>)</Absent> : <Absent>{k.unknown(l)}</Absent>;
}

export function Unexpected({ l, code }: { l: Locale; code: Locator }) {
  return <span className="p05-unexpected" role="note"><Template text={k.unexpected(l)} slots={{ code: ltr(code) }} /></span>;
}

export function State({ l, value, table, say, locator }: { l: Locale; value: string; table: Words; say: (l: Locale, s: string) => string; locator: Locator }) {
  if (!value) return <Absent>{k.unknown(l)}</Absent>;
  return has(table, value) ? <>{say(l, value)}</> : <Unexpected l={l} code={locator} />;
}

export function Fixed({ l, value, locator, expected = "not_connected", say = k.connection }: { l: Locale; value: string; locator: Locator; expected?: string; say?: (l: Locale, s: string) => string }) {
  return value === expected ? <>{say(l, value)}</> : <Unexpected l={l} code={locator} />;
}

export function Num({ l, n }: { l: Locale; n: number | null }) {
  return n === null ? <Absent>{k.unknown(l)}</Absent> : <bdi dir="ltr" className="p05-num">{n}</bdi>;
}

export function personText(p: Person): string {
  return p ? p.login || p.id : "";
}

export function Who({ l, p, empty }: { l: Locale; p: Person; empty?: string }) {
  if (!p) return <Absent>{empty ?? k.unknown(l)}</Absent>;
  if (!p.id && !p.login) return <Absent>{k.unknown(l)}</Absent>;
  if (p.directory === false) return <Template text={k.notDirectory(l)} slots={{ id: <bdi dir="ltr" className="live-mono p05-login">{p.id || k.unknown(l)}</bdi> }} />;
  return <bdi dir="ltr" className="live-mono p05-login">{p.login || p.id}</bdi>;
}

export function Free({ l, value, empty }: { l: Locale; value: string; empty?: string }) {
  return value ? <span dir="auto" className="p05-free">{value}</span> : <Absent>{empty ?? k.unknown(l)}</Absent>;
}

export function Digest({ l, value }: { l: Locale; value: string }) {
  return digestPattern.test(value) ? <span dir="ltr" className="live-mono p05-digest-value">{value}</span> : <Absent>{k.malformed(l)}</Absent>;
}

export function Commit({ l, value, short = false }: { l: Locale; value: string | null; short?: boolean }) {
  if (value === null) return <Absent>{k.noImplementation(l)}</Absent>;
  if (!commitPattern.test(value)) return <Absent>{k.malformed(l)}</Absent>;
  return short ? <bdi dir="ltr" className="live-mono" title={value}>{value.slice(0, 12)}</bdi> : ltr(value, true);
}

export function ListOr<T>({ l, items, empty, children }: { l: Locale; items: T[] | null; empty?: string; children: (items: T[]) => ReactNode }) {
  if (items === null) return <Absent>{k.unknown(l)}</Absent>;
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

export function plainFill(text: string, slots: Record<string, string>): string {
  return fill(text, slots);
}

export function MoreLine({ l, more, total, forms }: { l: Locale; more: boolean | null; total: number | null; forms: Plural }) {
  if (more === false) return null;
  const text = more === true && total !== null ? counted(l, total, forms) : k.maybeMore(l);
  return <p className="p05-small p05-muted">{text}</p>;
}

export function href(l: Locale, view: string, extra: Record<string, string> = {}): string {
  const q = new URLSearchParams({ lang: l, view, ...extra });
  return "#/release-evidence?" + q.toString();
}

export function measure(el: HTMLElement | null): number {
  if (!el || !el.isConnected) return 0;
  return el.getBoundingClientRect().width;
}

export function useAvailableWidth(ref: RefObject<HTMLElement | null>, ready: unknown): number {
  const [width, setWidth] = useState(0);
  useLayoutEffect(() => {
    const el = ref.current;
    setWidth(measure(el));
    if (!el || typeof ResizeObserver !== "function") return;
    const o = new ResizeObserver((entries) => {
      for (const e of entries) setWidth(e.contentRect.width);
    });
    o.observe(el);
    return () => o.disconnect();
  }, [ref, ready]);
  return width;
}

export type Intent = { kind: "open" | "close"; opener: string; pointer?: boolean } | null;
let intent: Intent = null;

export function setIntent(next: Intent): void {
  intent = next;
}

export function takeIntent(): Intent {
  const was = intent;
  intent = null;
  return was;
}

export function EvidenceLink({ l, label, to, id, controls }: { l: Locale; label: string; to: string; id: string; controls?: string }) {
  return (
    <a className="p05-evidence" id={id} href={to} aria-label={k.evidenceFor(l, label)} aria-controls={controls} onClick={(e) => setIntent({ kind: "open", opener: id, pointer: e.detail > 0 })}>
      {k.evidence(l)}
    </a>
  );
}

export function DetailShell({ l, mode, title, sub, closeTo, opener, children }: { l: Locale; mode: "split" | "step"; title: ReactNode; sub?: ReactNode; closeTo: string; opener: string; children: ReactNode }) {
  const leave = (e?: { detail: number }) => setIntent({ kind: "close", opener, pointer: !!e && e.detail > 0 });
  const onKey = (e: { key: string; preventDefault: () => void }) => {
    if (e.key !== "Escape") return;
    e.preventDefault();
    leave();
    location.hash = closeTo.slice(1);
  };
  const head = (
    <div className="p05-detail-head">
      <div>
        <h2 id="p05-detail-title" className="p05-detail-title" tabIndex={-1}>{title}</h2>
        {sub && <p className="p05-detail-sub">{sub}</p>}
      </div>
      {mode === "split" && <a className="p05-close" href={closeTo} onClick={leave}>{k.close(l)}</a>}
    </div>
  );
  if (mode === "step") {
    return (
      <section className="p05-step" id="p05-detail" aria-labelledby="p05-detail-title" onKeyDown={onKey}>
        <a className="p05-back" href={closeTo} onClick={leave}>{k.back(l)}</a>
        {head}
        {children}
      </section>
    );
  }
  return (
    <aside className="p05-detail" id="p05-detail" tabIndex={0} aria-labelledby="p05-detail-title" onKeyDown={onKey}>
      {head}
      {children}
    </aside>
  );
}

export type ProfilesList = { more: boolean; status: "idle" | "loading" | "failed"; reset: boolean; onMore: () => void };

export type FootCopy = { shown: (l: Locale, n: number, m: number) => string; more: (l: Locale) => string; failed: (l: Locale) => string; reset: (l: Locale) => string };

export function ListFoot({ l, kind, list, shown, total, copy }: { l: Locale; kind: string; list: ProfilesList; shown: number; total: number | null; copy: FootCopy }) {
  const failed = list.status === "failed";
  const c = "p05-" + kind;
  return (
    <div className={"p05-list-foot " + c + "-foot"}>
      {total !== null && shown > 0 && shown < total && <p className={"p05-small p05-muted " + c + "-count"}>{copy.shown(l, shown, total)}</p>}
      {list.reset && <p className={"p05-small " + c + "-reset"} role="status">{copy.reset(l)}</p>}
      {(list.more || failed) && (
        <div className={"p05-req-more " + c + "-more"}>
          {failed && <p role="alert">{copy.failed(l)}</p>}
          <button type="button" className={"live-secondary p05-req-more-button " + c + "-more-button"} aria-disabled={list.status === "loading" || undefined} onClick={() => { if (list.status !== "loading") list.onMore(); }}>
            {failed ? k.reqTryAgain(l) : list.status === "loading" ? k.reqLoading(l) : copy.more(l)}
          </button>
        </div>
      )}
    </div>
  );
}
