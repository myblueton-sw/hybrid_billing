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
import SessionOperations, { summarySentence, ReferencedOperation } from "./SessionOperations";
import AccessHistory from "./AccessHistory";
import { SessionSettings } from "./SessionSettings";
import { AppliedFilters, ListPager, ListSummary, type Chip } from "../../../../shared/list/ListParts";
import { lc } from "../../../../shared/list/list-copy";
import { qcopy } from "../service-accounts/query-copy";
import "./session-query.css";
import "../../../../shared/list/list-pattern.css";

export function SessionTimeHint({ l }: { l: Locale }) {
  return <p id="mysec-time-hint" className="live-meta">{sq.timeHint(l)} {sq.example(l)} <bdi dir="ltr">2026-11-20T09:00:00+09:00</bdi></p>;
}

export function SessionRows({ l,rows,locked,onEnd,onOperation }: { l: Locale; rows: SessionRow[]; locked: boolean; onEnd: (handle: string) => void; onOperation: (reference: string) => void }) {
  return <div className="security-table-scroll live-table"><table><thead><tr><th scope="col">{m.colSession(l)}</th><th scope="col">{m.colSignedIn(l)}</th><th scope="col">{m.colEnds(l)}</th><th scope="col">{sq.state(l)}</th><th scope="col">{m.colActions(l)}</th></tr></thead>
    <tbody>{rows.map(row => <tr key={row.handle}><td><strong>{row.current ? m.thisSession(l) : m.otherSession(l)}</strong><span className="live-mono live-sub"><bdi dir="ltr">{row.handle}</bdi></span></td>
      <td data-label={m.colSignedIn(l)}><time dateTime={row.signed_in}>{whenSeconds(l,row.signed_in)}</time></td><td data-label={m.colEnds(l)}><time dateTime={row.ends}>{whenSeconds(l,row.ends)}</time></td><td data-label={sq.state(l)}>{sessionStateLabel(l,row.state)}<ExclusionTag l={l} reference={row.not_included_operation} onOperation={onOperation} /></td>
      <td data-label={m.colActions(l)}>{row.current ? m.thisSession(l) : <button type="button" className="live-secondary" disabled={locked || !row.can_end} onClick={() => onEnd(row.handle)}>{m.end(l)}</button>}</td></tr>)}</tbody></table></div>;
}

export function ExclusionTag({ l, reference, onOperation }: { l: Locale; reference: string; onOperation: (reference: string) => void }) {
  return reference ? <span className="live-sub mysec-exclusion-tag">{sq.notIncluded(l)} <button type="button" className="live-secondary" onClick={() => onOperation(reference)}>{sq.viewOperation(l)}</button></span> : null;
}

export default function LiveSessions({ l,api,self,params,onParams,onRefresh,onSignOut,onSignedOut }: {
  l: Locale; api: AdminClient; self: SelfPage; params: URLSearchParams; onParams: (extra: Record<string,string | null>,push?: boolean) => void; onRefresh: () => void; onSignOut: () => void; onSignedOut: () => void;
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
  const serialized = new URLSearchParams([...params].filter(([key]) => sessionKeys.includes(key))).toString();
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
  const [operationData,setOperationData] = useState<{ scope: typeof store; reference: string } | null>(null);
  const operation = operationData?.scope === store ? operationData.reference : "";
  const viewOperation = (reference: string) => setOperationData({ scope:store,reference });
  const serial = useRef(0);
  const parents = useRef(new Map<string,string>());
  const invoker = useRef<HTMLElement | null>(null);
  const title = useRef<HTMLHeadingElement>(null);
  const previousFilter = useRef(filterKey);
  const lastAccepted = useRef(0);
  const summary = useRef<HTMLDivElement>(null);
  const focusAfter = useRef<"size" | "summary" | null>(null);
  const [filtersOpen,setFiltersOpen] = useState(false);
  useEffect(() => { setDraft(navigation.filters); setFormError(false); if (previousFilter.current !== filterKey) { parents.current.clear(); previousFilter.current = filterKey; } },[filterKey]);
  useEffect(() => {
    const n = ++serial.current;
    const valid = guard.begin();
    setPage(null);setFailure(null);
    if (navigation.bad) { focusAfter.current = null;setBusy(false);setFormError(true);return; }
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
  useEffect(() => {
    if (busy || focusAfter.current === null) return;
    if (failure !== null) { focusAfter.current = null;return; }
    if (!page) return;
    const want = focusAfter.current;
    focusAfter.current = null;
    if (want === "size") document.getElementById("mysec-size")?.focus();
    else summary.current?.focus();
  },[busy,page,failure]);
  const apply = (event: FormEvent) => {
    event.preventDefault();
    const candidate = new URLSearchParams();
    for (const [key,value] of Object.entries(sessionLink(draft))) if (value !== null) candidate.set(key,value);
    if (parseSessionLink(candidate).bad) { setFormError(true);return; }
    setFiltersOpen(false);
    navigate({ ...draft,limit:navigation.filters.limit });
    if (JSON.stringify(draft) === filterKey && !navigation.after) setRefresh(value => value+1);
  };
  const open = (handle?: string) => { if (locked) return;invoker.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;setTarget(handle ? { handle } : {}); };
  const close = () => { setTarget(null);requestAnimationFrame(() => { const element = invoker.current;if (element?.isConnected) element.focus();else title.current?.focus(); }); };
  const activeFilters: SessionFilters = page ? { session:page.query.session,q:page.query.q,since:page.query.since,until:page.query.until,limit:navigation.filters.limit } : navigation.filters;
  const chips: Chip[] = [
    ...(activeFilters.session !== "all" ? [{ key:"session",label:m.colSession(l),value:({ all:sq.all,current:sq.current,other:sq.other })[activeFilters.session](l) }] : []),
    ...(activeFilters.q ? [{ key:"q",label:sq.search(l),value:activeFilters.q }] : []),
    ...(activeFilters.since ? [{ key:"since",label:sq.since(l),value:activeFilters.since,mono:true }] : []),
    ...(activeFilters.until ? [{ key:"until",label:sq.until(l),value:activeFilters.until,mono:true }] : []),
  ];
  const current = self.current_session;
  const reset = () => navigate({ session:"all",q:"",since:"",until:"",limit:navigation.filters.limit });
  return <div className="live-users mysec-query-layout" data-live="sessions" data-wide={wide ? "true" : "false"}>
    <div className="live-list live-sections">
      <section aria-labelledby="mysec-sessions-title">
        <div className="live-toolbar"><div><h2 id="mysec-sessions-title" ref={title} tabIndex={-1}>{m.sessionsTitle(l)}</h2><p className="live-meta">{m.workspace(l)}: <bdi dir="ltr">{self.workspace}</bdi></p></div>
          <div className="live-tools"><button type="button" className="live-secondary mysec-end-others" disabled={locked} onClick={() => open()}>{e.entry(l)}</button><button type="button" className="live-secondary" onClick={() => { setRefresh(value => value+1);onRefresh(); }}>{w.reload(l)}</button></div></div>
        <p>{sq.fullScope(l)}</p><IntentRecovery l={l} api={api} store={store} />
        {notice && <p role="status">{notice}</p>}
        <div className="mysec-current"><h3>{m.thisSession(l)}</h3><bdi dir="ltr" className="live-mono">{current.handle}</bdi><p><time dateTime={current.signed_in}>{whenSeconds(l,current.signed_in)}</time>{" · "+sessionStateLabel(l,current.state)}</p><ExclusionTag l={l} reference={current.not_included_operation} onOperation={viewOperation} /><button type="button" className="live-secondary" onClick={onSignOut}>{m.signOut(l)}</button></div>
        <section className="mysec-query-form list-filters" data-open={filtersOpen ? "true" : "false"} aria-labelledby="mysec-filters-title">
          <h3 id="mysec-filters-title" className="list-filters-title" tabIndex={-1}>{sq.filters(l)}</h3>
          <button type="button" className="live-secondary list-filters-toggle" aria-expanded={filtersOpen} aria-controls="mysec-filter-form" onClick={() => setFiltersOpen(!filtersOpen)}>{lc.toggle(l,chips.length)}</button>
          {(formError || navigation.bad) && <p id="mysec-filter-error" className="live-error" role="alert">{sq.invalid(l)}</p>}
          <form id="mysec-filter-form" className="list-filter-form" onSubmit={apply} noValidate>
            <div className="list-filter-grid">
              <div className="service-field"><label htmlFor="mysec-f-session">{m.colSession(l)}</label><select id="mysec-f-session" className="live-select" value={draft.session} onChange={event => setDraft(value => ({ ...value,session:event.target.value as SessionFilters["session"] }))}><option value="all">{sq.all(l)}</option><option value="current">{sq.current(l)}</option><option value="other">{sq.other(l)}</option></select></div>
              <div className="service-field"><label htmlFor="mysec-f-q">{sq.search(l)}</label><input id="mysec-f-q" type="search" value={draft.q} onChange={event => setDraft(value => ({ ...value,q:event.target.value }))} /></div>
              <div className="service-field"><label htmlFor="mysec-f-since">{sq.since(l)}</label><input id="mysec-f-since" type="text" dir="ltr" value={draft.since} aria-describedby={formError || navigation.bad ? "mysec-time-hint mysec-filter-error" : "mysec-time-hint"} aria-invalid={formError || navigation.bad || undefined} onChange={event => setDraft(value => ({ ...value,since:event.target.value }))} /></div>
              <div className="service-field"><label htmlFor="mysec-f-until">{sq.until(l)}</label><input id="mysec-f-until" type="text" dir="ltr" value={draft.until} aria-describedby={formError || navigation.bad ? "mysec-time-hint mysec-filter-error" : "mysec-time-hint"} aria-invalid={formError || navigation.bad || undefined} onChange={event => setDraft(value => ({ ...value,until:event.target.value }))} /></div>
            </div><SessionTimeHint l={l} />
            <div className="live-buttons"><button type="submit" className="live-primary" disabled={busy}>{sq.apply(l)}</button><button type="button" className="live-secondary" onClick={reset}>{qcopy(l,"reset")}</button></div>
          </form>
        </section>
        <AppliedFilters l={l} id="mysec-applied" chips={chips} onRemove={key => navigate({ ...activeFilters,[key]:key === "session" ? "all" : "" })} onClear={reset} fallback={filterFocus} />
        {busy && <p role="status">{m.loading(l)}</p>}
        {failure !== null && <section className="live-error" role="alert"><h3>{m.loadFailed(l)}</h3><p>{failure instanceof AdminError && failure.code === "revision_conflict" ? sq.expired(l) : failure instanceof AdminError && failure.code === "invalid_request" ? sq.invalid(l) : m.loadRetry(l)}</p><button type="button" className="live-secondary" onClick={() => { navigate(navigation.filters);setRefresh(value => value+1); }}>{sq.first(l)}</button></section>}
        {page && <><ListSummary l={l} count={String(page.count.value)} exact={page.count.certainty === "exact"} reference={<time dateTime={page.reference_at}>{whenSeconds(l,page.reference_at)}</time>} summaryRef={summary} />
          {page.scan_stopped && <p role="status">{sq.stopped(l)}</p>}
          <SessionSettings l={l} settings={page.settings} />
          <SessionRows l={l} rows={page.rows} locked={locked} onEnd={open} onOperation={viewOperation} />{page.rows.length === 0 && <p>{sq.empty(l)}</p>}
          <ListPager l={l} title={m.sessionsTitle(l)} first={!navigation.after} hasNext={!!page.next} rows={page.rows.length} previousUnavailable={!parents.current.has(navigation.after)}
            size={{ id:"mysec-size",requested:navigation.filters.limit ?? page.query.limit,current:page.query.limit,limit:100,onChange:n => { focusAfter.current = "size";navigate({ ...navigation.filters,limit:n }); } }}
            onFirst={() => { focusAfter.current = "summary";navigate(navigation.filters); }}
            onPrevious={() => { focusAfter.current = "summary";navigate(navigation.filters,parents.current.get(navigation.after) ?? ""); }}
            onNext={() => { focusAfter.current = "summary";parents.current.set(page.next,navigation.after);if (parents.current.size > 32) parents.current.delete(parents.current.keys().next().value!);navigate(navigation.filters,page.next); }} /></>}
      </section>
      {operation && <ReferencedOperation l={l} api={api} reference={operation} onEnd={open} locked={locked} onSignedOut={onSignedOut} />}
      <SessionOperations l={l} api={api} params={params} onParams={onParams} refreshKey={refresh} onEnd={open} locked={locked} onSignedOut={onSignedOut} />
      <AccessHistory key={scopeKey(self)} l={l} api={api} self={self} reloadKey={refresh} say={setNotice} onSignedOut={onSignedOut} />
    </div>
    <aside className="live-panel mysec-session-limits" aria-labelledby="mysec-limits-title"><h2 id="mysec-limits-title">{m.limitsTitle(l)}</h2><dl className="live-facts">
      {[[m.idle(l),m.minutes(l,self.idle_minutes)],[m.longest(l),m.longestValue(l,self.absolute_minutes)],[m.maxAge(l),self.max_age_seconds ? m.age(l,self.max_age_seconds) : m.noMaxAge(l)],[m.youSignedIn(l),whenSeconds(l,self.signed_in_at)],[m.lastConfirmed(l),whenSeconds(l,self.authenticated_at)]].map(([label,value]) => <div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}
    </dl></aside>
    {target && <SessionSelection key={target.handle ?? "others"} l={l} api={api} handle={target.handle} store={store} onClose={close} onSignedOut={onSignedOut} />}
  </div>;
}

const filterFocus = () => {
  const toggle = document.querySelector<HTMLElement>(".mysec-query-form .list-filters-toggle");
  return toggle && toggle.offsetParent ? toggle : document.getElementById("mysec-filters-title");
};

const sessionKeys = ["session", "sq", "ssince", "suntil", "slimit", "safter"];

const scopeKey = (self: SelfPage) => JSON.stringify([self.user_id,self.workspace,self.current_session.handle]);
