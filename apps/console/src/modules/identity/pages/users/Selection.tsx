import { useEffect, useRef, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import type { UserRow } from "../../../../service/admin";
import { b } from "./bulk-copy";
import { headerState, notLoaded, type Selection } from "./bulk-model";
import { m } from "./match-copy";
import { fill } from "./Stamp";

export function CheckCell({ l, u, self, selected, atLimit, onToggle, keeping = false }: {
  l: Locale; u: UserRow; self: string; selected: Selection; atLimit: boolean; onToggle: (id: string) => void; keeping?: boolean;
}) {
  if (u.user_id === self) {
    return (
      <td role="cell" className="m01-check">
        <span className="m01-self">{b.you(l)}<span className="visually-hidden"> {keeping ? b.selfKept(l) : b.selfReason(l)}</span></span>
      </td>
    );
  }
  const checked = selected.has(u.user_id);
  const blocked = atLimit && !checked;
  return (
    <td role="cell" className="m01-check">
      <label className="m01-hit">
        <input type="checkbox" checked={checked} aria-disabled={blocked || undefined} aria-describedby={blocked && !keeping ? "m01-limit" : undefined}
          data-select={u.user_id} data-keep={keeping ? "true" : undefined} onChange={() => onToggle(u.user_id)} />
        <span className="visually-hidden">{keeping ? b.keepRow(l, u.login || u.user_id) : b.selectRow(l, u.login || u.user_id)}</span>
      </label>
    </td>
  );
}

export function HeaderCheck({ l, selected, loaded, onAll, cell, keeping = false }: { l: Locale; selected: Selection; loaded: string[]; onAll: () => void; cell: boolean; keeping?: boolean }) {
  const box = useRef<HTMLInputElement>(null);
  const state = headerState(selected, loaded);
  useEffect(() => { if (box.current) box.current.indeterminate = state === "mixed"; }, [state]);
  if (loaded.length === 0) return cell ? <th scope="col" className="m01-check" /> : null;
  const input = (
    <label className={cell ? "m01-hit" : "live-check m01-select-loaded"}>
      <input type="checkbox" ref={box} checked={state === "checked"} onChange={onAll} data-select-loaded="true" />
      <span className={cell ? "visually-hidden" : undefined}>{keeping ? b.keepLoaded(l, loaded.length) : b.selectLoaded(l, loaded.length)}</span>
    </label>
  );
  return cell ? <th scope="col" className="m01-check">{input}</th> : input;
}

export function SelectionBar({ l, selected, loaded, limit, atLimit, names, match, onPrepare, onClear, onRemove, onAll }: {
  l: Locale; selected: Selection; loaded: string[]; limit: number; atLimit: boolean; names: Record<string, string>; match?: ReactNode;
  onPrepare: () => void; onClear: () => void; onRemove: (id: string) => void; onAll: () => void;
}) {
  const missing = notLoaded(selected, loaded);
  return (
    <section className="m01-selbar" aria-labelledby="m01-selbar-h">
      <h3 id="m01-selbar-h" className="visually-hidden">{b.barHeading(l)}</h3>
      <HeaderCheck l={l} selected={selected} loaded={loaded} onAll={onAll} cell={false} />
      <p className={"m01-selcount" + (selected.size ? "" : " m01-muted")} tabIndex={-1}>
        {selected.size ? fill(b.selectedRest(l, limit), <strong>{selected.size}</strong>, "{n}") : match ? m.none(l) : b.none(l)}
      </p>
      {selected.size > 0 && (
        <div className="m01-selactions">
          <button type="button" className="live-secondary m01-prepare" onClick={onPrepare}>{b.prepare(l)}</button>
          <button type="button" className="live-secondary" onClick={onClear}>{b.clear(l)}</button>
        </div>
      )}
      {match}
      {atLimit && <p className="m01-limit" id="m01-limit">{b.limit(l, limit)}</p>}
      {missing.length > 0 && (
        <details className="m01-notloaded">
          <summary>{b.notLoaded(l, missing.length)}</summary>
          <ul className="m01-ids">
            {missing.map((id) => (
              <li key={id}>
                <bdi dir="ltr" className={names[id] ? undefined : "live-mono"}>{names[id] || id}</bdi>{" "}
                <button type="button" className="service-button-link" onClick={() => onRemove(id)}>{b.remove(l)}<span className="visually-hidden"> {names[id] || id}</span></button>
              </li>
            ))}
          </ul>
        </details>
      )}
    </section>
  );
}
