import { useState, type FormEvent } from "react";
import type { Locale } from "../../../../shared/types";
import { counted } from "../../../../shared/plural";
import type { AdminClient } from "../../../../service/admin";
import { trustGroups, type ListedTrustChange, type TrustChangePage, type TrustFilter } from "../../../../service/trust-changes";
import { ListFailure, Paging, Reference, usePagedList, type PagedList } from "../access/paged";
import { qcopy } from "../service-accounts/query-copy";
import { p } from "./live-copy";
import { when } from "./live-model";
import { ty } from "./trust-copy";
import { tl } from "./trust-list-copy";
import {
  byStatePriority, closedPageSize, searchFits, evidenceLeftToDetail, filtered, listedBlocked, listedNext, noFilter, openPageSize, sectionState, wholeOnOnePage, type ListFilter,
} from "./trust-list-model";
import { versionOf } from "./trust-model";
import { actor, ltr } from "./TrustChanges";

export function ChangeRows({ l, rows, title, selected, reduced, onOpen }: {
  l: Locale; rows: ListedTrustChange[]; title: string; selected: string; reduced: boolean; onOpen: (id: string, number: number) => void;
}) {
  return (
    <div className="security-table-scroll m02-table-wrap">
      <table className={"m02-table" + (reduced ? " m02-reduced" : "")} role="table">
        <caption className="visually-hidden">{ty.caption(l, title)}</caption>
        {!reduced && <colgroup><col className="m02-c1" /><col className="m02-c2" /><col className="m02-c3" /><col className="m02-c4" /><col className="m02-c5" /></colgroup>}
        <thead role="rowgroup" className={reduced ? "visually-hidden" : undefined}>
          <tr role="row">
            <th scope="col" role="columnheader">{ty.colChange(l)}</th>
            {!reduced && <><th scope="col" role="columnheader">{ty.colState(l)}</th><th scope="col" role="columnheader">{ty.proposedBy(l)}</th><th scope="col" role="columnheader">{ty.colBased(l)}</th><th scope="col" role="columnheader">{ty.colNext(l)}</th></>}
          </tr>
        </thead>
        <tbody role="rowgroup">
          {rows.map((c) => {
            const strong = c.state === "activation_unknown" || (c.state === "draft" && listedBlocked(c));
            const state = <span className={"m02-state-word" + (strong ? " live-strong" : "")}>{ty.state(l, c.state)}</span>;
            const opener = (
              <>
                <button type="button" className="m02-open" data-change={c.id} onClick={() => onOpen(c.id, c.number)}>{ty.change(l, c.number)}</button>
                {selected === c.id && <span className="m02-tag" aria-hidden="true">{ty.selected(l)}</span>}
                {c.earlier_run && <><br /><span className="m02-sub">{ty.earlierRun(l, when(l, c.run_started_at))}</span></>}
              </>
            );
            return (
              <tr key={c.id} role="row" data-state={c.state} aria-current={selected === c.id ? "true" : undefined}>
                {reduced ? (
                  <td role="cell">{opener}<br />{state}<br /><span className="m02-sub">{listedNext(l, c)}</span></td>
                ) : (
                  <>
                    <td role="cell">{opener}</td>
                    <td role="cell" data-label={ty.colState(l)}>{state}</td>
                    <td role="cell" data-label={ty.proposedBy(l)}>{c.proposer ? actor(c.proposer) : <span className="live-quiet">{ty.notRecorded(l)}</span>}</td>
                    <td role="cell" data-label={ty.colBased(l)}>{ltr(versionOf(c.old_epoch))}<br /><span className="m02-sub">{ty.policyNumber(l, c.old_policy)}</span></td>
                    <td role="cell" data-label={ty.colNext(l)}>{listedNext(l, c)}</td>
                  </>
                )}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function Section({ l, group, list, filter, selected, reduced, onOpen, onSignOut }: {
  l: Locale; group: "open" | "closed"; list: PagedList<TrustChangePage>; filter: ListFilter; selected: string; reduced: boolean; onOpen: (id: string, number: number) => void; onSignOut: () => void;
}) {
  const open = group === "open";
  const page = list.read.state === "ready" ? list.read.page : null;
  const whole = !!page && wholeOnOnePage(page, list.first);
  const rows = page ? (open && whole ? byStatePriority(page.rows) : page.rows) : [];
  const title = open ? ty.open(l) : ty.closed(l);
  return (
    <section className="m02-list-section" data-group={group} aria-labelledby={"m02-" + group + "-list"}>
      <h3 id={"m02-" + group + "-list"} data-count={page ? page.count.value : undefined}>{page ? counted(l, Number(page.count.value), open ? ty.openCount() : ty.closedCount()) : title}</h3>
      {list.read.state === "loading" && <p role="status" className="live-meta">{p.loading(l)}</p>}
      <ListFailure l={l} title={open ? tl.openFailed(l) : tl.closedFailed(l)} list={list} onSignOut={onSignOut} />
      {page && open && !whole && <p className="live-meta m02-order-note">{tl.spansPages(l)}</p>}
      {page && (rows.length === 0
        ? <p className="live-empty">{filtered(filter) ? (open ? tl.noOpenMatch(l) : tl.noClosedMatch(l)) : (open ? ty.noOpen(l) : ty.noClosed(l))}</p>
        : <ChangeRows l={l} rows={rows} title={title} selected={selected} reduced={reduced} onOpen={onOpen} />)}
      {page && open && evidenceLeftToDetail(rows) && <p className="live-note m02-evidence-note">{tl.evidenceOnOpen(l)}</p>}
      {page && <Reference l={l} page={page} list={list} />}
      <Paging l={l} label={title} list={list} />
    </section>
  );
}

function PagedSection({ l, api, group, state, filter, generation, selected, reduced, onOpen, onSignOut }: {
  l: Locale; api: AdminClient; group: "open" | "closed"; state: TrustFilter; filter: ListFilter; generation: number; selected: string; reduced: boolean;
  onOpen: (id: string, number: number) => void; onSignOut: () => void;
}) {
  const limit = group === "open" ? openPageSize : closedPageSize;
  const list = usePagedList((after) => api.trustChanges({ state, limit, ...(filter.q ? { q: filter.q } : {}), ...(after ? { after } : {}) }), group + "/" + state + "/" + filter.q, generation);
  return <Section l={l} group={group} list={list} filter={filter} selected={selected} reduced={reduced} onOpen={onOpen} onSignOut={onSignOut} />;
}

function Filters({ l, applied, onApply }: { l: Locale; applied: ListFilter; onApply: (next: ListFilter) => void }) {
  const [draft, setDraft] = useState(applied);
  const submit = (event: FormEvent) => { event.preventDefault(); onApply({ q: draft.q.trim(), state: draft.state }); };
  const reset = () => { setDraft(noFilter); onApply(noFilter); };
  return (
    <form className="m02-filters" aria-label={tl.filters(l)} onSubmit={submit} noValidate>
      <div className="service-field">
        <label htmlFor="m02-filter-q">{tl.searchLabel(l)}</label>
        <input id="m02-filter-q" name="change-id" dir="ltr" value={draft.q} aria-describedby="m02-filter-q-hint" autoComplete="off" spellCheck={false}
          onChange={(e) => { if (searchFits(e.target.value)) setDraft({ ...draft, q: e.target.value }); }} />
        <p className="service-hint" id="m02-filter-q-hint">{tl.searchHint(l)}</p>
      </div>
      <div className="service-field">
        <label htmlFor="m02-filter-state">{qcopy(l, "state")}</label>
        <select id="m02-filter-state" name="state" className="live-select" value={draft.state} onChange={(e) => setDraft({ ...draft, state: e.target.value as ListFilter["state"] })}>
          <option value="">{qcopy(l, "all")}</option>
          {[...trustGroups.open, ...trustGroups.closed].map((s) => <option key={s} value={s}>{ty.state(l, s)}</option>)}
        </select>
      </div>
      <div className="live-buttons m02-filter-buttons">
        <button type="submit" className="live-secondary">{qcopy(l, "apply")}</button>
        <button type="button" className="live-secondary" onClick={reset}>{qcopy(l, "reset")}</button>
      </div>
    </form>
  );
}

export function TrustLists({ l, api, generation, selected, reduced, onOpen, onSignOut }: {
  l: Locale; api: AdminClient; generation: number; selected: string; reduced: boolean; onOpen: (id: string, number: number) => void; onSignOut: () => void;
}) {
  const [filter, setFilter] = useState<ListFilter>(noFilter);
  const [applied, setApplied] = useState(0);
  const open = sectionState(filter, "open");
  const closed = sectionState(filter, "closed");
  const shared = { l, api, filter, generation: generation * 1000 + applied, selected, reduced, onOpen, onSignOut };
  return (
    <div className="m02-lists">
      <Filters l={l} applied={filter} onApply={(next) => { setFilter(next); setApplied((n) => n + 1); }} />
      {open && <PagedSection key="open" group="open" state={open} {...shared} />}
      {closed && <PagedSection key="closed" group="closed" state={closed} {...shared} />}
    </div>
  );
}
