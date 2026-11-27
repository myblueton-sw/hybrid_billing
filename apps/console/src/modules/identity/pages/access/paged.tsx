import { ListPager, ListSummary } from "../../../../shared/list/ListParts";
import { useEffect, useMemo, useRef, useState, type RefObject } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError } from "../../../../service/admin";
import { w } from "../users/live-copy";
import { when } from "../authentication/live-model";
import { qcopy } from "../service-accounts/query-copy";
import { cursorTrail } from "../service-accounts/query-model";
import { readFailure } from "./Permissions";
import { sizeRefused } from "../../../../service/query-settings";

type Listed = { next: string; reference_at: string; rows: unknown[]; count: { value: string; certainty: string }; query: { limit: number } };
type Read<P> = { state: "loading" } | { state: "error"; error: unknown } | { state: "ready"; page: P };
export type PagedList<P> = {
  read: Read<P>; reference: RefObject<HTMLDivElement | null>; first: boolean;
  reload: () => void; toFirst: () => void; toPrevious: () => void; toNext: (next: string) => void;
};

export function usePagedList<P extends Listed>(fetch: (after: string) => Promise<P>, key: string, generation: number): PagedList<P> {
  const [at, setAt] = useState({ generation, key, after: "" });
  const [local, setLocal] = useState(0);
  const after = at.generation === generation && at.key === key ? at.after : "";
  const token = useMemo(() => ({}), [key, generation, after, local]);
  const [result, setResult] = useState<{ token: object; read: Read<P> } | null>(null);
  const [trail] = useState(cursorTrail);
  useEffect(() => { if (at.generation !== generation || at.key !== key) trail.clear(); }, [generation, key]); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    let active = true;
    fetch(after).then(
      (page) => { if (active) setResult({ token, read: { state: "ready", page } }); },
      (error: unknown) => { if (active) setResult({ token, read: { state: "error", error } }); },
    );
    return () => { active = false; };
  }, [token]); // eslint-disable-line react-hooks/exhaustive-deps
  const reference = useRef<HTMLDivElement>(null);
  const moved = useRef(false);
  const read: Read<P> = result?.token === token ? result.read : { state: "loading" };
  useEffect(() => {
    if (read.state === "ready" && moved.current) { moved.current = false; reference.current?.focus(); }
  }, [read]);
  const go = (next: string) => { moved.current = true; setAt({ generation, key, after: next }); };
  return {
    read, reference, first: after === "",
    reload: () => setLocal((n) => n + 1),
    toFirst: () => { trail.clear(); go(""); },
    toPrevious: () => go(trail.previous(key) || ""),
    toNext: (next: string) => { if (trail.next(key, after, next)) go(next); },
  };
}

export function ListFailure<P>({ l, title, list, onSignOut, onRefresh }: { l: Locale; title: string; list: PagedList<P>; onSignOut: () => void; onRefresh: () => void }) {
  if (list.read.state !== "error") return null;
  const error = list.read.error;
  return (
    <section className="live-error access-list-failure" role="alert">
      <h3>{title}</h3>
      <p>{readFailure(l, error)}</p>
      <div className="live-buttons">
        {error instanceof AdminError && error.failure === "unauthenticated"
          ? <button type="button" className="live-secondary" onClick={onSignOut}>{w.signInAgain(l)}</button>
          : <button type="button" className="live-secondary" onClick={sizeRefused(error) ? onRefresh : list.reload}>{w.reload(l)}</button>}
        {!list.first && !sizeRefused(error) && <button type="button" className="live-secondary" onClick={list.toFirst}>{qcopy(l, "first")}</button>}
      </div>
    </section>
  );
}

export type PageSize = { requested: number; limit: number | null; onChange: (n: number) => void; id: string };

export function Paging<P extends Listed>({ l, label, list, size }: { l: Locale; label: string; list: PagedList<P>; size?: PageSize }) {
  const sizeFocus = useRef(false);
  const state = list.read.state;
  useEffect(() => {
    if (state === "loading" || !sizeFocus.current) return;
    sizeFocus.current = false;
    if (state === "ready" && size) document.getElementById(size.id)?.focus();
  }, [state, size]);
  if (list.read.state !== "ready") return null;
  const page = list.read.page;
  return (
    <ListPager l={l} title={label} first={list.first} hasNext={!!page.next} rows={page.rows.length}
      size={size && { ...size, current: page.query.limit, onChange: (n) => { sizeFocus.current = true; size.onChange(n); } }} onFirst={list.toFirst} onPrevious={list.toPrevious} onNext={() => list.toNext(page.next)} />
  );
}

export function Reference<P extends Listed>({ l, page, list }: { l: Locale; page: P; list: PagedList<P> }) {
  return <ListSummary l={l} count={page.count.value} exact={page.count.certainty === "exact"} reference={<time dateTime={page.reference_at} title={page.reference_at}>{when(l, page.reference_at)}</time>} summaryRef={list.reference} countInHeading />;
}
