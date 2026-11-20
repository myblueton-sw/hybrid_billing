import { useEffect, useId, useMemo, useRef, useState, useSyncExternalStore } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, type AdminClient } from "../../../../service/admin";
import type { Selection, SelectionPreview, SessionRow } from "../../../../service/session-query";
import { c } from "../../../../service/copy";
import { m } from "./live-copy";
import { e, whenSeconds } from "./end-copy";
import { sq } from "./query-copy";
import { intentLocked, submitSelected, recoverSelected, replaySelected, type IntentStore } from "./session-intent";
import { useScopedRead, type ScopedValue } from "./scoped-read";
import "./end.css";

export const sessionStateLabel = (l: Locale,state: SessionRow["state"]) => ({ active:sq.active,blocked:sq.blocked,expired:sq.ended,revoked:sq.revoked,unavailable:sq.unavailable })[state](l);

export function SelectionRows({ l, rows }: { l: Locale; rows: SessionRow[] }) {
  return <div className="security-table-scroll live-table"><table><thead><tr><th scope="col">{m.colSession(l)}</th><th scope="col">{m.colSignedIn(l)}</th><th scope="col">{m.colEnds(l)}</th><th scope="col">{sq.state(l)}</th></tr></thead>
    <tbody>{rows.map(row => <tr key={row.handle}><td className="live-mono"><bdi dir="ltr">{row.handle}</bdi></td><td data-label={m.colSignedIn(l)}><bdi dir="ltr">{whenSeconds(l,row.signed_in)}</bdi></td><td data-label={m.colEnds(l)}><bdi dir="ltr">{whenSeconds(l,row.ends)}</bdi></td><td data-label={sq.state(l)}>{sessionStateLabel(l,row.state)}</td></tr>)}</tbody></table></div>;
}

export function IntentRecovery({ l, api, store }: { l: Locale; api: AdminClient; store: IntentStore }) {
  const titleId = useId();
  const snapshot = useSyncExternalStore(store.subscribe,store.get,store.get);
  const intent = snapshot.intent;
  if (!intentLocked(snapshot) || !intent) return null;
  const busy = intent.state === "submitting" || intent.state === "recovering";
  return <section className="service-summary mysec-intent" aria-labelledby={titleId} aria-busy={busy}>
    <h3 id={titleId}>{e.titleUnknown(l)}</h3><p role="status">{intent.state === "unavailable" ? sq.resultUnavailable(l) : sq.unknown(l)}</p>
    <p className="live-meta">{sq.retention(l)}</p>
    <div className="live-buttons"><button type="button" className="live-secondary" disabled={busy} onClick={() => void recoverSelected(store,api)}>{sq.check(l)}</button>
      <button type="button" className="live-secondary" disabled={busy} onClick={() => void replaySelected(store,api)}>{sq.replay(l)}</button></div>
  </section>;
}

export default function SessionSelection({ l, api, handle, store, onClose, onSignedOut }: {
  l: Locale; api: AdminClient; handle?: string; store: IntentStore; onClose: () => void; onSignedOut: () => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const heading = useRef<HTMLHeadingElement>(null);
  const serial = useRef(0);
  const busyRef = useRef(false);
  const readKey = useMemo(() => ({ api,handle,store }),[api,handle,store]);
  const guard = useScopedRead(readKey);
  const [preparedData,setPreparedData] = useState<ScopedValue<Selection> | null>(null);
  const [previewData,setPreviewData] = useState<ScopedValue<SelectionPreview> | null>(null);
  const prepared = preparedData?.scope === guard.token ? preparedData.value : null;
  const preview = previewData?.scope === guard.token ? previewData.value : null;
  const setPrepared = (value: Selection | null) => setPreparedData(value ? { scope:guard.token,value } : null);
  const setPreview = (value: SelectionPreview | null) => setPreviewData(value ? { scope:guard.token,value } : null);
  const [after,setAfter] = useState("");
  const [trail,setTrail] = useState<string[]>([]);
  const [busy,setBusy] = useState(false);
  const [failure,setFailure] = useState<unknown>(null);
  const failed = failure !== null;
  const [confirmed,setConfirmed] = useState(false);
  const snapshot = useSyncExternalStore(store.subscribe,store.get,store.get);
  const kind = handle ? "one" : "others";
  const error = (cause: unknown) => {
    if (cause instanceof AdminError && cause.failure === "unauthenticated") onSignedOut();
    else {
      if (!intentLocked(store.get()) && cause instanceof AdminError && ["revision_conflict","unsupported_command"].includes(cause.code)) {
        setPrepared(null);setPreview(null);setAfter("");setTrail([]);setConfirmed(false);
      }
      setFailure(cause);
    }
  };
  const readPreview = async (selection: Selection, cursor = "", history: string[] = []) => {
    const n = ++serial.current;
    const valid = guard.begin();
    busyRef.current = true; setBusy(true); setFailure(null); setConfirmed(false);
    try {
      const got = await api.previewSessionSelection(selection.selection,{ ...(cursor ? { after:cursor } : {}) });
      if (n !== serial.current || !valid()) return;
      if (got.selection !== selection.selection || got.digest !== selection.digest || got.count !== selection.count || got.kind !== kind) throw new AdminError("invalid");
      setPreview(got); setAfter(cursor); setTrail(history);
    } catch (cause) { if (n === serial.current && valid()) error(cause); }
    finally { if (n === serial.current && valid()) { busyRef.current = false; setBusy(false); } }
  };
  const prepare = async (resume?: Selection) => {
    if (busyRef.current || intentLocked(store.get())) return;
    const n = ++serial.current;
    const valid = guard.begin();
    if (!store.get().intent && store.get().error !== null) store.update({ ...store.get(),error:null,version:store.get().version+1 });
    busyRef.current = true; setBusy(true); setFailure(null); setConfirmed(false);
    let next: Selection | null = null;
    try {
      if (!resume && handle) {
        const detail = await api.sessionDetail(handle);
        if (n !== serial.current || !valid()) return;
        if (!detail.session.can_end || detail.session.current || detail.session.handle !== handle) throw new AdminError("invalid");
      }
      const got = await api.prepareSessionSelection(resume ? { selection:resume.selection,after:resume.after } : handle ? { session:handle } : undefined);
      if (n !== serial.current || !valid()) return;
      if (got.kind !== kind) throw new AdminError("invalid");
      setPrepared(got); setPreview(null); setAfter(""); setTrail([]);
      if (got.state === "sealed") next = got;
    } catch (cause) { if (n === serial.current && valid()) error(cause); }
    finally { if (n === serial.current && valid()) { busyRef.current = false; setBusy(false); } }
    if (next && n === serial.current && valid()) await readPreview(next);
  };
  useEffect(() => {
    busyRef.current = false;
    if (dialog.current && !dialog.current.open) dialog.current.showModal(); heading.current?.focus(); void prepare();
    return () => { serial.current++; busyRef.current = false; };
  },[guard]);
  useEffect(() => {
    if (prepared && snapshot.intent?.fingerprint.selection === prepared.selection && snapshot.intent.state === "accepted") onClose();
  },[prepared,snapshot.intent]);
  const ready = !!prepared && prepared.state === "sealed" && prepared.count !== null && prepared.count > 0 && !!preview && !busy && !failed && !intentLocked(snapshot);
  const submit = () => {
    if (!ready || !confirmed || !prepared || prepared.count === null) return;
    void submitSelected(store,api,{ selection:prepared.selection,digest:prepared.digest,count:prepared.count },prepared.kind);
  };
  return <dialog ref={dialog} className="mysec-end" aria-labelledby="mysec-selection-title" onCancel={event => { event.preventDefault(); onClose(); }}>
    <h2 id="mysec-selection-title" tabIndex={-1} ref={heading}>{handle ? m.endOneTitle(l) : e.othersTitle(l)}</h2>
    <p className="mysec-end-scope">{e.scope(l)}</p><p className="mysec-end-note">{handle ? m.endOneImpact(l) : sq.fullScope(l)}</p>
    {busy && <p role="status">{e.reading(l)}</p>}
    {failed && <div className="service-summary" role="alert"><p>{failure instanceof AdminError && failure.code === "revision_conflict" ? sq.expired(l) : failure instanceof AdminError && failure.failure === "limited" ? sq.capacity(l) : sq.readFailed(l)}</p><button type="button" className="live-secondary" disabled={busy} onClick={() => prepared?.state === "sealed" ? void readPreview(prepared,after,trail) : void prepare(prepared ?? undefined)}>{!prepared && failure instanceof AdminError && ["revision_conflict","unsupported_command"].includes(failure.code) ? sq.fresh(l) : c.tryAgain(l)}</button></div>}
    {prepared?.state === "building" && <div role="status"><p>{sq.preparing(l,prepared.observed)}</p><button type="button" className="live-secondary" disabled={busy} onClick={() => void prepare(prepared)}>{sq.continue(l)}</button></div>}
    {prepared && prepared.count !== null && <><h3>{sq.recorded(l,prepared.count)}</h3><p className="live-meta">{e.listTime(l,whenSeconds(l,prepared.reference_at))}</p><p>{sq.frozen(l)}</p>
      {preview && <><SelectionRows l={l} rows={preview.rows} /><p className="live-meta">{sq.shown(l,preview.rows.length)}</p>
        <div className="live-buttons"><button type="button" className="live-secondary" disabled={busy || trail.length === 0} onClick={() => void readPreview(prepared,trail[trail.length-1],trail.slice(0,-1))}>{sq.previous(l)}</button>
          <button type="button" className="live-secondary" disabled={busy || !preview.next} onClick={() => void readPreview(prepared,preview.next,[...trail,after].slice(-32))}>{sq.next(l)}</button></div></>}
      {kind === "others" && <p className="mysec-end-note">{e.note(l)}</p>}
      <label className="mysec-confirm"><input type="checkbox" checked={confirmed} disabled={!ready} onChange={event => setConfirmed(event.target.checked)} />{handle ? m.endOneButton(l) : e.button(l,prepared.count)}</label>
      <button type="button" className="live-primary" disabled={!ready || !confirmed} onClick={submit}>{handle ? m.endOneButton(l) : e.button(l,prepared.count)}</button></>}
    <IntentRecovery l={l} api={api} store={store} />
    {!snapshot.intent && snapshot.error !== null && <div role="alert"><p>{snapshot.error instanceof AdminError && snapshot.error.code === "revision_conflict" ? sq.expired(l) : snapshot.error instanceof AdminError && snapshot.error.failure === "limited" ? sq.capacity(l) : sq.refused(l)}</p><button type="button" className="live-secondary" disabled={busy} onClick={() => void prepare()}>{sq.fresh(l)}</button></div>}
    <div className="live-buttons"><button type="button" className="live-secondary" onClick={onClose}>{e.close(l)}</button></div>
  </dialog>;
}
