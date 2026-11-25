import { useEffect, useRef, useState, type RefObject } from "react";

type Numbered = { number: number | null };
export type OlderFacts = { total: number; next: string };
export type OlderPage<T> = { rows: T[]; next: string; total: number };
type State<T> = { key: string; rows: T[]; next: string | null; total: number | null; status: "idle" | "loading" | "failed" };

export type OlderRows<T> = {
  rows: T[] | null;
  cut: boolean;
  all: boolean;
  total: number;
  counted: number;
  more: boolean;
  status: "idle" | "loading" | "failed";
  reset: boolean;
  load: () => void;
  countRef: RefObject<HTMLParagraphElement | null>;
};

export function newestFirst<T extends Numbered>(first: T[], older: T[]): T[] {
  const newest = [...first].reverse();
  const seen = new Set(newest.map((x) => x.number));
  return [...newest, ...older.filter((x) => x.number === null || !seen.has(x.number))];
}

export function useOlderRows<T extends Numbered>(
  key: string,
  facts: OlderFacts | null,
  first: T[] | null,
  read: ((after: string) => Promise<OlderPage<T>>) | undefined,
  onRefused: (err: unknown) => boolean,
  recorded = true,
): OlderRows<T> {
  const blank = (): State<T> => ({ key, rows: [], next: null, total: null, status: "idle" });
  const [state, setState] = useState<State<T>>(blank);
  const [reset, setReset] = useState(false);
  const [landed, setLanded] = useState(0);
  const token = useRef(0);
  const loaded = useRef({ key, rows: 0 });
  const countRef = useRef<HTMLParagraphElement | null>(null);
  useEffect(() => {
    if (landed > 0) countRef.current?.focus();
  }, [landed]);
  useEffect(() => {
    token.current += 1;
    if (loaded.current.key !== key && loaded.current.rows > 0) setReset(true);
    loaded.current = { key, rows: 0 };
    setState((was) => (was.key === key ? was : blank()));
  }, [key]); // eslint-disable-line react-hooks/exhaustive-deps
  const current = state.key === key ? state : blank();
  const next = facts && read ? current.next ?? facts.next : "";
  const rows = first === null ? null : newestFirst(recorded ? first : [...first].reverse(), current.rows);
  const total = Math.max(current.total ?? facts?.total ?? 0, rows?.length ?? 0);
  const load = () => {
    if (current.status === "loading" || !next || !read) return;
    const mine = ++token.current;
    setReset(false);
    setState({ ...current, status: "loading" });
    read(next)
      .then((got) => {
        if (mine !== token.current) return;
        loaded.current = { key, rows: current.rows.length + got.rows.length };
        setState({ key, rows: [...current.rows, ...got.rows], next: got.next, total: got.total, status: "idle" });
        if (!got.next) setLanded((n) => n + 1);
      })
      .catch((err: unknown) => {
        if (mine !== token.current) return;
        setState({ ...current, status: onRefused(err) ? "idle" : "failed" });
      });
  };
  const cut = rows !== null && facts !== null && total > (first?.length ?? 0);
  return { rows, cut, all: !next && (rows?.length ?? 0) >= total, total, counted: current.total ?? facts?.total ?? 0, more: next !== "", status: current.status, reset, load, countRef };
}
