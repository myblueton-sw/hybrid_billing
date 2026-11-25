import { useEffect, useRef, useState, type ReactNode } from "react";
import type { Locale } from "./types";
import type { KeptFacts, KeptPage } from "../service/kept";
import { kept as c } from "./kept-copy";

type Numbered = { number: number | null; state: string };

const closedIn = (rows: Numbered[]): number => rows.filter((p) => p.state !== "pending").length;
type State<T> = { revision: string; rows: T[]; next: string | null; total: number | null; status: "idle" | "loading" | "failed" };

export type Older<T> = { rows: T[]; head: ReactNode; count: ReactNode; foot: ReactNode };

export function merged<T extends Numbered>(first: T[], older: T[]): T[] {
  const seen = new Set(first.map((p) => p.number));
  return [...first, ...older.filter((p) => !seen.has(p.number))];
}

export function useOlder<T extends Numbered>(
  l: Locale,
  revision: string,
  facts: KeptFacts | null,
  first: T[],
  read: (after: string) => Promise<KeptPage<T>>,
  onRefused: (err: unknown) => boolean,
): Older<T> {
  const blank = (): State<T> => ({ revision, rows: [], next: null, total: null, status: "idle" });
  const [state, setState] = useState<State<T>>(blank);
  const [reset, setReset] = useState(false);
  const token = useRef(0);
  const [landed, setLanded] = useState(0);
  const endRef = useRef<HTMLParagraphElement>(null);
  const countRef = useRef<HTMLParagraphElement>(null);
  useEffect(() => {
    if (landed > 0) (endRef.current ?? countRef.current)?.focus();
  }, [landed]);
  const current = state.revision === revision ? state : blank();
  const loaded = useRef({ revision, rows: 0 });
  useEffect(() => {
    token.current += 1;
    if (loaded.current.revision !== revision && loaded.current.rows > 0) setReset(true);
    loaded.current = { revision, rows: 0 };
    setState((was) => (was.revision === revision ? was : blank()));
  }, [revision]); // eslint-disable-line react-hooks/exhaustive-deps
  if (!facts) return { rows: first, head: null, count: null, foot: null };
  const next = current.next ?? facts.next;
  const rows = merged(first, current.rows);
  const listed = closedIn(rows);
  const total = Math.max(current.total ?? facts.total, listed);
  const load = async () => {
    if (current.status === "loading" || !next) return;
    const mine = ++token.current;
    setReset(false);
    setState({ ...current, status: "loading" });
    try {
      const got = await read(next);
      if (mine !== token.current) return;
      loaded.current = { revision, rows: current.rows.length + got.rows.length };
      setState({ revision, rows: [...current.rows, ...got.rows], next: got.next, total: got.history.total, status: "idle" });
      if (!got.next) setLanded((n) => n + 1);
    } catch (err) {
      if (mine !== token.current) return;
      setState({ ...current, status: onRefused(err) ? "idle" : "failed" });
    }
  };
  const cut = total > closedIn(first);
  const failed = current.status === "failed";
  const count = cut ? <p className="live-meta kept-count" ref={countRef} tabIndex={-1}>{!next && listed >= total ? c.all(l, total) : c.shown(l, listed, total)}</p> : null;
  const head = reset ? <p className="live-note kept-reset" role="status">{c.reset(l)}</p> : null;
  const foot = (
    <>
      {next && (
        <div className="kept-more">
          {failed && <p role="alert">{c.failed(l)}</p>}
          <button type="button" className="live-secondary kept-more-button" aria-disabled={current.status === "loading" || undefined} onClick={load}>
            {failed ? c.tryAgain(l) : current.status === "loading" ? c.loading(l) : c.more(l)}
          </button>
        </div>
      )}
      {!next && facts.dropped > 0 && <p className="live-meta kept-end" ref={endRef} tabIndex={-1}>{c.end(l, facts.kept)}</p>}
    </>
  );
  return { rows, head, count, foot };
}
