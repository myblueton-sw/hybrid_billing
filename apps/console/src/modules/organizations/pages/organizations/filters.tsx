import { useEffect, useState, type FormEvent, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import type { OrgParty, RelationKind } from "../../../../service/admin";
import close from "../../../../../../../packages/design-system/assets/icons/close.svg";
import chevron from "../../../../../../../packages/design-system/assets/icons/chevron-down.svg";
import { o } from "./live-copy";
import { appliedKeys, label, relationKinds, validSearch, type Count, type FilterKey, type Filters } from "./live-model";

export function entityName(entities: OrgParty[], id: string, l: Locale): string {
  const p = entities.find((e) => e.id === id);
  if (!p) return id;
  return label(p.name, p.id) + (p.state !== "active" ? ` · ${o.stateSuspended(l)}` : "");
}

export function chipText(l: Locale, view: string, key: FilterKey, f: Filters, entities: OrgParty[]): string {
  if (view === "groups") return key === "entity" ? o.membersChip(l, entityName(entities, f.entity, l)) : o.groupsChip(l, f.q);
  if (key === "entity") return o.chip(o.entity(l), entityName(entities, f.entity, l));
  if (key === "rk") return o.chip(o.relKind(l), o.kind(l, f.rk));
  if (key === "ua") return o.onlyUnattributed(l);
  return o.chip(o.searchButton(l), f.q);
}

export function countText(l: Locale, c: Count, groups = false): string {
  if (c.n === 0) return "";
  if (groups) return c.more ? o.groupsShownMore(l, c.n) : o.groupsShownAll(l, c.n);
  return c.more ? o.shownMore(l, c.n) : o.shownAll(l, c.n);
}

export function SearchForm({ l, id, labelText, value, onSearch }: { l: Locale; id: string; labelText: string; value: string; onSearch: (q: string) => void }) {
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
    <form role="search" className="service-field o01-search" onSubmit={submit} noValidate>
      <label htmlFor={id}>{labelText}</label>
      {error && <p className="service-field-error" id={id + "-error"}>{o.searchTooLong(l)}</p>}
      <div className="o01-search-row">
        <input id={id} type="search" dir="auto" maxLength={64} autoComplete="off" value={draft} aria-invalid={error || undefined} aria-describedby={error ? id + "-error" : undefined} onChange={(e) => setDraft(e.target.value)} />
        <button type="submit" className="live-secondary">{o.searchButton(l)}</button>
      </div>
    </form>
  );
}

function EntitySelect({ l, value, entities, onChange }: { l: Locale; value: string; entities: OrgParty[]; onChange: (id: string) => void }) {
  return (
    <div className="service-field o01-field-entity">
      <label htmlFor="o01-entity">{o.entity(l)}</label>
      <select id="o01-entity" className="live-select" value={value} onChange={(e) => onChange(e.target.value)}>
        <option value="">{o.allEntities(l)}</option>
        {value && !entities.some((e) => e.id === value) && <option value={value}>{value}</option>}
        {entities.map((e) => <option key={e.id} value={e.id}>{entityName(entities, e.id, l)}</option>)}
      </select>
    </div>
  );
}

export function KindSelect({ l, value, onChange }: { l: Locale; value: RelationKind; onChange: (k: RelationKind) => void }) {
  return (
    <div className="service-field o01-field-kind">
      <label htmlFor="o01-kind">{o.relKind(l)}</label>
      <select id="o01-kind" className="live-select" value={value} onChange={(e) => onChange(e.target.value as RelationKind)}>
        {relationKinds.map((k) => <option key={k} value={k}>{o.kind(l, k)}</option>)}
      </select>
    </div>
  );
}

function Unattributed({ l, value, onChange }: { l: Locale; value: boolean; onChange: (v: boolean) => void }) {
  return <label className="live-check o01-field-ua"><input id="o01-ua" type="checkbox" checked={value} onChange={(e) => onChange(e.target.checked)} />{o.onlyUnattributed(l)}</label>;
}

function Provider({ l }: { l: Locale }) {
  return <p className="o01-provider"><span className="o01-provider-label">{o.provider(l)}</span> <span className="live-quiet">{o.providerNote(l)}</span></p>;
}

export function Chips({ l, view, filters, entities, onRemove, onClearAll }: {
  l: Locale; view: string; filters: Filters; entities: OrgParty[]; onRemove: (key: FilterKey) => void; onClearAll: () => void;
}) {
  const keys = appliedKeys(view, filters);
  if (keys.length === 0) return null;
  return (
    <div className="o01-applied">
      <span className="o01-applied-lead">{o.appliedFilters(l)}</span>
      <ul className="o01-chips">
        {keys.map((k) => {
          const text = chipText(l, view, k, filters, entities);
          return (
            <li key={k}>
              <button type="button" id={"o01-chip-" + k} className="o01-chip" aria-label={o.removeFilter(l, text)} onClick={() => onRemove(k)}>
                <span>{k === "entity" || k === "q" ? <bdi>{text}</bdi> : text}</span>
                <img className="icon" src={close} alt="" width={16} height={16} />
              </button>
            </li>
          );
        })}
      </ul>
      <button type="button" id="o01-clear-all" className="service-button-link" onClick={onClearAll}>{o.clearAll(l)}</button>
    </div>
  );
}

export type BandProps = {
  l: Locale; view: string; wide: boolean; open: boolean; onToggle: () => void; filters: Filters; entities: OrgParty[];
  onChange: (next: Filters) => void; onRemove: (key: FilterKey) => void; onClearAll: () => void;
  groupKind: RelationKind; onGroupKind: (k: RelationKind) => void; history?: ReactNode;
};

export function FilterBand(p: BandProps) {
  const { l, view, wide, open, filters, entities, onChange } = p;
  if (view === "tree") return <section className="o01-filters o01-filters-note" aria-label={o.filters(l)}><p className="live-note">{o.treeFilters(l)}</p></section>;
  const applied = appliedKeys(view, filters).length;
  const set = (change: Partial<Filters>) => onChange({ ...filters, ...change });
  const entity = <EntitySelect l={l} value={filters.entity} entities={entities} onChange={(entity) => set({ entity })} />;
  const ua = <Unattributed l={l} value={filters.ua} onChange={(v) => set({ ua: v })} />;
  const fields = view === "accounts" ? (
    <>
      <div className="o01-filter-fields">
        <SearchForm l={l} id="o01-q" labelText={o.search(l)} value={filters.q} onSearch={(q) => set({ q })} />
        {entity}
        <KindSelect l={l} value={filters.rk} onChange={(rk) => set({ rk })} />
        {ua}
      </div>
      <Provider l={l} />
    </>
  ) : view === "groups" ? (
    <>
      <div className="o01-filter-fields">
        <fieldset className="live-roles o01-kinds">
          <legend>{o.relationship(l)}</legend>
          {(["view_group", "allocation_pool"] as RelationKind[]).map((k) => (
            <label key={k}><input type="radio" name="o01-group-kind" value={k} checked={p.groupKind === k} onChange={() => p.onGroupKind(k)} />{o.kind(l, k)}</label>
          ))}
        </fieldset>
        <SearchForm l={l} id="o01-q" labelText={o.searchGroups(l)} value={filters.q} onSearch={(q) => set({ q })} />
        <fieldset className="o01-member-filters">
          <legend>{o.filterMembers(l)}</legend>
          <div className="o01-filter-fields">{entity}</div>
        </fieldset>
      </div>
      <Provider l={l} />
    </>
  ) : (
    <div className="o01-filter-fields">
      <KindSelect l={l} value={filters.rk} onChange={(rk) => set({ rk })} />
      {p.history}
    </div>
  );
  return (
    <section className="o01-filters" aria-labelledby={wide ? "o01-filters-title" : "o01-filters-toggle"}>
      {wide ? <h2 id="o01-filters-title">{o.filters(l)}</h2> : (
        <button type="button" id="o01-filters-toggle" className="o01-filters-toggle" aria-expanded={open} aria-controls="o01-filter-body" onClick={p.onToggle}>
          <img className="icon" src={chevron} alt="" width={16} height={16} />
          <span>{applied > 0 ? o.filtersApplied(l, applied) : o.filters(l)}</span>
        </button>
      )}
      <div id="o01-filter-body" className="o01-filter-body" hidden={!wide && !open}>{fields}</div>
      <Chips l={l} view={view} filters={filters} entities={entities} onRemove={p.onRemove} onClearAll={p.onClearAll} />
    </section>
  );
}

export function SearchedThrough({ l, id }: { l: Locale; id: string }) {
  const [before, after] = o.searchedThrough(l).split("{id}");
  return <>{before}<bdi dir="ltr" className="live-mono">{id}</bdi>{after}</>;
}

export function MoreBar({ l, next, stopped, status, count, compact, onMore }: {
  l: Locale; next: string; stopped: boolean; status: string; count: string; compact: boolean; onMore: () => void;
}) {
  const busy = status === "more";
  if (status === "moreFailed") {
    return (
      <div className="o01-more" role="alert">
        <p>{o.nextFailed(l)}</p>
        <button type="button" className="live-secondary o01-more-button" onClick={onMore}>{o.tryAgain(l)}</button>
      </div>
    );
  }
  if (!next) return null;
  const press = () => { if (!busy) onMore(); };
  return (
    <div className="o01-more">
      {stopped ? (
        <>
          <p role="status"><SearchedThrough l={l} id={next} /></p>
          <button type="button" className="live-secondary o01-more-button" aria-disabled={busy || undefined} onClick={press}>{busy ? o.loadingMore(l) : o.continueSearching(l)}</button>
        </>
      ) : (
        <button type="button" className="live-secondary o01-more-button" aria-disabled={busy || undefined} onClick={press}>{busy ? o.loadingMore(l) : o.loadMore(l)}</button>
      )}
      {compact && <span className="live-meta o01-more-count">{count}</span>}
    </div>
  );
}

export function FilteredEmpty({ l, date, onClear }: { l: Locale; date: string; onClear: () => void }) {
  return (
    <div className="live-empty o01-filtered-empty">
      <p>{o.filteredEmpty(l, date)}</p>
      <button type="button" className="service-button-link" onClick={onClear}>{o.clearAll(l)}</button>
    </div>
  );
}
