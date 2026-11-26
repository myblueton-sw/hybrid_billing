import { useEffect, useMemo, useRef, useState, type RefObject } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError } from "../../../../service/admin";
import { w } from "../users/live-copy";
import { when } from "../authentication/live-model";
import { qcopy } from "../service-accounts/query-copy";
import { cursorTrail } from "../service-accounts/query-model";
import { readFailure } from "./Permissions";

type Listed = { next: string; reference_at: string };
type Read<P> = { state: "loading" } | { state: "error"; error: unknown } | { state: "ready"; page: P };
export type PagedList<P> = {
  read: Read<P>; reference: RefObject<HTMLParagraphElement | null>; first: boolean;
  reload: () => void; toFirst: () => void; toPrevious: () => void; toNext: (next: string) => void;
};

export function usePagedList<P extends Listed>(fetch: (after: string) => Promise<P>, key: string, generation: number): PagedList<P> {
  const [at, setAt] = useState({ generation, after: "" });
  const [local, setLocal] = useState(0);
  const after = at.generation === generation ? at.after : "";
  const token = useMemo(() => ({}), [key, generation, after, local]);
  const [result, setResult] = useState<{ token: object; read: Read<P> } | null>(null);
  const [trail] = useState(cursorTrail);
  useEffect(() => { if (at.generation !== generation) trail.clear(); }, [generation]); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    let active = true;
    fetch(after).then(
      (page) => { if (active) setResult({ token, read: { state: "ready", page } }); },
      (error: unknown) => { if (active) setResult({ token, read: { state: "error", error } }); },
    );
    return () => { active = false; };
  }, [token]); // eslint-disable-line react-hooks/exhaustive-deps
  const reference = useRef<HTMLParagraphElement>(null);
  const moved = useRef(false);
  const read: Read<P> = result?.token === token ? result.read : { state: "loading" };
  useEffect(() => {
    if (read.state === "ready" && moved.current) { moved.current = false; reference.current?.focus(); }
  }, [read]);
  const go = (next: string) => { moved.current = true; setAt({ generation, after: next }); };
  return {
    read, reference, first: after === "",
    reload: () => setLocal((n) => n + 1),
    toFirst: () => { trail.clear(); go(""); },
    toPrevious: () => go(trail.previous(key) || ""),
    toNext: (next: string) => { if (trail.next(key, after, next)) go(next); },
  };
}

export function ListFailure<P>({ l, title, list, onSignOut }: { l: Locale; title: string; list: PagedList<P>; onSignOut: () => void }) {
  if (list.read.state !== "error") return null;
  const error = list.read.error;
  return (
    <section className="live-error access-list-failure" role="alert">
      <h3>{title}</h3>
      <p>{readFailure(l, error)}</p>
      <div className="live-buttons">
        {error instanceof AdminError && error.failure === "unauthenticated"
          ? <button type="button" className="live-secondary" onClick={onSignOut}>{w.signInAgain(l)}</button>
          : <button type="button" className="live-secondary" onClick={list.reload}>{w.reload(l)}</button>}
        {!list.first && <button type="button" className="live-secondary" onClick={list.toFirst}>{qcopy(l, "first")}</button>}
      </div>
    </section>
  );
}

export function Paging<P extends Listed>({ l, label, list }: { l: Locale; label: string; list: PagedList<P> }) {
  if (list.read.state !== "ready" || (list.first && !list.read.page.next)) return null;
  const next = list.read.page.next;
  return (
    <nav className="live-buttons access-paging" aria-label={label}>
      <button type="button" className="live-secondary" disabled={list.first} onClick={list.toFirst}>{qcopy(l, "first")}</button>
      <button type="button" className="live-secondary" disabled={list.first} onClick={list.toPrevious}>{qcopy(l, "previous")}</button>
      <button type="button" className="live-secondary" disabled={!next} onClick={() => list.toNext(next)}>{qcopy(l, "next")}</button>
    </nav>
  );
}

export function Reference<P extends Listed>({ l, page, list }: { l: Locale; page: P; list: PagedList<P> }) {
  return <p className="live-meta access-reference" tabIndex={-1} ref={list.reference}>{qcopy(l, "reference")}: <time dateTime={page.reference_at} title={page.reference_at}>{when(l, page.reference_at)}</time></p>;
}
