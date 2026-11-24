import { useCallback, useEffect, useRef, useState } from "react";
import { appended, firstPage, type Paged } from "./live-model";

export type Fetched<T> = { rows: T[]; next: string; stopped: boolean };
export type Pager<T> = Paged<T> & { key: string | null; current: boolean; more: () => void; retry: () => void };

export function usePages<T>(key: string | null, fetch: (after: string) => Promise<Fetched<T>>, onError: (err: unknown) => boolean): Pager<T> {
  const [state, setState] = useState<Paged<T> & { key: string | null }>(() => ({ ...firstPage<T>(), key: null }));
  const token = useRef(0);
  const fetchRef = useRef(fetch);
  fetchRef.current = fetch;
  const errorRef = useRef(onError);
  errorRef.current = onError;
  const run = useCallback(async (owner: string, after: string, prev: Paged<T> | null) => {
    const mine = ++token.current;
    try {
      const page = await fetchRef.current(after);
      if (mine !== token.current) return;
      setState({ ...appended(prev ?? firstPage<T>(), page.rows, page.next, page.stopped), key: owner });
    } catch (err) {
      if (mine !== token.current) return;
      if (errorRef.current(err)) return;
      setState(prev ? { ...prev, status: "moreFailed", key: owner } : { ...firstPage<T>(), status: "error", error: err, key: owner });
    }
  }, []);
  useEffect(() => {
    token.current += 1;
    if (key === null) return;
    setState({ ...firstPage<T>(), key });
    void run(key, "", null);
  }, [key, run]);
  const current = key !== null && state.key === key;
  const more = () => {
    if (!current || state.status === "more" || !state.next) return;
    const prev = { ...state, status: "ready" as const };
    setState({ ...state, status: "more" });
    void run(state.key as string, state.next, prev);
  };
  const retry = () => {
    if (key === null) return;
    setState({ ...firstPage<T>(), key });
    void run(key, "", null);
  };
  const shown: Paged<T> = current ? state : firstPage<T>();
  return { ...shown, key, current, more, retry };
}
