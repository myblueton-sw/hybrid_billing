import { useEffect, useMemo, useState } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, type AdminClient, type TrustChange } from "../../../../service/admin";
import { c } from "../../../../service/copy";
import { w } from "../users/live-copy";
import { readFailure } from "../access/Permissions";
import { ty } from "./trust-copy";
import { tl } from "./trust-list-copy";

export type DetailRead = { state: "loading" } | { state: "ready"; change: TrustChange } | { state: "gone" } | { state: "error"; error: unknown };
export type Opened = { id: string; number: number | null };

export function goneFrom(error: unknown): boolean {
  return error instanceof AdminError && error.code === "invalid_request";
}

export function shownRead(id: string, token: object, result: { id: string; token: object; read: DetailRead } | null): DetailRead {
  if (!id || !result || result.id !== id) return { state: "loading" };
  if (result.token === token || result.read.state === "ready") return result.read;
  return { state: "loading" };
}

export function useChangeDetail(api: AdminClient, id: string, generation: number): { read: DetailRead; stale: boolean; retry: () => void } {
  const [local, setLocal] = useState(0);
  const token = useMemo(() => ({}), [id, generation, local]);
  const [result, setResult] = useState<{ id: string; token: object; read: DetailRead } | null>(null);
  useEffect(() => {
    if (!id) return;
    let active = true;
    api.trustChange(id).then(
      (got) => { if (active) setResult({ id, token, read: { state: "ready", change: got.change } }); },
      (error: unknown) => { if (active) setResult({ id, token, read: goneFrom(error) ? { state: "gone" } : { state: "error", error } }); },
    );
    return () => { active = false; };
  }, [token]);
  return { read: shownRead(id, token, result), stale: !!id && !!result && result.id === id && result.token !== token, retry: () => setLocal((n) => n + 1) };
}

export function DetailPending({ l, opened, read, split, onClose, onRetry, onSignOut }: {
  l: Locale; opened: Opened; read: Exclude<DetailRead, { state: "ready" }>; split: boolean; onClose: () => void; onRetry: () => void; onSignOut: () => void;
}) {
  const signedOut = read.state === "error" && read.error instanceof AdminError && read.error.failure === "unauthenticated";
  return (
    <div className="m02-detail-body" data-detail={read.state}>
      <div className="m02-detail-head">
        <h2 className="m02-detail-title" tabIndex={-1}>
          {opened.number === null ? <bdi dir="ltr" className="live-mono">{opened.id}</bdi> : ty.change(l, opened.number)}
        </h2>
        {split && read.state === "loading" && <button type="button" className="service-button-link m02-close" onClick={onClose}>{ty.close(l)}</button>}
      </div>
      {read.state === "loading" && <p role="status" className="live-meta">{tl.detailLoading(l)}</p>}
      {read.state === "gone" && (
        <div className="m02-detail-gone" role="status">
          <p>{tl.noLongerKept(l)}</p>
          <div className="live-buttons"><button type="button" className="live-secondary" onClick={onClose}>{ty.close(l)}</button></div>
        </div>
      )}
      {read.state === "error" && (
        <section className="live-error" role="alert">
          <h3>{tl.detailFailed(l)}</h3>
          <p>{readFailure(l, read.error)}</p>
          <div className="live-buttons">
            {signedOut
              ? <button type="button" className="live-secondary" onClick={onSignOut}>{w.signInAgain(l)}</button>
              : <button type="button" className="live-secondary" onClick={onRetry}>{c.tryAgain(l)}</button>}
            <button type="button" className="live-secondary" onClick={onClose}>{ty.close(l)}</button>
          </div>
        </section>
      )}
    </div>
  );
}
