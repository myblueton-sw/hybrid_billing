import { useEffect, useRef, useState } from "react";

export type MorePage<T> = { rows: T[]; next: string };
export type MoreRows<T> = { rows: T[] | null; more: boolean; status: "idle" | "loading" | "failed"; reset: boolean; load: () => void };
type State<T> = { key: string; rows: T[]; next: string | null; status: "idle" | "loading" | "failed" };

export function appended<T>(first: T[], more: T[], idOf: (row: T) => string): T[] {
  const seen = new Set(first.map(idOf));
  const out = [...first];
  for (const row of more) {
    const id = idOf(row);
    if (id === "" || seen.has(id)) continue;
    seen.add(id);
    out.push(row);
  }
  return out;
}

export function useMoreRows<T>(
  key: string,
  first: T[] | null,
  next: string | null,
  read: ((after: string) => Promise<MorePage<T>>) | undefined,
  idOf: (row: T) => string,
  onRefused: (err: unknown) => boolean,
): MoreRows<T> {
  const blank = (): State<T> => ({ key, rows: [], next: null, status: "idle" });
  const [state, setState] = useState<State<T>>(blank);
  const [reset, setReset] = useState(false);
  const token = useRef(0);
  const loaded = useRef({ key, rows: 0 });
  useEffect(() => {
    token.current += 1;
    if (loaded.current.key !== key && loaded.current.rows > 0) setReset(true);
    loaded.current = { key, rows: 0 };
    setState((was) => (was.key === key ? was : blank()));
  }, [key]); // eslint-disable-line react-hooks/exhaustive-deps
  const current = state.key === key ? state : blank();
  const after = first !== null && read ? current.next ?? next ?? "" : "";
  const load = () => {
    if (current.status === "loading" || !after || !read) return;
    const mine = ++token.current;
    setReset(false);
    setState({ ...current, status: "loading" });
    read(after)
      .then((got) => {
        if (mine !== token.current) return;
        loaded.current = { key, rows: current.rows.length + got.rows.length };
        setState({ key, rows: [...current.rows, ...got.rows], next: got.next, status: "idle" });
      })
      .catch((err: unknown) => {
        if (mine !== token.current) return;
        setState({ ...current, status: onRefused(err) ? "idle" : "failed" });
      });
  };
  return { rows: first === null ? null : appended(first, current.rows, idOf), more: after !== "", status: current.status, reset, load };
}
