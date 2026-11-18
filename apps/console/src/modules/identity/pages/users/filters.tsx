import { useEffect, useState, type FormEvent } from "react";
import type { Locale } from "../../../../shared/types";
import close from "../../../../../../../packages/design-system/assets/icons/close.svg";
import chevron from "../../../../../../../packages/design-system/assets/icons/chevron-down.svg";
import { c } from "../../../../service/copy";
import { w } from "./live-copy";
import { accountName, roleName } from "./live-model";
import { p } from "./period-copy";
import { fc, type AuthMethods } from "./filter-copy";
import { appliedKeys, choices, cursorUser, validSearch, type FilterKey, type UserFilters } from "./filter-model";
import { fill } from "./Stamp";

export function optionName(l: Locale, key: FilterKey, value: string, days: number): string {
  if (key === "st") return accountName(l, value);
  if (key === "role") return roleName(l, value);
  if (key === "ms") return ({ active: fc.msActive(l), suspended: fc.msSuspended(l), ended: fc.membershipEnded(l), none: w.noMembership(l) } as Record<string, string>)[value];
  if (key === "per") return ({ open: fc.periodOpen(l), bounded: fc.periodBounded(l, days), ends_soon: fc.periodSoon(l, days), ended: p.ended(l) } as Record<string, string>)[value];
  return ({ password_set: fc.passwordSet(l), set_link: fc.setLink(l), reset_link: fc.resetLink(l), not_set: fc.notSet(l) } as Record<string, string>)[value];
}

export function fieldName(l: Locale, key: FilterKey): string {
  return ({ q: fc.searchChip(l), st: w.accountStatus(l), ms: fc.membership(l), role: w.workspaceRole(l), per: p.rolePeriod(l), si: w.signInSetup(l), adm: fc.holder(l) } as Record<FilterKey, string>)[key];
}

export function chipText(l: Locale, view: string, key: FilterKey, f: UserFilters, days: number): string {
  if (key === "adm") return fc.holder(l);
  if (key === "q") return fc.chip(fc.searchChip(l), f.q);
  const text = fc.chip(fieldName(l, key), optionName(l, key, f[key], days));
  return view === "membership" && (key === "role" || key === "per") ? `${text} ${fc.withMembership(l)}` : text;
}

export function ScopeLine({ l, workspace }: { l: Locale; workspace: string }) {
  return (
    <p className="m01-scopeline">
      <span className="m01-scopelabel">{fc.scopeLabel(l)}</span> <bdi dir="ltr" className="live-mono m01-scopevalue">{workspace}</bdi>
    </p>
  );
}

function SearchForm({ l, value, onSearch }: { l: Locale; value: string; onSearch: (q: string) => void }) {
  const [draft, setDraft] = useState(value);
  const [error, setError] = useState(false);
  useEffect(() => { setDraft(value); setError(false); }, [value]);
  const submit = (e: FormEvent) => {
    e.preventDefault();
    const q = draft.trim();
    if (!validSearch(q)) { setError(true); return; }
    setError(false);
    onSearch(q);
  };
  return (
    <form role="search" className="service-field m01-search" onSubmit={submit} noValidate>
      <label htmlFor="m01-q">{fc.search(l)}</label>
      {error && <p className="service-field-error" id="m01-q-error">{fc.searchTooLong(l)}</p>}
      <div className="m01-search-row">
        <input id="m01-q" type="search" dir="auto" autoComplete="off" value={draft} aria-invalid={error || undefined} aria-describedby={error ? "m01-q-error" : undefined}
          onChange={(e) => setDraft(e.target.value)} />
        <button type="submit" className="live-secondary">{fc.searchButton(l)}</button>
      </div>
    </form>
  );
}

function Choice({ l, k, value, days, onChange }: { l: Locale; k: Exclude<FilterKey, "q" | "adm">; value: string; days: number; onChange: (v: string) => void }) {
  return (
    <div className={"service-field m01-field-" + k}>
      <label htmlFor={"m01-" + k}>{fieldName(l, k)}</label>
      <select id={"m01-" + k} className="live-select" value={value} onChange={(e) => onChange(e.target.value)}>
        <option value="">{fc.all(l)}</option>
        {choices[k].map((v) => <option key={v} value={v}>{optionName(l, k, v, days)}</option>)}
      </select>
    </div>
  );
}

export function Chips({ l, view, filters, days, onRemove, onClearAll }: {
  l: Locale; view: string; filters: UserFilters; days: number; onRemove: (key: FilterKey) => void; onClearAll: () => void;
}) {
  const keys = appliedKeys(filters);
  const fixed = view === "invitations";
  if (keys.length === 0 && !fixed) return null;
  return (
    <div className="m01-applied">
      <span className="m01-applied-lead">{fc.appliedFilters(l)}</span>
      <ul className="m01-chips">
        {fixed && <li><span className="m01-chip m01-chip-fixed">{w.invitationsTitle(l)}</span></li>}
        {keys.map((k) => {
          const text = chipText(l, view, k, filters, days);
          return (
            <li key={k}>
              <button type="button" id={"m01-chip-" + k} className="m01-chip" aria-label={fc.removeFilter(l, text)} onClick={() => onRemove(k)}>
                <span>{k === "q" ? fill(text, <bdi dir="auto">{filters.q}</bdi>, filters.q) : text}</span>
                <img className="icon" src={close} alt="" width={16} height={16} />
              </button>
            </li>
          );
        })}
        {keys.length > 0 && <li className="m01-clear-li"><button type="button" id="m01-clear-all" className="service-button-link m01-clear-all" onClick={onClearAll}>{fc.clearAll(l)}</button></li>}
      </ul>
    </div>
  );
}

export function FilterBand({ l, view, wide, open, filters, days, auth, onToggle, onChange, onRemove, onClearAll }: {
  l: Locale; view: string; wide: boolean; open: boolean; filters: UserFilters; days: number; auth: AuthMethods;
  onToggle: () => void; onChange: (next: UserFilters) => void; onRemove: (key: FilterKey) => void; onClearAll: () => void;
}) {
  const applied = appliedKeys(filters).length;
  const set = (change: Partial<UserFilters>) => onChange({ ...filters, ...change });
  return (
    <section className="m01-filters" aria-labelledby="m01-filters-title">
      <h2 id="m01-filters-title" className={wide ? undefined : "visually-hidden"}>{fc.filters(l)}</h2>
      {!wide && (
        <button type="button" id="m01-filters-toggle" className="m01-filters-toggle" aria-expanded={open} aria-controls="m01-filter-body" onClick={onToggle}>
          <img className="icon" src={chevron} alt="" width={16} height={16} />
          <span>{applied > 0 ? fc.filtersApplied(l, applied) : fc.filters(l)}</span>
        </button>
      )}
      <div id="m01-filter-body" className="m01-filter-body" hidden={!wide && !open}>
        <div className="m01-filter-fields">
          <SearchForm l={l} value={filters.q} onSearch={(q) => set({ q })} />
          <Choice l={l} k="st" value={filters.st} days={days} onChange={(st) => set({ st })} />
          <Choice l={l} k="ms" value={filters.ms} days={days} onChange={(ms) => set({ ms })} />
          <Choice l={l} k="role" value={filters.role} days={days} onChange={(role) => set({ role })} />
          <Choice l={l} k="per" value={filters.per} days={days} onChange={(per) => set({ per })} />
          <Choice l={l} k="si" value={filters.si} days={days} onChange={(si) => set({ si })} />
          <label className="live-check m01-field-adm"><input id="m01-adm" type="checkbox" checked={filters.adm} onChange={(e) => set({ adm: e.target.checked })} />{fc.holder(l)}</label>
        </div>
        {fc.method(l, auth) && <p className="m01-method"><span className="m01-method-label">{fc.methodLabel(l)}</span> <span className="live-quiet">{fc.method(l, auth)}</span></p>}
      </div>
      <Chips l={l} view={view} filters={filters} days={days} onRemove={onRemove} onClearAll={onClearAll} />
    </section>
  );
}

export function SearchedUpTo({ l, next, login }: { l: Locale; next: string; login: string }) {
  const at = cursorUser(next);
  if (at.departed || !login) return <>{fill(fc.searchedUser(l), <bdi dir="ltr" className="live-mono">{at.user}</bdi>, "{id}")}</>;
  return <>{fill(fc.searchedUpTo(l), <bdi dir="ltr">{login}</bdi>, "{login}")}</>;
}

export type Paging = "idle" | "more" | "failed" | "restart";

export function PagingArea({ l, next, nextLogin, stopped, paging, changed, count, compact, onMore, onReload, onRestart }: {
  l: Locale; next: string; nextLogin: string; stopped: boolean; paging: Paging; changed: boolean; count: string; compact: boolean;
  onMore: () => void; onReload: () => void; onRestart: () => void;
}) {
  const busy = paging === "more";
  const press = () => { if (!busy) onMore(); };
  return (
    <>
      {changed && (
        <div className="m01-listnote" role="status">
          <p>{fc.changed(l).map((s, i) => <span key={i} className={i === 0 ? "live-strong" : undefined}>{i ? " " : ""}{s}</span>)}</p>
          <button type="button" className="live-secondary m01-more-button" onClick={onReload}>{w.reload(l)}</button>
        </div>
      )}
      {paging === "restart" ? (
        <div className="m01-restart m01-more" role="alert">
          <p>{fc.restart(l).map((s, i) => <span key={i} className={i === 0 ? "live-strong" : undefined}>{i ? " " : ""}{s}</span>)}</p>
          <button type="button" id="m01-start-again" className="live-secondary m01-more-button" onClick={onRestart}>{fc.startAgain(l)}</button>
        </div>
      ) : paging === "failed" ? (
        <div className="m01-more" role="alert">
          <p>{fc.nextFailed(l)}</p>
          <button type="button" className="live-secondary m01-more-button" onClick={onMore}>{c.tryAgain(l)}</button>
        </div>
      ) : next ? (
        <div className="m01-more">
          {stopped && <p role="status"><SearchedUpTo l={l} next={next} login={nextLogin} /></p>}
          <button type="button" className="live-secondary m01-more-button live-more" aria-disabled={busy || undefined} onClick={press}>
            {busy ? fc.loadingMore(l) : stopped ? fc.continueSearching(l) : w.more(l)}
          </button>
          {compact && <span className="live-meta m01-more-count">{count}</span>}
        </div>
      ) : null}
    </>
  );
}
