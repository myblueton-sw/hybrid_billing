import { useCallback, useEffect, useRef, useState } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, type OrgParty } from "../../../../service/admin";
import { useAdminClient } from "../../../../service/context";
import type { ContractDetail, ContractRow, ContractsPage } from "../../../../service/contracts";
import type { Session } from "../../../../service/api";
import { c as sc, retryPhrase } from "../../../../service/copy";
import { useWide } from "../../../identity/pages/users/Live";
import { k } from "./live-copy";
import { activeFilters, filtered, filtersOf, forget, legalEntities, paths, periodOf, setScope, settlements, states, validDay, type Filters } from "./live-model";
import { Detail, Settlement } from "./LiveDetail";
import { day, ltr, nameOf } from "./live-format";
import "../../../identity/pages/users/live.css";
import "./live.css";

type Props = {
  locale: Locale;
  view: string;
  params: URLSearchParams;
  session: Session;
  onSignOut: () => void;
  onView: (view: string, extra?: Record<string, string>) => void;
  onParams: (extra: Record<string, string | null>) => void;
};

export function failureText(l: Locale, e: unknown): { title: string; body: string } {
  if (e instanceof AdminError) {
    if (e.failure === "limited") return { title: sc.failed(l), body: sc.limited(l, retryPhrase(l, e.retryAfter || 60)) };
    if (e.failure === "busy" || e.failure === "unavailable") return { title: sc.failed(l), body: sc.unavailable(l) };
  }
  return { title: sc.failed(l), body: "" };
}

export default function LiveContracts({ locale: l, view, params, session, onSignOut, onParams }: Props) {
  const api = useAdminClient();
  setScope(session.workspace, session.user, session.csrf);
  const wide = useWide();
  const [page, setPage] = useState<ContractsPage | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [loadError, setLoadError] = useState<unknown>(null);
  const [detail, setDetail] = useState<ContractDetail | "missing" | null>(null);
  const [detailError, setDetailError] = useState<unknown>(null);
  const [parties, setParties] = useState<OrgParty[]>([]);
  const [denied, setDenied] = useState(false);
  const [signedOut, setSignedOut] = useState(false);
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);
  const seq = useRef(0);
  const detailSeq = useRef(0);
  const selectedId = params.get("contract") ?? "";
  const on = params.get("on") ?? "";
  const onValid = !on || validDay(on);
  const period = periodOf(params);
  const filters = filtersOf(params);

  const fail = (e: unknown) => {
    if (e instanceof AdminError && e.failure === "denied") setDenied(true);
    if (e instanceof AdminError && e.failure === "unauthenticated") setSignedOut(true);
  };

  const load = useCallback(async () => {
    if (!onValid) return;
    const n = ++seq.current;
    setBusy(true);
    try {
      const got = await api.contracts(on);
      if (n !== seq.current) return;
      setPage(got);
      setStatus("ready");
      setLoadError(null);
    } catch (e) {
      if (n !== seq.current) return;
      fail(e);
      setLoadError(e);
      setStatus("error");
    } finally {
      if (n === seq.current) setBusy(false);
    }
  }, [api, on, onValid]);

  const shownId = useRef("");
  const loadDetail = useCallback(async () => {
    const n = ++detailSeq.current;
    if (shownId.current !== selectedId) {
      setDetail(null);
      setDetailError(null);
      setNotice("");
      shownId.current = selectedId;
    }
    if (!selectedId) return;
    try {
      const got = period.problem ? await api.contract(selectedId) : await api.contract(selectedId, period.from, period.until);
      if (n !== detailSeq.current) return;
      setDetail(got);
      setDetailError(null);
    } catch (e) {
      if (n !== detailSeq.current) return;
      fail(e);
      setDetailError(e);
    }
  }, [api, selectedId, period.from, period.until, period.problem]);

  useEffect(() => { void load(); }, [load]);
  useEffect(() => { void loadDetail(); }, [loadDetail]);
  useEffect(() => {
    let live = true;
    api.organizations({ limit: 1 }).then((p) => { if (live) setParties(p.parties); }, () => undefined);
    return () => { live = false; };
  }, [api]);

  const reloadAll = async () => {
    await Promise.all([load(), loadDetail()]);
  };

  if (denied) {
    return (
      <div className="o03-live live-users">
        <section className="live-unavailable" aria-labelledby="o03-denied">
          <h2 id="o03-denied">{k.deniedTitle(l)}</h2>
          <p>{k.denied(l)}</p>
        </section>
      </div>
    );
  }
  if (signedOut) {
    return (
      <div className="o03-live">
        <div className="live-error" role="alert">
          <h2>{k.sessionEnded(l)}</h2>
          <button type="button" className="live-secondary" onClick={() => { forget(); onSignOut(); }}>{k.signInAgain(l)}</button>
        </div>
      </div>
    );
  }

  const rows = page?.contracts ?? [];
  const shown = filtered(rows, filters);
  const selected = rows.find((r) => r.id === selectedId);
  const split = wide && !!selectedId;
  const stepped = !wide && !!selectedId;
  const impactKnown = page ? Object.values(page.impact).every((v) => v === "not_computed") : true;
  const select = (id: string) => onParams({ contract: id, from: null, until: null });

  const list = (
    <div className="live-list o03-list" aria-busy={busy || undefined}>
      <FilterPanel l={l} filters={filters} rows={rows} parties={parties} wide={wide} rail={split} onParams={onParams} />
      <p className="live-counter">{k.countOf(l, shown.length, rows.length)}</p>
      {status === "loading" && !page && <p className="live-meta" role="status">{k.loading(l)}</p>}
      {page && rows.length === 0 && <p className="live-empty">{k.emptyAll(l)}</p>}
      {page && rows.length > 0 && shown.length === 0 && (
        <p className="live-empty">
          {k.emptyFiltered(l)}{" "}
          <button type="button" className="service-button-link" onClick={() => onParams({ state: null, settlement: null, path: null, buyer: null })}>{k.clearFilters(l)}</button>
        </p>
      )}
      {shown.length > 0 && (split ? <Rail l={l} rows={shown} parties={parties} selected={selectedId} onSelect={select} /> : <Table l={l} rows={shown} parties={parties} on={page?.on ?? ""} selected={selectedId} onSelect={select} />)}
      {split && <button type="button" className="service-button-link o03-show-all" onClick={() => onParams({ contract: null })}>{k.showAll(l)}</button>}
    </div>
  );

  return (
    <div className="o03-live" data-live={view}>
      <div className="o03-scope">
        <OnField l={l} value={on || page?.today || ""} valid={onValid} onChange={(v) => onParams({ on: v || null })} />
        <dl className="o03-scope-facts">
          <div><dt>{k.today(l)}</dt><dd>{page ? day(l, page.today) : "—"}</dd></div>
          <div><dt>{k.revision(l)}</dt><dd>{page ? ltr(page.revision) : "—"}</dd></div>
          <div><dt>{k.contractsCount(l)}</dt><dd>{page ? rows.length : "—"}</dd></div>
        </dl>
        <button type="button" className="live-secondary o03-reload" onClick={() => void reloadAll()}>{k.reload(l)}</button>
      </div>
      <p className="live-note-rule o03-impact-note"><strong>{k.impactLead(l)}</strong> {impactKnown ? k.impactBody(l) : k.impactUnknown(l)}</p>
      {notice && <div id="o03-notice" className="service-status" tabIndex={-1} role="status"><p>{notice}</p></div>}
      {status === "error" && (
        <div className="live-error o03-failure" role="alert">
          <h2>{failureText(l, loadError).title}</h2>
          {failureText(l, loadError).body && <p>{failureText(l, loadError).body}</p>}
          <button type="button" className="live-secondary" onClick={() => void load()}>{sc.tryAgain(l)}</button>
        </div>
      )}
      <div className="live-toolbar"><h2 id="o03-view-title" tabIndex={-1}>{k.viewTitle(l, view)}</h2></div>
      <div className="live-users o03-open" data-wide={split ? "true" : "false"}>
        {!stepped && list}
        {selectedId && (
          <aside className="live-panel o03-panel" aria-busy={busy || undefined} aria-labelledby="o03-contract-title">
            {stepped && <button type="button" className="service-button-link live-back" onClick={() => onParams({ contract: null })}>{k.back(l)}</button>}
            {detail === "missing" || (page && !selected && !detail && !detailError) ? (
              <div className="live-note-rule" role="status">
                <p>{k.notFound(l)}</p>
                <button type="button" className="service-button-link" onClick={() => onParams({ contract: null })}>{k.showAll(l)}</button>
              </div>
            ) : detailError && !detail ? (
              <div className="live-error" role="alert">
                <h2>{failureText(l, detailError).title}</h2>
                <button type="button" className="live-secondary" onClick={() => void loadDetail()}>{sc.tryAgain(l)}</button>
              </div>
            ) : detail && detail.contract.id === selectedId ? (
              <Detail
                key={selectedId}
                l={l} view={view} detail={detail} parties={parties} api={api} session={session} params={params} period={period}
                onParams={onParams} onReload={reloadAll}
                onDone={(text) => { setNotice(text); void reloadAll(); requestAnimationFrame(() => document.getElementById("o03-notice")?.focus()); }}
                onDenied={() => setDenied(true)} onSignedOut={() => setSignedOut(true)}
              />
            ) : (
              <p className="live-meta" role="status">{k.loading(l)}</p>
            )}
          </aside>
        )}
      </div>
    </div>
  );
}

function OnField({ l, value, valid, onChange }: { l: Locale; value: string; valid: boolean; onChange: (v: string) => void }) {
  const [draft, setDraft] = useState(value);
  useEffect(() => setDraft(value), [value]);
  const bad = draft !== "" && !validDay(draft);
  return (
    <div className="service-field o03-on">
      <label htmlFor="o03-on">{k.stateOn(l)}</label>
      <p className="service-hint" id="o03-on-hint">{k.stateOnHint(l)}</p>
      {(bad || !valid) && <p className="service-field-error" id="o03-on-error">{k.onInvalid(l)}</p>}
      <input
        id="o03-on" type="date" className="o03-date" value={draft} aria-invalid={bad || !valid || undefined}
        aria-describedby={["o03-on-hint", bad || !valid ? "o03-on-error" : ""].filter(Boolean).join(" ")}
        onChange={(e) => { setDraft(e.target.value); if (e.target.value === "" || validDay(e.target.value)) onChange(e.target.value); }}
      />
    </div>
  );
}

function FilterPanel({ l, filters, rows, parties, wide, rail, onParams }: {
  l: Locale; filters: Filters; rows: ContractRow[]; parties: OrgParty[]; wide: boolean; rail: boolean; onParams: (extra: Record<string, string | null>) => void;
}) {
  const n = activeFilters(filters);
  const buyers = legalEntities(rows);
  const select = (id: string, label: string, value: string, options: [string, string][], key: string) => (
    <div className="service-field">
      <label htmlFor={id}>{label}</label>
      <select id={id} className="live-select o03-filter" value={value} onChange={(e) => onParams({ [key]: e.target.value || null })}>
        <option value="">{k.any(l)}</option>
        {options.map(([v, text]) => <option key={v} value={v}>{text}</option>)}
      </select>
    </div>
  );
  return (
    <details className="o03-filter-fold" open={(wide && !rail) || n > 0 || undefined}>
      <summary>{k.filters(l, n)}</summary>
      <div className="o03-filters">
        {select("o03-f-state", k.contractState(l), filters.state, states.map((s) => [s, k.state(l, s)]), "state")}
        {select("o03-f-settlement", k.finalSettlement(l), filters.settlement, settlements.map((s) => [s, k.settlement(l, s)]), "settlement")}
        {select("o03-f-path", k.colPath(l), filters.path, paths.map((s) => [s, k.path(l, s)]), "path")}
        {select("o03-f-buyer", k.colBuyer(l), filters.buyer, buyers.map((b) => [b, nameOf(b, parties)]), "buyer")}
        {n > 0 && <button type="button" className="service-button-link" onClick={() => onParams({ state: null, settlement: null, path: null, buyer: null })}>{k.clearFilters(l)}</button>}
      </div>
    </details>
  );
}

function Rail({ l, rows, parties, selected, onSelect }: { l: Locale; rows: ContractRow[]; parties: OrgParty[]; selected: string; onSelect: (id: string) => void }) {
  return (
    <ul className="o03-rail">
      {rows.map((r) => (
        <li key={r.id}>
          <button type="button" className="o03-item" aria-current={r.id === selected || undefined} onClick={() => onSelect(r.id)}>
            <span className="o03-item-name"><bdi>{r.name || r.id}</bdi></span>
            <span className="o03-item-line">
              <span className="visually-hidden">{k.contractState(l)}: </span><span className="o03-tag">{k.state(l, r.state)}</span>{" "}
              <span className="visually-hidden">{k.finalSettlement(l)}: </span><Settlement l={l} value={r.settlement} />
            </span>
            <span className="o03-item-line live-quiet"><bdi>{nameOf(r.buyer, parties)}</bdi>{r.open_gaps === null ? " · " + k.colGaps(l) + ": " + k.unknown(l) : r.open_gaps > 0 ? " · " + k.colGaps(l) + ": " + r.open_gaps : ""}</span>
          </button>
        </li>
      ))}
    </ul>
  );
}

function Table({ l, rows, parties, on, selected, onSelect }: {
  l: Locale; rows: ContractRow[]; parties: OrgParty[]; on: string; selected: string; onSelect: (id: string) => void;
}) {
  return (
    <div className="live-table">
      <table className="o03-table">
        <colgroup>
          <col className="o03-c18" /><col className="o03-c14" /><col className="o03-c12" /><col className="o03-c10" />
          <col className="o03-c12" /><col className="o03-c12" /><col className="o03-c14" /><col className="o03-c8" />
        </colgroup>
        <thead>
          <tr>
            <th scope="col">{k.colContract(l)}</th><th scope="col">{k.colBuyer(l)}</th><th scope="col">{k.colPayer(l)}</th><th scope="col">{k.colPath(l)}</th>
            <th scope="col">{k.colPeriod(l)}</th><th scope="col">{k.colStateOnLabel(l)} <span className="o03-nowrap">{"\u2068" + day(l, on) + "\u2069"}</span></th><th scope="col">{k.colSettlement(l)}</th><th scope="col">{k.colGaps(l)}</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.id} className={r.id === selected ? "live-selected" : undefined} aria-current={r.id === selected || undefined}>
              <td>
                <button type="button" className="live-open" onClick={() => onSelect(r.id)}><bdi>{r.name || r.id}</bdi></button>
                <span className="live-sub live-mono">{ltr(r.id)}</span>
              </td>
              <td data-label={k.colBuyer(l)}><bdi>{nameOf(r.buyer, parties) || k.notRecorded(l)}</bdi></td>
              <td data-label={k.colPayer(l)}><bdi>{nameOf(r.payer, parties) || k.notRecorded(l)}</bdi></td>
              <td data-label={k.colPath(l)}>{k.path(l, r.path)}</td>
              <td data-label={k.colPeriod(l)}>
                {r.start ? k.since(l, day(l, r.start)) : k.notRecorded(l)}
                <span className="live-sub">{r.end ? k.until(l, day(l, r.end)) : k.noEnd(l)}</span>
              </td>
              <td data-label={k.colStateOn(l, day(l, on))}><span className="o03-tag">{k.state(l, r.state)}</span></td>
              <td data-label={k.colSettlement(l)}><Settlement l={l} value={r.settlement} /></td>
              <td data-label={k.colGaps(l)} className="o03-num">{r.open_gaps === null ? k.unknown(l) : r.open_gaps === 0 ? k.none(l) : r.open_gaps}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
