import { useEffect, useId, useRef, useState, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import type { OrgMember, OrgParty, OrgTreeNode } from "../../../../service/admin";
import { o } from "./live-copy";
import { label, type Paged } from "./live-model";
import { fill } from "../changes/live-model";
import { usePages, type Fetched } from "./use-pages";

export const parentList = "parent:";

export type TreeContext = {
  l: Locale; date: string; turn: string; collapse: number; selected: string;
  read: (list: string, after: string) => Promise<Fetched<OrgTreeNode>>;
  onError: (err: unknown) => boolean;
  onSelect: (node: OrgTreeNode) => void;
};

export function kindName(l: Locale, kind: string): string {
  return kind === "legal_entity" ? o.legalEntity(l) : kind === "internal_unit" ? o.internalUnit(l) : kind === "account" ? o.account(l) : o.unknown(l);
}

export function unplaced(m: OrgMember, parties: OrgParty[]): OrgTreeNode {
  const p = parties.find((x) => x.id === m.member);
  return { id: m.member, name: m.name, kind: p ? p.kind : "account", state: p ? p.state : "", source: "none", children: "unchecked" };
}

export const planMember = (party: OrgParty["kind"] | undefined, member: boolean) => (party ? party === "internal_unit" : member);

export function distinct(rows: OrgTreeNode[]): OrgTreeNode[] {
  const seen = new Set<string>();
  return rows.filter((n) => !seen.has(n.id) && !!seen.add(n.id));
}

export const canOpen = (node: OrgTreeNode, path: string[]) => node.children !== "none" && !path.includes(node.id);

function Row({ c, node, depth, path, first }: { c: TreeContext; node: OrgTreeNode; depth: number; path: string[]; first: boolean }) {
  const { l } = c;
  const openable = canOpen(node, path);
  const [open, setOpen] = useState(first && openable);
  const seen = useRef(c.collapse);
  useEffect(() => {
    if (seen.current === c.collapse) return;
    seen.current = c.collapse;
    setOpen(false);
  }, [c.collapse]);
  const under = useId();
  const name = label(node.name, node.id);
  return (
    <li>
      <div className={"o01-row" + (c.selected === node.id ? " live-selected" : "")} style={{ ["--o01-indent" as string]: `${Math.min(depth, 4) * 24}px` }}>
        {openable ? (
          <button type="button" className="o01-toggle" aria-expanded={open} aria-controls={open ? under : undefined} aria-label={open ? o.collapse(l, name) : o.expand(l, name)} onClick={() => setOpen((v) => !v)}>
            <span aria-hidden="true">{open ? "▼" : "▶"}</span>
          </button>
        ) : <span className="o01-toggle-space" aria-hidden="true" />}
        <button type="button" className={"live-open o01-name" + (node.kind === "legal_entity" ? " o01-strong" : "")} data-id={node.id} aria-current={c.selected === node.id ? "true" : undefined} onClick={() => c.onSelect(node)}>
          <bdi>{name}</bdi>
        </button>
        <span className="o01-kind">
          {kindName(l, node.kind)}{node.state && node.state !== "active" ? ` · ${o.stateSuspended(l)}` : ""}{node.source === "registry" ? ` · ${o.registryParent(l)}` : ""}{path.includes(node.id) ? ` · ${o.shownAbove(l)}` : ""}
        </span>
        <span className="o01-id live-mono"><bdi dir="ltr">{node.id}</bdi></span>
      </div>
      {open && openable && (
        <div id={under} className="o01-under" style={{ ["--o01-indent" as string]: `${Math.min(depth + 1, 4) * 24}px` }}>
          <Branch c={c} list={parentList + node.id} depth={depth + 1} path={[...path, node.id]} name={name} />
        </div>
      )}
    </li>
  );
}

export type BranchProps = { c: TreeContext; list: string; depth: number; path: string[]; name?: string; labelled?: boolean; empty?: ReactNode; onReady?: () => void; onRows?: (shown: boolean) => void };

export function BranchView({ c, depth, path, name, labelled, empty, state, onMore, onRetry }: BranchProps & { state: Paged<OrgTreeNode>; onMore: () => void; onRetry: () => void }) {
  const { l } = c;
  const named = (text: string) => fill(text, { n: <bdi>{name}</bdi> });
  if (state.status === "loading") return <p role="status" className="live-meta o01-branch-note">{name ? named(o.loadingUnder(l, "{n}")) : o.loadingMore(l)}</p>;
  if (state.status === "error") {
    return (
      <p role="alert" className="o01-branch-note">
        {name ? named(o.underFailed(l, "{n}")) : o.listFailed(l)} <button type="button" className="service-button-link" onClick={onRetry}>{o.tryAgain(l)}</button>
      </p>
    );
  }
  const rows = distinct(state.rows);
  const busy = state.status === "more";
  const press = () => { if (!busy) onMore(); };
  return (
    <>
      {rows.length === 0 && !state.next && (empty === undefined ? <p role="status" className="live-note o01-branch-note">{o.nothingUnder(l, c.date)}</p> : empty)}
      {rows.length > 0 && (
        <ul className={depth === 0 ? "o01-tree" : "o01-branch"} aria-label={depth === 0 && labelled ? o.treeTitle(l) : name ? o.underLabel(l, name) : undefined}>
          {rows.map((n) => <Row key={n.id} c={c} node={n} depth={depth} path={path} first={depth === 0 && !!labelled && rows.length === 1 && !state.next} />)}
        </ul>
      )}
      {state.status === "moreFailed" ? (
        <div className="o01-more o01-branch-note" role="alert">
          <p>{o.nextFailed(l)}</p>
          <button type="button" className="live-secondary o01-more-button" onClick={onMore}>{o.tryAgain(l)}</button>
        </div>
      ) : state.next ? (
        <div className="o01-more o01-branch-note">
          {state.stopped && <p role="status">{o.treeStopped(l)}</p>}
          <button type="button" className="live-secondary o01-more-button" aria-disabled={busy || undefined} aria-label={name && !busy && !state.stopped ? o.moreUnder(l, name) : undefined} onClick={press}>
            {busy ? o.loadingMore(l) : state.stopped ? o.continueSearching(l) : o.loadMore(l)}
          </button>
        </div>
      ) : null}
    </>
  );
}

export function Branch(p: BranchProps) {
  const { c, list, onReady, onRows } = p;
  const pager = usePages<OrgTreeNode>(`${c.turn}|${list}`, (after) => c.read(list, after), c.onError);
  const settled = pager.current && pager.status !== "loading";
  const shown = !settled || pager.status === "error" || pager.rows.length > 0 || !!pager.next;
  useEffect(() => { onRows?.(shown); }, [shown]); // eslint-disable-line react-hooks/exhaustive-deps
  const ready = pager.current && pager.status === "ready";
  useEffect(() => { if (ready) onReady?.(); }, [ready, pager.key]); // eslint-disable-line react-hooks/exhaustive-deps
  return <BranchView {...p} state={pager} onMore={pager.more} onRetry={pager.retry} />;
}

export function TreeGroup({ c, list, id, title, note, onShown }: { c: TreeContext; list: string; id: string; title: string; note?: string; onShown?: (shown: boolean) => void }) {
  const [shown, setShown] = useState(false);
  const report = (now: boolean) => { setShown(now); onShown?.(now); };
  return (
    <section className="o01-unattributed" aria-labelledby={id} hidden={!shown}>
      <h3 id={id}>{title}</h3>
      {note && <p className="live-note">{note}</p>}
      <Branch c={c} list={list} depth={0} path={[]} empty={null} onRows={report} />
    </section>
  );
}
