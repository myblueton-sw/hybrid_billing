import { sizeFor, sizeRefused } from "../../../../service/query-settings";
import { useEffect, useMemo, useRef, useState, type FormEvent, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, type AdminClient } from "../../../../service/admin";
import type { Permission, PermissionPage, PermissionQuery, PermissionSelector, PermissionSource, PermissionState } from "../../../../service/access-query";
import { w } from "../users/live-copy";
import { when } from "../authentication/live-model";
import { qcopy } from "../service-accounts/query-copy";
import { cursorTrail } from "../service-accounts/query-model";
import { a, actionLabel, kindLabel } from "./live-copy";
import { accessText, revocable, stateLine } from "./live-model";
import { p } from "./permission-copy";
import { q, v } from "./request-copy";
import { breakable } from "./who";

type Change = (extra: Record<string, string | null>, push?: boolean) => void;
const prefix = "acc_";
const fields = ["q", "subject", "action", "customer", "legal_entity", "source", "state", "before"] as const;
type Field = (typeof fields)[number];
const states: PermissionState[] = ["active", "pending", "revocation_unconfirmed", "expired", "revoked", "ended"];
const identifier = /^[A-Za-z0-9._:-]{1,128}$/;
export const pageSize = 50;

export type PermissionFilters = { valid: true; query: PermissionQuery; applied: number; draft: Record<Field, string> } | { valid: false };

export type IssuedCursor = { after: string; size: number | undefined } | null;

export function withoutStaleCursor(filters: PermissionFilters, issued: IssuedCursor, size: number | undefined): { filters: PermissionFilters; stale: boolean } {
  if (!filters.valid || !filters.query.after || !issued || issued.after !== filters.query.after || issued.size === size) return { filters, stale: false };
  const { after: _dropped, ...query } = filters.query;
  return { filters: { ...filters, query }, stale: true };
}

export function permissionFilters(params: URLSearchParams, size: number | undefined): PermissionFilters {
  const draft = Object.fromEntries(fields.map((f) => [f, ""])) as Record<Field, string>;
  const query: PermissionQuery = size === undefined ? {} : { limit: size };
  let applied = 0;
  for (const [name, value] of params) {
    if (!name.startsWith(prefix)) continue;
    const key = name.slice(prefix.length);
    if (params.getAll(name).length !== 1 || value === "") return { valid: false };
    if (key === "after") {
      if (value.length > 4096) return { valid: false };
      query.after = value;
      continue;
    }
    if (!(fields as readonly string[]).includes(key)) return { valid: false };
    if (key === "q") {
      if (value.trim() === "" || new TextEncoder().encode(value.trim()).length > 64) return { valid: false };
      query.q = value.trim();
    } else if (key === "source") {
      if (value !== "grant" && value !== "delegation") return { valid: false };
      query.source = value as PermissionSource;
    } else if (key === "state") {
      if (!(states as string[]).includes(value)) return { valid: false };
      query.state = value as PermissionState;
    } else if (key === "before") {
      if (!/^\d{4}-\d{2}-\d{2}$/.test(value) || Number.isNaN(Date.parse(value + "T00:00:00Z")) || new Date(value + "T00:00:00Z").toISOString().slice(0, 10) !== value) return { valid: false };
      query.expires_before = value + "T00:00:00Z";
    } else {
      if (!identifier.test(value)) return { valid: false };
      query[key as "subject" | "action" | "customer" | "legal_entity"] = value;
    }
    draft[key as Field] = key === "q" ? value.trim() : value;
    applied++;
  }
  return { valid: true, query, applied, draft };
}

type Read = { state: "loading" } | { state: "error"; error: unknown } | { state: "ready"; page: PermissionPage };

export function usePermissions(api: AdminClient, filters: PermissionFilters, generation: number, settled = true): { read: Read; reload: () => void } {
  const key = filters.valid ? JSON.stringify(filters.query) : "";
  const [local, setLocal] = useState(0);
  const token = useMemo(() => ({}), [api, key, generation, local, settled]);
  const [result, setResult] = useState<{ token: object; read: Read } | null>(null);
  useEffect(() => {
    if (!filters.valid || !settled) return;
    let active = true;
    api.accessPermissions(filters.query).then(
      (page) => { if (active) setResult({ token, read: { state: "ready", page } }); },
      (error: unknown) => { if (active) setResult({ token, read: { state: "error", error } }); },
    );
    return () => { active = false; };
  }, [token]); // eslint-disable-line react-hooks/exhaustive-deps
  return { read: result?.token === token ? result.read : { state: "loading" }, reload: () => setLocal((n) => n + 1) };
}

export function readFailure(l: Locale, error: unknown): string {
  if (sizeRefused(error)) return qcopy(l, "readSizeRefused");
  if (error instanceof AdminError && error.failure === "denied") return qcopy(l, "readDenied");
  const changed = error instanceof AdminError && (error.code === "revision_conflict" || error.code === "invalid_request");
  return qcopy(l, changed ? "readQueryChanged" : "readUnavailable");
}

function Time({ l, iso }: { l: Locale; iso: string }) {
  return <time dateTime={iso} title={iso}>{when(l, iso)}</time>;
}

function selectorText(l: Locale, s: PermissionSelector): ReactNode {
  return s.any ? qcopy(l, "all") : <bdi dir="ltr" className="live-mono">{s.id}</bdi>;
}

export function effectiveLines(l: Locale, row: Permission): [string, string] {
  return row.state === "ended" ? [p.endedChain(l), v.endedByChain(l)] : stateLine(l, row);
}

function stateWord(l: Locale, state: PermissionState): string {
  return state === "active" ? a.stActive(l) : state === "pending" ? a.stPending(l) : state === "expired" ? a.stExpired(l) : state === "revoked" ? a.stRevoked(l)
    : state === "ended" ? p.endedChain(l) : a.stUnconfirmed(l);
}

function Filters({ l, filters, actions, onParams, onSent }: { l: Locale; filters: Extract<PermissionFilters, { valid: true }>; actions: string[]; onParams: Change; onSent: () => void }) {
  const [draft, setDraft] = useState(filters.draft);
  const set = (key: Field, value: string) => setDraft({ ...draft, [key]: value });
  const submit = (event: FormEvent) => {
    event.preventDefault();
    onSent();
    onParams({ ...Object.fromEntries(fields.map((key) => [prefix + key, draft[key].trim() || null])), [prefix + "after"]: null }, true);
  };
  const reset = () => {
    setDraft(Object.fromEntries(fields.map((f) => [f, ""])) as Record<Field, string>);
    onSent();
    onParams(Object.fromEntries([...fields, "after"].map((key) => [prefix + key, null])), true);
  };
  const text = (key: Field, label: string, hint?: string, ltr = true) => (
    <div className="service-field">
      <label htmlFor={"acc-" + key}>{label}</label>
      {hint && <p className="service-hint" id={"acc-" + key + "-hint"}>{hint}</p>}
      <input id={"acc-" + key} dir={ltr ? "ltr" : undefined} value={draft[key]} aria-describedby={hint ? "acc-" + key + "-hint" : undefined} autoComplete="off"
        onChange={(e) => { if (e.target.value.length <= 128) set(key, e.target.value); }} />
    </div>
  );
  const select = (key: Field, label: string, options: [string, string][]) => (
    <div className="service-field">
      <label htmlFor={"acc-" + key}>{label}</label>
      <select id={"acc-" + key} className="live-select" value={draft[key]} onChange={(e) => set(key, e.target.value)}>
        <option value="">{qcopy(l, "all")}</option>
        {options.map(([value, name]) => <option key={value} value={value}>{name}</option>)}
      </select>
    </div>
  );
  const [open, setOpen] = useState(filters.applied > 0);
  return (
    <div className="access-filters" data-open={open ? "true" : "false"}>
      <button type="button" className="live-secondary access-filters-toggle" aria-expanded={open} aria-controls="access-filter-form" onClick={() => setOpen(!open)}>{p.filters(l, filters.applied)}</button>
      <form id="access-filter-form" className="access-filter-form" onSubmit={submit} noValidate>
      <p className="live-meta access-filter-note">{p.byName(l)}</p>
      <div className="access-filter-grid">
        {text("q", qcopy(l, "search"), p.searchHint(l), false)}
        {text("subject", p.holderId(l), p.holderHint(l))}
        {select("action", qcopy(l, "action"), actions.map((id) => [id, actionLabel(l, id) ?? id]))}
        {text("customer", p.customerId(l))}
        {text("legal_entity", p.legalEntityId(l))}
        {select("source", p.colSource(l), [["grant", p.grant(l)], ["delegation", a.colDelegation(l)]])}
        {select("state", qcopy(l, "state"), states.map((s) => [s, stateWord(l, s)]))}
        <div className="service-field">
          <label htmlFor="acc-before">{p.endsBefore(l)}</label>
          <p className="service-hint" id="acc-before-hint">{p.endsBeforeHint(l)}</p>
          <input id="acc-before" type="date" dir="ltr" value={draft.before} aria-describedby="acc-before-hint" onChange={(e) => set("before", e.target.value)} />
        </div>
      </div>
      <div className="live-buttons">
        <button type="submit" className="live-secondary access-apply">{qcopy(l, "apply")}</button>
        <button type="button" className="live-secondary access-reset" onClick={reset}>{qcopy(l, "reset")}</button>
      </div>
      </form>
    </div>
  );
}

function Rows({ l, rows, self, selected, onRevokeGrant, onRevokeDelegation }: {
  l: Locale; rows: Permission[]; self: string; selected: string; onRevokeGrant: (row: Permission) => void; onRevokeDelegation: (row: Permission) => void;
}) {
  return (
    <div className="security-table-scroll live-table live-access access-permissions">
      <table role="table">
        <colgroup><col className="acc-col-holder" /><col className="acc-col-access" /><col className="acc-col-scope" /><col className="acc-col-source" /><col className="acc-col-parent" /><col className="acc-col-validity" /><col className="acc-col-effective" /><col className="acc-col-action" /></colgroup>
        <thead role="rowgroup"><tr role="row">
          <th scope="col" role="columnheader">{p.colHolder(l)}</th>
          <th scope="col" role="columnheader">{a.colAccess(l)}</th>
          <th scope="col" role="columnheader">{p.colScope(l)}</th>
          <th scope="col" role="columnheader">{p.colSource(l)}</th>
          <th scope="col" role="columnheader" className="acc-parent">{p.colParent(l)}</th>
          <th scope="col" role="columnheader" className="acc-validity">{p.colValidity(l)}</th>
          <th scope="col" role="columnheader">{p.colEffective(l)}</th>
          <th scope="col" role="columnheader"><span className="visually-hidden">{a.colActions(l)}</span></th>
        </tr></thead>
        <tbody role="rowgroup">
          {rows.map((row) => {
            const grant = row.source === "grant";
            const client = grant && row.subject === "";
            const who = row.subject_login || row.subject;
            const [state, why] = effectiveLines(l, row);
            const from = p.from(l, when(l, row.not_before));
            return (
              <tr key={row.source + "/" + row.id} role="row" data-permission={row.source + "/" + row.id} className={selected === row.id ? "live-selected" : undefined}>
                <td role="cell" className="live-user" data-label={p.colHolder(l)}>
                  {client ? <>{p.console(l)}<span className="live-sub">{a.ceilingTitle(l)}</span></>
                    : <><bdi dir="ltr" className="acc-login">{breakable(who)}</bdi>{grant ? (row.subject_login && <span className="live-sub live-mono" dir="ltr">{row.subject}</span>)
                      : <span className="live-sub">{p.delegatedBy(l, "⁨" + (row.delegator_login || row.delegator) + "⁩")}</span>}</>}
                </td>
                <td role="cell" data-label={a.colAccess(l)}>{accessText(l, row.action, row.fields)}<span className="live-sub">{kindLabel(l, row.kind) ?? row.kind}</span></td>
                <td role="cell" data-label={p.colScope(l)}>
                  <span>{a.colCustomer(l)}: {selectorText(l, row.scope.customer)}</span>
                  <span className="live-sub">{qcopy(l, "legalEntity")}: {selectorText(l, row.scope.legal_entity)}</span>
                  {!row.scope.contract.any && <span className="live-sub">{qcopy(l, "contract")}: {selectorText(l, row.scope.contract)}</span>}
                  {row.scope.period !== "any" && <span className="live-sub">{qcopy(l, "period")}: <bdi dir="ltr">{row.scope.period}</bdi></span>}
                </td>
                <td role="cell" data-label={p.colSource(l)}>
                  {grant ? p.grant(l) : a.colDelegation(l)}
                  {row.redelegate && <span className="live-sub">{a.canRedelegate(l)}</span>}
                  {row.parent && <span className="live-sub acc-folded">{a.fromParent(l, "\u0001").split("\u0001")[0]}<bdi dir="ltr" className="live-mono">{row.parent}</bdi>{a.fromParent(l, "\u0001").split("\u0001")[1]}</span>}
                </td>
                <td role="cell" className="acc-parent" data-label={p.colParent(l)}>{row.parent ? <bdi dir="ltr" className="live-mono">{row.parent}</bdi> : <span className="live-quiet">{p.none(l)}</span>}</td>
                <td role="cell" className="acc-validity" data-label={p.colValidity(l)}><span>{from}</span><span className="live-sub">{a.until(l, when(l, row.not_after))}</span></td>
                <td role="cell" data-label={p.colEffective(l)} data-state={row.state} className={row.state !== "active" || !row.effective ? "live-strong" : ""}>
                  <span className="acc-state">{state}</span><span className="live-sub">{why}</span>
                  {row.revocation_requested && <span className="live-sub" data-requested={row.id}>{q.revocationRequested(l)}</span>}
                  <span className="live-sub acc-folded acc-folded-validity">{from}</span>
                </td>
                <td role="cell" className="acc-action">
                  {client ? <span className="live-quiet">{p.setByInstallation(l)}</span>
                    : grant ? (row.subject === self ? <span className="live-quiet">{a.yourGrant(l)}</span>
                      : row.revocation_requested || !revocable(row.state) ? null
                        : <button type="button" className="live-secondary live-revoke" data-grant={row.id} onClick={() => onRevokeGrant(row)}>{q.requestRevocation(l)}<span className="visually-hidden"> {who}</span></button>)
                      : revocable(row.state) ? <button type="button" className="live-secondary live-revoke" data-delegation={row.id} onClick={() => onRevokeDelegation(row)}>{a.revoke(l)}<span className="visually-hidden"> {who}</span></button> : null}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

export function PermissionList({ l, api, self, params, onParams, generation, pageLimit, settled, onRefresh, actions, selected, onSignOut, onRevokeGrant, onRevokeDelegation }: {
  l: Locale; api: AdminClient; self: string; params: URLSearchParams; onParams: Change; generation: number; pageLimit: number | null; settled: boolean; onRefresh: () => void;
  actions: string[]; selected: string; onSignOut: () => void;
  onRevokeGrant: (row: Permission) => void; onRevokeDelegation: (row: Permission) => void;
}) {
  const sent = sizeFor(pageSize, pageLimit);
  const issued = useRef<IssuedCursor>(null);
  const base = useMemo(() => permissionFilters(params, sent), [params, sent]);
  const { filters, stale } = withoutStaleCursor(base, issued.current, sent);
  const { read, reload } = usePermissions(api, filters, generation, settled);
  const [trail] = useState(cursorTrail);
  useEffect(() => {
    if (!stale) return;
    issued.current = null;
    trail.clear();
    onParams({ [prefix + "after"]: null });
  }, [stale, trail, onParams]);
  const count = useRef<HTMLDivElement>(null);
  const wantFocus = useRef(false);
  const key = filters.valid ? JSON.stringify({ ...filters.query, after: undefined, limit: undefined }) : "";
  useEffect(() => {
    if (read.state !== "loading" && wantFocus.current) { wantFocus.current = false; (count.current ?? document.querySelector<HTMLElement>(".access-list-failure"))?.focus(); }
  }, [read]);
  const clear = () => onParams(Object.fromEntries([...fields, "after"].map((f) => [prefix + f, null])), true);
  if (!filters.valid)
    return <section className="access-list-body"><p role="alert">{p.invalidFilters(l)}</p><button type="button" className="live-secondary" onClick={clear}>{qcopy(l, "reset")}</button></section>;
  const go = (after: string | null) => { wantFocus.current = true; issued.current = after ? { after, size: sent } : null; onParams({ [prefix + "after"]: after }, true); };
  const first = !filters.query.after;
  return (
    <div className="access-list-body">
      <Filters key={key} l={l} filters={filters} actions={actions} onParams={onParams} onSent={() => { trail.clear(); wantFocus.current = true; }} />
      {read.state === "loading" && <p role="status" className="live-meta">{a.loading(l)}</p>}
      {read.state === "error" && (
        <section className="live-error access-list-failure" role="alert" tabIndex={-1}>
          <h3>{a.loadFailed(l)}</h3>
          <p>{readFailure(l, read.error)}</p>
          <div className="live-buttons">
            {read.error instanceof AdminError && read.error.failure === "unauthenticated"
              ? <button type="button" className="live-secondary" onClick={onSignOut}>{w.signInAgain(l)}</button>
              : <button type="button" className="live-secondary" onClick={sizeRefused(read.error) ? onRefresh : reload}>{w.reload(l)}</button>}
            {!sizeRefused(read.error) && <button type="button" className="live-secondary" onClick={() => { trail.clear(); go(null); }}>{qcopy(l, "first")}</button>}
          </div>
        </section>
      )}
      {read.state === "ready" && (
        <>
          <div className="access-count" tabIndex={-1} ref={count}>
            <p className="live-meta">{qcopy(l, read.page.count.certainty === "exact" ? "exact" : "lower")}: {read.page.count.value}</p>
            <p className="live-meta">{qcopy(l, "reference")}: <Time l={l} iso={read.page.reference_at} /></p>
          </div>
          <p className="visually-hidden" role="status" aria-live="polite">{qcopy(l, read.page.count.certainty === "exact" ? "exact" : "lower")}: {read.page.count.value}</p>
          {read.page.rows.length === 0
            ? <p className="live-empty">{qcopy(l, read.page.next ? "incomplete" : "empty")}</p>
            : <Rows l={l} rows={read.page.rows} self={self} selected={selected} onRevokeGrant={onRevokeGrant} onRevokeDelegation={onRevokeDelegation} />}
          <nav className="live-buttons access-paging" aria-label={p.title(l)}>
            <button type="button" className="live-secondary" disabled={first} onClick={() => { trail.clear(); go(null); }}>{qcopy(l, "first")}</button>
            <button type="button" className="live-secondary" disabled={first} onClick={() => go(trail.previous(key) || null)}>{qcopy(l, "previous")}</button>
            <button type="button" className="live-secondary" disabled={!read.page.next} onClick={() => { if (trail.next(key, filters.query.after ?? "", read.page.next)) go(read.page.next); }}>{qcopy(l, "next")}</button>
          </nav>
        </>
      )}
    </div>
  );
}
