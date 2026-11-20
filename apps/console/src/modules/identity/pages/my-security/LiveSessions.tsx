import { useEffect, useMemo, useRef, useState, useSyncExternalStore, type FormEvent } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, type AdminClient, type SelfPage } from "../../../../service/admin";
import type { SessionPage, SessionRow } from "../../../../service/session-query";
import { c } from "../../../../service/copy";
import { w } from "../users/live-copy";
import { useScopedRead, type ScopedValue } from "./scoped-read";
import { useWide } from "../users/Live";
import { m } from "./live-copy";
import { e, whenSeconds } from "./end-copy";
import { sq } from "./query-copy";
import { parseSessionLink, sessionLink, sessionRequest, type SessionFilters } from "./session-query-model";
import { intentStore, intentLocked } from "./session-intent";
import SessionSelection, { IntentRecovery, sessionStateLabel } from "./SessionSelection";
import SessionOperations, { summarySentence } from "./SessionOperations";
import AccessHistory from "./AccessHistory";
import { SessionSettings } from "./SessionSettings";
import "./session-query.css";

export function SessionTimeHint({ l }: { l: Locale }) {
  return <p id="mysec-time-hint" className="live-meta">{sq.timeHint(l)} {sq.example(l)} <bdi dir="ltr">2026-11-20T09:00:00+09:00</bdi></p>;
}

function SessionRows({ l,rows,locked,onEnd }: { l: Locale; rows: SessionRow[]; locked: boolean; onEnd: (handle: string) => void }) {
  return <div className="security-table-scroll live-table"><table><thead><tr><th scope="col">{m.colSession(l)}</th><th scope="col">{m.colSignedIn(l)}</th><th scope="col">{m.colEnds(l)}</th><th scope="col">{sq.state(l)}</th><th scope="col">{m.colActions(l)}</th></tr></thead>
    <tbody>{rows.map(row => <tr key={row.handle}><td><strong>{row.current ? m.thisSession(l) : m.otherSession(l)}</strong><span className="live-mono live-sub"><bdi dir="ltr">{row.handle}</bdi></span></td>
      <td data-label={m.colSignedIn(l)}><bdi dir="ltr">{whenSeconds(l,row.signed_in)}</bdi></td><td data-label={m.colEnds(l)}><bdi dir="ltr">{whenSeconds(l,row.ends)}</bdi></td><td data-label={sq.state(l)}>{sessionStateLabel(l,row.state)}</td>
      <td data-label={m.colActions(l)}>{row.current ? m.thisSession(l) : <button type="button" className="live-secondary" disabled={locked || !row.can_end} onClick={() => onEnd(row.handle)}>{m.end(l)}</button>}</td></tr>)}</tbody></table></div>;
}

export default function LiveSessions({ l,api,self,params,onParams,onRefresh,onSignOut,onSignedOut }: {
  l: Locale; api: AdminClient; self: SelfPage; params: URLSearchParams; onParams: (extra: Record<string,string | null>) => void; onRefresh: () => void; onSignOut: () => void; onSignedOut: () => void;
}) {
  const scope = JSON.stringify([self.user_id,self.workspace,self.current_session.handle]);
  const store = useMemo(() => intentStore(api,scope),[api,scope]);
  if (!store) return <section className="live-error" role="alert"><p>{sq.scopeChanged(l)}</p><button type="button" className="live-secondary" onClick={onSignOut}>{m.signOut(l)}</button></section>;
  return <ScopedSessions l={l} api={api} self={self} params={params} onParams={onParams} onRefresh={onRefresh} onSignOut={onSignOut} onSignedOut={onSignedOut} store={store} />;
}

function ScopedSessions({ l,api,self,params,onParams,onRefresh,onSignOut,onSignedOut,store }: Parameters<typeof LiveSessions>[0] & { store: NonNullable<ReturnType<typeof intentStore>> }) {
  const wide = useWide();
  const snapshot = useSyncExternalStore(store.subscribe,store.get,store.get);
  const locked = intentLocked(snapshot);
  const serialized = params.toString();
  const navigation = useMemo(() => parseSessionLink(new URLSearchParams(serialized)),[serialized]);
  const filterKey = JSON.stringify(navigation.filters);
  const [draft,setDraft] = useState<SessionFilters>(navigation.filters);
  const [refresh,setRefresh] = useState(0);
  const readKey = useMemo(() => ({ api,scope:store.scope,serialized,refresh }),[api,store.scope,serialized,refresh]);
  const guard = useScopedRead(readKey);
  const [pageData,setPageData] = useState<ScopedValue<SessionPage> | null>(null);
  const page = pageData?.scope === guard.token ? pageData.value : null;
  const setPage = (value: SessionPage | null) => setPageData(value ? { scope:guard.token,value } : null);
  const [busy,setBusy] = useState(false);
  const [failure,setFailure] = useState<unknown>(null);
  const [formError,setFormError] = useState(false);
  const [target,setTarget] = useState<{ handle?: string } | null>(null);
  const [notice,setNotice] = useState("");
  const serial = useRef(0);
  const parents = useRef(new Map<string,string>());
  const invoker = useRef<HTMLElement | null>(null);
  const title = useRef<HTMLHeadingElement>(null);
  const previousFilter = useRef(filterKey);
  const lastAccepted = useRef(0);
  useEffect(() => { setDraft(navigation.filters); setFormError(false); if (previousFilter.current !== filterKey) { parents.current.clear(); previousFilter.current = filterKey; } },[filterKey]);
  useEffect(() => {
    const n = ++serial.current;
    const valid = guard.begin();
    setPage(null);setFailure(null);
    if (navigation.bad) { setBusy(false);setFormError(true);return; }
    setBusy(true);
    api.sessionQuery(sessionRequest(navigation)).then(got => {
      if (n !== serial.current || !valid()) return;
      if (got.query.workspace !== self.workspace || got.rows.some(row => row.current && row.handle !== self.current_session.handle)) throw new AdminError("invalid");
      setPage(got);setBusy(false);
    }).catch(error => { if (n !== serial.current || !valid()) return;setFailure(error);setBusy(false);if (error instanceof AdminError && error.failure === "unauthenticated") onSignedOut(); });
    return () => { serial.current++; };
  },[guard,self.workspace,self.current_session.handle]);
  useEffect(() => {
    const intent = snapshot.intent;
    if (intent?.state === "accepted" && lastAccepted.current !== intent.id && intent.operation) {
      lastAccepted.current = intent.id;setNotice((intent.operation.replay ? e.replay(l)+" " : "")+summarySentence(l,intent.operation));setRefresh(value => value+1);onRefresh();
    }
  },[snapshot.intent,l]);
  const navigate = (filters: SessionFilters,after = "") => { onParams(sessionLink(filters,after));setFormError(false); };
  const apply = (event: FormEvent) => {
    event.preventDefault();
    const candidate = new URLSearchParams();
    for (const [key,value] of Object.entries(sessionLink(draft))) if (value !== null) candidate.set(key,value);
    if (parseSessionLink(candidate).bad) { setFormError(true);return; }
    navigate(draft);
    if (JSON.stringify(draft) === filterKey && !navigation.after) setRefresh(value => value+1);
  };
  const open = (handle?: string) => { if (locked) return;invoker.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;setTarget(handle ? { handle } : {}); };
  const close = () => { setTarget(null);requestAnimationFrame(() => { const element = invoker.current;if (element?.isConnected) element.focus();else title.current?.focus(); }); };
  const activeFilters: SessionFilters = page ? { session:page.query.session,q:page.query.q,since:page.query.since,until:page.query.until,limit:navigation.filters.limit } : navigation.filters;
  const fields = (Object.entries(activeFilters) as [keyof SessionFilters,string | number | undefined][]).filter(([key,value]) => value !== "" && value !== undefined && !(key === "session" && value === "all"));
  const filterLabel = { session:m.colSession,q:sq.search,since:sq.since,until:sq.until,limit:sq.size };
  const current = self.current_session;
  const reset = () => navigate({ session:"all",q:"",since:"",until:"" });
  return <div className="live-users mysec-query-layout" data-live="sessions" data-wide={wide ? "true" : "false"}>
    <div className="live-list live-sections">
      <section aria-labelledby="mysec-sessions-title">
        <div className="live-toolbar"><div><h2 id="mysec-sessions-title" ref={title} tabIndex={-1}>{m.sessionsTitle(l)}</h2><p className="live-meta">{m.workspace(l)}: <bdi dir="ltr">{self.workspace}</bdi></p></div>
          <div className="live-tools"><button type="button" className="live-secondary mysec-end-others" disabled={locked} onClick={() => open()}>{e.entry(l)}</button><button type="button" className="live-secondary" onClick={() => { setRefresh(value => value+1);onRefresh(); }}>{w.reload(l)}</button></div></div>
        <p>{sq.fullScope(l)}</p><IntentRecovery l={l} api={api} store={store} />
        {notice && <p role="status">{notice}</p>}
        <div className="mysec-current"><h3>{m.thisSession(l)}</h3><bdi dir="ltr" className="live-mono">{current.handle}</bdi><p><bdi dir="ltr">{whenSeconds(l,current.signed_in)}</bdi>{" · "+sessionStateLabel(l,current.state)}</p><button type="button" className="live-secondary" onClick={onSignOut}>{m.signOut(l)}</button></div>
        <form className="mysec-query-form" onSubmit={apply} noValidate aria-labelledby="mysec-filters-title">
          <h3 id="mysec-filters-title">{sq.filters(l)}</h3>
          {(formError || navigation.bad) && <p id="mysec-filter-error" role="alert">{sq.invalid(l)}</p>}
          <div className="mysec-query-fields">
            <label>{m.colSession(l)}<select value={draft.session} onChange={event => setDraft(value => ({ ...value,session:event.target.value as SessionFilters["session"] }))}><option value="all">{sq.all(l)}</option><option value="current">{sq.current(l)}</option><option value="other">{sq.other(l)}</option></select></label>
            <label>{sq.search(l)}<input type="search" value={draft.q} onChange={event => setDraft(value => ({ ...value,q:event.target.value }))} /></label>
            <label>{sq.since(l)}<input type="text" dir="ltr" value={draft.since} aria-describedby={formError || navigation.bad ? "mysec-time-hint mysec-filter-error" : "mysec-time-hint"} aria-invalid={formError || navigation.bad || undefined} onChange={event => setDraft(value => ({ ...value,since:event.target.value }))} /></label>
            <label>{sq.until(l)}<input type="text" dir="ltr" value={draft.until} aria-describedby={formError || navigation.bad ? "mysec-time-hint mysec-filter-error" : "mysec-time-hint"} aria-invalid={formError || navigation.bad || undefined} onChange={event => setDraft(value => ({ ...value,until:event.target.value }))} /></label>
            <label>{sq.size(l)}<input type="number" min={1} max={100} step={1} placeholder={page ? `${sq.pageDefault(l)} (${page.query.limit})` : sq.pageDefault(l)} value={draft.limit ?? ""} onChange={event => setDraft(value => ({ ...value,limit:event.target.value === "" ? undefined : event.target.valueAsNumber }))} /></label>
          </div><SessionTimeHint l={l} />
          <div className="live-buttons"><button type="submit" className="live-primary" disabled={busy}>{sq.apply(l)}</button><button type="button" className="live-secondary" onClick={reset}>{sq.clear(l)}</button></div>
        </form>
        {fields.length > 0 && <ul className="mysec-applied">{fields.map(([key,value]) => <li key={key}><button type="button" className="live-secondary" onClick={() => navigate({ ...activeFilters,[key]:key === "session" ? "all" : key === "limit" ? undefined : "" })}>{filterLabel[key](l)}: {key === "session" ? ({ all:sq.all,current:sq.current,other:sq.other })[value as SessionFilters["session"]](l) : <bdi dir="ltr">{String(value)}</bdi>} · {sq.remove(l)}</button></li>)}</ul>}
        {busy && <p role="status">{m.loading(l)}</p>}
        {failure !== null && <section className="live-error" role="alert"><h3>{m.loadFailed(l)}</h3><p>{failure instanceof AdminError && failure.code === "revision_conflict" ? sq.expired(l) : failure instanceof AdminError && failure.code === "invalid_request" ? sq.invalid(l) : m.loadRetry(l)}</p><button type="button" className="live-secondary" onClick={() => { navigate(navigation.filters);setRefresh(value => value+1); }}>{sq.first(l)}</button></section>}
        {page && <><div className="mysec-page-meta"><p>{sq.count(l,page.count.value,page.count.certainty === "exact")}</p><p className="live-meta">{sq.shown(l,page.rows.length)} · {e.listTime(l,whenSeconds(l,page.reference_at))}</p></div>
          {page.scan_stopped && <p role="status">{sq.stopped(l)}</p>}
          <SessionSettings l={l} settings={page.settings} />
          <SessionRows l={l} rows={page.rows} locked={locked} onEnd={open} />{page.rows.length === 0 && <p>{sq.empty(l)}</p>}
          <div className="live-buttons"><button type="button" className="live-secondary" disabled={!navigation.after || !parents.current.has(navigation.after)} onClick={() => navigate(navigation.filters,parents.current.get(navigation.after) ?? "")}>{sq.previous(l)}</button>
            <button type="button" className="live-secondary" disabled={!navigation.after} onClick={() => navigate(navigation.filters)}>{sq.first(l)}</button>
            <button type="button" className="live-secondary" disabled={!page.next} onClick={() => { parents.current.set(page.next,navigation.after);if (parents.current.size > 32) parents.current.delete(parents.current.keys().next().value!);navigate(navigation.filters,page.next); }}>{sq.next(l)}</button></div></>}
      </section>
      <SessionOperations l={l} api={api} refreshKey={refresh} onEnd={open} locked={locked} onSignedOut={onSignedOut} />
      <AccessHistory key={scopeKey(self)} l={l} api={api} self={self} reloadKey={refresh} say={setNotice} onSignedOut={onSignedOut} />
    </div>
    <aside className="live-panel mysec-session-limits" aria-labelledby="mysec-limits-title"><h2 id="mysec-limits-title">{m.limitsTitle(l)}</h2><dl className="live-facts">
      {[[m.idle(l),m.minutes(l,self.idle_minutes)],[m.longest(l),m.longestValue(l)],[m.maxAge(l),self.max_age_seconds ? m.age(l,self.max_age_seconds) : m.noMaxAge(l)],[m.youSignedIn(l),whenSeconds(l,self.signed_in_at)],[m.lastConfirmed(l),whenSeconds(l,self.authenticated_at)]].map(([label,value]) => <div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}
    </dl></aside>
    {target && <SessionSelection key={target.handle ?? "others"} l={l} api={api} handle={target.handle} store={store} onClose={close} onSignedOut={onSignedOut} />}
  </div>;
}

const scopeKey = (self: SelfPage) => JSON.stringify([self.user_id,self.workspace,self.current_session.handle]);
