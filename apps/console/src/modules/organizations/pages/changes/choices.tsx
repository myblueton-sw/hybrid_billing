import { useEffect, useState, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, type AdminClient, type OrgHistory, type OrgParty, type RelationKind } from "../../../../service/admin";
import { o } from "../organizations/live-copy";
import { usePages, type Fetched } from "../organizations/use-pages";
import type { Paged } from "../organizations/live-model";
import { c } from "./live-copy";
import { validId } from "../organizations/live-model";
import type { Choice } from "./live-model";

export const resultPage = 10;

export function Named({ id, name }: { id: string; name: string }) {
  return name === id ? <bdi dir="ltr" className="o02-id-name live-mono">{id}</bdi> : <><bdi>{name}</bdi><span className="live-sub live-mono"><bdi dir="ltr">{id}</bdi></span></>;
}

export function unitChoices(parties: OrgParty[], kind: RelationKind | "", term: string): Choice[] {
  if (kind && kind !== "hierarchy") return [];
  const q = term.toLowerCase();
  return parties.filter((p) => p.kind === "internal_unit" && p.state === "active" && (p.id.toLowerCase().includes(q) || p.name.toLowerCase().includes(q)))
    .map((p) => ({ id: p.id, name: p.name || p.id })).sort((a, b) => a.name.localeCompare(b.name) || a.id.localeCompare(b.id));
}

export function listed(lead: Choice[], found: Choice[], exclude: string): Choice[] {
  const seen = new Set<string>([exclude]);
  return [...lead, ...found].filter((r) => !seen.has(r.id) && !!seen.add(r.id));
}

export type PickerProps = {
  l: Locale; id: string; label: string; hint: string; empty: string; value: string; name: string; error?: ReactNode; invalid: boolean; turn: string; exclude?: string;
  lead?: (term: string) => Choice[];
  search: (term: string, after: string) => Promise<Fetched<Choice>>;
  onError: (err: unknown) => boolean;
  onChoose: (choice: Choice) => void;
  onSeen: (rows: Choice[]) => void;
  onFocus: (selector: string) => void;
  children?: ReactNode;
};

export function Results({ l, id, value, empty, state, rows, onChoose, onMore, onRetry }: {
  l: Locale; id: string; value: string; empty: string; state: Paged<Choice>; rows: Choice[]; onChoose: (choice: Choice) => void; onMore: () => void; onRetry: () => void;
}) {
  const busy = state.status === "more";
  if (state.status === "loading") return <p role="status" className="live-meta">{c.searching(l)}</p>;
  if (state.status === "error") return <p role="alert">{c.searchFailed(l)} <button type="button" className="service-button-link" onClick={onRetry}>{o.tryAgain(l)}</button></p>;
  return (
    <>
      {rows.length === 0 && !state.next && <p role="status" className="live-note">{empty}</p>}
      {rows.length > 0 && (
        <ul className="o02-results" aria-label={c.results(l)}>
          {rows.map((r) => (
            <li key={r.id}>
              <button type="button" className="o02-result" data-result={id} data-id={r.id} aria-current={r.id === value ? "true" : undefined} onClick={() => onChoose(r)}>
                <Named id={r.id} name={r.name || r.id} />
              </button>
            </li>
          ))}
        </ul>
      )}
      {state.status === "moreFailed" ? (
        <p role="alert">{o.nextFailed(l)} <button type="button" className="service-button-link" onClick={onMore}>{o.tryAgain(l)}</button></p>
      ) : state.next ? (
        <div className="o02-more">
          {state.stopped && <p role="status">{o.treeStopped(l)}</p>}
          <button type="button" className="live-secondary" aria-disabled={busy || undefined} onClick={() => { if (!busy) onMore(); }}>{busy ? o.loadingMore(l) : state.stopped ? o.continueSearching(l) : c.showMore(l)}</button>
        </div>
      ) : null}
    </>
  );
}

export function Picker(p: PickerProps) {
  const { l, id, value } = p;
  const [changing, setChanging] = useState(false);
  const [opened, setOpened] = useState(0);
  const [draft, setDraft] = useState("");
  const [term, setTerm] = useState("");
  const open = !value || changing;
  const pager = usePages<Choice>(open ? `${p.turn}|${opened}|${term}` : null, (after) => p.search(term, after), p.onError);
  const rows = listed(p.lead ? p.lead(term) : [], pager.rows, p.exclude ?? "");
  useEffect(() => { if (pager.rows.length > 0) p.onSeen(pager.rows); }, [pager.rows]); // eslint-disable-line react-hooks/exhaustive-deps
  const run = () => setTerm(draft.trim());
  const described = [id + "-hint", p.invalid ? id + "-error" : ""].filter(Boolean).join(" ");
  return (
    <div className="service-field o02-picker">
      {open ? <label htmlFor={id} id={id + "-label"}>{p.label}</label> : <span id={id + "-label"} className="o02-field-label">{p.label}</span>}
      <p className="service-hint" id={id + "-hint"}>{p.hint}</p>
      {p.error}
      {value && (
        <p className="o02-chosen">
          <span className="o02-chosen-name" id={id + "-name"}><Named id={value} name={p.name || value} /></span>
          {open
            ? <button type="button" id={id + "-cancel"} className="service-button-link o02-id-toggle" aria-labelledby={`${id}-cancel ${id}-label`} onClick={() => { setChanging(false); p.onFocus("#" + id); }}>{o.cancel(l)}</button>
            : <button type="button" id={id} className="service-button-link o02-id-toggle" aria-labelledby={`${id} ${id}-label ${id}-name`} aria-describedby={described} onClick={() => { setOpened((n) => n + 1); setChanging(true); p.onFocus("#" + id); }}>{o.change(l)}</button>}
        </p>
      )}
      {open && (
        <>
          <div className="o02-search">
            <input id={id} type="search" dir="auto" value={draft} maxLength={64} aria-invalid={p.invalid || undefined} aria-describedby={described} onChange={(e) => setDraft(e.target.value)}
              onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); run(); } }} />
            <button type="button" className="live-secondary" onClick={run}>{c.search(l)}</button>
          </div>
          <Results l={l} id={id} value={value} empty={p.empty} state={pager} rows={rows} onChoose={(r) => { p.onChoose(r); setChanging(false); p.onFocus("#" + id); }} onMore={pager.more} onRetry={pager.retry} />
        </>
      )}
      {p.children}
    </div>
  );
}

export type Present = { key: string; status: "loading" | "ready" | "failed"; history: OrgHistory[] | null };

export function usePresent(api: AdminClient, member: string, kind: RelationKind | "", revision: number | null, onError: (err: unknown) => boolean) {
  const id = member.trim();
  const key = kind && revision !== null && validId(id) ? `${revision}|${kind}|${id}` : "";
  const [state, setState] = useState<Present>({ key: "", status: "loading", history: null });
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    if (!key || !kind || revision === null) return;
    let live = true;
    setState({ key, status: "loading", history: null });
    api.organizations({ kind, member: [id], revision })
      .then((page) => { if (live) setState({ key, status: "ready", history: page.members[0] ? page.members[0].history : null }); })
      .catch((err) => { if (live && !(err instanceof AdminError && onError(err))) setState({ key, status: "failed", history: null }); });
    return () => { live = false; };
  }, [key, attempt]); // eslint-disable-line react-hooks/exhaustive-deps
  const current: Present = !key ? { key: "", status: "ready", history: null } : state.key === key ? state : { key, status: "loading", history: null };
  return { ...current, valid: !!key, retry: () => setAttempt((n) => n + 1) };
}
