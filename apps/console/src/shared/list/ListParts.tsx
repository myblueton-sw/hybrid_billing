import { useEffect, useRef, type ReactNode, type RefObject } from "react";
import type { Locale } from "../types";
import type { AccessSetting } from "../../service/admin";
import { counted } from "../plural";
import { lc } from "./list-copy";
import { lowered, nextFocusAfterRemoval, pageSizeOptions } from "./list-model";

export type Chip = { key: string; label: string; value: string; mono?: boolean };

export function ListSummary({ l, count, exact, reference, summaryRef, countInHeading = false }: {
  l: Locale; count: string; exact: boolean; reference: ReactNode; summaryRef?: RefObject<HTMLDivElement | null>; countInHeading?: boolean;
}) {
  const line = `${(exact ? lc.exact(l) : lc.lower(l))}: ${count}`;
  return (
    <>
      <div className="list-summary" tabIndex={-1} ref={summaryRef} data-count={count} data-certainty={exact ? "exact" : "lower_bound"}>
        {!countInHeading && <p>{line}</p>}
        <p>{lc.reference(l)}: {reference}</p>
        <p className="visually-hidden" role="status" aria-live="polite">{line}</p>
      </div>
    </>
  );
}

export function ListPager({ l, title, first, hasNext, rows, size, onFirst, onPrevious, onNext, previousUnavailable = false }: {
  l: Locale; title: string; first: boolean; hasNext: boolean; rows: number; previousUnavailable?: boolean;
  size?: { requested: number; current: number; limit: number | null; onChange: (n: number) => void; id: string };
  onFirst: () => void; onPrevious: () => void; onNext: () => void;
}) {
  const options = size ? pageSizeOptions(size.limit, size.current) : [];
  const select = options.length > 1;
  const note = !!size && lowered(size.requested, size.limit);
  const nav = !(first && !hasNext);
  if (!nav && !select && !note) return null;
  return (
    <div className="list-pager">
      <div className="list-pager-size">
        {select && size && (
          <div className="list-pager-select">
            <label htmlFor={size.id}>{lc.size(l)}</label>
            <select id={size.id} className="live-select" value={size.current} onChange={(e) => size.onChange(Number(e.target.value))}>
              {options.map((n) => <option key={n} value={n}>{n}</option>)}
            </select>
          </div>
        )}
        <p className="list-pager-rows">{lc.onPage(l, rows)}</p>
        {note && size && size.limit !== null && <p className="list-pager-note">{lc.lowered(l, size.limit)}</p>}
      </div>
      {nav && (
        <nav className="list-pager-nav" aria-label={lc.pages(l, title)}>
          <button type="button" className="live-secondary" disabled={first} onClick={onFirst}>{lc.first(l)}</button>
          <button type="button" className="live-secondary" disabled={first || previousUnavailable} onClick={onPrevious}>{lc.previous(l)}</button>
          <button type="button" className="live-secondary" disabled={!hasNext} onClick={onNext}>{lc.next(l)}</button>
        </nav>
      )}
    </div>
  );
}

export function AppliedFilters({ l, id, chips, onRemove, onClear, fallback }: {
  l: Locale; id: string; chips: Chip[]; onRemove: (key: string) => void; onClear: () => void; fallback: () => HTMLElement | null;
}) {
  const list = useRef<HTMLUListElement>(null);
  const pending = useRef<string | null>(null);
  const keys = chips.map((c) => c.key).join("|");
  useEffect(() => {
    const target = pending.current;
    if (target === null) return;
    pending.current = null;
    const button = target ? list.current?.querySelector<HTMLButtonElement>(`[data-chip="${CSS.escape(target)}"] .list-chip-remove`) : null;
    (button ?? fallback())?.focus();
  }, [keys, fallback]);
  if (chips.length === 0) return null;
  const remove = (key: string) => { pending.current = nextFocusAfterRemoval(chips.map((c) => c.key), key) ?? ""; onRemove(key); };
  return (
    <section className="list-applied" aria-labelledby={id}>
      <p className="list-applied-lead" id={id}>{lc.applied(l)}</p>
      <ul ref={list}>
        {chips.map((c) => (
          <li key={c.key} className="list-chip" data-chip={c.key}>
            <span className="list-chip-text">{c.label}: <strong>{c.mono ? <bdi dir="ltr" className="live-mono">{c.value}</bdi> : <bdi>{c.value}</bdi>}</strong></span>
            <button type="button" className="list-chip-remove" aria-label={`${lc.remove(l)} ${c.label}: ${c.value}`} onClick={() => remove(c.key)}>
              <span aria-hidden="true">×</span>
            </button>
          </li>
        ))}
        <li className="list-applied-clear">
          <button type="button" className="live-secondary" onClick={() => { pending.current = ""; onClear(); }}>{lc.clear(l)}</button>
        </li>
      </ul>
    </section>
  );
}

export function ListSettings({ l, rows }: { l: Locale; rows: AccessSetting[] }) {
  if (rows.length === 0) return null;
  return (
    <details className="live-ended list-settings">
      <summary>{lc.settingsTitle(l)}</summary>
      <p className="live-note">{lc.settingsLead(l)}</p>
      <div className="security-table-scroll live-table">
        <table role="table">
          <thead role="rowgroup"><tr role="row">
            <th scope="col" role="columnheader">{lc.colSetting(l)}</th>
            <th scope="col" role="columnheader">{lc.colValue(l)}</th>
            <th scope="col" role="columnheader">{lc.colRange(l)}</th>
            <th scope="col" role="columnheader">{lc.colSource(l)}</th>
          </tr></thead>
          <tbody role="rowgroup">
            {rows.map((x) => {
              const unit = lc.unit[x.unit];
              return (
                <tr key={x.name} role="row" data-setting={x.name}>
                  <td role="cell" className="live-user">{lc.name[x.name]?.(l) ?? x.name}<span className="live-sub live-mono"><bdi dir="ltr">{x.name}</bdi></span></td>
                  <td role="cell" data-label={lc.colValue(l)} className={x.source === "start_flag" ? "live-strong" : ""}>{unit ? counted(l, x.value, unit) : `${x.value} ${x.unit}`}</td>
                  <td role="cell" data-label={lc.colRange(l)}>{lc.range(l, x.min, x.max, x.default)}</td>
                  <td role="cell" data-label={lc.colSource(l)}>{lc.source(l, x.source)}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </details>
  );
}
